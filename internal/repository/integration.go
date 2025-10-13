package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RepositoryIntegration manages integration between repository and agent components
type RepositoryIntegration struct {
	logger            *zap.Logger
	repositoryManager RepositoryManager
	config            *RepositoryConfiguration
	initialized       bool
}

// NewRepositoryIntegration creates a new repository integration instance
func NewRepositoryIntegration(logger *zap.Logger) *RepositoryIntegration {
	return &RepositoryIntegration{
		logger: logger,
	}
}

// Initialize sets up the repository integration with sirius-agent-modules
func (ri *RepositoryIntegration) Initialize(ctx context.Context) error {
	ri.logger.Info("Initializing repository integration")

	// Create repository configuration for sirius-agent-modules
	config := &RepositoryConfiguration{
		RemoteURL:        "https://github.com/SiriusScan/sirius-agent-modules",
		LocalPath:        "/app-agent/sirius-agent-modules",
		UpdateInterval:   24 * time.Hour, // Daily updates
		UpdateStrategy:   UpdateStrategyIncremental,
		VerifySignatures: false, // Basic checksum validation for now
		CacheEnabled:     true,
		CacheSize:        100, // 100MB cache
		Timeout:          30 * time.Second,
		RetryAttempts:    3,
		UserAgent:        "Sirius-Agent/1.0",
	}

	ri.config = config

	// Create repository manager
	ri.repositoryManager = NewGitHubRepositoryManager(ri.logger)
	// Type assertion to access SetConfiguration method
	if githubManager, ok := ri.repositoryManager.(*GitHubRepositoryManager); ok {
		githubManager.SetConfiguration(config)
	}

	// Initialize repository
	if err := ri.repositoryManager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Perform initial repository download if local files don't exist
	manifestPath := filepath.Join(ri.config.LocalPath, "repository-manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		ri.logger.Info("Local repository not found, performing initial download")

		// Perform initial update to download repository content
		updateCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		updateResult, err := ri.repositoryManager.UpdateRepository(updateCtx)
		if err != nil {
			ri.logger.Warn("Initial repository download failed, continuing with local files only", zap.Error(err))
		} else {
			ri.logger.Info("Initial repository download completed",
				zap.Int("files_added", len(updateResult.FilesAdded)),
				zap.Int("files_updated", len(updateResult.FilesUpdated)))
		}
	} else {
		ri.logger.Info("Using existing local sirius-agent-modules repository")
	}

	ri.initialized = true
	ri.logger.Info("Repository integration initialized successfully")
	return nil
}

// LoadTemplatesFromRepository loads templates from the sirius-agent-modules repository
func (ri *RepositoryIntegration) LoadTemplatesFromRepository(ctx context.Context) ([]string, error) {
	if !ri.initialized {
		return nil, fmt.Errorf("repository integration not initialized")
	}

	ri.logger.Info("Loading templates from repository")

	// Check if repository-manifest.json exists
	manifestPath := filepath.Join(ri.config.LocalPath, "repository-manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		ri.logger.Warn("Repository manifest not found, checking for templates directly")
		// Fallback: check templates directory directly
		templatesDir := filepath.Join(ri.config.LocalPath, "templates")
		if _, err := os.Stat(templatesDir); err == nil {
			// Walk templates directory to find .yaml files
			var templatePaths []string
			err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".yaml") {
					templatePaths = append(templatePaths, path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk templates directory: %w", err)
			}
			ri.logger.Info("Templates loaded from directory",
				zap.Int("count", len(templatePaths)),
				zap.Strings("paths", templatePaths))
			return templatePaths, nil
		}
		return []string{}, nil
	}

	// Load repository manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository manifest: %w", err)
	}

	var manifest struct {
		Components struct {
			Templates struct {
				Path string `json:"path"`
			} `json:"templates"`
		} `json:"components"`
	}

	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse repository manifest: %w", err)
	}

	// Get templates directory
	templatesDir := filepath.Join(ri.config.LocalPath, manifest.Components.Templates.Path)
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		ri.logger.Warn("Templates directory not found", zap.String("path", templatesDir))
		return []string{}, nil
	}

	// Walk templates directory to find .yaml files
	var templatePaths []string
	err = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".yaml") {
			templatePaths = append(templatePaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk templates directory: %w", err)
	}

	ri.logger.Info("Templates loaded from repository",
		zap.Int("count", len(templatePaths)),
		zap.Strings("paths", templatePaths))

	return templatePaths, nil
}

