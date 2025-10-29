package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/template/agent"
	"go.uber.org/zap"
)

// RepositoryIntegration manages integration between repository and agent components
type RepositoryIntegration struct {
	logger            *zap.Logger
	repositoryManager RepositoryManager
	config            *RepositoryConfiguration
	initialized       bool
	syncManager       *agent.AgentSyncManager
}

// NewRepositoryIntegration creates a new repository integration instance
func NewRepositoryIntegration(logger *zap.Logger) *RepositoryIntegration {
	return &RepositoryIntegration{
		logger: logger,
	}
}

// Initialize sets up the repository integration with the new template system
func (ri *RepositoryIntegration) Initialize(ctx context.Context, agentID string, serverURL string) error {
	ri.logger.Info("Initializing repository integration with new template system")

	// Initialize the new template sync manager (without ValKey client)
	// The sync manager will communicate with the server via gRPC stream
	syncManager, err := agent.NewAgentSyncManager(ri.logger, serverURL, agentID)
	if err != nil {
		return fmt.Errorf("failed to create template sync manager: %w", err)
	}
	ri.syncManager = syncManager

	// Note: gRPC stream will be set later by the agent after connection is established
	// Initial template sync will be triggered by the agent after stream is ready

	ri.logger.Info("Template sync manager initialized, waiting for gRPC stream")

	// Create a minimal configuration for compatibility
	// Note: Repository URLs are now managed by the server via RepositoryManager
	// This configuration is maintained for backward compatibility
	ri.config = &RepositoryConfiguration{
		RemoteURL:        "",                               // Now managed by server's RepositoryManager
		LocalPath:        agent.GetAgentTemplateCacheDir(), // Use OS-agnostic cache directory
		UpdateInterval:   24 * time.Hour,
		UpdateStrategy:   UpdateStrategyIncremental,
		VerifySignatures: false,
		CacheEnabled:     true,
		CacheSize:        100,
		Timeout:          30 * time.Second,
		RetryAttempts:    3,
		UserAgent:        "Sirius-Agent/1.0",
	}

	ri.initialized = true
	ri.logger.Info("Repository integration initialized successfully with new template system")
	return nil
}

// GetSyncManager returns the sync manager instance
func (ri *RepositoryIntegration) GetSyncManager() *agent.AgentSyncManager {
	return ri.syncManager
}

// LoadTemplatesFromRepository loads templates from the cached template system
func (ri *RepositoryIntegration) LoadTemplatesFromRepository(ctx context.Context) ([]string, error) {
	if !ri.initialized {
		return nil, fmt.Errorf("repository integration not initialized")
	}

	ri.logger.Info("Loading templates from cached template system")

	// Use the new template sync manager to load templates
	templates, err := ri.syncManager.LoadTemplates(ctx)
	if err != nil {
		ri.logger.Warn("Failed to load templates from cache, falling back to directory scan", zap.Error(err))

		// Fallback: scan the cache directory for template files
		cacheDir := agent.GetAgentTemplateCacheDir()
		var templatePaths []string

		err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".yaml") {
				templatePaths = append(templatePaths, path)
			}
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan cache directory: %w", err)
		}

		ri.logger.Info("Templates loaded from cache directory scan",
			zap.Int("count", len(templatePaths)),
			zap.Strings("paths", templatePaths))
		return templatePaths, nil
	}

	// Convert templates to file paths
	var templatePaths []string
	for _, template := range templates {
		// For now, we'll use the template ID as a path-like identifier
		// In a real implementation, we might want to store the actual file path
		templatePaths = append(templatePaths, template.ID+".yaml")
	}

	ri.logger.Info("Templates loaded from template system",
		zap.Int("count", len(templatePaths)),
		zap.Strings("template_ids", templatePaths))

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

// GetRepositoryTemplateDirectories returns template directories including cache
func (ri *RepositoryIntegration) GetRepositoryTemplateDirectories() []string {
	if !ri.initialized {
		return []string{}
	}

	// Return the OS-agnostic cache directory
	cacheDir := agent.GetAgentTemplateCacheDir()
	if _, err := os.Stat(cacheDir); err == nil {
		return []string{cacheDir}
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

// UpdateRepositoryIfNeeded checks for and applies template updates from server
func (ri *RepositoryIntegration) UpdateRepositoryIfNeeded(ctx context.Context) error {
	if !ri.initialized {
		return fmt.Errorf("repository integration not initialized")
	}

	ri.logger.Info("Checking for template updates from server")

	updateCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	// Use the new template sync manager to sync from server
	if err := ri.syncManager.SyncFromServer(updateCtx); err != nil {
		ri.logger.Warn("Template sync failed", zap.Error(err))
		return fmt.Errorf("template sync failed: %w", err)
	}

	ri.logger.Info("Template sync completed successfully")
	return nil
}

// GetRepositoryStatus returns current template system status
func (ri *RepositoryIntegration) GetRepositoryStatus() (*RepositoryIntegrationStatus, error) {
	if !ri.initialized {
		return nil, fmt.Errorf("repository integration not initialized")
	}

	// For now, return basic status since we don't have a GetCacheStatistics method
	// In a real implementation, we would get this from the sync manager
	status := &RepositoryIntegrationStatus{
		Initialized:     ri.initialized,
		LocalPath:       ri.config.LocalPath,
		RemoteURL:       ri.config.RemoteURL,
		CurrentVersion:  "template-system-v2", // New template system version
		LastUpdate:      time.Now(),           // Placeholder
		TemplateCount:   0,                    // Will be populated when templates are loaded
		ScriptCount:     0,                    // Scripts are now part of templates
		TotalSize:       0,                    // Will be calculated from cache
		Status:          "active",
		ManifestVersion: "2.0.0",    // New manifest format
		ManifestUpdated: time.Now(), // Placeholder
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
