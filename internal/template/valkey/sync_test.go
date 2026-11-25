package valkey

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/repository"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

func TestSyncCheckoutRejectsIncompleteOrInvalidCandidateBeforeStorage(t *testing.T) {
	for _, name := range []string{"missing file", "unknown second step"} {
		t.Run(name, func(t *testing.T) {
			second := fixtureTemplate("SECOND")
			if name == "unknown second step" {
				second += "    - type: nonexistent\n"
			}
			directory := writeSyncCheckout(t, map[string]string{
				"a.yaml":       fixtureTemplate("FIRST"),
				"nested/b.yml": second,
			})
			if name == "missing file" {
				if err := os.Remove(filepath.Join(directory, "templates", "nested", "b.yml")); err != nil {
					t.Fatal(err)
				}
			}
			_, offlineErr := repository.Load(directory)
			if offlineErr == nil {
				t.Fatal("expected offline validation to fail")
			}

			storage := &fixtureCatalog{}
			lastGood := time.Unix(10, 0)
			manager := &GitHubSyncManager{storage: storage, logger: zap.NewNop(), repoPath: directory, lastSync: lastGood}
			err := manager.syncCheckout(context.Background(), "fixture")
			if err == nil || !strings.Contains(err.Error(), `validate repository "fixture": `+offlineErr.Error()) {
				t.Fatalf("expected the offline diagnostics, got %v", err)
			}
			if len(storage.metadataReads) != 0 || storage.manifestReads != 0 || len(storage.contents) != 0 || storage.manifestWrites != 0 {
				t.Fatalf("invalid checkout accessed the catalog: %+v", storage)
			}
			if !manager.GetLastSyncTime().Equal(lastGood) {
				t.Fatalf("last successful sync changed: %v", manager.GetLastSyncTime())
			}
		})
	}
}

func TestSyncCheckoutUsesValidatedBytes(t *testing.T) {
	const name = "nested/check.yml"
	original := fixtureTemplate("CHECK")
	directory := writeSyncCheckout(t, map[string]string{name: original})
	storage := &fixtureCatalog{
		onManifestRead: func() {
			writeSyncFixture(t, directory, "templates/"+name, []byte("changed after validation"))
		},
	}
	manager := &GitHubSyncManager{storage: storage, logger: zap.NewNop(), repoPath: directory}

	if err := manager.syncCheckout(context.Background(), "fixture"); err != nil {
		t.Fatal(err)
	}
	if storage.contents["CHECK"] != original {
		t.Fatalf("wrong catalog bytes: %q", storage.contents["CHECK"])
	}
	if storage.manifestWrites != 1 || storage.manifest.Version != "fixture-version" ||
		storage.manifest.Statistics.StandardTemplates != 1 || storage.manifest.Statistics.ByType["file-content"] != 1 {
		t.Fatalf("unexpected manifest result: %+v", storage.manifest)
	}
	if manager.GetLastSyncTime().IsZero() {
		t.Fatal("last successful sync was not recorded")
	}
}

func TestSyncCheckoutPreservesCustomPrecedence(t *testing.T) {
	directory := writeSyncCheckout(t, map[string]string{
		"a.yaml": fixtureTemplate("CUSTOM"), "b.yaml": fixtureTemplate("STANDARD"),
	})
	storage := &fixtureCatalog{
		metadata: map[string]*TemplateInfo{"CUSTOM": {ID: "CUSTOM", IsCustom: true}},
		manifest: TemplateManifest{Statistics: TemplateStatistics{CustomTemplates: 1}},
	}
	manager := &GitHubSyncManager{storage: storage, logger: zap.NewNop(), repoPath: directory}

	if err := manager.syncCheckout(context.Background(), "fixture"); err != nil {
		t.Fatal(err)
	}
	if len(storage.contents) != 1 || storage.contents["STANDARD"] != fixtureTemplate("STANDARD") {
		t.Fatalf("unexpected writes: %v", storage.contents)
	}
	stats := storage.manifest.Statistics
	if stats.CustomTemplates != 1 || stats.StandardTemplates != 1 || stats.TotalTemplates != 2 {
		t.Fatalf("unexpected counts: %+v", stats)
	}
}

func TestSyncCheckoutPreflightsOwnership(t *testing.T) {
	directory := writeSyncCheckout(t, map[string]string{
		"a.yaml": fixtureTemplate("FIRST"), "b.yaml": fixtureTemplate("SECOND"),
	})
	storage := &fixtureCatalog{
		metadata: map[string]*TemplateInfo{
			"SECOND": {ID: "SECOND", Metadata: map[string]string{"repository_id": "another-repo"}},
		},
	}
	manager := &GitHubSyncManager{storage: storage, logger: zap.NewNop(), repoPath: directory}

	err := manager.syncCheckout(context.Background(), "fixture")
	if err == nil || !strings.Contains(err.Error(), `templates/b.yaml: template "SECOND" is owned by repository "another-repo"`) {
		t.Fatalf("expected ownership diagnostics, got %v", err)
	}
	if len(storage.contents) != 0 || storage.manifestWrites != 0 {
		t.Fatalf("ownership conflict reached publication: %+v", storage)
	}
}

