package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetTemplateBaseDir returns the platform-specific template directory.
// Priority: SIRIUS_TEMPLATE_DIR env var > agent cache directory
//
// This MUST match the agent sync manager's cache directory to ensure
// templates synced from the server are properly discovered.
//
// Platform-specific paths:
//   - Windows: %ProgramData%\Sirius\template-cache
//   - macOS: ~/Library/Application Support/Sirius/template-cache
//   - Linux: ~/.local/share/sirius/template-cache (user) or /var/lib/sirius/template-cache (root)
func GetTemplateBaseDir() (string, error) {
	// Check environment variable override first (highest priority)
	if dir := os.Getenv("SIRIUS_TEMPLATE_DIR"); dir != "" {
		return dir, nil
	}

	// Import the agent path function to ensure consistency
	// This is the same path used by the agent sync manager
	return getAgentTemplateBaseDirForStorage(), nil
}

// getAgentTemplateBaseDirForStorage returns the agent cache directory
// This duplicates logic from internal/template/agent/paths.go to avoid circular imports
func getAgentTemplateBaseDirForStorage() string {
	switch runtime.GOOS {
	case "windows":
		// Use ProgramData on Windows
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "Sirius", "template-cache")
	case "darwin":
		// Use Application Support on macOS
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE") // Fallback for some systems
		}
		return filepath.Join(home, "Library", "Application Support", "Sirius", "template-cache")
	default: // linux and other unix-like systems
		// Use /var/lib/sirius/template-cache for system-wide installation
		// or ~/.local/share/sirius/template-cache for user installation
		if os.Getuid() == 0 {
			// Running as root, use system directory
			return "/var/lib/sirius/template-cache"
		}
		// Running as user, use user directory
		home := os.Getenv("HOME")
		if home == "" {
			home = "/tmp" // Fallback
		}
		return filepath.Join(home, ".local", "share", "sirius", "template-cache")
	}
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
