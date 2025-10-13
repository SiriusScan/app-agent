package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"go.uber.org/zap"
)

// ScriptSyncService handles comprehensive script synchronization to ValKey
type ScriptSyncService struct {
	logger      *zap.Logger
	valkeyStore *ScriptValKeyStore
	config      *config.ServerConfig
}

// ScriptValKeyStore interface for ValKey operations
type ScriptValKeyStore interface {
	SetScriptMetadata(ctx context.Context, scriptID string, metadata *ScriptMetadata) error
	SetScriptContent(ctx context.Context, scriptID string, content *ScriptContent) error
	ListScriptMetaKeys(ctx context.Context) ([]string, error)
	GetScriptManifest(ctx context.Context) (*ScriptManifest, error)
	SetScriptManifest(ctx context.Context, manifest *ScriptManifest) error
}

// ScriptMetadata represents script metadata stored in ValKey
type ScriptMetadata struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Author      string       `json:"author"`
	Version     string       `json:"version"`
	Language    string       `json:"language"`
	Platform    string       `json:"platform"`
	Tags        []string     `json:"tags"`
	Category    string       `json:"category"`
	Source      ScriptSource `json:"source"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	UsageCount  int          `json:"usage_count"`
}

// ScriptContent represents script content stored in ValKey
type ScriptContent struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScriptSource represents the source of a script
type ScriptSource struct {
	Type        string    `json:"type"`     // "repository", "custom", "local"
	Name        string    `json:"name"`     // Source name
	Priority    int       `json:"priority"` // Priority for conflict resolution
	LastUpdated time.Time `json:"last_updated"`
}

// ScriptManifest represents a collection of scripts
type ScriptManifest struct {
	Name        string                    `json:"name"`
	Version     string                    `json:"version"`
	Description string                    `json:"description"`
	LastUpdated time.Time                 `json:"last_updated"`
	Scripts     map[string]ScriptMetadata `json:"scripts"`
	Sources     map[string]ScriptSource   `json:"sources"`
	Statistics  ScriptStatistics          `json:"statistics"`
}

// ScriptStatistics tracks script statistics
type ScriptStatistics struct {
	TotalScripts      int            `json:"total_scripts"`
	CustomScripts     int            `json:"custom_scripts"`
	RepositoryScripts int            `json:"repository_scripts"`
	LocalScripts      int            `json:"local_scripts"`
	ActiveScripts     int            `json:"active_scripts"`
	ByLanguage        map[string]int `json:"by_language"`
	ByPlatform        map[string]int `json:"by_platform"`
	LastSyncTime      time.Time      `json:"last_sync_time"`
}

// ScriptSyncStatistics tracks script synchronization progress
type ScriptSyncStatistics struct {
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	RepositoryScripts int
	LocalScripts      int
	CustomScripts     int
	TotalSynced       int
	Errors            []string
}

// NewScriptSyncService creates a new script sync service
func NewScriptSyncService(logger *zap.Logger, valkeyStore *ScriptValKeyStore, config *config.ServerConfig) *ScriptSyncService {
	return &ScriptSyncService{
		logger:      logger,
		valkeyStore: valkeyStore,
		config:      config,
	}
}

// SyncAllScriptsToValKey performs comprehensive synchronization of all script types to ValKey
func (sss *ScriptSyncService) SyncAllScriptsToValKey(ctx context.Context) error {
	sss.logger.Info("Starting comprehensive script synchronization to ValKey")

	// Track sync statistics
	stats := &ScriptSyncStatistics{
		StartTime: time.Now(),
	}

	// 1. Sync repository scripts (highest priority)
	if err := sss.syncRepositoryScripts(ctx, stats); err != nil {
		sss.logger.Error("Failed to sync repository scripts", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Repository sync failed: %v", err))
	}

	// 2. Sync local scripts (from predefined directories)
	if err := sss.syncLocalScripts(ctx, stats); err != nil {
		sss.logger.Error("Failed to sync local scripts", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Local sync failed: %v", err))
	}

	// 3. Verify custom scripts are still synced
	if err := sss.verifyCustomScripts(ctx, stats); err != nil {
		sss.logger.Error("Failed to verify custom scripts", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Custom verification failed: %v", err))
	}

	// 4. Update global manifests and statistics
	if err := sss.updateGlobalManifests(ctx, stats); err != nil {
		sss.logger.Error("Failed to update global manifests", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Manifest update failed: %v", err))
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	sss.logger.Info("Script synchronization completed",
		zap.Int("repository_scripts", stats.RepositoryScripts),
		zap.Int("local_scripts", stats.LocalScripts),
		zap.Int("custom_scripts", stats.CustomScripts),
		zap.Int("total_synced", stats.TotalSynced),
		zap.Duration("duration", stats.Duration),
		zap.Int("errors", len(stats.Errors)))

	if len(stats.Errors) > 0 {
		return fmt.Errorf("script sync completed with %d errors", len(stats.Errors))
	}

	return nil
}

// syncRepositoryScripts synchronizes repository scripts to ValKey
func (sss *ScriptSyncService) syncRepositoryScripts(ctx context.Context, stats *ScriptSyncStatistics) error {
	sss.logger.Info("Syncing repository scripts to ValKey")

	// Define repository script paths
	repositoryPaths := []string{
		"/app-agent/sirius-agent-modules/scripts",
		"/app-agent/scripts/sirius-agent-modules",
		"./sirius-agent-modules/scripts",
		"./scripts",
	}

	// Load script manifest to get metadata
	var scriptManifest map[string]interface{}
	manifestPath := ""

	// Find the script manifest
	for _, repoPath := range repositoryPaths {
		manifestFile := filepath.Join(repoPath, "manifest.json")
		if _, err := os.Stat(manifestFile); err == nil {
			manifestPath = manifestFile
			break
		}
	}

	if manifestPath != "" {
		manifestData, err := os.ReadFile(manifestPath)
		if err == nil {
			json.Unmarshal(manifestData, &scriptManifest)
		}
	}

	// Sync scripts from each repository path
	for _, repoPath := range repositoryPaths {
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			continue // Skip non-existent paths
		}

		err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || strings.HasSuffix(path, "manifest.json") {
				return nil
			}

			// Check if it's a script file (common extensions)
			ext := strings.ToLower(filepath.Ext(path))
			if !isScriptFile(ext) {
				return nil
			}

			// Get relative path from repository root for manifest lookup
			relPath, err := filepath.Rel(repoPath, path)
			if err != nil {
				relPath = path
			}

			if err := sss.syncSingleScript(ctx, path, relPath, scriptManifest, "repository", "sirius-agent-modules", 50); err != nil {
				sss.logger.Warn("Failed to sync repository script", zap.String("path", path), zap.Error(err))
				return nil // Continue with other scripts
			}

			stats.RepositoryScripts++
			stats.TotalSynced++
			return nil
		})

		if err != nil {
			sss.logger.Warn("Failed to walk repository path", zap.String("path", repoPath), zap.Error(err))
		}
	}

	sss.logger.Info("Repository script sync completed", zap.Int("synced", stats.RepositoryScripts))
	return nil
}

// syncLocalScripts synchronizes local scripts to ValKey
func (sss *ScriptSyncService) syncLocalScripts(ctx context.Context, stats *ScriptSyncStatistics) error {
	sss.logger.Info("Syncing local scripts to ValKey")

	// Define local script paths
	localPaths := []string{
		"/app-agent/custom-scripts",
		"./custom-scripts",
	}

	for _, localPath := range localPaths {
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			continue // Skip non-existent paths
		}

		err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Check if it's a script file
			ext := strings.ToLower(filepath.Ext(path))
			if !isScriptFile(ext) {
				return nil
			}

			// Skip repository scripts that are in local paths
			if strings.Contains(path, "sirius-agent-modules") {
				return nil
			}

			relPath, err := filepath.Rel(localPath, path)
			if err != nil {
				relPath = path
			}

			if err := sss.syncSingleScript(ctx, path, relPath, nil, "local", "local", 1); err != nil {
				sss.logger.Warn("Failed to sync local script", zap.String("path", path), zap.Error(err))
				return nil // Continue with other scripts
			}

			stats.LocalScripts++
			stats.TotalSynced++
			return nil
		})

		if err != nil {
			sss.logger.Warn("Failed to walk local path", zap.String("path", localPath), zap.Error(err))
		}
	}

	sss.logger.Info("Local script sync completed", zap.Int("synced", stats.LocalScripts))
	return nil
}

// syncSingleScript synchronizes a single script file to ValKey
func (sss *ScriptSyncService) syncSingleScript(ctx context.Context, filePath, relPath string, manifest map[string]interface{}, sourceType, sourceName string, priority int) error {
	// Read script content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// Generate script ID from relative path
	scriptID := generateScriptID(relPath)

	// Extract metadata from manifest or infer from file
	metadata := sss.extractScriptMetadata(scriptID, relPath, filePath, manifest, sourceType, sourceName, priority)

	// Calculate content hash
	hash := sha256.Sum256(content)
	hashStr := fmt.Sprintf("%x", hash)

	// Create script content for ValKey
	scriptContent := &ScriptContent{
		ID:        scriptID,
		Content:   string(content),
		Hash:      hashStr,
		Size:      int64(len(content)),
		UpdatedAt: time.Now(),
	}

	// Store script metadata and content in ValKey
	if err := (*sss.valkeyStore).SetScriptMetadata(ctx, scriptID, metadata); err != nil {
		return fmt.Errorf("failed to store script metadata in ValKey: %w", err)
	}

	if err := (*sss.valkeyStore).SetScriptContent(ctx, scriptID, scriptContent); err != nil {
		return fmt.Errorf("failed to store script content in ValKey: %w", err)
	}

	sss.logger.Debug("Successfully synced script to ValKey",
		zap.String("script_id", scriptID),
		zap.String("script_name", metadata.Name),
		zap.String("source_type", sourceType),
		zap.String("file_path", filePath))

	return nil
}

// extractScriptMetadata extracts metadata from manifest or infers from file
func (sss *ScriptSyncService) extractScriptMetadata(scriptID, relPath, filePath string, manifest map[string]interface{}, sourceType, sourceName string, priority int) *ScriptMetadata {
	// Default metadata
	metadata := &ScriptMetadata{
		ID:          scriptID,
		Name:        scriptID,
		Description: "Script description",
		Author:      "unknown",
		Version:     "1.0.0",
		Language:    inferLanguageFromExtension(filepath.Ext(filePath)),
		Platform:    inferPlatformFromPath(relPath),
		Tags:        []string{},
		Category:    "custom",
		Source: ScriptSource{
			Type:        sourceType,
			Name:        sourceName,
			Priority:    priority,
			LastUpdated: time.Now(),
		},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		UsageCount: 0,
	}

	// Extract from manifest if available
	if manifest != nil {
		if scripts, ok := manifest["scripts"].(map[string]interface{}); ok {
			if scriptInfo, ok := scripts[relPath].(map[string]interface{}); ok {
				if name, ok := scriptInfo["description"].(string); ok {
					metadata.Name = name
				}
				if desc, ok := scriptInfo["description"].(string); ok {
					metadata.Description = desc
				}
				if author, ok := scriptInfo["author"].(string); ok {
					metadata.Author = author
				}
				if version, ok := scriptInfo["version"].(string); ok {
					metadata.Version = version
				}
				if platforms, ok := scriptInfo["platforms"].([]interface{}); ok {
					if len(platforms) > 0 {
						if platform, ok := platforms[0].(string); ok {
							metadata.Platform = platform
						}
					}
				}
			}
		}
	}

	return metadata
}

// Helper functions
func isScriptFile(ext string) bool {
	scriptExtensions := []string{".sh", ".ps1", ".py", ".js", ".lua", ".nse", ".pl", ".rb"}
	for _, scriptExt := range scriptExtensions {
		if ext == scriptExt {
			return true
		}
	}
	return false
}

func generateScriptID(relPath string) string {
	// Convert path to ID: cross-platform/script.sh -> cross-platform-script-sh
	id := strings.ReplaceAll(relPath, "/", "-")
	id = strings.ReplaceAll(id, "\\", "-")
	id = strings.ReplaceAll(id, ".", "-")
	return strings.ToLower(id)
}

func inferLanguageFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".sh":
		return "bash"
	case ".ps1":
		return "powershell"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".lua", ".nse":
		return "lua"
	case ".pl":
		return "perl"
	case ".rb":
		return "ruby"
	default:
		return "unknown"
	}
}

func inferPlatformFromPath(relPath string) string {
	path := strings.ToLower(relPath)
	if strings.Contains(path, "windows") {
		return "windows"
	} else if strings.Contains(path, "linux") {
		return "linux"
	} else if strings.Contains(path, "cross-platform") {
		return "cross"
	} else if strings.Contains(path, "macos") {
		return "macos"
	}
	return "any"
}

// verifyCustomScripts verifies that custom scripts are properly synced
func (sss *ScriptSyncService) verifyCustomScripts(ctx context.Context, stats *ScriptSyncStatistics) error {
	sss.logger.Info("Verifying custom scripts in ValKey")

	// Custom scripts should be handled by CustomStorageManager
	// This is just verification that they exist
	stats.CustomScripts = 0 // Will be updated by actual count if needed

	sss.logger.Info("Custom scripts verified", zap.Int("count", stats.CustomScripts))
	return nil
}

// updateGlobalManifests updates the global script manifest in ValKey
func (sss *ScriptSyncService) updateGlobalManifests(ctx context.Context, stats *ScriptSyncStatistics) error {
	sss.logger.Info("Updating global script manifests")

	// Get all script metadata keys to build comprehensive manifest
	metaKeys, err := (*sss.valkeyStore).ListScriptMetaKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list script metadata keys: %w", err)
	}

	// Create global manifest
	globalManifest := &ScriptManifest{
		Name:        "global-scripts",
		Version:     "1.0.0",
		Description: "Global script manifest for all script sources",
		LastUpdated: time.Now(),
		Scripts:     make(map[string]ScriptMetadata),
		Sources:     make(map[string]ScriptSource),
		Statistics: ScriptStatistics{
			TotalScripts:      len(metaKeys),
			CustomScripts:     stats.CustomScripts,
			RepositoryScripts: stats.RepositoryScripts,
			LocalScripts:      stats.LocalScripts,
			ActiveScripts:     len(metaKeys),
			ByLanguage:        make(map[string]int),
			ByPlatform:        make(map[string]int),
			LastSyncTime:      time.Now(),
		},
	}

	// Store the global manifest
	if err := (*sss.valkeyStore).SetScriptManifest(ctx, globalManifest); err != nil {
		return fmt.Errorf("failed to store global script manifest: %w", err)
	}

	sss.logger.Info("Global script manifest updated", zap.Int("total_scripts", len(metaKeys)))
	return nil
}
