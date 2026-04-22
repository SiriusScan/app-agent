package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/family/sirius"
	"github.com/SiriusScan/app-agent/internal/shell"
	"go.uber.org/zap"
)

type Context struct {
	AgentInfo        commands.AgentInfo
	PowerShellPath   string
	ScriptingEnabled bool
}

func NewContext(cfg *config.AgentConfig, logger *zap.Logger) *Context {
	psPath := cfg.PowerShellPath
	psFound := false
	var findErr error

	if psPath == "" {
		logger.Debug("PowerShell path not configured, attempting to find...")
		psPath, psFound, findErr = shell.FindPowerShell()
		if findErr != nil {
			logger.Warn("Error finding PowerShell", zap.String("executable", psPath), zap.Error(findErr))
		} else if psFound {
			logger.Info("Found PowerShell executable", zap.String("path", psPath))
		} else {
			logger.Info("PowerShell executable (pwsh/powershell.exe) not found in PATH")
		}
	} else {
		logger.Info("Using configured PowerShell path", zap.String("path", psPath))
		psFound = true
	}

	scriptingEnabled := cfg.EnableScripting && psFound
	logger.Info("Scripting capability determined",
		zap.Bool("config_enabled", cfg.EnableScripting),
		zap.Bool("powershell_found", psFound),
		zap.Bool("scripting_active", scriptingEnabled))

	startTime := time.Now()
	return &Context{
		AgentInfo: commands.AgentInfo{
			Logger:           logger,
			Config:           cfg,
			APIClient:        commands.NewAPIClientAdapter(),
			StartTime:        startTime,
			ScriptingEnabled: scriptingEnabled,
			PowerShellPath:   psPath,
		},
		PowerShellPath:   psPath,
		ScriptingEnabled: scriptingEnabled,
	}
}

func MergeConnectorConfig(base *config.AgentConfig, override sirius.ConnectorConfig) *config.AgentConfig {
	if base == nil {
		base = &config.AgentConfig{}
	}
	merged := *base

	if override.AgentID != "" {
		merged.AgentID = override.AgentID
	}
	if override.HostID != "" {
		merged.HostID = override.HostID
	}
	if override.ServerAddress != "" {
		merged.ServerAddress = override.ServerAddress
	}
	if override.APIBaseURL != "" {
		merged.ApiBaseURL = override.APIBaseURL
	}
	if override.PowerShellPath != "" {
		merged.PowerShellPath = override.PowerShellPath
	}
	if override.EnableScripting != nil {
		merged.EnableScripting = *override.EnableScripting
	}
	if merged.HostID == "" {
		merged.HostID = merged.AgentID
	}
	return &merged
}

func DispatchTemplateTask(ctx context.Context, info commands.AgentInfo, task sirius.TemplateTask) (*sirius.TaskResult, error) {
	command := buildTemplateCommand(task)
	return dispatchCommand(ctx, info, command, task.Format)
}

func DispatchInventoryTask(ctx context.Context, info commands.AgentInfo, task sirius.InventoryTask) (*sirius.TaskResult, error) {
	command := buildInventoryCommand(task)
	return dispatchCommand(ctx, info, command, task.Format)
}

func dispatchCommand(ctx context.Context, info commands.AgentInfo, command, format string) (*sirius.TaskResult, error) {
	output, err := commands.Dispatch(ctx, info, command)
	if format == "" {
		format = "json"
	}
	return &sirius.TaskResult{
		Output: output,
		Format: format,
	}, err
}

func buildTemplateCommand(task sirius.TemplateTask) string {
	parts := []string{"internal:template-scan"}

	switch {
	case task.TemplatePath != "":
		parts = append(parts, "--template", task.TemplatePath)
	case task.Directory != "":
		parts = append(parts, "--directory", task.Directory)
	default:
		parts = append(parts, "--all")
	}

	if task.Workers > 0 {
		parts = append(parts, "--workers", fmt.Sprintf("%d", task.Workers))
	}
	if task.TimeoutSeconds > 0 {
		parts = append(parts, "--timeout", fmt.Sprintf("%d", task.TimeoutSeconds))
	}
	if task.Format != "" {
		parts = append(parts, "--format", task.Format)
	}
	if task.ScanID != "" {
		parts = append(parts, "--scan-id="+task.ScanID)
	}

	return strings.Join(parts, " ")
}

func buildInventoryCommand(task sirius.InventoryTask) string {
	parts := []string{"internal:scan"}
	if len(task.Scripts) > 0 {
		parts = append(parts, "--scripts="+strings.Join(task.Scripts, ","))
	}
	return strings.Join(parts, " ")
}
