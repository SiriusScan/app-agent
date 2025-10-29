package agent

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetAgentTemplateCacheDir returns the OS-appropriate cache directory for agent templates
func GetAgentTemplateCacheDir() string {
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

// GetAgentTemplateCacheSubdirs returns the subdirectories within the cache
func GetAgentTemplateCacheSubdirs() map[string]string {
	baseDir := GetAgentTemplateCacheDir()
	return map[string]string{
		"standard":  filepath.Join(baseDir, "server"), // Changed from "standard" to "server" to match actual directory structure
		"custom":    filepath.Join(baseDir, "custom"),
		"manifest":  filepath.Join(baseDir, ".manifest.json"),
		"checksums": filepath.Join(baseDir, ".checksums.json"),
	}
}

// EnsureCacheDirectoryStructure creates the cache directory structure if it doesn't exist
func EnsureCacheDirectoryStructure() error {
	subdirs := GetAgentTemplateCacheSubdirs()

	// Create base directory
	baseDir := filepath.Dir(subdirs["standard"])
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	// Create subdirectories
	for name, path := range subdirs {
		if name == "manifest" || name == "checksums" {
			continue // Skip files
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

// GetCacheManifestPath returns the path to the cache manifest file
func GetCacheManifestPath() string {
	subdirs := GetAgentTemplateCacheSubdirs()
	return subdirs["manifest"]
}

// GetCacheChecksumsPath returns the path to the cache checksums file
func GetCacheChecksumsPath() string {
	subdirs := GetAgentTemplateCacheSubdirs()
	return subdirs["checksums"]
}

// GetStandardTemplatesPath returns the path to the standard templates directory
func GetStandardTemplatesPath() string {
	subdirs := GetAgentTemplateCacheSubdirs()
	return subdirs["standard"]
}

// GetCustomTemplatesPath returns the path to the custom templates directory
func GetCustomTemplatesPath() string {
	subdirs := GetAgentTemplateCacheSubdirs()
	return subdirs["custom"]
}
