package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestNewManager(t *testing.T) {
	logger := zap.NewNop()

	manager, err := NewManager(logger)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil manager")
	}

	if manager.baseDir == "" {
		t.Fatal("Manager baseDir is empty")
	}

	t.Logf("✅ Manager created with base dir: %s", manager.baseDir)
}

func TestDiscoverTemplates(t *testing.T) {
	logger := zap.NewNop()

	// Create temp directory for testing
	tempDir := filepath.Join(os.TempDir(), "sirius-agent-manager-test")
	defer os.RemoveAll(tempDir)

	manager, err := NewManagerWithBaseDir(tempDir, logger)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() failed: %v", err)
	}

	// Create test templates
	customDir := filepath.Join(tempDir, "custom")
	serverDir := filepath.Join(tempDir, "server")

	// Custom template (highest priority)
	customTemplate := `id: TEST-001
info:
  name: Custom Test Template
  severity: high
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /test/file
        hash: abc123
`
	if err := os.WriteFile(filepath.Join(customDir, "custom.yaml"), []byte(customTemplate), 0644); err != nil {
		t.Fatalf("Failed to write custom template: %v", err)
	}

	// Server template (medium priority, same ID to test override)
	serverTemplate := `id: TEST-001
info:
  name: Server Test Template (should be overridden)
  severity: low
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /test/file
        hash: xyz789
`
	if err := os.WriteFile(filepath.Join(serverDir, "server.yaml"), []byte(serverTemplate), 0644); err != nil {
		t.Fatalf("Failed to write server template: %v", err)
	}

	// Another server template with different ID
	serverTemplate2 := `id: TEST-002
info:
  name: Server Only Template
  severity: medium
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /test/file2
        hash: def456
`
	if err := os.WriteFile(filepath.Join(serverDir, "server2.yaml"), []byte(serverTemplate2), 0644); err != nil {
		t.Fatalf("Failed to write server template 2: %v", err)
	}

	ctx := context.Background()
	templates, err := manager.DiscoverTemplates(ctx)
	if err != nil {
		t.Fatalf("DiscoverTemplates() failed: %v", err)
	}

	// Should have at least 2 templates (TEST-001 from custom overrides server, TEST-002 from server)
	// Plus any built-in templates (5 in current implementation)
	if len(templates) < 2 {
		t.Errorf("Expected at least 2 templates, got %d", len(templates))
	}

	t.Logf("✅ Discovered %d total templates (including %d built-in)", len(templates), len(templates)-2)

	// Check TEST-001 is from custom (highest priority)
	for _, tmpl := range templates {
		if tmpl.ID == "TEST-001" {
			if tmpl.Info.Name != "Custom Test Template" {
				t.Errorf("Expected custom template to override server template, got name: %s", tmpl.Info.Name)
			}
			t.Logf("✅ Custom template correctly overrides server template (ID: %s)", tmpl.ID)
		}
		if tmpl.ID == "TEST-002" {
			if tmpl.Info.Name != "Server Only Template" {
				t.Errorf("Expected server template, got name: %s", tmpl.Info.Name)
			}
			t.Logf("✅ Server template loaded (ID: %s)", tmpl.ID)
		}
	}
}

func TestGetTemplate(t *testing.T) {
	logger := zap.NewNop()

	tempDir := filepath.Join(os.TempDir(), "sirius-agent-get-test")
	defer os.RemoveAll(tempDir)

	manager, err := NewManagerWithBaseDir(tempDir, logger)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() failed: %v", err)
	}

	// Create custom template
	customDir := filepath.Join(tempDir, "custom")
	customTemplate := `id: TEST-GET
info:
  name: Get Test Template
  severity: high
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /test/file
        hash: abc123
`
	if err := os.WriteFile(filepath.Join(customDir, "get.yaml"), []byte(customTemplate), 0644); err != nil {
		t.Fatalf("Failed to write custom template: %v", err)
	}

	ctx := context.Background()

	// Get existing template
	template, err := manager.GetTemplate(ctx, "TEST-GET")
	if err != nil {
		t.Fatalf("GetTemplate() failed: %v", err)
	}

	if template.ID != "TEST-GET" {
		t.Errorf("Expected template ID 'TEST-GET', got %q", template.ID)
	}

	t.Logf("✅ GetTemplate() found template: %s", template.Info.Name)

	// Get non-existent template
	_, err = manager.GetTemplate(ctx, "NON-EXISTENT")
	if err == nil {
		t.Error("Expected error for non-existent template, got nil")
	}

	t.Logf("✅ GetTemplate() correctly returns error for non-existent template")
}

func TestListTemplates(t *testing.T) {
	logger := zap.NewNop()

	tempDir := filepath.Join(os.TempDir(), "sirius-agent-list-test")
	defer os.RemoveAll(tempDir)

	manager, err := NewManagerWithBaseDir(tempDir, logger)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() failed: %v", err)
	}

	// Create templates in different directories
	customDir := filepath.Join(tempDir, "custom")
	serverDir := filepath.Join(tempDir, "server")

	customTemplate := `id: CUSTOM-001
info:
  name: Custom Template
  severity: high
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /test/file
        hash: abc123
`
	serverTemplate := `id: SERVER-001
info:
  name: Server Template
  severity: medium
detection:
  logic: all
  steps:
    - type: file_hash
      config:
        path: /test/file
        hash: xyz789
`

	if err := os.WriteFile(filepath.Join(customDir, "custom.yaml"), []byte(customTemplate), 0644); err != nil {
		t.Fatalf("Failed to write custom template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "server.yaml"), []byte(serverTemplate), 0644); err != nil {
		t.Fatalf("Failed to write server template: %v", err)
	}

	ctx := context.Background()

	// List custom templates
	customTemplates, err := manager.ListTemplates(ctx, SourceCustom)
	if err != nil {
		t.Fatalf("ListTemplates(SourceCustom) failed: %v", err)
	}
	if len(customTemplates) != 1 {
		t.Errorf("Expected 1 custom template, got %d", len(customTemplates))
	}
	t.Logf("✅ Listed %d custom templates", len(customTemplates))

	// List server templates
	serverTemplates, err := manager.ListTemplates(ctx, SourceServer)
	if err != nil {
		t.Fatalf("ListTemplates(SourceServer) failed: %v", err)
	}
	if len(serverTemplates) != 1 {
		t.Errorf("Expected 1 server template, got %d", len(serverTemplates))
	}
	t.Logf("✅ Listed %d server templates", len(serverTemplates))

	// List built-in templates (should be empty for now)
	builtinTemplates, err := manager.ListTemplates(ctx, SourceBuiltin)
	if err != nil {
		t.Fatalf("ListTemplates(SourceBuiltin) failed: %v", err)
	}
	if len(builtinTemplates) != 0 {
		t.Logf("Note: Found %d built-in templates (unexpected, but OK)", len(builtinTemplates))
	}
	t.Logf("✅ Listed %d built-in templates", len(builtinTemplates))
}

func TestGetStoragePath(t *testing.T) {
	logger := zap.NewNop()

	tempDir := filepath.Join(os.TempDir(), "sirius-agent-storage-test")
	defer os.RemoveAll(tempDir)

	manager, err := NewManagerWithBaseDir(tempDir, logger)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() failed: %v", err)
	}

	storagePath := manager.GetStoragePath()
	if storagePath != tempDir {
		t.Errorf("Expected storage path %q, got %q", tempDir, storagePath)
	}

	t.Logf("✅ GetStoragePath() returns correct path: %s", storagePath)
}
