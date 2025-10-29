package template

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/SiriusScan/app-agent/internal/commands"
	templateagent "github.com/SiriusScan/app-agent/internal/template/agent"
)

var (
	// Global sync manager reference - set by integration layer
	syncManagerMutex  sync.RWMutex
	globalSyncManager *templateagent.AgentSyncManager
)

func init() {
	commands.Register("internal:template sync", &SyncCommand{})
	commands.RegisterAlias("template sync", "internal:template sync")
	commands.RegisterAlias("sync templates", "internal:template sync")
}

// SetGlobalSyncManager sets the global sync manager reference
func SetGlobalSyncManager(manager *templateagent.AgentSyncManager) {
	syncManagerMutex.Lock()
	defer syncManagerMutex.Unlock()
	globalSyncManager = manager
}

// GetGlobalSyncManager gets the global sync manager reference
func GetGlobalSyncManager() *templateagent.AgentSyncManager {
	syncManagerMutex.RLock()
	defer syncManagerMutex.RUnlock()
	return globalSyncManager
}

// SyncCommand synchronizes templates from the server
type SyncCommand struct{}

// Execute runs the template sync command
func (c *SyncCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (string, error) {
	agentInfo.Logger.Info("Executing template sync command")

	// Get the global sync manager
	syncManager := GetGlobalSyncManager()
	if syncManager == nil {
		return "", fmt.Errorf("sync manager not initialized; cannot sync templates")
	}

	// Trigger template sync from server via gRPC
	if err := syncManager.SyncFromServer(ctx); err != nil {
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
	cacheStatus, err := syncManager.GetCacheStatus()
	if err != nil {
		agentInfo.Logger.Warn("Failed to get cache status after sync")
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": "Template sync completed successfully",
		"details": map[string]interface{}{
			"last_sync":          cacheStatus.LastSync,
			"total_templates":    cacheStatus.Statistics.TotalTemplates,
			"standard_templates": cacheStatus.Statistics.StandardTemplates,
			"custom_templates":   cacheStatus.Statistics.CustomTemplates,
			"cache_size":         cacheStatus.Statistics.CacheSize,
		},
	}

	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(jsonOutput), nil
}
