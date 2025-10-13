package template

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TemplateDiscoveryService provides comprehensive template discovery
type TemplateDiscoveryService struct {
	logger      *zap.Logger
	valkeyStore *ValKeyTemplateStore
	parser      *TemplateParser
}

// DiscoveredTemplate represents a discovered template with source information
type DiscoveredTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Severity    string         `json:"severity"`
	Type        string         `json:"type"`
	Content     string         `json:"content"`
	Source      TemplateSource `json:"source"`
	FilePath    string         `json:"file_path"`
	Hash        string         `json:"hash"`
	Size        int64          `json:"size"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// DiscoveryResult represents the result of template discovery
type DiscoveryResult struct {
	Templates     []*DiscoveredTemplate
	Sources       map[string]TemplateSource
	Statistics    TemplateStatistics
	Errors        []string
	LastDiscovery time.Time
}

// NewTemplateDiscoveryService creates a new template discovery service
func NewTemplateDiscoveryService(logger *zap.Logger, valkeyStore *ValKeyTemplateStore, parser *TemplateParser) *TemplateDiscoveryService {
	return &TemplateDiscoveryService{
		logger:      logger,
		valkeyStore: valkeyStore,
		parser:      parser,
	}
}

// DiscoverAllTemplates performs comprehensive template discovery from all sources
func (tds *TemplateDiscoveryService) DiscoverAllTemplates(ctx context.Context, templateDirs []string) (*DiscoveryResult, error) {
	tds.logger.Info("Starting comprehensive template discovery using ValKey as primary source")

	result := &DiscoveryResult{
		Templates:     []*DiscoveredTemplate{},
		Sources:       make(map[string]TemplateSource),
		Statistics:    TemplateStatistics{},
		Errors:        []string{},
		LastDiscovery: time.Now(),
	}

	// 1. Discover all templates from ValKey (primary source of truth)
	valkeyTemplates, err := tds.discoverTemplatesFromValKey(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("ValKey template discovery failed: %v", err))
		tds.logger.Error("Failed to discover templates from ValKey", zap.Error(err))
		// Fall back to filesystem discovery if ValKey fails
		return tds.discoverFromFilesystemFallback(ctx, templateDirs)
	}

	result.Templates = append(result.Templates, valkeyTemplates...)

	// Categorize templates by source type for statistics
	for _, template := range valkeyTemplates {
		switch template.Source.Type {
		case TemplateSourceCustom:
			result.Statistics.CustomTemplates++
		case TemplateSourceRepository:
			result.Statistics.RepositoryTemplates++
		case TemplateSourceLocal:
			result.Statistics.LocalTemplates++
		}
		result.Sources[template.Source.Name] = template.Source
	}

	// Update statistics
	result.Statistics.TotalTemplates = len(result.Templates)
	result.Statistics.ActiveTemplates = len(result.Templates)
	result.Statistics.LastSyncTime = time.Now()

	tds.logger.Info("Template discovery completed from ValKey",
		zap.Int("total", result.Statistics.TotalTemplates),
		zap.Int("custom", result.Statistics.CustomTemplates),
		zap.Int("repository", result.Statistics.RepositoryTemplates),
		zap.Int("local", result.Statistics.LocalTemplates))

	return result, nil
}

// discoverCustomTemplates discovers custom templates from ValKey
func (tds *TemplateDiscoveryService) discoverCustomTemplates(ctx context.Context) ([]*DiscoveredTemplate, error) {
	tds.logger.Debug("Discovering custom templates from ValKey")

	manifest, err := tds.valkeyStore.GetCustomTemplateManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom template manifest: %w", err)
	}

	var templates []*DiscoveredTemplate
	for templateID, metadata := range manifest.Templates {
		// Get template content
		content, err := tds.valkeyStore.GetTemplateContent(ctx, templateID)
		if err != nil {
			tds.logger.Warn("Failed to get custom template content", zap.String("template_id", templateID), zap.Error(err))
			continue
		}

		template := &DiscoveredTemplate{
			ID:          templateID,
			Name:        metadata.Name,
			Description: metadata.Description,
			Severity:    metadata.Severity,
			Type:        metadata.Category,
			Content:     content.Content,
			Source:      metadata.Source,
			Hash:        content.Hash,
			Size:        content.Size,
			CreatedAt:   metadata.CreatedAt,
			UpdatedAt:   metadata.UpdatedAt,
		}

		templates = append(templates, template)
	}

	tds.logger.Debug("Discovered custom templates", zap.Int("count", len(templates)))
	return templates, nil
}

// discoverRepositoryTemplates discovers templates from repositories
func (tds *TemplateDiscoveryService) discoverRepositoryTemplates(ctx context.Context) ([]*DiscoveredTemplate, error) {
	tds.logger.Debug("Discovering repository templates")

	repoList, err := tds.valkeyStore.GetRepositoryList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository list: %w", err)
	}

	var templates []*DiscoveredTemplate
	for _, repo := range repoList.Repositories {
		if !repo.Enabled {
			continue
		}

		repoTemplates, err := tds.discoverTemplatesFromRepository(ctx, repo)
		if err != nil {
			tds.logger.Warn("Failed to discover templates from repository", zap.String("repo", repo.Name), zap.Error(err))
			continue
		}

		templates = append(templates, repoTemplates...)
	}

	tds.logger.Debug("Discovered repository templates", zap.Int("count", len(templates)))
	return templates, nil
}

// discoverTemplatesFromRepository discovers templates from a specific repository
func (tds *TemplateDiscoveryService) discoverTemplatesFromRepository(ctx context.Context, repo Repository) ([]*DiscoveredTemplate, error) {
	// This would integrate with the existing repository system
	// For now, we'll scan the repository directory structure
	repoPath := fmt.Sprintf("/app-agent/templates/%s", repo.Name)

	var templates []*DiscoveredTemplate

	// Walk repository directory
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".yaml") {
			return nil
		}

		// Load and parse template
		template, err := tds.parser.LoadTemplate(path)
		if err != nil {
			tds.logger.Warn("Failed to load repository template", zap.String("path", path), zap.Error(err))
			return nil
		}

		// Calculate hash and size
		hash, size, err := tds.calculateFileHashAndSize(path)
		if err != nil {
			tds.logger.Warn("Failed to calculate file hash", zap.String("path", path), zap.Error(err))
			hash = ""
			size = 0
		}

		discoveredTemplate := &DiscoveredTemplate{
			ID:          template.ID,
			Name:        template.Info.Name,
			Description: template.Info.Description,
			Severity:    string(template.Info.Severity),
			Type:        string(template.Detection.Type),
			Content:     "", // Would need to read file content
			Source: TemplateSource{
				Type:        TemplateSourceRepository,
				Name:        repo.Name,
				URL:         repo.URL,
				Priority:    repo.Priority,
				LastUpdated: time.Now(),
			},
			FilePath:  path,
			Hash:      hash,
			Size:      size,
			CreatedAt: template.LoadedAt,
			UpdatedAt: time.Now(),
		}

		templates = append(templates, discoveredTemplate)
		return nil
	})

	return templates, err
}

// discoverLocalTemplates discovers templates from local directories
func (tds *TemplateDiscoveryService) discoverLocalTemplates(ctx context.Context, templateDirs []string) ([]*DiscoveredTemplate, error) {
	tds.logger.Debug("Discovering local templates", zap.Strings("dirs", templateDirs))

	var templates []*DiscoveredTemplate

	for _, dir := range templateDirs {
		dirTemplates, err := tds.discoverTemplatesFromDirectory(ctx, dir)
		if err != nil {
			tds.logger.Warn("Failed to discover templates from directory", zap.String("dir", dir), zap.Error(err))
			continue
		}

		templates = append(templates, dirTemplates...)
	}

	tds.logger.Debug("Discovered local templates", zap.Int("count", len(templates)))
	return templates, nil
}

// discoverTemplatesFromDirectory discovers templates from a specific directory
func (tds *TemplateDiscoveryService) discoverTemplatesFromDirectory(ctx context.Context, dir string) ([]*DiscoveredTemplate, error) {
	var templates []*DiscoveredTemplate

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".yaml") {
			return nil
		}

		// Load and parse template
		template, err := tds.parser.LoadTemplate(path)
		if err != nil {
			tds.logger.Warn("Failed to load local template", zap.String("path", path), zap.Error(err))
			return nil
		}

		// Calculate hash and size
		hash, size, err := tds.calculateFileHashAndSize(path)
		if err != nil {
			tds.logger.Warn("Failed to calculate file hash", zap.String("path", path), zap.Error(err))
			hash = ""
			size = 0
		}

		discoveredTemplate := &DiscoveredTemplate{
			ID:          template.ID,
			Name:        template.Info.Name,
			Description: template.Info.Description,
			Severity:    string(template.Info.Severity),
			Type:        string(template.Detection.Type),
			Content:     "", // Would need to read file content
			Source: TemplateSource{
				Type:        TemplateSourceLocal,
				Name:        "local",
				Priority:    1,
				LastUpdated: time.Now(),
			},
			FilePath:  path,
			Hash:      hash,
			Size:      size,
			CreatedAt: template.LoadedAt,
			UpdatedAt: time.Now(),
		}

		templates = append(templates, discoveredTemplate)
		return nil
	})

	return templates, err
}

// calculateFileHashAndSize calculates SHA256 hash and size of a file
func (tds *TemplateDiscoveryService) calculateFileHashAndSize(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	size := stat.Size()

	// Calculate hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

// resolveConflictsAndUpdate resolves template conflicts and updates ValKey
func (tds *TemplateDiscoveryService) resolveConflictsAndUpdate(ctx context.Context, result *DiscoveryResult) error {
	// Group templates by ID to identify conflicts
	templateGroups := make(map[string][]*DiscoveredTemplate)
	for _, template := range result.Templates {
		templateGroups[template.ID] = append(templateGroups[template.ID], template)
	}

	// Resolve conflicts and update ValKey
	for templateID, templates := range templateGroups {
		if len(templates) == 1 {
			// No conflict, just update
			if err := tds.updateTemplateInValKey(ctx, templates[0]); err != nil {
				tds.logger.Error("Failed to update template in ValKey", zap.String("template_id", templateID), zap.Error(err))
			}
		} else {
			// Conflict detected, resolve based on priority
			resolvedTemplate := tds.resolveTemplateConflict(templates)
			if err := tds.updateTemplateInValKey(ctx, resolvedTemplate); err != nil {
				tds.logger.Error("Failed to update resolved template in ValKey", zap.String("template_id", templateID), zap.Error(err))
			}
		}
	}

	// Update global manifest
	manifest := &TemplateManifest{
		Name:        "agent-templates",
		Version:     "1.0.0",
		Description: "Global template manifest",
		LastUpdated: time.Now(),
		Templates:   make(map[string]TemplateMetadata),
		Sources:     result.Sources,
		Statistics:  result.Statistics,
	}

	// Convert discovered templates to metadata
	for _, template := range result.Templates {
		metadata := TemplateMetadata{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			Author:      "System",
			Version:     "1.0.0",
			Severity:    template.Severity,
			Tags:        []string{template.Type},
			Category:    template.Type,
			Source:      template.Source,
			CreatedAt:   template.CreatedAt,
			UpdatedAt:   template.UpdatedAt,
		}
		manifest.Templates[template.ID] = metadata
	}

	return tds.valkeyStore.SetTemplateManifest(ctx, manifest)
}

// resolveTemplateConflict resolves conflicts between templates with the same ID
func (tds *TemplateDiscoveryService) resolveTemplateConflict(templates []*DiscoveredTemplate) *DiscoveredTemplate {
	if len(templates) == 0 {
		return nil
	}

	// Sort by priority (custom > repository > local)
	var highestPriority *DiscoveredTemplate
	highestPriorityValue := -1

	for _, template := range templates {
		priority := 0
		switch template.Source.Type {
		case TemplateSourceCustom:
			priority = 100
		case TemplateSourceRepository:
			priority = 50 + template.Source.Priority
		case TemplateSourceLocal:
			priority = 1
		}

		if priority > highestPriorityValue {
			highestPriorityValue = priority
			highestPriority = template
		}
	}

	return highestPriority
}

// updateTemplateInValKey updates a template in ValKey storage
func (tds *TemplateDiscoveryService) updateTemplateInValKey(ctx context.Context, template *DiscoveredTemplate) error {
	// Update template content
	content := &TemplateContent{
		ID:        template.ID,
		Content:   template.Content,
		Hash:      template.Hash,
		Size:      template.Size,
		UpdatedAt: template.UpdatedAt,
	}

	if err := tds.valkeyStore.SetTemplateContent(ctx, template.ID, content); err != nil {
		return fmt.Errorf("failed to set template content: %w", err)
	}

	// Update template metadata
	metadata := &TemplateMetadata{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Author:      "System",
		Version:     "1.0.0",
		Severity:    template.Severity,
		Tags:        []string{template.Type},
		Category:    template.Type,
		Source:      template.Source,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}

	if err := tds.valkeyStore.SetTemplateMetadata(ctx, template.ID, metadata); err != nil {
		return fmt.Errorf("failed to set template metadata: %w", err)
	}

	return nil
}

// GetTemplateByID retrieves a specific template by ID
func (tds *TemplateDiscoveryService) GetTemplateByID(ctx context.Context, templateID string) (*DiscoveredTemplate, error) {
	return tds.loadTemplateFromValKey(ctx, templateID)
}

// ListAllTemplates lists all templates from ValKey (optimized for UI listing)
func (tds *TemplateDiscoveryService) ListAllTemplates(ctx context.Context) ([]*DiscoveredTemplate, error) {
	tds.logger.Debug("Listing all templates from ValKey")

	// Use the same ValKey discovery method but optimize for listing
	return tds.discoverTemplatesFromValKey(ctx)
}

// GetTemplateSummary returns a summary of template statistics without loading full content
func (tds *TemplateDiscoveryService) GetTemplateSummary(ctx context.Context) (*TemplateStatistics, error) {
	tds.logger.Debug("Getting template summary from ValKey")

	// Get global manifest if available
	manifest, err := tds.valkeyStore.GetTemplateManifest(ctx)
	if err != nil {
		// Fallback to counting templates directly
		metaKeys, err := tds.valkeyStore.ListTemplateMetaKeys(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get template count: %w", err)
		}

		// Count by type
		stats := &TemplateStatistics{
			TotalTemplates: len(metaKeys),
			LastSyncTime:   time.Now(),
		}

		// Count by source type
		for _, metaKey := range metaKeys {
			templateID := strings.TrimPrefix(metaKey, "agent:template:meta:")
			metadata, err := tds.valkeyStore.GetTemplateMetadata(ctx, templateID)
			if err != nil {
				continue
			}

			switch metadata.Source.Type {
			case TemplateSourceCustom:
				stats.CustomTemplates++
			case TemplateSourceRepository:
				stats.RepositoryTemplates++
			case TemplateSourceLocal:
				stats.LocalTemplates++
			}
		}

		stats.ActiveTemplates = stats.TotalTemplates
		return stats, nil
	}

	return &manifest.Statistics, nil
}

// discoverTemplatesFromValKey discovers all templates from ValKey store
func (tds *TemplateDiscoveryService) discoverTemplatesFromValKey(ctx context.Context) ([]*DiscoveredTemplate, error) {
	tds.logger.Debug("Discovering templates from ValKey store")

	// Get all template metadata keys
	metaKeys, err := tds.valkeyStore.ListTemplateMetaKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list template metadata keys: %w", err)
	}

	var templates []*DiscoveredTemplate
	for _, metaKey := range metaKeys {
		// Extract template ID from metadata key
		templateID := strings.TrimPrefix(metaKey, "agent:template:meta:")

		template, err := tds.loadTemplateFromValKey(ctx, templateID)
		if err != nil {
			tds.logger.Warn("Failed to load template from ValKey", zap.String("template_id", templateID), zap.Error(err))
			continue
		}

		templates = append(templates, template)
	}

	tds.logger.Debug("Discovered templates from ValKey", zap.Int("count", len(templates)))
	return templates, nil
}

// loadTemplateFromValKey loads a single template from ValKey
func (tds *TemplateDiscoveryService) loadTemplateFromValKey(ctx context.Context, templateID string) (*DiscoveredTemplate, error) {
	// Get template metadata
	metadata, err := tds.valkeyStore.GetTemplateMetadata(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template metadata: %w", err)
	}

	// Get template content
	content, err := tds.valkeyStore.GetTemplateContent(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template content: %w", err)
	}

	// Create discovered template from ValKey data
	discoveredTemplate := &DiscoveredTemplate{
		ID:          metadata.ID,
		Name:        metadata.Name,
		Description: metadata.Description,
		Severity:    metadata.Severity,
		Type:        metadata.Category,
		Content:     content.Content,
		Source:      metadata.Source,
		FilePath:    "", // ValKey templates don't have file paths
		Hash:        content.Hash,
		Size:        content.Size,
		CreatedAt:   metadata.CreatedAt,
		UpdatedAt:   metadata.UpdatedAt,
	}

	return discoveredTemplate, nil
}

// discoverFromFilesystemFallback falls back to filesystem discovery if ValKey is not available
func (tds *TemplateDiscoveryService) discoverFromFilesystemFallback(ctx context.Context, templateDirs []string) (*DiscoveryResult, error) {
	tds.logger.Warn("Falling back to filesystem template discovery")

	result := &DiscoveryResult{
		Templates:     []*DiscoveredTemplate{},
		Sources:       make(map[string]TemplateSource),
		Statistics:    TemplateStatistics{},
		Errors:        []string{},
		LastDiscovery: time.Now(),
	}

	// 1. Discover custom templates from filesystem
	customTemplates, err := tds.discoverCustomTemplates(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Filesystem custom template discovery failed: %v", err))
	} else {
		result.Templates = append(result.Templates, customTemplates...)
		result.Statistics.CustomTemplates = len(customTemplates)
	}

	// 2. Discover repository templates from filesystem
	repoTemplates, err := tds.discoverRepositoryTemplates(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Filesystem repository template discovery failed: %v", err))
	} else {
		result.Templates = append(result.Templates, repoTemplates...)
		result.Statistics.RepositoryTemplates = len(repoTemplates)
	}

	// 3. Discover local templates from filesystem
	localTemplates, err := tds.discoverLocalTemplates(ctx, templateDirs)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Filesystem local template discovery failed: %v", err))
	} else {
		result.Templates = append(result.Templates, localTemplates...)
		result.Statistics.LocalTemplates = len(localTemplates)
	}

	// Update statistics
	result.Statistics.TotalTemplates = len(result.Templates)
	result.Statistics.ActiveTemplates = len(result.Templates)
	result.Statistics.LastSyncTime = time.Now()

	return result, nil
}