// LoadScriptsFromRepository loads scripts from the sirius-agent-modules repository
func (ri *RepositoryIntegration) LoadScriptsFromRepository(ctx context.Context) ([]string, error) {
	if !ri.initialized {
		return nil, fmt.Errorf("repository integration not initialized")
	}

	ri.logger.Info("Loading scripts from repository")

	// Check if repository-manifest.json exists
	manifestPath := filepath.Join(ri.config.LocalPath, "repository-manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		ri.logger.Warn("Repository manifest not found, checking for scripts directly")
		// Fallback: check scripts directory directly
		scriptsDir := filepath.Join(ri.config.LocalPath, "scripts")
		if _, err := os.Stat(scriptsDir); err == nil {
			// Walk scripts directory to find script files
			var scriptPaths []string
			err := filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					ext := strings.ToLower(filepath.Ext(path))
					if ext == ".sh" || ext == ".ps1" || ext == ".py" {
						scriptPaths = append(scriptPaths, path)
					}
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk scripts directory: %w", err)
			}
			ri.logger.Info("Scripts loaded from directory",
				zap.Int("count", len(scriptPaths)),
				zap.Strings("paths", scriptPaths))
			return scriptPaths, nil
		}
		return []string{}, nil
	}

	// Load repository manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository manifest: %w", err)
	}

	var manifest struct {
		Components struct {
			Scripts struct {
				Path string `json:"path"`
			} `json:"scripts"`
		} `json:"components"`
	}

	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse repository manifest: %w", err)
	}

	// Get scripts directory
	scriptsDir := filepath.Join(ri.config.LocalPath, manifest.Components.Scripts.Path)
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		ri.logger.Warn("Scripts directory not found", zap.String("path", scriptsDir))
		return []string{}, nil
	}

	// Walk scripts directory to find script files
	var scriptPaths []string
	err = filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".sh" || ext == ".ps1" || ext == ".py" {
				scriptPaths = append(scriptPaths, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk scripts directory: %w", err)
	}

	ri.logger.Info("Scripts loaded from repository",
		zap.Int("count", len(scriptPaths)),
		zap.Strings("paths", scriptPaths))

	return scriptPaths, nil
}

// GetRepositoryTemplateDirectories returns template directories including repository
func (ri *RepositoryIntegration) GetRepositoryTemplateDirectories() []string {
	if !ri.initialized {
		return []string{}
	}

	// Return repository template directory
	repoTemplateDir := filepath.Join(ri.config.LocalPath, "templates")
	if _, err := os.Stat(repoTemplateDir); err == nil {
		return []string{repoTemplateDir}
	}

	return []string{}
}

// GetRepositoryScriptDirectories returns script directories including repository
func (ri *RepositoryIntegration) GetRepositoryScriptDirectories() []string {
	if !ri.initialized {
		return []string{}
	}

	// Return repository script directory
	repoScriptDir := filepath.Join(ri.config.LocalPath, "scripts")
	if _, err := os.Stat(repoScriptDir); err == nil {
		return []string{repoScriptDir}
	}

	return []string{}
}

// UpdateRepositoryIfNeeded checks for and applies repository updates
func (ri *RepositoryIntegration) UpdateRepositoryIfNeeded(ctx context.Context) error {
	if !ri.initialized {
		return fmt.Errorf("repository integration not initialized")
	}

	ri.logger.Info("Checking for repository updates")

	updateCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	result, err := ri.repositoryManager.UpdateRepository(updateCtx)
	if err != nil {
		return fmt.Errorf("repository update failed: %w", err)
	}

	if result.Success {
		if len(result.FilesAdded) > 0 || len(result.FilesUpdated) > 0 {
			ri.logger.Info("Repository updated successfully",
				zap.Int("files_added", len(result.FilesAdded)),
				zap.Int("files_updated", len(result.FilesUpdated)),
				zap.Duration("duration", result.Duration))
		} else {
			ri.logger.Debug("No repository updates available")
		}
	} else {
		ri.logger.Warn("Repository update had issues",
			zap.Strings("errors", result.Errors))
	}

	return nil
}

