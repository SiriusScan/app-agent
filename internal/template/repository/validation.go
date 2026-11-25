// Package repository stages the templates in a local repository checkout for offline validation.
package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/template/parser"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

const (
	repositoryManifestPath = "repository-manifest.json"
	templateDirectory      = "templates"
	templateManifestPath   = "templates/manifest.json"
	maxManifestBytes       = 8 << 20
	maxTemplateBytes       = 1 << 20
)

type templateManifest struct {
	FormatVersion *int                      `json:"format_version"`
	Templates     map[string]*manifestEntry `json:"templates"`
}

type manifestEntry struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
	Size     *int64 `json:"size"`
	Severity string `json:"severity"`
	Author   string `json:"author"`
}

// Candidate holds a validated template inventory and any legacy compatibility warnings.
// Templates is populated only when the complete repository passes validation.
type Candidate struct {
	RepositoryVersion string
	Templates         []Template
	Warnings          []error
}

// Template holds the parsed definition and the exact bytes used to validate it.
type Template struct {
	Path       string
	Definition *types.Template
	Content    []byte
}

// Load checks the root and template manifests and stages all listed templates.
// It uses checkout files and static schema data and never runs detection steps.
func Load(directory string) (Candidate, error) {
	var candidate Candidate
	root, err := os.OpenRoot(directory)
	if err != nil {
		return candidate, fmt.Errorf("open repository %q: %w", directory, err)
	}
	defer root.Close()

	var repository struct {
		Version string `json:"version"`
	}
	if err := readManifest(root, repositoryManifestPath, &repository); err != nil {
		return candidate, err
	}
	candidate.RepositoryVersion = repository.Version

	templateDir, err := root.Lstat(templateDirectory)
	if err != nil {
		return candidate, fmt.Errorf("%s: %w", templateDirectory, err)
	}
	if !templateDir.IsDir() {
		return candidate, fmt.Errorf("%s: must be a directory", templateDirectory)
	}

	var manifest templateManifest
	if err := readManifest(root, templateManifestPath, &manifest); err != nil {
		return candidate, err
	}
	if manifest.Templates == nil {
		return candidate, fmt.Errorf("%s: templates must be a JSON object", templateManifestPath)
	}
	strict := false
	if manifest.FormatVersion != nil {
		switch *manifest.FormatVersion {
		case 1:
		case 2:
			strict = true
		default:
			return candidate, fmt.Errorf("%s: unsupported format_version %d", templateManifestPath, *manifest.FormatVersion)
		}
	}

	inventory := make(map[string]struct{})
	var issues []error
	if err := fs.WalkDir(root.FS(), templateDirectory, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			issues = append(issues, fmt.Errorf("%s: %w", filePath, err))
			return nil
		}
		if !entry.IsDir() && isTemplatePath(filePath) {
			inventory[strings.TrimPrefix(filePath, templateDirectory+"/")] = struct{}{}
		}
		return nil
	}); err != nil {
		return candidate, err
	}

	manifestIDs := make(map[string]string)
	yamlIDs := make(map[string]string)
	var validated []Template
	for _, name := range slices.Sorted(maps.Keys(manifest.Templates)) {
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\:") || !isTemplatePath(name) {
			issues = append(issues, fmt.Errorf("%s: invalid template path %q", templateManifestPath, name))
			continue
		}
		filePath := path.Join(templateDirectory, name)
		entry := manifest.Templates[name]
		if entry == nil {
			issues = append(issues, fmt.Errorf("%s: manifest entry must be a JSON object", filePath))
			continue
		}
		issueCount := len(issues)
		if strings.TrimSpace(entry.ID) == "" {
			issues = append(issues, fmt.Errorf("%s: manifest ID is required", filePath))
		} else if previous, found := manifestIDs[entry.ID]; found {
			issues = append(issues, fmt.Errorf("%s: duplicate manifest ID %q also listed for %s", filePath, entry.ID, previous))
		} else {
			manifestIDs[entry.ID] = filePath
		}
		if _, found := inventory[name]; !found {
			issues = append(issues, fmt.Errorf("%s: listed in %s but missing from the template inventory", filePath, templateManifestPath))
			continue
		}
		content, err := readRegularFile(root, filePath, maxTemplateBytes)
		if err != nil {
			issues = append(issues, fmt.Errorf("%s: %w", filePath, err))
			continue
		}

		if entry.Size == nil && strict {
			issues = append(issues, fmt.Errorf("%s: manifest size is required in format_version 2", filePath))
		}
		if entry.Size != nil && *entry.Size != int64(len(content)) {
			issues = append(issues, fmt.Errorf("%s: manifest size is %d bytes, file is %d bytes", filePath, *entry.Size, len(content)))
		}
		warning, err := validateChecksum(entry.Checksum, content, strict)
		if warning != "" {
			candidate.Warnings = append(candidate.Warnings, fmt.Errorf("%s: %s", filePath, warning))
		}
		if err != nil {
			issues = append(issues, fmt.Errorf("%s: %w", filePath, err))
		}

		definition, err := parseTemplate(content, entry, strict)
		if err != nil {
			issues = append(issues, fmt.Errorf("%s: %w", filePath, err))
		}
		if definition != nil && strings.TrimSpace(definition.ID) != "" {
			if previous, found := yamlIDs[definition.ID]; found {
				if manifestIDs[definition.ID] != previous || entry.ID != definition.ID {
					issues = append(issues, fmt.Errorf("%s: duplicate YAML ID %q also used by %s", filePath, definition.ID, previous))
				}
			} else {
				yamlIDs[definition.ID] = filePath
			}
		}
		if len(issues) == issueCount {
			validated = append(validated, Template{Path: filePath, Definition: definition, Content: content})
		}
	}

	for _, name := range slices.Sorted(maps.Keys(inventory)) {
		if _, listed := manifest.Templates[name]; !listed {
			issues = append(issues, fmt.Errorf("%s: missing entry in %s", path.Join(templateDirectory, name), templateManifestPath))
		}
	}

	if err := errors.Join(issues...); err != nil {
		return candidate, err
	}
	candidate.Templates = validated
	return candidate, nil
}

