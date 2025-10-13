package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

// validTemplateYAML is a complete valid template for testing
const validTemplateYAML = `
id: test-template-001
info:
  name: Test Template
  author: Test Author
  severity: high
  description: A test template for unit testing
  references:
    - https://example.com/vuln-1
  cve:
    - CVE-2023-1234
  tags:
    - test
    - example
  version: "1.0"
detection:
  logic: all
  steps:
    - type: file_hash
      platforms:
        - linux
        - darwin
      weight: 0.8
      config:
        path: /usr/bin/ssh
        hash: abc123def456
        algorithm: sha256
    - type: file_content
      platforms:
        - windows
      weight: 1.0
      config:
        path: C:\Windows\system32\notepad.exe
        regex: "vulnerable.*version"
`

// invalidYAML is malformed YAML that should fail parsing
const invalidYAML = `
id: bad-template
info:
  name: Bad Template
  indentation is wrong here
    severity: high
`

// minimalTemplateYAML has only required fields
const minimalTemplateYAML = `
id: minimal-001
info:
  name: Minimal Template
  severity: medium
  description: Minimal template
detection:
  steps:
    - type: file_hash
      config:
        path: /test/file
`

func TestParseTemplateBytes(t *testing.T) {
	t.Log("\n🔍 Testing ParseTemplateBytes()...")

	t.Run("valid template", func(t *testing.T) {
		t.Log("\n  Testing valid template parsing...")

		template, err := ParseTemplateBytes([]byte(validTemplateYAML))
		if err != nil {
			t.Fatalf("❌ Failed to parse valid template: %v", err)
		}

		// Verify basic fields
		if template.ID != "test-template-001" {
			t.Errorf("Expected ID 'test-template-001', got '%s'", template.ID)
		}

		if template.Info.Name != "Test Template" {
			t.Errorf("Expected name 'Test Template', got '%s'", template.Info.Name)
		}

		if template.Info.Severity != types.SeverityHigh {
			t.Errorf("Expected severity 'high', got '%s'", template.Info.Severity)
		}

		// Verify LoadedAt was set
		if template.LoadedAt.IsZero() {
			t.Error("❌ LoadedAt was not set")
		}

		if time.Since(template.LoadedAt) > time.Second {
			t.Error("❌ LoadedAt timestamp seems incorrect (too old)")
		}

		// Verify detection config
		if template.Detection.Logic != types.LogicAll {
			t.Errorf("Expected logic 'all', got '%s'", template.Detection.Logic)
		}

		if len(template.Detection.Steps) != 2 {
			t.Errorf("Expected 2 steps, got %d", len(template.Detection.Steps))
		}

		// Verify first step
		if template.Detection.Steps[0].Type != "file_hash" {
			t.Errorf("Expected first step type 'file_hash', got '%s'", template.Detection.Steps[0].Type)
		}

		if template.Detection.Steps[0].Weight != 0.8 {
			t.Errorf("Expected first step weight 0.8, got %f", template.Detection.Steps[0].Weight)
		}

		// Verify step platforms
		if len(template.Detection.Steps[0].Platforms) != 2 {
			t.Errorf("Expected 2 platforms in first step, got %d", len(template.Detection.Steps[0].Platforms))
		}

		// Verify config was parsed
		if template.Detection.Steps[0].Config == nil {
			t.Error("❌ Step config is nil")
		}

		t.Log("  ✅ Valid template parsed successfully")
	})

	t.Run("minimal template with defaults", func(t *testing.T) {
		t.Log("\n  Testing minimal template with default values...")

		template, err := ParseTemplateBytes([]byte(minimalTemplateYAML))
		if err != nil {
			t.Fatalf("❌ Failed to parse minimal template: %v", err)
		}

		// Verify default logic was set
		if template.Detection.Logic != types.LogicAll {
			t.Errorf("Expected default logic 'all', got '%s'", template.Detection.Logic)
		}

		// Verify default weight was set
		if template.Detection.Steps[0].Weight != 1.0 {
			t.Errorf("Expected default weight 1.0, got %f", template.Detection.Steps[0].Weight)
		}

		t.Log("  ✅ Minimal template with defaults parsed successfully")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		t.Log("\n  Testing invalid YAML...")

		_, err := ParseTemplateBytes([]byte(invalidYAML))
		if err == nil {
			t.Fatal("❌ Expected error for invalid YAML, got nil")
		}

		if !strings.Contains(err.Error(), "unmarshal") {
			t.Errorf("Expected error to mention 'unmarshal', got: %v", err)
		}

		t.Logf("  ✅ Invalid YAML rejected with error: %v", err)
	})

	t.Run("empty data", func(t *testing.T) {
		t.Log("\n  Testing empty data...")

		_, err := ParseTemplateBytes([]byte{})
		if err == nil {
			t.Fatal("❌ Expected error for empty data, got nil")
		}

		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Expected error about empty data, got: %v", err)
		}

		t.Logf("  ✅ Empty data rejected with error: %v", err)
	})

	t.Log("\n✅ ParseTemplateBytes() tests completed")
}

