package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	valkey "github.com/valkey-io/valkey-go"
	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/parser"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// AgentSyncManager manages template synchronization for agents
type AgentSyncManager struct {
	cacheDir     string
	valkeyClient valkey.Client
	logger       *zap.Logger
	serverURL    string
}

// CacheManifest represents the local cache manifest
type CacheManifest struct {
	Version      string                 `json:"version"`
	LastSync     time.Time              `json:"last_sync"`
	ServerURL    string                 `json:"server_url"`
	Templates    map[string]*CacheTemplateInfo `json:"templates"`
	Statistics   CacheStatistics        `json:"statistics"`
}

// CacheTemplateInfo represents template information in the cache
type CacheTemplateInfo struct {
	ID           string    `json:"id"`
	Version      string    `json:"version"`
	Checksum     string    `json:"checksum"`
	Size         int64     `json:"size"`
	Severity     string    `json:"severity"`
	Platforms    []string  `json:"platforms"`
	DetectionType string   `json:"detection_type"`
	Author       string    `json:"author"`
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated"`
	FilePath     string    `json:"file_path"`
	IsCustom     bool      `json:"is_custom"`
}

// CacheStatistics contains cache statistics
type CacheStatistics struct {
	TotalTemplates     int `json:"total_templates"`
	StandardTemplates  int `json:"standard_templates"`
	CustomTemplates    int `json:"custom_templates"`
	LastSyncDuration   time.Duration `json:"last_sync_duration"`
	CacheSize          int64 `json:"cache_size"`
}

// NewAgentSyncManager creates a new agent sync manager
func NewAgentSyncManager(valkeyClient valkey.Client, logger *zap.Logger, serverURL string) (*AgentSyncManager, error) {
	cacheDir := GetAgentTemplateCacheDir()
	
	// Ensure cache directory structure exists
	if err := EnsureCacheDirectoryStructure(); err != nil {
		return nil, fmt.Errorf("failed to create cache directory structure: %w", err)
	}
	
	return &AgentSyncManager{
		cacheDir:     cacheDir,
		valkeyClient: valkeyClient,
		logger:       logger,
		serverURL:    serverURL,
	}, nil
}

// SyncFromServer pulls templates from server and updates local cache
func (asm *AgentSyncManager) SyncFromServer(ctx context.Context) error {
	startTime := time.Now()
	asm.logger.Info("Starting template sync from server",
		zap.String("server_url", asm.serverURL))

	// Get server manifest
	serverManifest, err := asm.getServerManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to get server manifest: %w", err)
	}

	// Load local cache manifest
	localManifest, err := asm.loadCacheManifest()
	if err != nil {
		asm.logger.Warn("Failed to load local cache manifest, creating new one", zap.Error(err))
		localManifest = &CacheManifest{
			Version:   "1.0.0",
			Templates: make(map[string]*CacheTemplateInfo),
			Statistics: CacheStatistics{},
		}
	}

	// Compare manifests and download changed templates
	var downloadedCount int
	var errorCount int

	for templateID, serverTemplate := range serverManifest.Templates {
		localTemplate, exists := localManifest.Templates[templateID]
		
		// Check if template needs update
		needsUpdate := !exists || 
			localTemplate.Checksum != serverTemplate.Checksum ||
			localTemplate.Version != serverTemplate.Version

		if needsUpdate {
			if err := asm.downloadTemplate(ctx, templateID, serverTemplate); err != nil {
				asm.logger.Error("Failed to download template",
					zap.String("id", templateID),
					zap.Error(err))
				errorCount++
				continue
			}
			downloadedCount++
		}
	}

	// Update local manifest
	localManifest.Version = serverManifest.Version
	localManifest.LastSync = time.Now()
	localManifest.ServerURL = asm.serverURL
	localManifest.Statistics.LastSyncDuration = time.Since(startTime)
	localManifest.Statistics.TotalTemplates = len(serverManifest.Templates)

	// Count by type
	localManifest.Statistics.StandardTemplates = 0
	localManifest.Statistics.CustomTemplates = 0
	for _, template := range localManifest.Templates {
		if template.IsCustom {
			localManifest.Statistics.CustomTemplates++
		} else {
			localManifest.Statistics.StandardTemplates++
		}
	}

	// Calculate cache size
	cacheSize, err := asm.calculateCacheSize()
	if err != nil {
		asm.logger.Warn("Failed to calculate cache size", zap.Error(err))
	} else {
		localManifest.Statistics.CacheSize = cacheSize
	}

	// Save updated manifest
	if err := asm.saveCacheManifest(localManifest); err != nil {
		return fmt.Errorf("failed to save cache manifest: %w", err)
	}

	asm.logger.Info("Template sync completed",
		zap.Int("downloaded", downloadedCount),
		zap.Int("errors", errorCount),
		zap.Duration("duration", time.Since(startTime)))

	return nil
}

// ValidateCache verifies checksums of cached templates
func (asm *AgentSyncManager) ValidateCache(ctx context.Context) error {
	asm.logger.Info("Validating template cache")

	manifest, err := asm.loadCacheManifest()
	if err != nil {
		return fmt.Errorf("failed to load cache manifest: %w", err)
	}

	var validCount int
	var invalidCount int

	for templateID, templateInfo := range manifest.Templates {
		// Read template file
		content, err := os.ReadFile(templateInfo.FilePath)
		if err != nil {
			asm.logger.Error("Failed to read template file",
				zap.String("id", templateID),
				zap.String("path", templateInfo.FilePath),
				zap.Error(err))
			invalidCount++
			continue
		}

		// Calculate checksum
		hash := sha256.Sum256(content)
		actualChecksum := "sha256:" + hex.EncodeToString(hash[:])

		// Compare checksums
		if actualChecksum != templateInfo.Checksum {
			asm.logger.Error("Template checksum mismatch",
				zap.String("id", templateID),
				zap.String("expected", templateInfo.Checksum),
				zap.String("actual", actualChecksum))
			invalidCount++
		} else {
			validCount++
		}
	}

	asm.logger.Info("Cache validation completed",
		zap.Int("valid", validCount),
		zap.Int("invalid", invalidCount))

	if invalidCount > 0 {
		return fmt.Errorf("cache validation failed: %d invalid templates", invalidCount)
	}

	return nil
}

