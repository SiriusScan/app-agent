package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createTestTemplate creates a valid template YAML file for testing
func createTestTemplate(t *testing.T, dir string, filename string, id string) string {
	t.Helper()

	content := fmt.Sprintf(`
id: %s
info:
  name: Test Template %s
  severity: high
  description: Test template for discovery
detection:
  logic: all
  steps:
    - type: file_hash
      weight: 1.0
      config:
        path: /test/file
`, id, id)

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	return path
}

// createInvalidTemplate creates an invalid template file
func createInvalidTemplate(t *testing.T, dir string, filename string) string {
	t.Helper()

	content := `
id: invalid-template
info:
  name: Invalid Template
  # Missing required severity field
detection:
  steps:
    - type: file_hash
`

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create invalid template: %v", err)
	}

	return path
}

// createMalformedYAML creates a malformed YAML file
func createMalformedYAML(t *testing.T, dir string, filename string) string {
	t.Helper()

	content := `
id: malformed
info:
  name: Bad YAML
    indentation is all wrong
      severity: high
`

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create malformed YAML: %v", err)
	}

	return path
}

func TestDiscoverTemplates(t *testing.T) {
	t.Log("\n🔍 Testing DiscoverTemplates()...")

	t.Run("empty directory", func(t *testing.T) {
		t.Log("\n  Testing empty directory...")

		tmpDir := t.TempDir()

		templates, errs := DiscoverTemplates(tmpDir)

		if len(templates) != 0 {
			t.Errorf("Expected 0 templates, got %d", len(templates))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
		}

		t.Log("  ✅ Empty directory handled correctly")
	})

	t.Run("single valid template", func(t *testing.T) {
		t.Log("\n  Testing single valid template...")

		tmpDir := t.TempDir()
		createTestTemplate(t, tmpDir, "template-001.yaml", "test-001")

		templates, errs := DiscoverTemplates(tmpDir)

		if len(templates) != 1 {
			t.Fatalf("Expected 1 template, got %d", len(templates))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
		}

		// Verify template was loaded correctly
		if templates[0].ID != "test-001" {
			t.Errorf("Expected template ID 'test-001', got '%s'", templates[0].ID)
		}

		t.Log("  ✅ Single valid template discovered")
	})

	t.Run("multiple valid templates", func(t *testing.T) {
		t.Log("\n  Testing multiple valid templates...")

		tmpDir := t.TempDir()
		createTestTemplate(t, tmpDir, "template-001.yaml", "test-001")
		createTestTemplate(t, tmpDir, "template-002.yml", "test-002")
		createTestTemplate(t, tmpDir, "template-003.yaml", "test-003")

		templates, errs := DiscoverTemplates(tmpDir)

		if len(templates) != 3 {
			t.Fatalf("Expected 3 templates, got %d", len(templates))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
		}

		// Verify all templates were loaded
		ids := make(map[string]bool)
		for _, tmpl := range templates {
			ids[tmpl.ID] = true
		}

		expectedIDs := []string{"test-001", "test-002", "test-003"}
		for _, id := range expectedIDs {
			if !ids[id] {
				t.Errorf("Expected to find template with ID '%s'", id)
			}
		}

		t.Log("  ✅ Multiple valid templates discovered")
	})

	t.Run("nested directories", func(t *testing.T) {
		t.Log("\n  Testing nested directories...")

		tmpDir := t.TempDir()

		// Create nested structure
		subDir1 := filepath.Join(tmpDir, "linux")
		subDir2 := filepath.Join(tmpDir, "windows", "critical")

		os.MkdirAll(subDir1, 0755)
		os.MkdirAll(subDir2, 0755)

		createTestTemplate(t, tmpDir, "root-template.yaml", "test-root")
		createTestTemplate(t, subDir1, "linux-template.yaml", "test-linux")
		createTestTemplate(t, subDir2, "windows-template.yaml", "test-windows")

		templates, errs := DiscoverTemplates(tmpDir)

		if len(templates) != 3 {
			t.Fatalf("Expected 3 templates from nested dirs, got %d", len(templates))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
		}

		t.Log("  ✅ Nested directories handled correctly")
	})

	t.Run("mixed valid and invalid templates", func(t *testing.T) {
		t.Log("\n  Testing mix of valid and invalid templates...")

		tmpDir := t.TempDir()

		createTestTemplate(t, tmpDir, "valid-001.yaml", "test-valid-001")
		createInvalidTemplate(t, tmpDir, "invalid-001.yaml")
		createTestTemplate(t, tmpDir, "valid-002.yaml", "test-valid-002")
		createMalformedYAML(t, tmpDir, "malformed.yaml")

		templates, errs := DiscoverTemplates(tmpDir)

		// Should have 2 valid templates
		if len(templates) != 2 {
			t.Errorf("Expected 2 valid templates, got %d", len(templates))
		}

		// Should have 2 errors
		if len(errs) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(errs))
		}

		// Verify valid templates were loaded
		ids := make(map[string]bool)
		for _, tmpl := range templates {
			ids[tmpl.ID] = true
		}

		if !ids["test-valid-001"] || !ids["test-valid-002"] {
			t.Error("Expected to find both valid templates")
		}

		t.Logf("  ✅ Mixed templates handled: %d valid, %d errors", len(templates), len(errs))
	})

	t.Run("non-template files ignored", func(t *testing.T) {
		t.Log("\n  Testing that non-template files are ignored...")

		tmpDir := t.TempDir()

		createTestTemplate(t, tmpDir, "template.yaml", "test-001")

		// Create non-template files
		os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# README"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "script.sh"), []byte("#!/bin/bash"), 0644)
		os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log"), 0644)

		templates, errs := DiscoverTemplates(tmpDir)

		if len(templates) != 1 {
			t.Errorf("Expected 1 template (non-YAML files should be ignored), got %d", len(templates))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
		}

		t.Log("  ✅ Non-template files correctly ignored")
	})

	t.Run("directory not found", func(t *testing.T) {
		t.Log("\n  Testing non-existent directory...")

		templates, errs := DiscoverTemplates("/nonexistent/directory/path")

		if len(templates) != 0 {
			t.Errorf("Expected 0 templates, got %d", len(templates))
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(errs))
		}

		if !strings.Contains(errs[0].Error(), "does not exist") {
			t.Errorf("Expected error about non-existent directory, got: %v", errs[0])
		}

		t.Logf("  ✅ Non-existent directory error: %v", errs[0])
	})

	t.Run("empty path", func(t *testing.T) {
		t.Log("\n  Testing empty path...")

		templates, errs := DiscoverTemplates("")

		if len(templates) != 0 {
			t.Errorf("Expected 0 templates, got %d", len(templates))
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(errs))
		}

		if !strings.Contains(errs[0].Error(), "cannot be empty") {
			t.Errorf("Expected error about empty path, got: %v", errs[0])
		}

		t.Logf("  ✅ Empty path error: %v", errs[0])
	})

	t.Run("path is a file not directory", func(t *testing.T) {
		t.Log("\n  Testing when path is a file not a directory...")

		tmpDir := t.TempDir()
		filePath := createTestTemplate(t, tmpDir, "template.yaml", "test-001")

		templates, errs := DiscoverTemplates(filePath)

		if len(templates) != 0 {
			t.Errorf("Expected 0 templates, got %d", len(templates))
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(errs))
		}

		if !strings.Contains(errs[0].Error(), "not a directory") {
			t.Errorf("Expected error about file not being a directory, got: %v", errs[0])
		}

		t.Logf("  ✅ File path error: %v", errs[0])
	})

	t.Log("\n✅ DiscoverTemplates() tests completed")
}