func readManifest(root *os.Root, name string, target any) error {
	data, err := readRegularFile(root, name, maxManifestBytes)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if len(data) == 0 {
		return fmt.Errorf("%s: manifest must be a JSON object", name)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if strings.TrimSpace(string(data)) == "null" {
		return fmt.Errorf("%s: manifest must be a JSON object", name)
	}
	return nil
}

func readRegularFile(root *os.Root, name string, limit int64) ([]byte, error) {
	localName, err := filepath.Localize(name)
	if err != nil {
		return nil, err
	}
	info, err := root.Lstat(localName)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("must be a regular file")
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("file exceeds the %d-byte limit", limit)
	}
	file, err := root.Open(localName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("file exceeds the %d-byte limit", limit)
	}
	return content, nil
}

func isTemplatePath(name string) bool {
	return strings.EqualFold(path.Ext(name), ".yaml") || strings.EqualFold(path.Ext(name), ".yml")
}

func validateChecksum(checksum string, content []byte, strict bool) (string, error) {
	if checksum == "" || isLegacyPlaceholder(checksum) {
		if strict {
			return "", fmt.Errorf("format_version 2 requires a SHA-256 checksum for every template")
		}
		return "legacy manifest has no verifiable SHA-256 checksum", nil
	}
	value, ok := strings.CutPrefix(checksum, "sha256:")
	if !ok || len(value) != 2*sha256.Size {
		return "", fmt.Errorf("manifest checksum must use sha256:<64 hex characters>")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", fmt.Errorf("manifest checksum must use sha256:<64 hex characters>")
	}
	if strict && value != strings.ToLower(value) {
		return "", fmt.Errorf("format_version 2 requires lowercase SHA-256 checksums")
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(content))
	if !strings.EqualFold(value, actual) {
		return "", fmt.Errorf("manifest checksum mismatch: declared sha256:%s, file is sha256:%s", value, actual)
	}
	return "", nil
}

func isLegacyPlaceholder(checksum string) bool {
	switch checksum {
	case "sha256:pending",
		"sha256:demo1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		"sha256:demo5678901234cdef5678901234cdef5678901234cdef5678901234cdef5678",
		"sha256:demo9012345678efgh9012345678efgh9012345678efgh9012345678efgh9012":
		return true
	}
	return false
}

func parseTemplate(content []byte, entry *manifestEntry, strict bool) (*types.Template, error) {
	definition, err := parser.ParseTemplateBytes(content)
	if err != nil {
		return nil, err
	}
	if !strict && definition.ID == "" {
		definition.ID = entry.ID
	}
	if definition.Info.Version == "" {
		definition.Info.Version = entry.Version
	}
	if definition.Info.Author == "" {
		definition.Info.Author = entry.Author
	}
	if definition.Info.Severity == "" {
		definition.Info.Severity = types.Severity(entry.Severity)
	}

	var issues []error
	if definition.ID != entry.ID {
		issues = append(issues, fmt.Errorf("manifest ID %q does not match YAML ID %q", entry.ID, definition.ID))
	}
	if err := parser.ValidateTemplate(definition); err != nil {
		issues = append(issues, err)
	}
	for i, step := range definition.Detection.Steps {
		if strings.TrimSpace(step.Type) != "" && !modules.IsKnownDetectionType(step.Type) {
			issues = append(issues, fmt.Errorf(
				"detection.steps[%d].type %q is unsupported (known types: %s)",
				i, step.Type, strings.Join(modules.KnownDetectionTypeNames(), ", "),
			))
		}
	}
	return definition, errors.Join(issues...)
}
