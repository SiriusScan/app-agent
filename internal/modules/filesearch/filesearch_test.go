package filesearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
)

// buildTree creates a temporary directory tree for testing.
// It returns the root path. Paths are relative to root; directories end with "/".
func buildTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

func TestBasicFilenameMatch(t *testing.T) {
	root := buildTree(t, map[string]string{
		"project/package.json":          `{"name":"myapp","version":"1.0.0"}`,
		"project/src/index.js":          `console.log("hi")`,
		"project/node_modules/a/pkg.js": `module.exports = {}`,
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "package.json",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected match for package.json")
	}
	matches := result.Evidence["matches"].([]interface{})
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
}

func TestPathRegexFilter(t *testing.T) {
	root := buildTree(t, map[string]string{
		"app/node_modules/axios/package.json": `{"version":"1.14.1"}`,
		"app/node_modules/lodash/package.json": `{"version":"4.17.21"}`,
		"app/package.json":                     `{"version":"2.0.0"}`,
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "package.json",
		"path_regex":       `node_modules/axios/package\.json$`,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected match for axios package.json")
	}
	matches := result.Evidence["matches"].([]interface{})
	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 match (axios only), got %d", len(matches))
	}
}

func TestContentRegex(t *testing.T) {
	root := buildTree(t, map[string]string{
		"a/node_modules/axios/package.json": `{"name":"axios","version":"1.14.1"}`,
		"b/node_modules/axios/package.json": `{"name":"axios","version":"1.7.9"}`,
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "package.json",
		"path_regex":       `node_modules/axios/`,
		"content_regex":    `"version":\s*"(1\.14\.1|0\.30\.4)"`,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected match for compromised version")
	}
	matches := result.Evidence["matches"].([]interface{})
	if len(matches) != 1 {
		t.Fatalf("expected 1 content match, got %d", len(matches))
	}

	first := matches[0].(map[string]interface{})
	if first["matched_text"] == nil || first["matched_text"] == "" {
		t.Error("expected matched_text in evidence")
	}
	if first["line_number"] == nil {
		t.Error("expected line_number in evidence")
	}
}

func TestContentRegexNoMatch(t *testing.T) {
	root := buildTree(t, map[string]string{
		"node_modules/axios/package.json": `{"name":"axios","version":"1.7.9"}`,
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":     root,
		"content_regex": `"version":\s*"(1\.14\.1|0\.30\.4)"`,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Matched {
		t.Fatal("should not match safe version")
	}
}

func TestMaxDepthEnforced(t *testing.T) {
	root := buildTree(t, map[string]string{
		"a/b/c/deep.txt":   "deep",
		"a/shallow.txt":    "shallow",
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "*.txt",
		"max_depth":        1,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected match for shallow.txt at depth 1")
	}
	matches := result.Evidence["matches"].([]interface{})
	for _, raw := range matches {
		entry := raw.(map[string]interface{})
		path := entry["path"].(string)
		if filepath.Base(path) == "deep.txt" {
			t.Errorf("deep.txt at depth 3 should have been skipped with max_depth=1")
		}
	}
}

func TestExcludeDirs(t *testing.T) {
	root := buildTree(t, map[string]string{
		"src/main.txt":         "source",
		"vendor/dep/main.txt":  "vendored",
		".cache/tmp/main.txt":  "cached",
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "main.txt",
		"exclude_dirs":     []interface{}{"vendor", ".cache"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected match from src/")
	}
	matches := result.Evidence["matches"].([]interface{})
	if len(matches) != 1 {
		t.Fatalf("expected 1 match (vendor and .cache excluded), got %d", len(matches))
	}
}

func TestMaxResultsTruncation(t *testing.T) {
	tree := map[string]string{}
	for i := 0; i < 10; i++ {
		tree[filepath.Join("dir", string(rune('a'+i))+".txt")] = "content"
	}
	root := buildTree(t, tree)

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "*.txt",
		"max_results":      3,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected match")
	}
	matches := result.Evidence["matches"].([]interface{})
	if len(matches) != 3 {
		t.Fatalf("expected 3 collected matches, got %d", len(matches))
	}
	if result.Evidence["truncated"] != true {
		t.Error("expected truncated=true when max_results exceeded")
	}
}

func TestMissingRootPath(t *testing.T) {
	m := &FileSearchModule{}
	_, err := m.Execute(context.Background(), modules.StepConfig{
		"filename_pattern": "*.txt",
	})
	if err == nil {
		t.Fatal("expected error for missing root_path")
	}
}

func TestNoFilterProvided(t *testing.T) {
	m := &FileSearchModule{}
	_, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path": "/tmp",
	})
	if err == nil {
		t.Fatal("expected error when no filter is provided")
	}
}

func TestNonexistentRootPath(t *testing.T) {
	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        "/nonexistent/path/abc123",
		"filename_pattern": "*.txt",
	})
	if err != nil {
		t.Fatalf("Execute should not return error, got: %v", err)
	}
	if result.Matched {
		t.Error("should not match with nonexistent root")
	}
	if result.Error == "" {
		t.Error("expected error message in result")
	}
}

func TestInvalidPathRegex(t *testing.T) {
	m := &FileSearchModule{}
	_, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":  "/tmp",
		"path_regex": "[invalid(regex",
	})
	if err == nil {
		t.Fatal("expected error for invalid path_regex")
	}
}

