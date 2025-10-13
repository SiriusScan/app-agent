package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/detect/template"
	"go.uber.org/zap"
)

// TemplateSyncService handles comprehensive template synchronization to ValKey
type TemplateSyncService struct {
	logger      *zap.Logger
	valkeyStore *template.ValKeyTemplateStore
	config      *config.ServerConfig
}

// NewTemplateSyncService creates a new template sync service
func NewTemplateSyncService(logger *zap.Logger, valkeyStore *template.ValKeyTemplateStore, config *config.ServerConfig) *TemplateSyncService {
	return &TemplateSyncService{
		logger:      logger,
		valkeyStore: valkeyStore,
		config:      config,
	}
}

// SyncAllTemplatesToValKey performs comprehensive synchronization of all template types to ValKey
func (tss *TemplateSyncService) SyncAllTemplatesToValKey(ctx context.Context) error {
	tss.logger.Info("Starting comprehensive template synchronization to ValKey")

	// Track sync statistics
	stats := &SyncStatistics{
		StartTime: time.Now(),
	}

	// 1. Sync repository templates (highest priority for missing types)
	if err := tss.syncRepositoryTemplates(ctx, stats); err != nil {
		tss.logger.Error("Failed to sync repository templates", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Repository sync failed: %v", err))
	}

	// 2. Sync local templates (from predefined directories)
	if err := tss.syncLocalTemplates(ctx, stats); err != nil {
		tss.logger.Error("Failed to sync local templates", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Local sync failed: %v", err))
	}

	// 3. Verify custom templates are still synced (they should be via CustomStorageManager)
	if err := tss.verifyCustomTemplates(ctx, stats); err != nil {
		tss.logger.Error("Failed to verify custom templates", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Custom verification failed: %v", err))
	}

	// 4. Update global manifests and statistics
	if err := tss.updateGlobalManifests(ctx, stats); err != nil {
		tss.logger.Error("Failed to update global manifests", zap.Error(err))
		stats.Errors = append(stats.Errors, fmt.Sprintf("Manifest update failed: %v", err))
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	tss.logger.Info("Template synchronization completed",
		zap.Int("repository_templates", stats.RepositoryTemplates),
		zap.Int("local_templates", stats.LocalTemplates),
		zap.Int("custom_templates", stats.CustomTemplates),
		zap.Int("total_synced", stats.TotalSynced),
		zap.Duration("duration", stats.Duration),
		zap.Int("errors", len(stats.Errors)))

	if len(stats.Errors) > 0 {
		return fmt.Errorf("template sync completed with %d errors", len(stats.Errors))
	}

	return nil
}

// syncRepositoryTemplates synchronizes repository templates to ValKey
func (tss *TemplateSyncService) syncRepositoryTemplates(ctx context.Context, stats *SyncStatistics) error {
	tss.logger.Info("Syncing repository templates to ValKey")

	// Define repository template paths
	repositoryPaths := []string{
		"/app-agent/sirius-agent-modules/templates",
		"/app-agent/templates/sirius-agent-modules",
		"./sirius-agent-modules/templates",
		"./templates",
	}

	// Initialize repository list in ValKey if it doesn't exist
	repoList, err := tss.valkeyStore.GetRepositoryList(ctx)
	if err != nil {
		// Create default repository list
		repoList = &template.RepositoryList{
			Repositories: []template.Repository{
				{
					Name:     "sirius-agent-modules",
					URL:      "https://github.com/SiriusScan/sirius-agent-modules",
					Priority: 50,
					Enabled:  true,
				},
			},
			LastUpdated: time.Now(),
		}
		if err := tss.valkeyStore.SetRepositoryList(ctx, repoList); err != nil {
			tss.logger.Warn("Failed to initialize repository list", zap.Error(err))
		}
	}

	// Sync templates from each repository path
	for _, repoPath := range repositoryPaths {
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			continue // Skip non-existent paths
		}

		err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".yaml") {
				return nil
			}

			if err := tss.syncSingleTemplate(ctx, path, template.TemplateSourceRepository, "sirius-agent-modules", 50); err != nil {
				tss.logger.Warn("Failed to sync repository template", zap.String("path", path), zap.Error(err))
				return nil // Continue with other templates
			}

			stats.RepositoryTemplates++
			stats.TotalSynced++
			return nil
		})

		if err != nil {
			tss.logger.Warn("Failed to walk repository path", zap.String("path", repoPath), zap.Error(err))
		}
	}

	tss.logger.Info("Repository template sync completed", zap.Int("synced", stats.RepositoryTemplates))
	return nil
}

// syncLocalTemplates synchronizes local templates to ValKey
func (tss *TemplateSyncService) syncLocalTemplates(ctx context.Context, stats *SyncStatistics) error {
	tss.logger.Info("Syncing local templates to ValKey")

	// Define local template paths
	localPaths := []string{
		"/app-agent/templates",
		"./templates",
	}

	for _, localPath := range localPaths {
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			continue // Skip non-existent paths
		}

		err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".yaml") {
				return nil
			}

			// Skip repository templates that are in local paths
			if strings.Contains(path, "sirius-agent-modules") {
				return nil
			}

			if err := tss.syncSingleTemplate(ctx, path, template.TemplateSourceLocal, "local", 1); err != nil {
				tss.logger.Warn("Failed to sync local template", zap.String("path", path), zap.Error(err))
				return nil // Continue with other templates
			}

			stats.LocalTemplates++
			stats.TotalSynced++
			return nil
		})

		if err != nil {
			tss.logger.Warn("Failed to walk local path", zap.String("path", localPath), zap.Error(err))
		}
	}

	tss.logger.Info("Local template sync completed", zap.Int("synced", stats.LocalTemplates))
	return nil
}

