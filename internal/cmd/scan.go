package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/template/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewScanCommand creates the scan command for CLI usage.
func NewScanCommand() *cobra.Command {
	var (
		scripts      []string
		listOnly     bool
		outputFormat string
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
  sirius-agent scan --format text                      # Output as human-readable text`,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			output, err := commands.Dispatch(ctx, agentInfo, commandString+" "+argsString)
			if err != nil {
				return fmt.Errorf("scan execution failed: %w", err)
			}

			// Parse and display output
			if outputFormat == "text" {
				return displayScanResultText(output)
			}

			// Default: JSON output
			fmt.Println(output)
			return nil
		},
	}

	// Flags
	scanCmd.Flags().StringSliceVar(&scripts, "scripts", []string{}, "Comma-separated list of custom scripts to execute")
	scanCmd.Flags().BoolVar(&listOnly, "list-templates", false, "List available templates (shortcut for 'template list')")
	scanCmd.Flags().StringVar(&outputFormat, "format", "json", "Output format (json, text)")

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

	// Output template list
	fmt.Printf("📋 Found %d template(s):\n\n", len(templates))
	for i, t := range templates {
		fmt.Printf("%d. %s (%s)\n", i+1, t.Info.Name, t.ID)
		fmt.Printf("   Severity: %s\n", t.Info.Severity)
		fmt.Printf("   Steps: %d\n", len(t.Detection.Steps))
		if t.FilePath != "" {
			fmt.Printf("   File: %s\n", t.FilePath)
		}
		if i < len(templates)-1 {
			fmt.Println()
		}
	}

	return nil
}

// displayScanResultText formats scan results as human-readable text
func displayScanResultText(jsonOutput string) error {
	var result struct {
		OSInfo struct {
			OS        string `json:"os"`
			Version   string `json:"version"`
			Hostname  string `json:"hostname"`
			PrimaryIP string `json:"primary_ip"`
		} `json:"os_info"`
		Packages      []interface{}          `json:"packages"`
		CustomResults map[string]interface{} `json:"custom_results"`
		ScanErrors    []string               `json:"scan_errors"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		return fmt.Errorf("failed to parse scan results: %w", err)
	}

	fmt.Println("📊 System Scan Results")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// OS Information
	fmt.Println("🖥️  System Information:")
	fmt.Printf("   OS: %s\n", result.OSInfo.OS)
	fmt.Printf("   Version: %s\n", result.OSInfo.Version)
	fmt.Printf("   Hostname: %s\n", result.OSInfo.Hostname)
	fmt.Printf("   Primary IP: %s\n", result.OSInfo.PrimaryIP)
	fmt.Println()

	// Packages
	if len(result.Packages) > 0 {
		fmt.Printf("📦 Installed Packages: %d\n", len(result.Packages))
		fmt.Println()
	}

	// Custom Scripts
	if len(result.CustomResults) > 0 {
		fmt.Printf("🔧 Custom Script Results: %d\n", len(result.CustomResults))
		for scriptName, scriptResult := range result.CustomResults {
			fmt.Printf("   • %s\n", scriptName)
			if scriptResultMap, ok := scriptResult.(map[string]interface{}); ok {
				if exitCode, ok := scriptResultMap["exit_code"].(float64); ok {
					if exitCode == 0 {
						fmt.Println("     ✅ Success")
					} else {
						fmt.Printf("     ❌ Exit code: %.0f\n", exitCode)
					}
				}
			}
		}
		fmt.Println()
	}

	// Errors
	if len(result.ScanErrors) > 0 {
		fmt.Println("⚠️  Scan Errors:")
		for _, err := range result.ScanErrors {
			fmt.Printf("   • %s\n", err)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("✅ Scan complete")

	return nil
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
