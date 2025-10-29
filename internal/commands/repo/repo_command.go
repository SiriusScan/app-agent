package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/repository"
	"go.uber.org/zap"
)

// RepoCommand implements repository management commands
type RepoCommand struct {
	repoIntegration *repository.RepositoryIntegration
}

// Ensures RepoCommand implements the Command interface at compile time.
var _ commands.Command = (*RepoCommand)(nil)

func init() {
	commands.Register("internal:repo", &RepoCommand{})
}

// Execute handles repository management commands
func (c *RepoCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (output string, err error) {
	agentInfo.Logger.Info("Executing repository command", zap.String("args", args))

	// Initialize repository integration if not already done
	if c.repoIntegration == nil {
		c.repoIntegration = repository.NewRepositoryIntegration(agentInfo.Logger)
		if err := c.repoIntegration.Initialize(ctx, agentInfo.Config.AgentID, agentInfo.Config.ServerAddress); err != nil {
			return "", fmt.Errorf("failed to initialize repository: %w", err)
		}
	}

	// Parse subcommand
	argsList := strings.Fields(args)
	if len(argsList) == 0 {
		return c.showHelp(), nil
	}

	subcommand := argsList[0]
	subArgs := strings.Join(argsList[1:], " ")

	switch subcommand {
	case "status":
		return c.handleStatus(ctx, agentInfo)
	case "update":
		return c.handleUpdate(ctx, agentInfo)
	case "list":
		return c.handleList(ctx, agentInfo, subArgs)
	case "validate":
		return c.handleValidate(ctx, agentInfo)
	case "help":
		return c.showHelp(), nil
	default:
		return fmt.Sprintf("Unknown subcommand: %s\n%s", subcommand, c.showHelp()), nil
	}
}

// handleStatus shows repository status
func (c *RepoCommand) handleStatus(ctx context.Context, agentInfo commands.AgentInfo) (string, error) {
	agentInfo.Logger.Info("Getting repository status")

	status, err := c.repoIntegration.GetRepositoryStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get repository status: %w", err)
	}

	// Format status output
	output := fmt.Sprintf("Repository Status:\n")
	output += fmt.Sprintf("  Initialized: %t\n", status.Initialized)
	output += fmt.Sprintf("  Local Path: %s\n", status.LocalPath)
	output += fmt.Sprintf("  Remote URL: %s\n", status.RemoteURL)
	output += fmt.Sprintf("  Current Version: %s\n", status.CurrentVersion)
	output += fmt.Sprintf("  Last Update: %s\n", status.LastUpdate.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("  Template Count: %d\n", status.TemplateCount)
	output += fmt.Sprintf("  Script Count: %d\n", status.ScriptCount)
	output += fmt.Sprintf("  Total Size: %d bytes\n", status.TotalSize)
	output += fmt.Sprintf("  Status: %s\n", status.Status)
	if status.ManifestVersion != "" {
		output += fmt.Sprintf("  Manifest Version: %s\n", status.ManifestVersion)
		output += fmt.Sprintf("  Manifest Updated: %s\n", status.ManifestUpdated.Format("2006-01-02 15:04:05"))
	}

	return output, nil
}

// handleUpdate performs repository update
func (c *RepoCommand) handleUpdate(ctx context.Context, agentInfo commands.AgentInfo) (string, error) {
	agentInfo.Logger.Info("Starting repository update")

	updateCtx, cancel := context.WithTimeout(ctx, 2*time.Minute) // 2 minute timeout
	defer cancel()

	err := c.repoIntegration.UpdateRepositoryIfNeeded(updateCtx)
	if err != nil {
		return "", fmt.Errorf("repository update failed: %w", err)
	}

	return "Repository update completed successfully", nil
}

// handleList lists repository content
func (c *RepoCommand) handleList(ctx context.Context, agentInfo commands.AgentInfo, args string) (string, error) {
	agentInfo.Logger.Info("Listing repository content", zap.String("args", args))

	// Parse list arguments
	argsList := strings.Fields(args)
	contentType := "all"
	if len(argsList) > 0 {
		contentType = argsList[0]
	}

	output := "Repository Content:\n\n"

	// List templates
	if contentType == "all" || contentType == "templates" {
		templatePaths, err := c.repoIntegration.LoadTemplatesFromRepository(ctx)
		if err != nil {
			agentInfo.Logger.Warn("Failed to load templates", zap.Error(err))
		} else {
			output += fmt.Sprintf("Templates (%d):\n", len(templatePaths))
			for i, path := range templatePaths {
				// Extract template name from path
				parts := strings.Split(path, "/")
				name := parts[len(parts)-1]
				output += fmt.Sprintf("  %d. %s\n", i+1, name)
			}
			output += "\n"
		}
	}

	// List scripts
	if contentType == "all" || contentType == "scripts" {
		scriptPaths, err := c.repoIntegration.LoadScriptsFromRepository(ctx)
		if err != nil {
			agentInfo.Logger.Warn("Failed to load scripts", zap.Error(err))
		} else {
			output += fmt.Sprintf("Scripts (%d):\n", len(scriptPaths))
			for i, path := range scriptPaths {
				// Extract script name from path
				parts := strings.Split(path, "/")
				name := parts[len(parts)-1]
				output += fmt.Sprintf("  %d. %s\n", i+1, name)
			}
			output += "\n"
		}
	}

	return output, nil
}

// handleValidate validates repository content
func (c *RepoCommand) handleValidate(ctx context.Context, agentInfo commands.AgentInfo) (string, error) {
	agentInfo.Logger.Info("Validating repository content")

	validationCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // 30 second timeout
	defer cancel()

	result, err := c.repoIntegration.ValidateRepositoryContent(validationCtx)
	if err != nil {
		return "", fmt.Errorf("repository validation failed: %w", err)
	}

	output := fmt.Sprintf("Repository Validation Results:\n")
	output += fmt.Sprintf("  Valid: %t\n", result.Valid)
	output += fmt.Sprintf("  Validation Type: %s\n", result.ValidationType)
	output += fmt.Sprintf("  Validated At: %s\n", result.ValidatedAt.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("  Error Count: %d\n", len(result.Errors))

	if len(result.Errors) > 0 {
		output += "\nErrors:\n"
		for i, err := range result.Errors {
			output += fmt.Sprintf("  %d. %s: %s\n", i+1, err.Type, err.Message)
			if err.Location != "" {
				output += fmt.Sprintf("     Location: %s\n", err.Location)
			}
		}
	}

	return output, nil
}

// showHelp shows command help
func (c *RepoCommand) showHelp() string {
	return `Repository Management Commands:

Usage: internal:repo <subcommand> [options]

Subcommands:
  status     - Show repository status and information
  update     - Update repository from remote source
  list       - List repository content (templates/scripts)
  validate   - Validate repository content integrity
  help       - Show this help message

Examples:
  internal:repo status
  internal:repo update
  internal:repo list
  internal:repo list templates
  internal:repo list scripts
  internal:repo validate`
}
