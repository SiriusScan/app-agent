package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/output"
	"github.com/SiriusScan/app-agent/internal/template/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewScanCommand creates the scan command for CLI usage.
func NewScanCommand() *cobra.Command {
	var (
		scripts  []string
		listOnly bool
	)

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Perform system scanning and inventory collection",
		Long: `Execute system scanning to gather installed packages and run custom scripts.

By default, scans installed packages for the current operating system.
Use --scripts to execute custom detection scripts.

Examples:
  sirius-agent scan                                    # Scan installed packages
  sirius-agent scan --scripts=check-suid.sh           # Run custom script
  sirius-agent scan --scripts=script1.sh,script2.sh   # Run multiple scripts
  sirius-agent scan --list-templates                   # List available templates (alias)
  sirius-agent scan --format json                      # Output as JSON (default)
  sirius-agent scan --format table                     # Output as table
  sirius-agent scan --format text                      # Output as human-readable text`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ValidateFormat(); err != nil {
				return err
			}

			// Handle --list-templates alias
			if listOnly {
				return listTemplatesShortcut()
			}

			// Initialize logger
			logger, err := initLogger(logLevel)
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			defer logger.Sync()

			// Build agent config and info
			agentConfig := &config.AgentConfig{
				AgentID:         getHostname(),
				EnableScripting: true,
				PowerShellPath:  "powershell.exe", // Default for Windows
			}

			agentInfo := commands.AgentInfo{
				Config:           agentConfig,
				Logger:           logger,
				ScriptingEnabled: agentConfig.EnableScripting,
				PowerShellPath:   agentConfig.PowerShellPath,
			}

			// Build command string
			commandString := "internal:scan"
			var commandArgs []string
			if len(scripts) > 0 {
				commandArgs = append(commandArgs, fmt.Sprintf("--scripts=%s", strings.Join(scripts, ",")))
			}
			argsString := strings.Join(commandArgs, " ")

			// Execute the scan command
			ctx := context.Background()
			jsonOutput, err := commands.Dispatch(ctx, agentInfo, commandString+" "+argsString)
			if err != nil {
				return fmt.Errorf("scan execution failed: %w", err)
			}

			// Parse JSON output into our output types
			var scanResult output.SystemScanResult
			if err := json.Unmarshal([]byte(jsonOutput), &scanResult); err != nil {
				// If parsing fails, fall back to raw output for JSON format
				if format == "json" {
					fmt.Println(jsonOutput)
					return nil
				}
				return fmt.Errorf("failed to parse scan results: %w", err)
			}

			// Format and output using the selected formatter
			formatter := GetFormatter()
			formatted, err := formatter.FormatSystemScan(&scanResult)
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			return WriteOutput(formatted)
		},
	}

	// Flags
	scanCmd.Flags().StringSliceVar(&scripts, "scripts", []string{}, "Comma-separated list of custom scripts to execute")
	scanCmd.Flags().BoolVar(&listOnly, "list-templates", false, "List available templates (shortcut for 'template list')")

	return scanCmd
}

// listTemplatesShortcut provides a shortcut to template list command
func listTemplatesShortcut() error {
	fmt.Println("💡 Tip: Use 'sirius-agent template list' for more options")
	fmt.Println()

	// Use template manager to discover from all sources
	ctx := context.Background()
	logger := zap.NewNop()
	manager, err := storage.NewManager(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize template manager: %w", err)
	}

	templates, err := manager.DiscoverTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found")
		return nil
	}

	// Use the formatter for output
	formatter := GetFormatter()
	formatted, err := formatter.FormatTemplateList(templates)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return WriteOutput(formatted)
}

// initLogger creates a zap logger with the specified level
func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	config := zap.Config{
		Level:            zapLevel,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// getHostname returns the system hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