func TestParseTemplate(t *testing.T) {
	t.Log("\n🔍 Testing ParseTemplate()...")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("valid template file", func(t *testing.T) {
		t.Log("\n  Testing valid template file...")

		// Create a test file
		testFile := filepath.Join(tmpDir, "valid-template.yaml")
		if err := os.WriteFile(testFile, []byte(validTemplateYAML), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Parse the file
		template, err := ParseTemplate(testFile)
		if err != nil {
			t.Fatalf("❌ Failed to parse template file: %v", err)
		}

		// Verify FilePath was set
		if template.FilePath != testFile {
			t.Errorf("Expected FilePath '%s', got '%s'", testFile, template.FilePath)
		}

		// Verify LoadedAt was set
		if template.LoadedAt.IsZero() {
			t.Error("❌ LoadedAt was not set")
		}

		// Verify content was parsed correctly
		if template.ID != "test-template-001" {
			t.Errorf("Expected ID 'test-template-001', got '%s'", template.ID)
		}

		t.Logf("  ✅ Valid template file parsed successfully: %s", testFile)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Log("\n  Testing file not found...")

		nonExistentFile := filepath.Join(tmpDir, "nonexistent.yaml")

		_, err := ParseTemplate(nonExistentFile)
		if err == nil {
			t.Fatal("❌ Expected error for non-existent file, got nil")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected error to mention 'not found', got: %v", err)
		}

		t.Logf("  ✅ Non-existent file rejected with error: %v", err)
	})

	t.Run("empty path", func(t *testing.T) {
		t.Log("\n  Testing empty path...")

		_, err := ParseTemplate("")
		if err == nil {
			t.Fatal("❌ Expected error for empty path, got nil")
		}

		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Expected error about empty path, got: %v", err)
		}

		t.Logf("  ✅ Empty path rejected with error: %v", err)
	})

	t.Run("invalid YAML file", func(t *testing.T) {
		t.Log("\n  Testing invalid YAML file...")

		// Create a file with invalid YAML
		invalidFile := filepath.Join(tmpDir, "invalid.yaml")
		if err := os.WriteFile(invalidFile, []byte(invalidYAML), 0644); err != nil {
			t.Fatalf("Failed to create invalid test file: %v", err)
		}

		_, err := ParseTemplate(invalidFile)
		if err == nil {
			t.Fatal("❌ Expected error for invalid YAML file, got nil")
		}

		if !strings.Contains(err.Error(), "failed to parse") {
			t.Errorf("Expected error to mention 'failed to parse', got: %v", err)
		}

		t.Logf("  ✅ Invalid YAML file rejected with error: %v", err)
	})

	t.Run("unreadable file", func(t *testing.T) {
		t.Log("\n  Testing unreadable file...")

		// Create a file with no read permissions
		unreadableFile := filepath.Join(tmpDir, "unreadable.yaml")
		if err := os.WriteFile(unreadableFile, []byte(validTemplateYAML), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Remove read permissions
		if err := os.Chmod(unreadableFile, 0200); err != nil {
			t.Logf("⚠️  Could not set permissions (may be running as root): %v", err)
			t.Skip("Skipping permission test")
		}

		// Ensure we can't read it
		_, readErr := os.ReadFile(unreadableFile)
		if readErr == nil {
			t.Skip("⚠️  File is still readable (may be running as root), skipping permission test")
		}

		// Try to parse
		_, err := ParseTemplate(unreadableFile)
		if err == nil {
			t.Fatal("❌ Expected error for unreadable file, got nil")
		}

		if !strings.Contains(err.Error(), "permission") && !strings.Contains(err.Error(), "failed to read") {
			t.Logf("⚠️  Got error but not permission-related: %v", err)
		} else {
			t.Logf("  ✅ Unreadable file rejected with error: %v", err)
		}

		// Restore permissions for cleanup
		os.Chmod(unreadableFile, 0644)
	})

	t.Log("\n✅ ParseTemplate() tests completed")
}

func TestParseTemplateWeightDefaults(t *testing.T) {
	t.Log("\n🔍 Testing weight default values...")

	// Template with explicit and missing weights
	yamlWithMixedWeights := `
id: weight-test
info:
  name: Weight Test
  severity: low
  description: Testing weight defaults
detection:
  steps:
    - type: file_hash
      weight: 0.5
      config:
        path: /test1
    - type: file_content
      config:
        path: /test2
`

	template, err := ParseTemplateBytes([]byte(yamlWithMixedWeights))
	if err != nil {
		t.Fatalf("❌ Failed to parse template: %v", err)
	}

	// First step should have explicit weight
	if template.Detection.Steps[0].Weight != 0.5 {
		t.Errorf("Expected explicit weight 0.5, got %f", template.Detection.Steps[0].Weight)
	}

	// Second step should have default weight
	if template.Detection.Steps[1].Weight != 1.0 {
		t.Errorf("Expected default weight 1.0, got %f", template.Detection.Steps[1].Weight)
	}

	t.Log("  ✅ Weight defaults applied correctly")
}

func TestParseTemplateLogicDefault(t *testing.T) {
	t.Log("\n🔍 Testing detection logic default value...")

	// Template without explicit logic
	yamlWithoutLogic := `
id: logic-test
info:
  name: Logic Test
  severity: info
  description: Testing logic default
detection:
  steps:
    - type: file_hash
      config:
        path: /test
`

	template, err := ParseTemplateBytes([]byte(yamlWithoutLogic))
	if err != nil {
		t.Fatalf("❌ Failed to parse template: %v", err)
	}

	// Should default to "all" logic
	if template.Detection.Logic != types.LogicAll {
		t.Errorf("Expected default logic 'all', got '%s'", template.Detection.Logic)
	}

	t.Log("  ✅ Detection logic defaults to 'all' correctly")
}

