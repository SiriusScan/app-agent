package template

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SiriusScan/app-agent/internal/commands"
)

func init() {
	commands.Register("internal:template sync", &SyncCommand{})
	commands.RegisterAlias("template sync", "internal:template sync")
	commands.RegisterAlias("sync templates", "internal:template sync")
}

// SyncCommand synchronizes templates from the server
type SyncCommand struct{}

// Execute runs the template sync command
func (c *SyncCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (string, error) {
	agentInfo.Logger.Info("Executing template sync command")

	if agentInfo.TemplateSync == nil {
		return "", fmt.Errorf("sync manager not initialized; cannot sync templates")
	}

	// Trigger template sync from server via gRPC
	if err := agentInfo.TemplateSync.SyncFromServer(ctx); err != nil {
		agentInfo.Logger.Error("Template sync failed")

		result := map[string]interface{}{
			"status":  "error",
			"message": "Template sync failed",
			"error":   err.Error(),
		}

		jsonOutput, _ := json.MarshalIndent(result, "", "  ")
		return string(jsonOutput), err
	}

	// Get cache status to show sync results
	cacheStatus, err := agentInfo.TemplateSync.GetStatus(ctx)
	if err != nil {
		agentInfo.Logger.Warn("Failed to get cache status after sync")
		cacheStatus = &commands.TemplateSyncStatus{}
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": "Template sync completed successfully",
		"details": map[string]interface{}{
			"last_sync":          cacheStatus.LastSync,
			"total_templates":    cacheStatus.TotalTemplates,
			"standard_templates": cacheStatus.StandardTemplates,
			"custom_templates":   cacheStatus.CustomTemplates,
			"cache_size":         cacheStatus.CacheSize,
		},
	}

	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(jsonOutput), nil
}