func TestDiscoverTemplatesWithTimeout(t *testing.T) {
	t.Log("\n🔍 Testing DiscoverTemplatesWithTimeout()...")

	t.Run("timeout with context cancellation", func(t *testing.T) {
		t.Log("\n  Testing context cancellation...")

		tmpDir := t.TempDir()

		// Create some templates
		for i := 0; i < 5; i++ {
			createTestTemplate(t, tmpDir, fmt.Sprintf("template-%d.yaml", i), fmt.Sprintf("test-%d", i))
		}

		// Create a context that's already canceled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		templates, errs := DiscoverTemplatesWithContext(ctx, tmpDir)

		// Should fail with context canceled error
		if len(errs) == 0 {
			t.Error("Expected error for canceled context")
		}

		hasContextError := false
		for _, err := range errs {
			if strings.Contains(err.Error(), "canceled") {
				hasContextError = true
				break
			}
		}

		if !hasContextError {
			t.Errorf("Expected context canceled error, got: %v", errs)
		}

		t.Logf("  ✅ Context cancellation handled: %d templates, %d errors", len(templates), len(errs))
	})

	t.Run("successful discovery with timeout", func(t *testing.T) {
		t.Log("\n  Testing successful discovery with generous timeout...")

		tmpDir := t.TempDir()

		createTestTemplate(t, tmpDir, "template-001.yaml", "test-001")
		createTestTemplate(t, tmpDir, "template-002.yaml", "test-002")

		// Use a very generous timeout
		templates, errs := DiscoverTemplatesWithTimeout(tmpDir, 1*time.Minute)

		if len(templates) != 2 {
			t.Errorf("Expected 2 templates, got %d", len(templates))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
		}

		t.Log("  ✅ Discovery completed successfully within timeout")
	})

	t.Log("\n✅ DiscoverTemplatesWithTimeout() tests completed")
}

