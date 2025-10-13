package script

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
)

// ScriptRepository manages vulnerability detection scripts with versioning and validation
type ScriptRepository struct {
	logger       *zap.Logger
	basePath     string
	manifestPath string
	manifest     *ScriptManifest
}

// ScriptManifest represents the repository manifest with version tracking
type ScriptManifest struct {
	Version    string                             `json:"version"`
	UpdatedAt  time.Time                          `json:"updated_at"`
	Scripts    map[string]*detect.DetectionScript `json:"scripts"`
	Checksums  map[string]string                  `json:"checksums"`
	Signatures map[string]string                  `json:"signatures,omitempty"`
}

// ScriptValidationResult represents script validation results
type ScriptValidationResult struct {
	Valid    bool                    `json:"valid"`
	Errors   []string                `json:"errors,omitempty"`
	Warnings []string                `json:"warnings,omitempty"`
	Script   *detect.DetectionScript `json:"script,omitempty"`
}

// RepositoryValidationResult represents the validation results for the entire repository
type RepositoryValidationResult struct {
	Valid          bool                               `json:"valid"`
	TotalScripts   int                                `json:"total_scripts"`
	ValidScripts   int                                `json:"valid_scripts"`
	InvalidScripts int                                `json:"invalid_scripts"`
	Scripts        map[string]*ScriptValidationResult `json:"scripts"`
}

// NewScriptRepository creates a new script repository manager
func NewScriptRepository(basePath string, logger *zap.Logger) *ScriptRepository {
	return &ScriptRepository{
		logger:       logger,
		basePath:     basePath,
		manifestPath: filepath.Join(basePath, "manifest.json"),
	}
}

// LoadRepository loads the script repository and validates all scripts
func (sr *ScriptRepository) LoadRepository() error {
	sr.logger.Info("Loading script repository", zap.String("path", sr.basePath))

	// Load or create manifest
	if err := sr.loadManifest(); err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Scan directory for scripts
	scripts, err := sr.scanForScripts()
	if err != nil {
		return fmt.Errorf("failed to scan for scripts: %w", err)
	}

	// Validate all scripts
	for scriptPath, script := range scripts {
		if err := sr.validateScript(scriptPath, script); err != nil {
			sr.logger.Warn("Script validation failed",
				zap.String("script", scriptPath),
				zap.Error(err))
		}
	}

	sr.logger.Info("Script repository loaded successfully",
		zap.Int("script_count", len(sr.manifest.Scripts)))

	return nil
}

// loadManifest loads the repository manifest or creates a new one
func (sr *ScriptRepository) loadManifest() error {
	if _, err := os.Stat(sr.manifestPath); os.IsNotExist(err) {
		// Create new manifest
		sr.logger.Info("Script manifest file not found, creating new manifest",
			zap.String("file", sr.manifestPath))
		sr.manifest = &ScriptManifest{
			Version:   "1.0.0",
			UpdatedAt: time.Now(),
			Scripts:   make(map[string]*detect.DetectionScript),
			Checksums: make(map[string]string),
		}
		return sr.saveManifest()
	}

	data, err := os.ReadFile(sr.manifestPath)
	if err != nil {
		sr.logger.Error("Failed to read script manifest file, creating new manifest",
			zap.String("file", sr.manifestPath),
			zap.Error(err))
		// Create new manifest on read error
		sr.manifest = &ScriptManifest{
			Version:   "1.0.0",
			UpdatedAt: time.Now(),
			Scripts:   make(map[string]*detect.DetectionScript),
			Checksums: make(map[string]string),
		}
		return sr.saveManifest()
	}

	var manifest ScriptManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		// Backup the corrupted file
		backupName := fmt.Sprintf("%s.bak-%d", sr.manifestPath, time.Now().Unix())
		if backupErr := os.WriteFile(backupName, data, 0644); backupErr != nil {
			sr.logger.Error("Failed to backup corrupted script manifest file",
				zap.String("file", sr.manifestPath),
				zap.String("backup", backupName),
				zap.Error(backupErr))
		} else {
			sr.logger.Warn("Backed up corrupted script manifest file",
				zap.String("file", sr.manifestPath),
				zap.String("backup", backupName))
		}

		sr.logger.Error("Failed to parse script manifest file, creating new manifest",
			zap.String("file", sr.manifestPath),
			zap.Error(err))

		// Create new manifest on parse error
		sr.manifest = &ScriptManifest{
			Version:   "1.0.0",
			UpdatedAt: time.Now(),
			Scripts:   make(map[string]*detect.DetectionScript),
			Checksums: make(map[string]string),
		}
		return sr.saveManifest()
	}

	// Validate the parsed manifest structure
	if manifest.Scripts == nil {
		manifest.Scripts = make(map[string]*detect.DetectionScript)
	}
	if manifest.Checksums == nil {
		manifest.Checksums = make(map[string]string)
	}
	if manifest.Version == "" {
		manifest.Version = "1.0.0"
	}

	sr.manifest = &manifest
	sr.logger.Info("Successfully loaded script manifest",
		zap.String("file", sr.manifestPath),
		zap.Int("scripts", len(manifest.Scripts)))

	return nil
}

