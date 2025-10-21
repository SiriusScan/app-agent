package valkey

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

// GitHubSyncManager manages synchronization with GitHub repositories
type GitHubSyncManager struct {
	storage     *ValKeyTemplateStorage
	logger      *zap.Logger
	repoPath    string
	repoURL     string
	lastSync    time.Time
}

// NewGitHubSyncManager creates a new GitHub sync manager
func NewGitHubSyncManager(storage *ValKeyTemplateStorage, logger *zap.Logger, repoPath, repoURL string) *GitHubSyncManager {
	return &GitHubSyncManager{
		storage:  storage,
		logger:   logger,
		repoPath: repoPath,
		repoURL:  repoURL,
	}
}

// RepositoryManifest represents the repository manifest structure
type RepositoryManifest struct {
	Version     string                 `json:"version"`
	Updated     string                 `json:"updated"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Components  map[string]Component   `json:"components"`
	Statistics  map[string]interface{} `json:"statistics"`
}

// Component represents a repository component
type Component struct {
	Path    string `json:"path"`
	Manifest string `json:"manifest"`
	Version string `json:"version"`
	Updated string `json:"updated"`
}

// GitHubTemplateManifest represents the templates manifest structure from GitHub
type GitHubTemplateManifest struct {
	Version    string                        `json:"version"`
	Updated    string                        `json:"updated"`
	Description string                       `json:"description"`
	Templates  map[string]*TemplateManifestEntry `json:"templates"`
	Statistics map[string]interface{}        `json:"statistics"`
}

// TemplateManifestEntry represents a template entry in the manifest
type TemplateManifestEntry struct {
	ID               string    `json:"id"`
	Version          string    `json:"version"`
	Checksum         string    `json:"checksum"`
	Size             int64     `json:"size"`
	Severity         string    `json:"severity"`
	Platforms        []string  `json:"platforms"`
	DetectionType    string    `json:"detection_type"`
	Author           string    `json:"author"`
	Created          string    `json:"created"`
	Updated          string    `json:"updated"`
	VulnerabilityIDs []string  `json:"vulnerability_ids"`
}

// SyncFromGitHub synchronizes templates from GitHub repository
func (g *GitHubSyncManager) SyncFromGitHub(ctx context.Context) error {
	g.logger.Info("Starting GitHub template sync",
		zap.String("repo_url", g.repoURL),
		zap.String("repo_path", g.repoPath))

	// Clone or update repository
	if err := g.ensureRepository(ctx); err != nil {
		return fmt.Errorf("failed to ensure repository: %w", err)
	}

	// Load repository manifest
	repoManifest, err := g.loadRepositoryManifest()
	if err != nil {
		return fmt.Errorf("failed to load repository manifest: %w", err)
	}

	// Load templates manifest
	templatesManifest, err := g.loadTemplatesManifest()
	if err != nil {
		return fmt.Errorf("failed to load templates manifest: %w", err)
	}

	// Process templates
	var processedCount int
	var errorCount int

	for templatePath, templateEntry := range templatesManifest.Templates {
		if err := g.processTemplate(ctx, templatePath, templateEntry); err != nil {
			g.logger.Error("Failed to process template",
				zap.String("path", templatePath),
				zap.Error(err))
			errorCount++
			continue
		}
		processedCount++
	}

	// Update global manifest
	if err := g.updateGlobalManifest(ctx, repoManifest, templatesManifest); err != nil {
		return fmt.Errorf("failed to update global manifest: %w", err)
	}

	g.lastSync = time.Now()
	g.logger.Info("GitHub template sync completed",
		zap.Int("processed", processedCount),
		zap.Int("errors", errorCount),
		zap.Time("last_sync", g.lastSync))

	return nil
}

// ensureRepository clones or updates the repository
func (g *GitHubSyncManager) ensureRepository(ctx context.Context) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(g.repoPath), 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(g.repoPath, ".git")); os.IsNotExist(err) {
		// Clone repository
		g.logger.Info("Cloning repository", zap.String("url", g.repoURL))
		cmd := exec.CommandContext(ctx, "git", "clone", g.repoURL, g.repoPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone repository: %s, error: %w", string(output), err)
		}
	} else {
		// Update existing repository
		g.logger.Info("Updating repository", zap.String("path", g.repoPath))
		cmd := exec.CommandContext(ctx, "git", "pull", "origin", "main")
		cmd.Dir = g.repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update repository: %s, error: %w", string(output), err)
		}
	}

	return nil
}

// loadRepositoryManifest loads the repository manifest
func (g *GitHubSyncManager) loadRepositoryManifest() (*RepositoryManifest, error) {
	manifestPath := filepath.Join(g.repoPath, "repository-manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository manifest: %w", err)
	}

	var manifest RepositoryManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse repository manifest: %w", err)
	}

	return &manifest, nil
}

// loadTemplatesManifest loads the templates manifest
func (g *GitHubSyncManager) loadTemplatesManifest() (*GitHubTemplateManifest, error) {
	manifestPath := filepath.Join(g.repoPath, "templates", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates manifest: %w", err)
	}

	var manifest GitHubTemplateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse templates manifest: %w", err)
	}

	return &manifest, nil
}

// processTemplate processes a single template
func (g *GitHubSyncManager) processTemplate(ctx context.Context, templatePath string, entry *TemplateManifestEntry) error {
	// Read template file
	fullPath := filepath.Join(g.repoPath, "templates", templatePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse template
	template, err := g.parseTemplate(content, entry)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Validate template
	if err := g.storage.ValidateTemplate(template, content); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Store template in ValKey
	if err := g.storage.StoreTemplate(ctx, template, content, false); err != nil {
		return fmt.Errorf("failed to store template: %w", err)
	}

	g.logger.Debug("Template processed successfully",
		zap.String("id", template.ID),
		zap.String("path", templatePath))

	return nil
}

// parseTemplate parses template content and creates a types.Template
func (g *GitHubSyncManager) parseTemplate(content []byte, entry *TemplateManifestEntry) (*types.Template, error) {
	// Note: created/updated times are not used in the current template structure

	// Create template
	template := &types.Template{
		ID: entry.ID,
		Info: types.TemplateInfo{
			Name:        entry.ID, // Use ID as name
			Author:      entry.Author,
			Severity:    types.Severity(entry.Severity),
			Version:     entry.Version,
			CVE:         entry.VulnerabilityIDs,
		},
		Detection: types.DetectionConfig{
			Steps: []types.DetectionStep{
				{
					Type: entry.DetectionType,
				},
			},
		},
	}

	return template, nil
}

// updateGlobalManifest updates the global template manifest
func (g *GitHubSyncManager) updateGlobalManifest(ctx context.Context, repoManifest *RepositoryManifest, templatesManifest *GitHubTemplateManifest) error {
	// Get current manifest
	currentManifest, err := g.storage.GetTemplateManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current manifest: %w", err)
	}

	// Update manifest
	currentManifest.Version = repoManifest.Version
	currentManifest.LastSync = time.Now()

	// Update statistics
	currentManifest.Statistics = TemplateStatistics{
		TotalTemplates:    len(templatesManifest.Templates),
		StandardTemplates: len(templatesManifest.Templates),
		CustomTemplates:   0, // Will be updated separately
		ByType:           make(map[string]int),
		ByPlatform:       make(map[string]int),
		BySeverity:       make(map[string]int),
	}

	// Count by type, platform, and severity
	for _, entry := range templatesManifest.Templates {
		// Count by type
		currentManifest.Statistics.ByType[entry.DetectionType]++

		// Count by platform
		for _, platform := range entry.Platforms {
			currentManifest.Statistics.ByPlatform[platform]++
		}

		// Count by severity
		currentManifest.Statistics.BySeverity[entry.Severity]++
	}

	// Store updated manifest
	if err := g.storage.UpdateTemplateManifest(ctx, currentManifest); err != nil {
		return fmt.Errorf("failed to update global manifest: %w", err)
	}

	return nil
}

// GetLastSyncTime returns the last sync time
func (g *GitHubSyncManager) GetLastSyncTime() time.Time {
	return g.lastSync
}

// IsSyncNeeded checks if a sync is needed based on time interval
func (g *GitHubSyncManager) IsSyncNeeded(interval time.Duration) bool {
	return time.Since(g.lastSync) > interval
}

// StartPeriodicSync starts a periodic sync goroutine
func (g *GitHubSyncManager) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	g.logger.Info("Starting periodic GitHub sync",
		zap.Duration("interval", interval))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial sync
	if err := g.SyncFromGitHub(ctx); err != nil {
		g.logger.Error("Initial GitHub sync failed", zap.Error(err))
	}

	// Periodic sync
	for {
		select {
		case <-ctx.Done():
			g.logger.Info("Periodic GitHub sync stopped")
			return
		case <-ticker.C:
			if err := g.SyncFromGitHub(ctx); err != nil {
				g.logger.Error("Periodic GitHub sync failed", zap.Error(err))
			}
		}
	}
}