func TestInvalidContentRegex(t *testing.T) {
	m := &FileSearchModule{}
	_, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":     "/tmp",
		"content_regex": "[invalid(regex",
	})
	if err == nil {
		t.Fatal("expected error for invalid content_regex")
	}
}

func TestContextCancellation(t *testing.T) {
	root := buildTree(t, map[string]string{
		"a.txt": "data",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond) // ensure timeout fires

	m := &FileSearchModule{}
	result, err := m.Execute(ctx, modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "*.txt",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Cancelled walk should still return a valid result (possibly empty).
	_ = result
}

func TestGlobPatternWildcard(t *testing.T) {
	root := buildTree(t, map[string]string{
		"data/report.csv":  "a,b,c",
		"data/report.json": `{"x":1}`,
		"data/report.txt":  "hello",
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "*.csv",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected match for *.csv")
	}
	matches := result.Evidence["matches"].([]interface{})
	if len(matches) != 1 {
		t.Fatalf("expected 1 csv match, got %d", len(matches))
	}
}

func TestDefaultExcludeGit(t *testing.T) {
	root := buildTree(t, map[string]string{
		".git/config":    "gitconfig",
		"src/main.txt":   "source",
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":        root,
		"filename_pattern": "*",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	matches := result.Evidence["matches"].([]interface{})
	for _, raw := range matches {
		entry := raw.(map[string]interface{})
		path := entry["path"].(string)
		if filepath.Base(filepath.Dir(path)) == ".git" {
			t.Error(".git directory should be excluded by default")
		}
	}
}

func TestFilesScannedCount(t *testing.T) {
	root := buildTree(t, map[string]string{
		"a/file1.txt": "hello",
		"a/file2.txt": "world",
		"a/file3.log": "log",
	})

	m := &FileSearchModule{}
	result, err := m.Execute(context.Background(), modules.StepConfig{
		"root_path":     root,
		"content_regex": "hello",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	scanned, ok := result.Evidence["files_scanned"].(int)
	if !ok {
		t.Fatal("files_scanned should be int")
	}
	if scanned != 3 {
		t.Errorf("expected 3 files scanned, got %d", scanned)
	}
	matched := result.Evidence["files_matched"].(int)
	if matched != 1 {
		t.Errorf("expected 1 file matched, got %d", matched)
	}
}

func TestRegistration(t *testing.T) {
	mod := registry.Get("file_search")
	if mod == nil {
		t.Fatal("file_search module not registered")
	}

	desc := registry.GetDescriptor("file_search")
	if desc == nil {
		t.Fatal("file_search descriptor not found")
	}
	if desc.Type != "file_search" {
		t.Errorf("expected type 'file_search', got %q", desc.Type)
	}
	if len(desc.SupportedOS) == 0 {
		t.Error("no supported OS specified")
	}
}
