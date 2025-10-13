package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetTemplateBaseDir(t *testing.T) {
	t.Run("default_path", func(t *testing.T) {
		// Unset env var to test default behavior
		os.Unsetenv("SIRIUS_TEMPLATE_DIR")

		dir, err := GetTemplateBaseDir()
		if err != nil {
			t.Fatalf("GetTemplateBaseDir() failed: %v", err)
		}

		if dir == "" {
			t.Fatal("GetTemplateBaseDir() returned empty string")
		}

		// Should contain "sirius-agent/templates"
		if !strings.Contains(dir, "sirius-agent") || !strings.Contains(dir, "templates") {
			t.Errorf("Expected path to contain 'sirius-agent/templates', got: %s", dir)
		}

		t.Logf("✅ Default template directory: %s", dir)
	})

	t.Run("env_var_override", func(t *testing.T) {
		customPath := "/custom/template/path"
		os.Setenv("SIRIUS_TEMPLATE_DIR", customPath)
		defer os.Unsetenv("SIRIUS_TEMPLATE_DIR")

		dir, err := GetTemplateBaseDir()
		if err != nil {
			t.Fatalf("GetTemplateBaseDir() failed: %v", err)
		}

		if dir != customPath {
			t.Errorf("Expected %q, got %q", customPath, dir)
		}

		t.Logf("✅ Environment variable override works: %s", dir)
	})
}

func TestEnsureDirectoryStructure(t *testing.T) {
	// Create temp directory for testing
	tempDir := filepath.Join(os.TempDir(), "sirius-agent-test-"+t.Name())
	defer os.RemoveAll(tempDir)

	err := EnsureDirectoryStructure(tempDir)
	if err != nil {
		t.Fatalf("EnsureDirectoryStructure() failed: %v", err)
	}

	// Check that all subdirectories exist
	expectedDirs := []string{"builtin", "custom", "server", "cache"}
	for _, dir := range expectedDirs {
		path := filepath.Join(tempDir, dir)
		stat, err := os.Stat(path)
		if err != nil {
			t.Errorf("Directory %q was not created: %v", dir, err)
			continue
		}
		if !stat.IsDir() {
			t.Errorf("%q is not a directory", dir)
		}
	}

	t.Logf("✅ All directories created successfully in %s", tempDir)
}

func TestGetCustomTemplateDir(t *testing.T) {
	os.Unsetenv("SIRIUS_TEMPLATE_DIR")

	dir, err := GetCustomTemplateDir()
	if err != nil {
		t.Fatalf("GetCustomTemplateDir() failed: %v", err)
	}

	if !strings.HasSuffix(dir, filepath.Join("sirius-agent", "templates", "custom")) {
		t.Errorf("Expected path to end with 'sirius-agent/templates/custom', got: %s", dir)
	}

	t.Logf("✅ Custom template directory: %s", dir)
}

func TestGetServerTemplateDir(t *testing.T) {
	os.Unsetenv("SIRIUS_TEMPLATE_DIR")

	dir, err := GetServerTemplateDir()
	if err != nil {
		t.Fatalf("GetServerTemplateDir() failed: %v", err)
	}

	if !strings.HasSuffix(dir, filepath.Join("sirius-agent", "templates", "server")) {
		t.Errorf("Expected path to end with 'sirius-agent/templates/server', got: %s", dir)
	}

	t.Logf("✅ Server template directory: %s", dir)
}

func TestGetCacheDir(t *testing.T) {
	os.Unsetenv("SIRIUS_TEMPLATE_DIR")

	dir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() failed: %v", err)
	}

	if !strings.HasSuffix(dir, filepath.Join("sirius-agent", "templates", "cache")) {
		t.Errorf("Expected path to end with 'sirius-agent/templates/cache', got: %s", dir)
	}

	t.Logf("✅ Cache directory: %s", dir)
}
