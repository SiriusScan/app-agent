package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetTemplateBaseDir returns the platform-specific template directory.
// Priority: SIRIUS_TEMPLATE_DIR env var > os.UserConfigDir() > temp fallback
//
// Platform-specific paths:
//   - Windows: %APPDATA%\sirius-agent\templates (e.g., C:\Users\John\AppData\Roaming\sirius-agent\templates)
//   - macOS: ~/Library/Application Support/sirius-agent/templates
//   - Linux: ~/.config/sirius-agent/templates
func GetTemplateBaseDir() (string, error) {
	// Check environment variable override first (highest priority)
	if dir := os.Getenv("SIRIUS_TEMPLATE_DIR"); dir != "" {
		return dir, nil
	}

	// Use os.UserConfigDir() for cross-platform support
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to temp directory if UserConfigDir fails
		tempDir := filepath.Join(os.TempDir(), "sirius-agent", "templates")
		return tempDir, nil
	}

	return filepath.Join(configDir, "sirius-agent", "templates"), nil
}

// EnsureDirectoryStructure creates the template directory layout.
// Creates: builtin/, custom/, server/, cache/ subdirectories.
func EnsureDirectoryStructure(baseDir string) error {
	dirs := []string{"builtin", "custom", "server", "cache"}

	for _, dir := range dirs {
		path := filepath.Join(baseDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	return nil
}

// GetCustomTemplateDir returns the custom template directory path
func GetCustomTemplateDir() (string, error) {
	baseDir, err := GetTemplateBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "custom"), nil
}

// GetServerTemplateDir returns the server-synced template directory path
func GetServerTemplateDir() (string, error) {
	baseDir, err := GetTemplateBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "server"), nil
}

// GetCacheDir returns the cache directory path
func GetCacheDir() (string, error) {
	baseDir, err := GetTemplateBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "cache"), nil
}
