package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	valkey "github.com/valkey-io/valkey-go"
	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/types"
	templatevalkey "github.com/SiriusScan/app-agent/internal/template/valkey"
)

// ServerTemplateManager manages templates on the server side
type ServerTemplateManager struct {
	valkeyClient valkey.Client
	storage      *templatevalkey.ValKeyTemplateStorage
	githubSync   *templatevalkey.GitHubSyncManager
	logger       *zap.Logger
	config       *TemplateConfig
	server       *Server // Reference to server for agent communication
}

// TemplateConfig contains configuration for template management
type TemplateConfig struct {
	RepoURL     string        `json:"repo_url"`
	RepoPath    string        `json:"repo_path"`
	SyncInterval time.Duration `json:"sync_interval"`
	MaxTemplateSize int64     `json:"max_template_size"`
}

// NewServerTemplateManager creates a new server template manager
func NewServerTemplateManager(valkeyClient valkey.Client, logger *zap.Logger, config *TemplateConfig, server *Server) *ServerTemplateManager {
	// Create ValKey storage
	storage := templatevalkey.NewValKeyTemplateStorage(valkeyClient, logger)

	// Create GitHub sync manager
	githubSync := templatevalkey.NewGitHubSyncManager(storage, logger, config.RepoPath, config.RepoURL)

	return &ServerTemplateManager{
		valkeyClient: valkeyClient,
		storage:      storage,
		githubSync:   githubSync,
		logger:       logger,
		config:       config,
		server:       server,
	}
}

// SyncFromGitHub pulls standard templates from sirius-agent-modules
func (tm *ServerTemplateManager) SyncFromGitHub(ctx context.Context) error {
	tm.logger.Info("Starting GitHub template sync")
	
	if err := tm.githubSync.SyncFromGitHub(ctx); err != nil {
		tm.logger.Error("GitHub sync failed", zap.Error(err))
		return fmt.Errorf("GitHub sync failed: %w", err)
	}

	tm.logger.Info("GitHub template sync completed successfully")
	return nil
}

// StoreCustomTemplate stores a user-uploaded custom template
func (tm *ServerTemplateManager) StoreCustomTemplate(ctx context.Context, template *types.Template, content []byte) error {
	tm.logger.Info("Storing custom template",
		zap.String("id", template.ID),
		zap.String("author", template.Info.Author))

	// Validate template
	if err := tm.storage.ValidateTemplate(template, content); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Check size limit
	if int64(len(content)) > tm.config.MaxTemplateSize {
		return fmt.Errorf("template size exceeds limit: %d bytes", len(content))
	}

	// Store template
	if err := tm.storage.StoreTemplate(ctx, template, content, true); err != nil {
		return fmt.Errorf("failed to store custom template: %w", err)
	}

	// Update global manifest
	if err := tm.updateCustomTemplateManifest(ctx); err != nil {
		tm.logger.Warn("Failed to update custom template manifest", zap.Error(err))
	}

	// Push to online agents immediately
	if err := tm.pushToAgents(ctx, template, content); err != nil {
		tm.logger.Warn("Failed to push template to agents", zap.Error(err))
	}

	tm.logger.Info("Custom template stored successfully",
		zap.String("id", template.ID))

	return nil
}

// GetTemplateManifest returns the manifest for agent sync
func (tm *ServerTemplateManager) GetTemplateManifest(ctx context.Context) (*templatevalkey.TemplateManifest, error) {
	return tm.storage.GetTemplateManifest(ctx)
}

// GetTemplate retrieves a template by ID
func (tm *ServerTemplateManager) GetTemplate(ctx context.Context, templateID string, isCustom bool) (*types.Template, []byte, error) {
	return tm.storage.GetTemplate(ctx, templateID, isCustom)
}

// ListTemplates lists all templates
func (tm *ServerTemplateManager) ListTemplates(ctx context.Context, isCustom bool) ([]*templatevalkey.TemplateInfo, error) {
	return tm.storage.ListTemplates(ctx, isCustom)
}

// DeleteCustomTemplate deletes a custom template
func (tm *ServerTemplateManager) DeleteCustomTemplate(ctx context.Context, templateID string) error {
	tm.logger.Info("Deleting custom template", zap.String("id", templateID))

	if err := tm.storage.DeleteTemplate(ctx, templateID, true); err != nil {
		return fmt.Errorf("failed to delete custom template: %w", err)
	}

	// Update global manifest
	if err := tm.updateCustomTemplateManifest(ctx); err != nil {
		tm.logger.Warn("Failed to update custom template manifest", zap.Error(err))
	}

	tm.logger.Info("Custom template deleted successfully",
		zap.String("id", templateID))

	return nil
}

// ValidateTemplate performs security and syntax validation
func (tm *ServerTemplateManager) ValidateTemplate(template *types.Template, content []byte) error {
	// Use storage validation
	if err := tm.storage.ValidateTemplate(template, content); err != nil {
		return err
	}

	// Additional security checks
	if err := tm.performSecurityScan(content); err != nil {
		return fmt.Errorf("security scan failed: %w", err)
	}

	return nil
}