// GetRepositoryStatus returns current repository status
func (ri *RepositoryIntegration) GetRepositoryStatus() (*RepositoryIntegrationStatus, error) {
	if !ri.initialized {
		return nil, fmt.Errorf("repository integration not initialized")
	}

	info, err := ri.repositoryManager.GetRepositoryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	manifest, err := ri.repositoryManager.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load repository manifest: %w", err)
	}

	status := &RepositoryIntegrationStatus{
		Initialized:    ri.initialized,
		LocalPath:      ri.config.LocalPath,
		RemoteURL:      ri.config.RemoteURL,
		CurrentVersion: info.CurrentVersion,
		LastUpdate:     info.LastUpdate,
		TemplateCount:  info.TemplateCount,
		ScriptCount:    info.ScriptCount,
		TotalSize:      info.TotalSize,
		Status:         string(info.Status),
	}

	if manifest != nil {
		status.ManifestVersion = manifest.Version
		status.ManifestUpdated = manifest.Updated
	}

	return status, nil
}

// ValidateRepositoryContent validates repository content integrity
func (ri *RepositoryIntegration) ValidateRepositoryContent(ctx context.Context) (*ValidationResult, error) {
	if !ri.initialized {
		return nil, fmt.Errorf("repository integration not initialized")
	}

	ri.logger.Info("Validating repository content")

	validationCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := ri.repositoryManager.ValidateRepository(validationCtx)
	if err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	if result.Valid {
		ri.logger.Info("Repository content validation passed")
	} else {
		ri.logger.Warn("Repository content validation failed",
			zap.Int("error_count", len(result.Errors)))
	}

	return result, nil
}

// RepositoryIntegrationStatus represents the current status of the repository integration
type RepositoryIntegrationStatus struct {
	Initialized     bool      `json:"initialized"`
	LocalPath       string    `json:"local_path"`
	RemoteURL       string    `json:"remote_url"`
	CurrentVersion  string    `json:"current_version"`
	LastUpdate      time.Time `json:"last_update"`
	TemplateCount   int       `json:"template_count"`
	ScriptCount     int       `json:"script_count"`
	TotalSize       int64     `json:"total_size"`
	Status          string    `json:"status"`
	ManifestVersion string    `json:"manifest_version,omitempty"`
	ManifestUpdated time.Time `json:"manifest_updated,omitempty"`
}

// MergeResourceDirectories merges repository directories with existing directories
func (ri *RepositoryIntegration) MergeResourceDirectories(existingDirs []string, resourceType string) []string {
	if !ri.initialized {
		return existingDirs
	}

	var repoDirs []string
	switch resourceType {
	case "templates":
		repoDirs = ri.GetRepositoryTemplateDirectories()
	case "scripts":
		repoDirs = ri.GetRepositoryScriptDirectories()
	default:
		return existingDirs
	}

	// If repository directories exist, use them exclusively
	// This prevents duplicate loading of the same content
	if len(repoDirs) > 0 {
		ri.logger.Info("Using repository directories exclusively",
			zap.String("type", resourceType),
			zap.Strings("repository_dirs", repoDirs))
		return repoDirs
	}

	// Fallback to existing directories if no repository content
	ri.logger.Info("No repository directories found, using existing directories",
		zap.String("type", resourceType),
		zap.Strings("existing_dirs", existingDirs))
	return existingDirs
}

// IsRepositoryFile checks if a file path belongs to the repository
func (ri *RepositoryIntegration) IsRepositoryFile(filePath string) bool {
	if !ri.initialized {
		return false
	}

	return strings.HasPrefix(filePath, ri.config.LocalPath)
}

// GetRepositoryFileInfo returns information about a repository file
func (ri *RepositoryIntegration) GetRepositoryFileInfo(filePath string) (*FileInfo, error) {
	if !ri.initialized {
		return nil, fmt.Errorf("repository integration not initialized")
	}

	if !ri.IsRepositoryFile(filePath) {
		return nil, fmt.Errorf("file is not in repository: %s", filePath)
	}

	// Get relative path from repository root
	relPath, err := filepath.Rel(ri.config.LocalPath, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	// Load manifest to get file info
	manifest, err := ri.repositoryManager.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// Check if file exists in manifest
	if fileInfo, exists := manifest.Templates[relPath]; exists {
		return fileInfo, nil
	}

	if fileInfo, exists := manifest.Scripts[relPath]; exists {
		return fileInfo, nil
	}

	return nil, fmt.Errorf("file not found in repository manifest: %s", relPath)
}
