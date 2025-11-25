package storage

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

func TestLoadBuiltinTemplates(t *testing.T) {
	logger := zap.NewNop()

	tempDir := filepath.Join(os.TempDir(), "sirius-agent-builtin-test")
	defer os.RemoveAll(tempDir)

	manager, err := NewManagerWithBaseDir(tempDir, logger)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() failed: %v", err)
	}

	ctx := context.Background()
	builtins, err := manager.loadBuiltinTemplates(ctx)
	if err != nil {
		t.Fatalf("loadBuiltinTemplates() failed: %v", err)
	}

	// Built-in templates are currently disabled by design
	// They return an empty list to use only server and custom templates
	if len(builtins) != 0 {
		t.Errorf("Expected 0 built-in templates (disabled by design), got %d", len(builtins))
	}

	// Verify templates are valid (if any are returned in the future)
	for _, tmpl := range builtins {
		if tmpl.ID == "" {
			t.Error("Built-in template has empty ID")
		}
		if tmpl.Info.Name == "" {
			t.Error("Built-in template has empty name")
		}
		if tmpl.FilePath == "" {
			t.Error("Built-in template has empty file path")
		}

		t.Logf("✅ Built-in template: %s (%s) from %s", tmpl.ID, tmpl.Info.Name, tmpl.FilePath)
	}

	t.Logf("✅ Built-in templates disabled as expected, loaded %d templates", len(builtins))
}

func TestBuiltinTemplatesPrecedence(t *testing.T) {
	logger := zap.NewNop()

	tempDir := filepath.Join(os.TempDir(), "sirius-agent-precedence-test")
	defer os.RemoveAll(tempDir)

	manager, err := NewManagerWithBaseDir(tempDir, logger)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() failed: %v", err)
	}

	ctx := context.Background()

	// First, get built-in templates
	builtins, err := manager.loadBuiltinTemplates(ctx)
	if err != nil {
		t.Fatalf("loadBuiltinTemplates() failed: %v", err)
	}

	if len(builtins) == 0 {
		t.Skip("No built-in templates to test precedence")
	}

	// Pick first built-in template ID
	builtinID := builtins[0].ID
	builtinName := builtins[0].Info.Name

	// Create custom template with same ID but different name
	customDir := filepath.Join(tempDir, "custom")
	customTemplate := `id: ` + builtinID + `
info:
  name: Custom Override Template
  severity: high
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /test/override
        hash: override123
`
	if err := os.WriteFile(filepath.Join(customDir, "override.yaml"), []byte(customTemplate), 0644); err != nil {
		t.Fatalf("Failed to write custom template: %v", err)
	}

	// Discover all templates
	all, err := manager.DiscoverTemplates(ctx)
	if err != nil {
		t.Fatalf("DiscoverTemplates() failed: %v", err)
	}

	// Find the template with built-in ID
	var found *types.Template
	for _, tmpl := range all {
		if tmpl.ID == builtinID {
			found = tmpl
			break
		}
	}

	if found == nil {
		t.Fatalf("Template with ID %q not found", builtinID)
	}

	// Should be the custom one, not the built-in
	if found.Info.Name == builtinName {
		t.Errorf("Custom template did not override built-in (still has name: %s)", found.Info.Name)
	}

	if found.Info.Name != "Custom Override Template" {
		t.Errorf("Expected custom template name 'Custom Override Template', got %q", found.Info.Name)
	}

	t.Logf("✅ Custom template correctly overrides built-in template (ID: %s)", builtinID)
}

func TestEmbeddedTemplatesInBinary(t *testing.T) {
	// This test verifies that templates are actually embedded in the binary
	// by checking the embed.FS is not empty

	if (embeddedTemplates == embed.FS{}) {
		t.Fatal("Embedded templates FS is empty - templates not embedded")
	}

	// Try to read a file
	entries, err := embeddedTemplates.ReadDir("templates/builtin")
	if err != nil {
		t.Fatalf("Failed to read embedded template directory: %v", err)
	}

	// Should have at least 1 template file
	if len(entries) < 1 {
		t.Errorf("Expected at least 1 embedded template file, got %d", len(entries))
	}

	for _, entry := range entries {
		t.Logf("✅ Embedded file: %s", entry.Name())
	}

	t.Logf("✅ %d templates are properly embedded in the binary", len(entries))
}