func TestSyncCheckoutFencesConcurrentCustomWrite(t *testing.T) {
	for _, name := range []string{"content first", "metadata also published"} {
		t.Run(name, func(t *testing.T) {
			directory := writeSyncCheckout(t, map[string]string{"check.yaml": fixtureTemplate("CHECK")})
			storage := &fixtureCatalog{
				metadata: map[string]*TemplateInfo{
					"CHECK": {ID: "CHECK", Metadata: map[string]string{"repository_id": "fixture"}},
				},
				customContents: make(map[string]string),
			}
			storage.onManifestRead = func() {
				storage.customContents["CHECK"] = "operator content"
				if name == "metadata also published" {
					storage.metadata["CHECK"] = &TemplateInfo{ID: "CHECK", IsCustom: true, Checksum: "operator-checksum"}
				}
			}
			lastGood := time.Unix(10, 0)
			manager := &GitHubSyncManager{storage: storage, logger: zap.NewNop(), repoPath: directory, lastSync: lastGood}

			err := manager.syncCheckout(context.Background(), "fixture")
			if !errors.Is(err, errFixtureCatalogChanged) || !strings.Contains(err.Error(), "templates/check.yaml") {
				t.Fatalf("expected the concurrent write to prevent publication, got %v", err)
			}
			if len(storage.contents) != 0 || storage.manifestWrites != 0 || !manager.GetLastSyncTime().Equal(lastGood) {
				t.Fatalf("standard write succeeded after the override: %+v", storage)
			}
			if storage.customContents["CHECK"] != "operator content" {
				t.Fatal("operator content was lost")
			}
			if name == "metadata also published" && !storage.metadata["CHECK"].IsCustom {
				t.Fatal("operator metadata was overwritten")
			}
		})
	}
}

func TestSyncCheckoutReturnsStorageFailure(t *testing.T) {
	directory := writeSyncCheckout(t, map[string]string{"check.yaml": fixtureTemplate("CHECK")})
	storeErr := errors.New("catalog write failed")
	storage := &fixtureCatalog{storeErr: storeErr}
	lastGood := time.Unix(10, 0)
	manager := &GitHubSyncManager{storage: storage, logger: zap.NewNop(), repoPath: directory, lastSync: lastGood}

	err := manager.syncCheckout(context.Background(), "fixture")
	if !errors.Is(err, storeErr) || !strings.Contains(err.Error(), "templates/check.yaml") {
		t.Fatalf("expected a path and the storage error, got %v", err)
	}
	if storage.manifestWrites != 0 || !manager.GetLastSyncTime().Equal(lastGood) {
		t.Fatalf("failed write was marked successful: %+v", storage)
	}
}

var errFixtureCatalogChanged = errors.New("template catalog changed after preflight")

type fixtureCatalog struct {
	metadata       map[string]*TemplateInfo
	manifest       TemplateManifest
	contents       map[string]string
	customContents map[string]string
	metadataReads  []string
	manifestReads  int
	manifestWrites int
	onManifestRead func()
	storeErr       error
}

func (s *fixtureCatalog) getTemplateMetadata(_ context.Context, id string) (*TemplateInfo, string, error) {
	s.metadataReads = append(s.metadataReads, id)
	metadata := s.metadata[id]
	raw, err := s.metadataSnapshot(id)
	return metadata, raw, err
}

func (s *fixtureCatalog) metadataSnapshot(id string) (string, error) {
	if s.metadata[id] == nil {
		return "", nil
	}
	raw, err := json.Marshal(s.metadata[id])
	return string(raw), err
}

func (s *fixtureCatalog) GetTemplateManifest(_ context.Context) (*TemplateManifest, error) {
	s.manifestReads++
	if s.onManifestRead != nil {
		s.onManifestRead()
	}
	copy := s.manifest
	return &copy, nil
}

func (s *fixtureCatalog) storeRepositoryTemplate(_ context.Context, definition *types.Template, content []byte, repoID, snapshot string) error {
	if repoID != "fixture" {
		return fmt.Errorf("unexpected catalog destination: %s", repoID)
	}
	current, err := s.metadataSnapshot(definition.ID)
	if err != nil {
		return err
	}
	_, custom := s.customContents[definition.ID]
	if snapshot != current || custom {
		return errFixtureCatalogChanged
	}
	if s.storeErr != nil {
		return s.storeErr
	}
	if s.contents == nil {
		s.contents = make(map[string]string)
	}
	s.contents[definition.ID] = string(content)
	return nil
}

func (s *fixtureCatalog) UpdateTemplateManifest(_ context.Context, manifest *TemplateManifest) error {
	s.manifestWrites++
	s.manifest = *manifest
	return nil
}

func fixtureTemplate(id string) string {
	return fmt.Sprintf("id: %s\ninfo:\n  name: Synthetic fixture\n  severity: low\ndetection:\n  steps:\n    - type: file_content\n", id)
}

func writeSyncCheckout(t *testing.T, files map[string]string) string {
	t.Helper()
	directory := t.TempDir()
	writeSyncFixture(t, directory, "repository-manifest.json", []byte(`{"version":"fixture-version"}`))

	templates := make(map[string]repository.ManifestEntry, len(files))
	for name, content := range files {
		data := []byte(content)
		size := int64(len(data))
		id, _, _ := strings.Cut(strings.TrimPrefix(content, "id: "), "\n")
		templates[name] = repository.ManifestEntry{
			ID: id, Checksum: fmt.Sprintf("sha256:%x", sha256.Sum256(data)), Size: &size,
			DetectionType: "file-content", Severity: "low",
		}
		writeSyncFixture(t, directory, "templates/"+name, data)
	}
	manifest, err := json.Marshal(map[string]any{"format_version": 2, "templates": templates})
	if err != nil {
		t.Fatal(err)
	}
	writeSyncFixture(t, directory, "templates/manifest.json", manifest)
	return directory
}

func writeSyncFixture(t *testing.T, directory, name string, data []byte) {
	t.Helper()
	fullPath := filepath.Join(directory, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