func TestDiscoverTemplatesPerformance(t *testing.T) {
	t.Log("\n🔍 Testing template discovery performance...")

	t.Run("many templates", func(t *testing.T) {
		t.Log("\n  Testing discovery of 100+ templates...")

		tmpDir := t.TempDir()

		// Create 150 templates
		numTemplates := 150
		for i := 0; i < numTemplates; i++ {
			createTestTemplate(t, tmpDir, fmt.Sprintf("template-%03d.yaml", i), fmt.Sprintf("test-%03d", i))
		}

		start := time.Now()
		templates, errs := DiscoverTemplates(tmpDir)
		duration := time.Since(start)

		if len(templates) != numTemplates {
			t.Errorf("Expected %d templates, got %d", numTemplates, len(templates))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d", len(errs))
		}

		t.Logf("  ✅ Discovered %d templates in %v", len(templates), duration)

		// Performance check - should be reasonably fast
		if duration > 10*time.Second {
			t.Logf("  ⚠️  Discovery took longer than expected: %v", duration)
		}
	})

	t.Log("\n✅ Performance tests completed")
}

func TestDiscoveryError(t *testing.T) {
	t.Log("\n🔍 Testing DiscoveryError type...")

	t.Run("error formatting", func(t *testing.T) {
		de := &DiscoveryError{
			Path: "/test/template.yaml",
			Err:  fmt.Errorf("parse error"),
		}

		errStr := de.Error()
		if !strings.Contains(errStr, "/test/template.yaml") {
			t.Error("Expected error to contain file path")
		}
		if !strings.Contains(errStr, "parse error") {
			t.Error("Expected error to contain original error message")
		}

		t.Logf("  ✅ DiscoveryError format: %s", errStr)
	})

	t.Log("\n✅ DiscoveryError tests completed")
}

func TestIsTemplateFile(t *testing.T) {
	t.Log("\n🔍 Testing isTemplateFile()...")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/path/to/template.yaml", true},
		{"/path/to/template.yml", true},
		{"/path/to/TEMPLATE.YAML", true},
		{"/path/to/TEMPLATE.YML", true},
		{"/path/to/template.json", false},
		{"/path/to/template.txt", false},
		{"/path/to/README.md", false},
		{"/path/to/script.sh", false},
		{"/path/to/config", false},
		{"/path/to/.gitignore", false},
	}

	for _, tt := range tests {
		result := isTemplateFile(tt.path)
		if result != tt.expected {
			t.Errorf("isTemplateFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}

	t.Log("  ✅ isTemplateFile() works correctly for all file types")
}

