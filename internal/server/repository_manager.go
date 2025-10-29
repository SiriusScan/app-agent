package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	valkey "github.com/valkey-io/valkey-go"
	"go.uber.org/zap"

	templatevalkey "github.com/SiriusScan/app-agent/internal/template/valkey"
)

const (
	repositoryManifestKey = "sirius:agent-templates:repositories"
)

// Repository represents a template repository
type Repository struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	URL           string  `json:"url"`
	Branch        string  `json:"branch"`
	Priority      int     `json:"priority"`
	Enabled       bool    `json:"enabled"`
	LastSync      *string `json:"last_sync"`
	TemplateCount int     `json:"template_count"`
	Status        string  `json:"status"`
	ErrorMessage  *string `json:"error_message"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// RepositoryManifest represents the repository list structure
type RepositoryManifest struct {
	Repositories []Repository `json:"repositories"`
	Version      string       `json:"version"`
	UpdatedAt    string       `json:"updated_at"`
}

// RepositoryManager manages multiple template repositories
type RepositoryManager struct {
	valkeyClient valkey.Client
	logger       *zap.Logger
	basePath     string
	syncManagers map[string]*templatevalkey.GitHubSyncManager
	syncMutex    sync.RWMutex
	storage      *templatevalkey.ValKeyTemplateStorage
	server       *Server
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(valkeyClient valkey.Client, logger *zap.Logger, basePath string, server *Server) *RepositoryManager {
	storage := templatevalkey.NewValKeyTemplateStorage(valkeyClient, logger)

	return &RepositoryManager{
		valkeyClient: valkeyClient,
		logger:       logger,
		basePath:     basePath,
		syncManagers: make(map[string]*templatevalkey.GitHubSyncManager),
		storage:      storage,
		server:       server,
	}
}

// InitializeDefaultRepository creates default repository if none exist
func (rm *RepositoryManager) InitializeDefaultRepository(ctx context.Context) error {
	rm.logger.Info("Checking for existing repositories")

	// Check if repository manifest exists
	cmd := rm.valkeyClient.B().Get().Key(repositoryManifestKey).Build()
	resp := rm.valkeyClient.Do(ctx, cmd)

	if err := resp.Error(); err == nil {
		rm.logger.Info("Repository manifest already exists, skipping initialization")
		return nil
	}

	rm.logger.Info("No repositories found, initializing default repository")

	// Create default repository
	now := time.Now().Format(time.RFC3339)
	defaultRepo := Repository{
		ID:            "default-sirius-official",
		Name:          "Sirius Official",
		URL:           "https://github.com/SiriusScan/sirius-agent-modules",
		Branch:        "main",
		Priority:      1,
		Enabled:       true,
		LastSync:      nil,
		TemplateCount: 0,
		Status:        "never_synced",
		ErrorMessage:  nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	manifest := RepositoryManifest{
		Repositories: []Repository{defaultRepo},
		Version:      "1.0",
		UpdatedAt:    now,
	}

	// Save to Valkey
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	cmd = rm.valkeyClient.B().Set().Key(repositoryManifestKey).Value(string(data)).Build()
	if err := rm.valkeyClient.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	rm.logger.Info("Default repository initialized successfully",
		zap.String("id", defaultRepo.ID),
		zap.String("url", defaultRepo.URL))

	return nil
}

// LoadRepositories reads repository list from Valkey
func (rm *RepositoryManager) LoadRepositories(ctx context.Context) ([]Repository, error) {
	cmd := rm.valkeyClient.B().Get().Key(repositoryManifestKey).Build()
	resp := rm.valkeyClient.Do(ctx, cmd)

	if err := resp.Error(); err != nil {
		return nil, fmt.Errorf("failed to get repository manifest: %w", err)
	}

	manifestData, err := resp.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest to string: %w", err)
	}

	var manifest RepositoryManifest
	if err := json.Unmarshal([]byte(manifestData), &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return manifest.Repositories, nil
}

// SyncRepository syncs a specific repository
func (rm *RepositoryManager) SyncRepository(ctx context.Context, repoID string) error {
	rm.logger.Info("Syncing repository", zap.String("repo_id", repoID))

	// Load repositories
	repos, err := rm.LoadRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to load repositories: %w", err)
	}

	// Find target repository
	var targetRepo *Repository
	for i := range repos {
		if repos[i].ID == repoID {
			targetRepo = &repos[i]
			break
		}
	}

	if targetRepo == nil {
		return fmt.Errorf("repository not found: %s", repoID)
	}

	if !targetRepo.Enabled {
		rm.logger.Info("Repository is disabled, skipping sync", zap.String("repo_id", repoID))
		return nil
	}

	// Update status to syncing
	if err := rm.updateRepositoryStatus(ctx, repoID, "syncing", nil, 0); err != nil {
		rm.logger.Warn("Failed to update repository status", zap.Error(err))
	}

	// Get or create sync manager for this repository
	repoPath := filepath.Join(rm.basePath, repoID)
	syncManager := rm.getOrCreateSyncManager(repoID, repoPath, targetRepo.URL, targetRepo.Branch)

	// Perform sync
	if err := syncManager.SyncFromGitHub(ctx, repoID); err != nil {
		rm.logger.Error("Repository sync failed",
			zap.String("repo_id", repoID),
			zap.String("repo_name", targetRepo.Name),
			zap.Error(err))

		errorMsg := err.Error()
		if updateErr := rm.updateRepositoryStatus(ctx, repoID, "error", &errorMsg, 0); updateErr != nil {
			rm.logger.Warn("Failed to update error status", zap.Error(updateErr))
		}

		return fmt.Errorf("sync failed: %w", err)
	}

	// Count templates from this repository
	templateCount, err := rm.countRepositoryTemplates(ctx, repoID)
	if err != nil {
		rm.logger.Warn("Failed to count templates", zap.Error(err))
		templateCount = 0
	}

	// Update status to synced
	if err := rm.updateRepositoryStatus(ctx, repoID, "synced", nil, templateCount); err != nil {
		rm.logger.Warn("Failed to update synced status", zap.Error(err))
	}

	rm.logger.Info("Repository sync completed",
		zap.String("repo_id", repoID),
		zap.String("repo_name", targetRepo.Name),
		zap.Int("template_count", templateCount))

	// Notify connected agents
	if rm.server != nil {
		rm.notifyAgents(ctx)
	}

	return nil
}

// SyncAllRepositories syncs all enabled repositories
func (rm *RepositoryManager) SyncAllRepositories(ctx context.Context) error {
	rm.logger.Info("Syncing all enabled repositories")

	repos, err := rm.LoadRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to load repositories: %w", err)
	}

	var syncErrors []error
	successCount := 0
	skipCount := 0

	for _, repo := range repos {
		if !repo.Enabled {
			rm.logger.Debug("Skipping disabled repository",
				zap.String("repo_id", repo.ID),
				zap.String("repo_name", repo.Name))
			skipCount++
			continue
		}

		if err := rm.SyncRepository(ctx, repo.ID); err != nil {
			rm.logger.Error("Failed to sync repository",
				zap.String("repo_id", repo.ID),
				zap.String("repo_name", repo.Name),
				zap.Error(err))
			syncErrors = append(syncErrors, err)
		} else {
			successCount++
		}
	}

	rm.logger.Info("Repository sync batch completed",
		zap.Int("success", successCount),
		zap.Int("errors", len(syncErrors)),
		zap.Int("skipped", skipCount))

	if len(syncErrors) > 0 {
		return fmt.Errorf("sync completed with %d errors", len(syncErrors))
	}

	return nil
}

// DeleteRepository removes repository files and Valkey entries
func (rm *RepositoryManager) DeleteRepository(ctx context.Context, repoID string) error {
	rm.logger.Info("Deleting repository", zap.String("repo_id", repoID))

	// Remove sync manager
	rm.syncMutex.Lock()
	delete(rm.syncManagers, repoID)
	rm.syncMutex.Unlock()

	// Delete repository directory
	repoPath := filepath.Join(rm.basePath, repoID)
	if err := os.RemoveAll(repoPath); err != nil {
		rm.logger.Warn("Failed to delete repository directory",
			zap.String("path", repoPath),
			zap.Error(err))
		// Don't fail, continue cleanup
	}

	// Note: Templates from this repository remain in Valkey
	// They will be removed during next priority resolution
	// or when another repository with same template IDs syncs

	rm.logger.Info("Repository deleted", zap.String("repo_id", repoID))
	return nil
}

// Helper methods

func (rm *RepositoryManager) getOrCreateSyncManager(repoID, repoPath, repoURL, branch string) *templatevalkey.GitHubSyncManager {
	rm.syncMutex.Lock()
	defer rm.syncMutex.Unlock()

	if manager, exists := rm.syncManagers[repoID]; exists {
		return manager
	}

	manager := templatevalkey.NewGitHubSyncManager(rm.storage, rm.logger, repoPath, repoURL, branch)
	rm.syncManagers[repoID] = manager
	return manager
}

func (rm *RepositoryManager) updateRepositoryStatus(ctx context.Context, repoID, status string, errorMsg *string, templateCount int) error {
	// Load current manifest
	cmd := rm.valkeyClient.B().Get().Key(repositoryManifestKey).Build()
	resp := rm.valkeyClient.Do(ctx, cmd)

	if err := resp.Error(); err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	manifestData, err := resp.ToString()
	if err != nil {
		return fmt.Errorf("failed to convert manifest: %w", err)
	}

	var manifest RepositoryManifest
	if err := json.Unmarshal([]byte(manifestData), &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// Update repository
	found := false
	now := time.Now().Format(time.RFC3339)

	for i := range manifest.Repositories {
		if manifest.Repositories[i].ID == repoID {
			manifest.Repositories[i].Status = status
			manifest.Repositories[i].ErrorMessage = errorMsg
			manifest.Repositories[i].UpdatedAt = now

			if status == "synced" {
				manifest.Repositories[i].LastSync = &now
				manifest.Repositories[i].TemplateCount = templateCount
			}

			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("repository not found in manifest: %s", repoID)
	}

	manifest.UpdatedAt = now

	// Save manifest
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	cmd = rm.valkeyClient.B().Set().Key(repositoryManifestKey).Value(string(data)).Build()
	if err := rm.valkeyClient.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	return nil
}

func (rm *RepositoryManager) countRepositoryTemplates(ctx context.Context, repoID string) (int, error) {
	// Count templates by scanning template keys
	// This is a simple implementation that counts all standard templates
	// In a production system, you'd want to track which templates came from which repository

	cmd := rm.valkeyClient.B().Keys().Pattern("template:standard:*").Build()
	resp := rm.valkeyClient.Do(ctx, cmd)

	if err := resp.Error(); err != nil {
		return 0, err
	}

	keys, err := resp.AsStrSlice()
	if err != nil {
		return 0, err
	}

	return len(keys), nil
}

func (rm *RepositoryManager) notifyAgents(ctx context.Context) {
	if rm.server == nil {
		return
	}

	rm.logger.Info("Notifying connected agents of template updates")

	// Send template sync command to all connected agents
	rm.server.agentsMutex.RLock()
	agentCount := len(rm.server.agents)
	for agentID := range rm.server.agents {
		if err := rm.server.SendCommandToAgent(agentID, "internal:template sync"); err != nil {
			rm.logger.Error("Failed to send template sync command to agent",
				zap.String("agent_id", agentID),
				zap.Error(err))
		}
	}
	rm.server.agentsMutex.RUnlock()

	rm.logger.Info("Agent notification completed", zap.Int("agents_notified", agentCount))
}
