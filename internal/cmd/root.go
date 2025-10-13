package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	logLevel   string
	outputFile string
	format     string
)

// NewRootCommand creates the root command for the sirius-agent CLI.
func NewRootCommand(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sirius-agent",
		Short: "Sirius vulnerability detection agent",
		Long: `Sirius Agent - Template-based vulnerability detection system

The agent can run in two modes:
1. Server mode (default): Connects to Sirius server and waits for commands
2. CLI mode: Execute templates locally via subcommands

Examples:
  sirius-agent                           # Start in server mode
  sirius-agent template run mytemplate.yaml   # Run a template
  sirius-agent module list                    # List available modules
  sirius-agent version                        # Show version`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: start server mode
			return fmt.Errorf("server mode not implemented in MVP - use template/module commands")
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "json", "Output format (json, jsonl, text)")

	// Add version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("sirius-agent version %s\n", version)
		},
	}
	rootCmd.AddCommand(versionCmd)

	// Add subcommand groups
	rootCmd.AddCommand(NewTemplateCommand())
	rootCmd.AddCommand(NewModuleCommand())
	rootCmd.AddCommand(NewServerCommand()) // Add server mode

	return rootCmd
}

// getOutputWriter returns the appropriate output writer based on the --output flag.
func getOutputWriter() (*os.File, error) {
	if outputFile == "" {
		return os.Stdout, nil
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	return file, nil
}