// StartPeriodicSync starts the periodic GitHub sync
func (tm *ServerTemplateManager) StartPeriodicSync(ctx context.Context) {
	tm.logger.Info("Starting periodic template sync",
		zap.Duration("interval", tm.config.SyncInterval))

	go tm.githubSync.StartPeriodicSync(ctx, tm.config.SyncInterval)
}

// GetSyncStatus returns the current sync status
func (tm *ServerTemplateManager) GetSyncStatus() map[string]interface{} {
	return map[string]interface{}{
		"last_sync":     tm.githubSync.GetLastSyncTime(),
		"sync_needed":   tm.githubSync.IsSyncNeeded(tm.config.SyncInterval),
		"repo_url":      tm.config.RepoURL,
		"repo_path":     tm.config.RepoPath,
		"sync_interval": tm.config.SyncInterval.String(),
	}
}

// updateCustomTemplateManifest updates the global manifest with custom template statistics
func (tm *ServerTemplateManager) updateCustomTemplateManifest(ctx context.Context) error {
	// Get current manifest
	manifest, err := tm.storage.GetTemplateManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current manifest: %w", err)
	}

	// Count custom templates
	customTemplates, err := tm.storage.ListTemplates(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to list custom templates: %w", err)
	}

	// Update statistics
	manifest.Statistics.CustomTemplates = len(customTemplates)
	manifest.Statistics.TotalTemplates = manifest.Statistics.StandardTemplates + manifest.Statistics.CustomTemplates

	// Update custom template counts
	for _, template := range customTemplates {
		// Count by type
		manifest.Statistics.ByType[template.DetectionType]++

		// Count by platform
		for _, platform := range template.Platforms {
			manifest.Statistics.ByPlatform[platform]++
		}

		// Count by severity
		manifest.Statistics.BySeverity[template.Severity]++
	}

	// Store updated manifest
	return tm.storage.UpdateTemplateManifest(ctx, manifest)
}

// pushToAgents pushes a template to all online agents using command system
func (tm *ServerTemplateManager) pushToAgents(ctx context.Context, template *types.Template, content []byte) error {
	if tm.server == nil {
		tm.logger.Warn("Cannot push template - server reference not available")
		return nil
	}

	tm.logger.Info("Pushing template to agents",
		zap.String("template_id", template.ID),
		zap.String("severity", string(template.Info.Severity)))

	// For the demo, we'll use the existing command system to trigger template sync
	// This is simpler than modifying proto files and still demonstrates the functionality
	command := "internal:template sync"
	
	// Send sync command to all connected agents
	tm.server.agentsMutex.RLock()
	agentCount := len(tm.server.agents)
	for agentID := range tm.server.agents {
		// Use the existing command sending mechanism
		if err := tm.server.SendCommandToAgent(agentID, command); err != nil {
			tm.logger.Error("Failed to send template sync command to agent",
				zap.String("agent_id", agentID),
				zap.String("template_id", template.ID),
				zap.Error(err))
		} else {
			tm.logger.Debug("Template sync command sent to agent",
				zap.String("agent_id", agentID),
				zap.String("template_id", template.ID))
		}
	}
	tm.server.agentsMutex.RUnlock()

	tm.logger.Info("Template push completed",
		zap.String("template_id", template.ID),
		zap.Int("agents_notified", agentCount))

	return nil
}

// calculateChecksum calculates SHA256 checksum of content
func (tm *ServerTemplateManager) calculateChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%x", hash)
}

// performSecurityScan performs security validation on template content
func (tm *ServerTemplateManager) performSecurityScan(content []byte) error {
	contentStr := string(content)

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"eval(",
		"exec(",
		"system(",
		"shell_exec(",
		"passthru(",
		"`",
		"$((",
		"${",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentStr, pattern) {
			return fmt.Errorf("potentially dangerous pattern detected: %s", pattern)
		}
	}

	// Check for script injection patterns
	scriptPatterns := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
	}

	for _, pattern := range scriptPatterns {
		if strings.Contains(strings.ToLower(contentStr), pattern) {
			return fmt.Errorf("potential script injection pattern detected: %s", pattern)
		}
	}

	return nil
}

// GetTemplateStatistics returns template statistics
func (tm *ServerTemplateManager) GetTemplateStatistics(ctx context.Context) (map[string]interface{}, error) {
	manifest, err := tm.storage.GetTemplateManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get template manifest: %w", err)
	}

	return map[string]interface{}{
		"total_templates":     manifest.Statistics.TotalTemplates,
		"standard_templates":  manifest.Statistics.StandardTemplates,
		"custom_templates":    manifest.Statistics.CustomTemplates,
		"by_type":            manifest.Statistics.ByType,
		"by_platform":        manifest.Statistics.ByPlatform,
		"by_severity":        manifest.Statistics.BySeverity,
		"last_updated":       manifest.Updated,
		"version":            manifest.Version,
	}, nil
}