// LoadTemplates loads cached templates for scanning
func (asm *AgentSyncManager) LoadTemplates(ctx context.Context) ([]*types.Template, error) {
	asm.logger.Info("Loading cached templates")

	manifest, err := asm.loadCacheManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load cache manifest: %w", err)
	}

	var templates []*types.Template
	var errorCount int

	for templateID, templateInfo := range manifest.Templates {
		// Load template using parser
		template, err := parser.ParseTemplate(templateInfo.FilePath)
		if err != nil {
			asm.logger.Error("Failed to parse template",
				zap.String("id", templateID),
				zap.String("path", templateInfo.FilePath),
				zap.Error(err))
			errorCount++
			continue
		}

		templates = append(templates, template)
	}

	asm.logger.Info("Template loading completed",
		zap.Int("loaded", len(templates)),
		zap.Int("errors", errorCount))

	return templates, nil
}

// ClearCache removes all cached templates
func (asm *AgentSyncManager) ClearCache() error {
	asm.logger.Info("Clearing template cache")

	// Remove cache directory
	if err := os.RemoveAll(asm.cacheDir); err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Recreate directory structure
	if err := EnsureCacheDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to recreate cache directory structure: %w", err)
	}

	asm.logger.Info("Template cache cleared successfully")
	return nil
}

// GetCacheStatus returns the current cache status
func (asm *AgentSyncManager) GetCacheStatus() (*CacheManifest, error) {
	return asm.loadCacheManifest()
}

// getServerManifest retrieves the template manifest from the server
func (asm *AgentSyncManager) getServerManifest(ctx context.Context) (*ServerManifest, error) {
	// This would typically make an HTTP request to the server
	// For now, we'll simulate getting it from ValKey directly
	// In a real implementation, this would be an HTTP API call
	
	// TODO: Implement HTTP client to get manifest from server
	// For now, return empty manifest
	return &ServerManifest{
		Version:   "1.0.0",
		Updated:   time.Now(),
		Templates: make(map[string]*ServerTemplateInfo),
	}, nil
}

// downloadTemplate downloads a template from the server
func (asm *AgentSyncManager) downloadTemplate(ctx context.Context, templateID string, templateInfo *ServerTemplateInfo) error {
	asm.logger.Debug("Downloading template",
		zap.String("id", templateID),
		zap.String("checksum", templateInfo.Checksum))

	// Determine cache path
	var cachePath string
	if templateInfo.IsCustom {
		cachePath = filepath.Join(GetCustomTemplatesPath(), templateID+".yaml")
	} else {
		cachePath = filepath.Join(GetStandardTemplatesPath(), templateID+".yaml")
	}

	// TODO: Implement actual template download from server
	// For now, create a placeholder file
	content := []byte(fmt.Sprintf("# Template %s\n# Placeholder content", templateID))
	
	// Write template file atomically
	tempPath := cachePath + ".tmp"
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}
	
	if err := os.Rename(tempPath, cachePath); err != nil {
		return fmt.Errorf("failed to rename template file: %w", err)
	}

	asm.logger.Debug("Template downloaded successfully",
		zap.String("id", templateID),
		zap.String("path", cachePath))

	return nil
}

// loadCacheManifest loads the local cache manifest
func (asm *AgentSyncManager) loadCacheManifest() (*CacheManifest, error) {
	manifestPath := GetCacheManifestPath()
	
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &CacheManifest{
				Version:   "1.0.0",
				Templates: make(map[string]*CacheTemplateInfo),
				Statistics: CacheStatistics{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read cache manifest: %w", err)
	}

	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse cache manifest: %w", err)
	}

	return &manifest, nil
}

// saveCacheManifest saves the local cache manifest
func (asm *AgentSyncManager) saveCacheManifest(manifest *CacheManifest) error {
	manifestPath := GetCacheManifestPath()
	
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache manifest: %w", err)
	}

	// Write manifest file atomically
	tempPath := manifestPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache manifest: %w", err)
	}
	
	if err := os.Rename(tempPath, manifestPath); err != nil {
		return fmt.Errorf("failed to rename cache manifest: %w", err)
	}

	return nil
}

// calculateCacheSize calculates the total size of the cache directory
func (asm *AgentSyncManager) calculateCacheSize() (int64, error) {
	var totalSize int64
	
	err := filepath.Walk(asm.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	
	return totalSize, err
}

// ServerManifest represents the server template manifest
type ServerManifest struct {
	Version   string                        `json:"version"`
	Updated   time.Time                     `json:"updated"`
	Templates map[string]*ServerTemplateInfo `json:"templates"`
}

// ServerTemplateInfo represents template information from the server
type ServerTemplateInfo struct {
	ID               string    `json:"id"`
	Version          string    `json:"version"`
	Checksum         string    `json:"checksum"`
	Size             int64     `json:"size"`
	Severity         string    `json:"severity"`
	Platforms        []string  `json:"platforms"`
	DetectionType    string    `json:"detection_type"`
	Author           string    `json:"author"`
	Created          time.Time `json:"created"`
	Updated          time.Time `json:"updated"`
	VulnerabilityIDs []string  `json:"vulnerability_ids"`
	IsCustom         bool      `json:"is_custom"`
}