// saveManifest saves the current manifest to disk
func (sr *ScriptRepository) saveManifest() error {
	sr.manifest.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(sr.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(sr.manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// scanForScripts scans the repository directory for script files
func (sr *ScriptRepository) scanForScripts() (map[string]*detect.DetectionScript, error) {
	scripts := make(map[string]*detect.DetectionScript)

	err := filepath.Walk(sr.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check for script files
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".ps1" || ext == ".sh" || ext == ".py" {
			relPath, err := filepath.Rel(sr.basePath, path)
			if err != nil {
				return err
			}

			// Skip manifest file
			if relPath == "manifest.json" {
				return nil
			}

			script, err := sr.extractScriptMetadata(path)
			if err != nil {
				sr.logger.Warn("Failed to extract metadata",
					zap.String("script", relPath),
					zap.Error(err))
				return nil
			}

			scripts[relPath] = script
		}

		return nil
	})

	return scripts, err
}

// extractScriptMetadata extracts metadata from script comments/headers
func (sr *ScriptRepository) extractScriptMetadata(scriptPath string) (*detect.DetectionScript, error) {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}

	// Calculate checksum
	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	script := &detect.DetectionScript{
		Path:     scriptPath,
		Checksum: checksum,
		LoadedAt: time.Now(),
		Timeout:  30 * time.Second, // Default timeout
	}

	// Determine platform and language from file path and extension
	sr.setPlatformFromPath(script, scriptPath)
	sr.setLanguageFromExtension(script, scriptPath)

	// Extract metadata based on file type
	ext := strings.ToLower(filepath.Ext(scriptPath))
	switch ext {
	case ".ps1":
		return sr.extractPowerShellMetadata(content, script)
	case ".sh":
		return sr.extractBashMetadata(content, script)
	case ".py":
		return sr.extractPythonMetadata(content, script)
	default:
		return nil, fmt.Errorf("unsupported script type: %s", ext)
	}
}

// setPlatformFromPath determines platform from file path
func (sr *ScriptRepository) setPlatformFromPath(script *detect.DetectionScript, scriptPath string) {
	if strings.Contains(scriptPath, "windows") {
		script.Platform = "windows"
	} else if strings.Contains(scriptPath, "linux") {
		script.Platform = "linux"
	} else if strings.Contains(scriptPath, "darwin") || strings.Contains(scriptPath, "macos") {
		script.Platform = "darwin"
	} else {
		// Default based on extension
		ext := strings.ToLower(filepath.Ext(scriptPath))
		if ext == ".ps1" {
			script.Platform = "windows"
		} else {
			script.Platform = "linux"
		}
	}
}

// setLanguageFromExtension determines language from file extension
func (sr *ScriptRepository) setLanguageFromExtension(script *detect.DetectionScript, scriptPath string) {
	ext := strings.ToLower(filepath.Ext(scriptPath))
	switch ext {
	case ".ps1":
		script.Language = detect.ScriptLanguagePowerShell
	case ".sh":
		script.Language = detect.ScriptLanguageBash
	case ".py":
		script.Language = detect.ScriptLanguagePython
	default:
		script.Language = detect.ScriptLanguageShell
	}
}

// extractPowerShellMetadata extracts metadata from PowerShell comment blocks
func (sr *ScriptRepository) extractPowerShellMetadata(content []byte, script *detect.DetectionScript) (*detect.DetectionScript, error) {
	contentStr := string(content)

	// Extract from comment block
	patterns := map[string]*regexp.Regexp{
		"vulnerability": regexp.MustCompile(`(?i)\.VULNERABILITY\s+([^\r\n]+)`),
		"severity":      regexp.MustCompile(`(?i)\.SEVERITY\s+([^\r\n]+)`),
		"description":   regexp.MustCompile(`(?i)\.DESCRIPTION\s+([^\r\n]+)`),
		"author":        regexp.MustCompile(`(?i)\.AUTHOR\s+([^\r\n]+)`),
		"version":       regexp.MustCompile(`(?i)\.VERSION\s+([^\r\n]+)`),
		"synopsis":      regexp.MustCompile(`(?i)\.SYNOPSIS\s+([^\r\n]+)`),
	}

	for field, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(contentStr); len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			switch field {
			case "vulnerability":
				script.VulnerabilityID = value
			case "severity":
				script.Severity = sr.parseSeverity(value)
			case "description":
				script.Description = value
			case "author":
				script.Author = value
			case "version":
				script.Version = value
			case "synopsis":
				if script.Name == "" {
					script.Name = value
				}
			}
		}
	}

	// Extract parameters
	paramPattern := regexp.MustCompile(`(?i)\.PARAMETER\s+(\w+)\s+([^\r\n]+)`)
	paramMatches := paramPattern.FindAllStringSubmatch(contentStr, -1)
	for _, match := range paramMatches {
		if len(match) >= 3 {
			param := detect.ScriptParameter{
				Name:        match[1],
				Description: strings.TrimSpace(match[2]),
				Type:        "string", // Default type
				Required:    false,    // Default to optional
			}
			script.Parameters = append(script.Parameters, param)
		}
	}

	return script, nil
}

