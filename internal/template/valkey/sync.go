package valkey

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/repository"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// repositoryStorage is the set of catalog operations used by GitHub sync.
type repositoryStorage interface {
	getTemplateMetadata(ctx context.Context, templateID string) (*TemplateInfo, string, error)
	GetTemplateManifest(ctx context.Context) (*TemplateManifest, error)
	storeRepositoryTemplate(ctx context.Context, template *types.Template, content []byte, repositoryID, metadataSnapshot string) error
	UpdateTemplateManifest(ctx context.Context, manifest *TemplateManifest) error
}

type stagedTemplate struct {
	repository.Template
	metadataSnapshot string
}

// GitHubSyncManager manages synchronization with GitHub repositories
type GitHubSyncManager struct {
	storage  repositoryStorage
	logger   *zap.Logger
	repoPath string
	repoURL  string
	branch   string
	lastSync time.Time
}

// NewGitHubSyncManager creates a new GitHub sync manager
func NewGitHubSyncManager(storage *ValKeyTemplateStorage, logger *zap.Logger, repoPath, repoURL, branch string) *GitHubSyncManager {
	return &GitHubSyncManager{
		storage:  storage,
		logger:   logger,
		repoPath: repoPath,
		repoURL:  repoURL,
		branch:   branch,
	}
}

// SyncFromGitHub synchronizes templates from a GitHub repository.
func (g *GitHubSyncManager) SyncFromGitHub(ctx context.Context, repoID string) error {
	g.logger.Info("Starting GitHub template sync",
		zap.String("repo_id", repoID),
		zap.String("repo_url", g.repoURL),
		zap.String("repo_path", g.repoPath))

	if err := g.ensureRepository(ctx); err != nil {
		return fmt.Errorf("prepare repository %q: %w", repoID, err)
	}
	return g.syncCheckout(ctx, repoID)
}

// syncCheckout validates the complete checkout before reading or updating its catalog records.
func (g *GitHubSyncManager) syncCheckout(ctx context.Context, repoID string) error {
	candidate, err := repository.Load(g.repoPath)
	for _, warning := range candidate.Warnings {
		g.logger.Warn("Legacy repository template validation",
			zap.String("repo_id", repoID), zap.Error(warning))
	}
	if err != nil {
		return fmt.Errorf("validate repository %q: %w", repoID, err)
	}

	// The standard and custom records currently share their metadata and
	// version keys. Keep the active custom record when it shadows a standard ID.
	templates := make([]stagedTemplate, 0, len(candidate.Templates))
	for _, template := range candidate.Templates {
		id := template.Definition.ID
		metadata, snapshot, err := g.storage.getTemplateMetadata(ctx, id)
		if err != nil {
			return fmt.Errorf("repository %q %s: check catalog metadata: %w", repoID, template.Path, err)
		}
		if metadata != nil {
			if metadata.IsCustom {
				g.logger.Debug("Custom template takes precedence over repository template",
					zap.String("repo_id", repoID), zap.String("id", id), zap.String("path", template.Path))
				continue
			}
			if owner := metadata.Metadata["repository_id"]; owner != "" && owner != repoID {
				return fmt.Errorf("repository %q %s: template %q is owned by repository %q", repoID, template.Path, id, owner)
			}
		}
		templates = append(templates, stagedTemplate{Template: template, metadataSnapshot: snapshot})
	}

	manifest, err := g.storage.GetTemplateManifest(ctx)
	if err != nil {
		return fmt.Errorf("repository %q: read catalog manifest: %w", repoID, err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("repository %q: %w", repoID, err)
	}

	// Publish the exact buffers read and checked above.
	for _, template := range templates {
		if err := g.storage.storeRepositoryTemplate(ctx, template.Definition, template.Content, repoID, template.metadataSnapshot); err != nil {
			return fmt.Errorf("repository %q %s: store template: %w", repoID, template.Path, err)
		}
	}

	now := time.Now()
	manifest.Version = candidate.RepositoryVersion
	manifest.LastSync = now
	manifest.Statistics = repositoryStatistics(templates, manifest.Statistics.CustomTemplates)
	if err := g.storage.UpdateTemplateManifest(ctx, manifest); err != nil {
		return fmt.Errorf("repository %q: update catalog manifest: %w", repoID, err)
	}

	g.lastSync = now
	g.logger.Info("GitHub template sync completed",
		zap.String("repo_id", repoID),
		zap.Int("processed", len(templates)),
		zap.Time("last_sync", g.lastSync))

	return nil
}

func repositoryStatistics(templates []stagedTemplate, customCount int) TemplateStatistics {
	stats := TemplateStatistics{
		TotalTemplates:    len(templates) + customCount,
		StandardTemplates: len(templates),
		CustomTemplates:   customCount,
		ByType:            make(map[string]int),
		ByPlatform:        make(map[string]int),
		BySeverity:        make(map[string]int),
	}
	for _, template := range templates {
		stats.ByType[template.Manifest.DetectionType]++
		for _, platform := range template.Manifest.Platforms {
			stats.ByPlatform[platform]++
		}
		stats.BySeverity[template.Manifest.Severity]++
	}
	return stats
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
		g.logger.Info("Updating repository", zap.String("path", g.repoPath), zap.String("branch", g.branch))
		cmd := exec.CommandContext(ctx, "git", "pull", "origin", g.branch)
		cmd.Dir = g.repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update repository: %s, error: %w", string(output), err)
		}
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
// Note: This method is deprecated in favor of queue-based sync triggered by the server
func (g *GitHubSyncManager) StartPeriodicSync(ctx context.Context, interval time.Duration, repoID string) {
	g.logger.Info("Starting periodic GitHub sync",
		zap.Duration("interval", interval),
		zap.String("repo_id", repoID))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial sync
	if err := g.SyncFromGitHub(ctx, repoID); err != nil {
		g.logger.Error("Initial GitHub sync failed", zap.Error(err))
	}

	// Periodic sync
	for {
		select {
		case <-ctx.Done():
			g.logger.Info("Periodic GitHub sync stopped")
			return
		case <-ticker.C:
			if err := g.SyncFromGitHub(ctx, repoID); err != nil {
				g.logger.Error("Periodic GitHub sync failed", zap.Error(err))
			}
		}
	}
}