// syncSingleTemplate synchronizes a single template file to ValKey
func (tss *TemplateSyncService) syncSingleTemplate(ctx context.Context, filePath, sourceType, sourceName string, priority int) error {
	// Read template content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse template to extract ID and metadata
	parser := template.NewTemplateParser(tss.logger, []string{})
	parsedTemplate, err := parser.LoadTemplate(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Calculate content hash
	hash := sha256.Sum256(content)
	hashStr := fmt.Sprintf("%x", hash)

	// Create template metadata for ValKey
	metadata := &template.TemplateMetadata{
		ID:          parsedTemplate.ID,
		Name:        parsedTemplate.Info.Name,
		Description: parsedTemplate.Info.Description,
		Author:      parsedTemplate.Info.Author,
		Version:     parsedTemplate.Info.Version,
		Severity:    string(parsedTemplate.Info.Severity),
		Tags:        parsedTemplate.Info.Tags,
		Category:    string(parsedTemplate.Detection.Type),
		Source: template.TemplateSource{
			Type:        sourceType,
			Name:        sourceName,
			Priority:    priority,
			LastUpdated: time.Now(),
		},
		CreatedAt:  parsedTemplate.LoadedAt,
		UpdatedAt:  time.Now(),
		UsageCount: 0,
	}

	// Create template content for ValKey
	templateContent := &template.TemplateContent{
		ID:        parsedTemplate.ID,
		Content:   string(content),
		Hash:      hashStr,
		Size:      int64(len(content)),
		UpdatedAt: time.Now(),
	}

	// Store template metadata and content in ValKey
	if err := tss.valkeyStore.SetTemplateMetadata(ctx, parsedTemplate.ID, metadata); err != nil {
		return fmt.Errorf("failed to store template metadata in ValKey: %w", err)
	}

	if err := tss.valkeyStore.SetTemplateContent(ctx, parsedTemplate.ID, templateContent); err != nil {
		return fmt.Errorf("failed to store template content in ValKey: %w", err)
	}

	tss.logger.Debug("Successfully synced template to ValKey",
		zap.String("template_id", parsedTemplate.ID),
		zap.String("template_name", parsedTemplate.Info.Name),
		zap.String("source_type", sourceType),
		zap.String("file_path", filePath))

	return nil
}

// verifyCustomTemplates verifies that custom templates are properly synced
func (tss *TemplateSyncService) verifyCustomTemplates(ctx context.Context, stats *SyncStatistics) error {
	tss.logger.Info("Verifying custom templates in ValKey")

	// Get custom template manifest from ValKey
	customManifest, err := tss.valkeyStore.GetCustomTemplateManifest(ctx)
	if err != nil {
		tss.logger.Info("No custom template manifest found in ValKey")
		return nil // Not an error - might not have custom templates
	}

	stats.CustomTemplates = len(customManifest.Templates)
	tss.logger.Info("Custom templates verified", zap.Int("count", stats.CustomTemplates))
	return nil
}

// updateGlobalManifests updates the global template manifest in ValKey
func (tss *TemplateSyncService) updateGlobalManifests(ctx context.Context, stats *SyncStatistics) error {
	tss.logger.Info("Updating global template manifests")

	// Get all template metadata keys to build comprehensive manifest
	metaKeys, err := tss.valkeyStore.ListTemplateMetaKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list template metadata keys: %w", err)
	}

	// Create global manifest
	globalManifest := &template.TemplateManifest{
		Name:        "global-templates",
		Version:     "1.0.0",
		Description: "Global template manifest for all template sources",
		LastUpdated: time.Now(),
		Templates:   make(map[string]template.TemplateMetadata),
		Sources:     make(map[string]template.TemplateSource),
		Statistics: template.TemplateStatistics{
			TotalTemplates:      0, // Will be updated below
			CustomTemplates:     stats.CustomTemplates,
			RepositoryTemplates: stats.RepositoryTemplates,
			LocalTemplates:      stats.LocalTemplates,
			ActiveTemplates:     0, // Will be updated below
			LastSyncTime:        time.Now(),
		},
	}

	// Populate template metadata in manifest (skip corrupted entries)
	validTemplateCount := 0
	for _, metaKey := range metaKeys {
		// Extract template ID from metadata key
		templateID := strings.TrimPrefix(metaKey, "agent:template:meta:")

		metadata, err := tss.valkeyStore.GetTemplateMetadata(ctx, templateID)
		if err != nil {
			tss.logger.Warn("Failed to get template metadata, skipping corrupted entry",
				zap.String("template_id", templateID),
				zap.Error(err))
			continue
		}

		globalManifest.Templates[templateID] = *metadata
		globalManifest.Sources[metadata.Source.Name] = metadata.Source
		validTemplateCount++
	}

	// Update statistics with actual valid template count
	globalManifest.Statistics.TotalTemplates = validTemplateCount
	globalManifest.Statistics.ActiveTemplates = validTemplateCount

	// Store global manifest
	if err := tss.valkeyStore.SetTemplateManifest(ctx, globalManifest); err != nil {
		return fmt.Errorf("failed to update global template manifest: %w", err)
	}

	tss.logger.Info("Global template manifest updated successfully",
		zap.Int("total_templates", len(globalManifest.Templates)),
		zap.Int("sources", len(globalManifest.Sources)),
		zap.Int("valid_templates", validTemplateCount),
		zap.Int("total_keys_found", len(metaKeys)))

	return nil
}

// SyncStatistics tracks synchronization progress
type SyncStatistics struct {
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	RepositoryTemplates int
	LocalTemplates      int
	CustomTemplates     int
	TotalSynced         int
	Errors              []string
}