// extractBashMetadata extracts metadata from Bash script comments
func (sr *ScriptRepository) extractBashMetadata(content []byte, script *detect.DetectionScript) (*detect.DetectionScript, error) {
	contentStr := string(content)

	// Extract from comment variables
	patterns := map[string]*regexp.Regexp{
		"vulnerability": regexp.MustCompile(`VULNERABILITY_ID=["']?([^"'\r\n]+)["']?`),
		"severity":      regexp.MustCompile(`SEVERITY=["']?([^"'\r\n]+)["']?`),
		"description":   regexp.MustCompile(`DESCRIPTION=["']?([^"'\r\n]+)["']?`),
		"author":        regexp.MustCompile(`AUTHOR=["']?([^"'\r\n]+)["']?`),
		"version":       regexp.MustCompile(`VERSION=["']?([^"'\r\n]+)["']?`),
	}

	for field, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(contentStr); len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			switch field {
			case "vulnerability":
				script.VulnerabilityID = value
			case "severity":
				script.Severity = sr.parseSeverity(value)
			case "description":
				script.Description = value
			case "author":
				script.Author = value
			case "version":
				script.Version = value
			}
		}
	}

	// Set name from file name if not provided
	if script.Name == "" {
		script.Name = strings.TrimSuffix(filepath.Base(script.Path), filepath.Ext(script.Path))
	}

	return script, nil
}

// extractPythonMetadata extracts metadata from Python docstrings
func (sr *ScriptRepository) extractPythonMetadata(content []byte, script *detect.DetectionScript) (*detect.DetectionScript, error) {
	contentStr := string(content)

	// Extract from docstring
	docstringPattern := regexp.MustCompile(`(?s)"""([^"]+)"""`)
	if matches := docstringPattern.FindStringSubmatch(contentStr); len(matches) > 1 {
		docstring := matches[1]

		patterns := map[string]*regexp.Regexp{
			"vulnerability": regexp.MustCompile(`(?i)vulnerability:?\s*([^\r\n]+)`),
			"severity":      regexp.MustCompile(`(?i)severity:?\s*([^\r\n]+)`),
			"description":   regexp.MustCompile(`(?i)description:?\s*([^\r\n]+)`),
			"author":        regexp.MustCompile(`(?i)author:?\s*([^\r\n]+)`),
			"version":       regexp.MustCompile(`(?i)version:?\s*([^\r\n]+)`),
		}

		for field, pattern := range patterns {
			if matches := pattern.FindStringSubmatch(docstring); len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				switch field {
				case "vulnerability":
					script.VulnerabilityID = value
				case "severity":
					script.Severity = sr.parseSeverity(value)
				case "description":
					script.Description = value
				case "author":
					script.Author = value
				case "version":
					script.Version = value
				}
			}
		}
	}

	// Set name from file name if not provided
	if script.Name == "" {
		script.Name = strings.TrimSuffix(filepath.Base(script.Path), filepath.Ext(script.Path))
	}

	return script, nil
}

