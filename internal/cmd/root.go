package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/SiriusScan/app-agent/internal/output"
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

Output Formats:
  json      Pretty-printed JSON (default, ideal for machine processing)
  jsonl     JSON Lines (one object per line, for streaming)
  table     ASCII table (ideal for terminal viewing)
  text      Human-readable text with emojis
  csv       Comma-separated values (for spreadsheets)
  quiet     Minimal output (for scripting and CI/CD)
  markdown  Markdown tables (for documentation and reports)

Examples:
  sirius-agent                                 # Start in server mode
  sirius-agent template run mytemplate.yaml    # Run a template
  sirius-agent template run-all --format table # Run all templates with table output
  sirius-agent module list                     # List available modules
  sirius-agent version                         # Show version`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: start server mode
			return NewServerCommand().RunE(cmd, args)
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "json",
		fmt.Sprintf("Output format (%s)", strings.Join(output.AvailableStrings(), ", ")))

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
	rootCmd.AddCommand(NewScanCommand())
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

// GetFormat returns the current output format.
func GetFormat() output.Format {
	return output.Format(format)
}

// GetFormatter returns the formatter for the current format.
// Falls back to JSON formatter if the format is invalid.
func GetFormatter() output.Formatter {
	f, err := output.GetByString(format)
	if err != nil {
		// Fallback to JSON
		return output.MustGet(output.FormatJSON)
	}
	return f
}

// ValidateFormat validates the format flag value.
func ValidateFormat() error {
	if !output.IsValidFormat(format) {
		return fmt.Errorf("invalid output format: %s (valid: %s)",
			format, strings.Join(output.AvailableStrings(), ", "))
	}
	return nil
}

// WriteOutput writes formatted output to the appropriate writer.
func WriteOutput(content string) error {
	writer, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer func() {
		if writer != os.Stdout {
			writer.Close()
		}
	}()

	_, err = writer.WriteString(content)
	return err
}
