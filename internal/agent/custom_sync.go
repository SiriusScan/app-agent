package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	pb "github.com/SiriusScan/app-agent/proto/hello"
	"go.uber.org/zap"
)

// CustomContentSync handles synchronization of custom templates and scripts from the server
type CustomContentSync struct {
	agent    *Agent
	logger   *zap.Logger
	config   *config.AgentConfig
	lastSync time.Time
}

// NewCustomContentSync creates a new custom content sync handler
func NewCustomContentSync(agent *Agent) *CustomContentSync {
	return &CustomContentSync{
		agent:    agent,
		logger:   agent.logger,
		config:   agent.config,
		lastSync: time.Time{}, // Zero time means never synced
	}
}

// SyncCustomContent synchronizes custom templates and scripts from the server
func (cs *CustomContentSync) SyncCustomContent(ctx context.Context) error {
	cs.logger.Info("Starting custom content synchronization")

	// Create custom directories if they don't exist
	if err := cs.ensureCustomDirectories(); err != nil {
		return fmt.Errorf("failed to ensure custom directories: %w", err)
	}

	// Sync templates
	if err := cs.syncCustomTemplates(ctx); err != nil {
		cs.logger.Error("Failed to sync custom templates", zap.Error(err))
		// Continue with scripts even if templates fail
	}

	// Sync scripts
	if err := cs.syncCustomScripts(ctx); err != nil {
		cs.logger.Error("Failed to sync custom scripts", zap.Error(err))
		return fmt.Errorf("failed to sync custom scripts: %w", err)
	}

	cs.lastSync = time.Now()
	cs.logger.Info("Custom content synchronization completed", zap.Time("last_sync", cs.lastSync))
	return nil
}

// ensureCustomDirectories creates the custom templates and scripts directories
func (cs *CustomContentSync) ensureCustomDirectories() error {
	dirs := []string{cs.config.CustomTemplatesDir, cs.config.CustomScriptsDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		cs.logger.Debug("Ensured custom directory exists", zap.String("directory", dir))
	}

	return nil
}

// syncCustomTemplates synchronizes custom templates from the server
func (cs *CustomContentSync) syncCustomTemplates(ctx context.Context) error {
	cs.logger.Info("Syncing custom templates from server")

	// Create request for custom templates
	req := &pb.CustomContentRequest{
		AgentId:           cs.config.AgentID,
		LastSyncTimestamp: cs.lastSync.Format(time.RFC3339),
	}

	// Call server to get custom templates
	resp, err := cs.agent.client.GetCustomTemplates(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get custom templates from server: %w", err)
	}

	cs.logger.Info("Received custom templates from server",
		zap.Int("template_count", len(resp.Templates)),
		zap.String("last_updated", resp.LastUpdated))

	// Save each template to local storage
	for _, template := range resp.Templates {
		if err := cs.saveCustomTemplate(template); err != nil {
			cs.logger.Error("Failed to save custom template",
				zap.String("template_id", template.Id),
				zap.Error(err))
			continue // Continue with other templates
		}
		cs.logger.Debug("Saved custom template", zap.String("template_id", template.Id))
	}

	return nil
}

// syncCustomScripts synchronizes custom scripts from the server
func (cs *CustomContentSync) syncCustomScripts(ctx context.Context) error {
	cs.logger.Info("Syncing custom scripts from server")

	// Create request for custom scripts
	req := &pb.CustomContentRequest{
		AgentId:           cs.config.AgentID,
		LastSyncTimestamp: cs.lastSync.Format(time.RFC3339),
	}

	// Call server to get custom scripts
	resp, err := cs.agent.client.GetCustomScripts(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get custom scripts from server: %w", err)
	}

	cs.logger.Info("Received custom scripts from server",
		zap.Int("script_count", len(resp.Scripts)),
		zap.String("last_updated", resp.LastUpdated))

	// Save each script to local storage
	for _, script := range resp.Scripts {
		if err := cs.saveCustomScript(script); err != nil {
			cs.logger.Error("Failed to save custom script",
				zap.String("script_id", script.Id),
				zap.Error(err))
			continue // Continue with other scripts
		}
		cs.logger.Debug("Saved custom script", zap.String("script_id", script.Id))
	}

	return nil
}

// saveCustomTemplate saves a custom template to local storage
func (cs *CustomContentSync) saveCustomTemplate(template *pb.CustomContent) error {
	// Determine file extension based on template type
	extension := ".yaml" // Default to YAML for templates

	// Create filename
	filename := filepath.Join(cs.config.CustomTemplatesDir, template.Id+extension)

	// Write template content to file
	if err := os.WriteFile(filename, []byte(template.Content), 0644); err != nil {
		return fmt.Errorf("failed to write template file %s: %w", filename, err)
	}

	cs.logger.Debug("Saved custom template to file",
		zap.String("template_id", template.Id),
		zap.String("filename", filename))

	return nil
}

// saveCustomScript saves a custom script to local storage
func (cs *CustomContentSync) saveCustomScript(script *pb.CustomContent) error {
	// Determine file extension based on script type
	extension := ".sh" // Default to shell script

	// Try to infer extension from script content or metadata
	if script.Metadata != nil {
		if scriptType, exists := script.Metadata["type"]; exists {
			switch scriptType {
			case "powershell":
				extension = ".ps1"
			case "python":
				extension = ".py"
			case "bash":
				extension = ".sh"
			}
		}
	}

	// Create filename
	filename := filepath.Join(cs.config.CustomScriptsDir, script.Id+extension)

	// Write script content to file
	if err := os.WriteFile(filename, []byte(script.Content), 0755); err != nil {
		return fmt.Errorf("failed to write script file %s: %w", filename, err)
	}

	cs.logger.Debug("Saved custom script to file",
		zap.String("script_id", script.Id),
		zap.String("filename", filename))

	return nil
}

// GetLastSyncTime returns the last sync time
func (cs *CustomContentSync) GetLastSyncTime() time.Time {
	return cs.lastSync
}

// GetCustomTemplatesDir returns the custom templates directory
func (cs *CustomContentSync) GetCustomTemplatesDir() string {
	return cs.config.CustomTemplatesDir
}

// GetCustomScriptsDir returns the custom scripts directory
func (cs *CustomContentSync) GetCustomScriptsDir() string {
	return cs.config.CustomScriptsDir
}