// parseSeverity converts string severity to SeverityLevel
func (sr *ScriptRepository) parseSeverity(severity string) detect.SeverityLevel {
	switch strings.ToLower(severity) {
	case "low":
		return detect.SeverityLevelLow
	case "medium":
		return detect.SeverityLevelMedium
	case "high":
		return detect.SeverityLevelHigh
	case "critical":
		return detect.SeverityLevelCritical
	case "info":
		return detect.SeverityLevelInfo
	default:
		return detect.SeverityLevelMedium // Default
	}
}

// validateScript validates a script file and its metadata
func (sr *ScriptRepository) validateScript(scriptPath string, script *detect.DetectionScript) error {
	// Basic validation
	if script.VulnerabilityID == "" {
		return fmt.Errorf("missing vulnerability ID")
	}

	if script.Severity == "" {
		return fmt.Errorf("missing severity level")
	}

	// Verify checksum
	fullPath := filepath.Join(sr.basePath, scriptPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read script for checksum verification: %w", err)
	}

	hash := sha256.Sum256(content)
	expectedChecksum := hex.EncodeToString(hash[:])
	if script.Checksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, script.Checksum)
	}

	// Update manifest
	sr.manifest.Scripts[scriptPath] = script
	sr.manifest.Checksums[scriptPath] = script.Checksum

	return nil
}

// GetScriptsByPlatform returns scripts filtered by platform
func (sr *ScriptRepository) GetScriptsByPlatform(platform string) map[string]*detect.DetectionScript {
	filtered := make(map[string]*detect.DetectionScript)

	for path, script := range sr.manifest.Scripts {
		if script.Platform == platform {
			filtered[path] = script
		}
	}

	return filtered
}

// GetScriptsBySeverity returns scripts filtered by severity level
func (sr *ScriptRepository) GetScriptsBySeverity(severity detect.SeverityLevel) map[string]*detect.DetectionScript {
	filtered := make(map[string]*detect.DetectionScript)

	for path, script := range sr.manifest.Scripts {
		if script.Severity == severity {
			filtered[path] = script
		}
	}

	return filtered
}

// ValidateRepository validates the entire repository
func (sr *ScriptRepository) ValidateRepository() (*RepositoryValidationResult, error) {
	result := &RepositoryValidationResult{
		Valid:        true,
		Scripts:      make(map[string]*ScriptValidationResult),
		TotalScripts: len(sr.manifest.Scripts),
	}

	for scriptPath, script := range sr.manifest.Scripts {
		scriptResult := &ScriptValidationResult{
			Valid:  true,
			Script: script,
		}

		if err := sr.validateScript(scriptPath, script); err != nil {
			scriptResult.Valid = false
			scriptResult.Errors = append(scriptResult.Errors, err.Error())
			result.Valid = false
			result.InvalidScripts++
		} else {
			result.ValidScripts++
		}

		result.Scripts[scriptPath] = scriptResult
	}

	return result, nil
}

// AddScript adds a new script to the repository with validation
func (sr *ScriptRepository) AddScript(scriptPath string, content []byte) error {
	fullPath := filepath.Join(sr.basePath, scriptPath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write script file
	if err := os.WriteFile(fullPath, content, 0755); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	// Extract and validate metadata
	script, err := sr.extractScriptMetadata(fullPath)
	if err != nil {
		os.Remove(fullPath) // Clean up on error
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	if err := sr.validateScript(scriptPath, script); err != nil {
		os.Remove(fullPath) // Clean up on error
		return fmt.Errorf("script validation failed: %w", err)
	}

	// Save updated manifest
	return sr.saveManifest()
}

// UpdateChecksum updates the checksum for a script in the manifest
func (sr *ScriptRepository) UpdateChecksum(scriptPath string) error {
	fullPath := filepath.Join(sr.basePath, scriptPath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	if script, exists := sr.manifest.Scripts[scriptPath]; exists {
		script.Checksum = checksum
		script.LoadedAt = time.Now()
	}

	sr.manifest.Checksums[scriptPath] = checksum

	return sr.saveManifest()
}

// GetAllScripts returns all scripts in the repository
func (sr *ScriptRepository) GetAllScripts() map[string]*detect.DetectionScript {
	return sr.manifest.Scripts
}
