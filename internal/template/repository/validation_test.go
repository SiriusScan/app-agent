package repository

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

const validTemplate = "id: example-one\ninfo:\n  name: Example\n  severity: low\ndetection:\n  steps:\n    - type: file_hash\n"

func TestLoadStrictRepository(t *testing.T) {
	const name = "nested/check.yml"
	directory := writeRepository(t, 2, map[string]string{name: validTemplate}, map[string]ManifestEntry{
		name: verifiedEntry("example-one", validTemplate),
	})

	candidate, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Warnings) != 0 || len(candidate.Templates) != 1 || candidate.RepositoryVersion != "2026.1" {
		t.Fatalf("unexpected candidate: %+v", candidate)
	}
	template := candidate.Templates[0]
	if template.Path != "templates/"+name || string(template.Content) != validTemplate {
		t.Fatalf("unexpected staged file: %+v", template)
	}
	if template.Definition.Detection.Logic != types.LogicAll || template.Definition.Detection.Steps[0].Weight != 1 {
		t.Fatalf("parser defaults were lost: %+v", template.Definition.Detection)
	}
}

func TestLoadLegacyDefaultsAndWarnings(t *testing.T) {
	const name = "legacy.yaml"
	legacyTemplate := strings.ReplaceAll(validTemplate, "id: example-one\n", "")
	legacyTemplate = strings.ReplaceAll(legacyTemplate, "  severity: low\n", "")
	directory := writeRepository(t, 0, map[string]string{name: legacyTemplate}, map[string]ManifestEntry{
		name: {ID: "example-one", Version: "1.5.0", Author: "Example", Severity: "low", Checksum: "sha256:pending"},
	})

	candidate, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Warnings) != 1 || !strings.Contains(candidate.Warnings[0].Error(), "templates/legacy.yaml: legacy manifest") {
		t.Fatalf("unexpected warnings: %v", candidate.Warnings)
	}
	if len(candidate.Templates) != 1 {
		t.Fatalf("expected a template, got %d", len(candidate.Templates))
	}
	template := candidate.Templates[0].Definition
	if template.ID != "example-one" || template.Info.Version != "1.5.0" ||
		template.Info.Author != "Example" || template.Info.Severity != types.SeverityLow {
		t.Fatalf("manifest fallback was lost: %+v", template)
	}
}

func TestLoadReportsRepositoryDrift(t *testing.T) {
	withUnknownStep := validTemplate + "    - type: unknown_module\n"
	files := map[string]string{
		"a.yaml":        validTemplate,
		"nested/b.yml":  withUnknownStep,
		"id.yaml":       validTemplate,
		"unlisted.yml":  validTemplate,
		"bad-hash.yaml": strings.ReplaceAll(validTemplate, "example-one", "bad-hash"),
	}
	incorrectSize := int64(1)
	entries := map[string]ManifestEntry{
		"a.yaml":        verifiedEntry("example-one", validTemplate),
		"nested/b.yml":  {ID: "example-one", Checksum: "sha256:" + strings.Repeat("0", 64), Size: &incorrectSize},
		"id.yaml":       {ID: "another-id", Checksum: "sha256:pending"},
		"bad-hash.yaml": {ID: "bad-hash", Checksum: "sha256:demo-invalid"},
		"missing.yaml":  {ID: "missing"},
		"../escape.yml": {ID: "outside"},
	}
	candidate, err := Load(writeRepository(t, 1, files, entries))
	if err == nil || len(candidate.Templates) != 0 {
		t.Fatalf("invalid checkout staged templates: %+v, %v", candidate, err)
	}
	for _, want := range []string{
		`templates/manifest.json: invalid template path "../escape.yml"`,
		`templates/id.yaml: manifest ID "another-id" does not match YAML ID "example-one"`,
		`templates/id.yaml: duplicate YAML ID "example-one"`,
		"templates/nested/b.yml: duplicate manifest ID",
		"templates/nested/b.yml: manifest size is",
		"templates/nested/b.yml: manifest checksum mismatch",
		"templates/nested/b.yml: detection.steps[1].type",
		"templates/bad-hash.yaml: manifest checksum must use sha256:<64 hex characters>",
		"templates/missing.yaml: listed in templates/manifest.json but missing",
		"templates/unlisted.yml: missing entry in templates/manifest.json",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q in:\n%s", want, err)
		}
	}
}

func TestLoadStrictRejectsMissingIntegrityMetadata(t *testing.T) {
	const name = "strict.yaml"
	const missingID = "info:\n  name: Example\n  severity: low\ndetection:\n  steps:\n    - type: file_hash\n"
	for _, checksum := range []string{"", "sha256:pending"} {
		t.Run(checksum, func(t *testing.T) {
			directory := writeRepository(t, 2, map[string]string{name: missingID}, map[string]ManifestEntry{
				name: {ID: "example-one", Checksum: checksum},
			})
			candidate, err := Load(directory)
			if err == nil || len(candidate.Warnings) != 0 {
				t.Fatalf("expected strict errors: %+v, %v", candidate, err)
			}
			for _, want := range []string{
				"manifest size is required",
				"requires a SHA-256 checksum",
				`does not match YAML ID ""`,
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected %q in:\n%s", want, err)
				}
			}
		})
	}
}

func verifiedEntry(id, content string) ManifestEntry {
	size := int64(len(content))
	return ManifestEntry{ID: id, Size: &size, Checksum: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content)))}
}

func writeRepository(t *testing.T, version int, files map[string]string, entries map[string]ManifestEntry) string {
	t.Helper()
	directory := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		fullPath := filepath.Join(directory, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(repositoryManifestPath, `{"version":"2026.1"}`)
	manifest := struct {
		FormatVersion int                      `json:"format_version,omitempty"`
		Templates     map[string]ManifestEntry `json:"templates"`
	}{version, entries}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	write(templateManifestPath, string(raw))
	for name, content := range files {
		write("templates/"+name, content)
	}
	return directory
}
