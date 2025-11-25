package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/output"
	"github.com/SiriusScan/app-agent/internal/template/executor"
	"github.com/SiriusScan/app-agent/internal/template/parser"
	"github.com/SiriusScan/app-agent/internal/template/storage"
	"github.com/SiriusScan/app-agent/internal/template/types"
	"github.com/spf13/cobra"
)

// NewTemplateCommand creates the template command group.
func NewTemplateCommand() *cobra.Command {
	templateCmd := &cobra.Command{
		Use:   "template",
		Short: "Template operations",
		Long:  "Commands for running, validating, and managing vulnerability detection templates",
	}

	templateCmd.AddCommand(newTemplateRunCommand())
	templateCmd.AddCommand(newTemplateRunAllCommand())
	templateCmd.AddCommand(newTemplateValidateCommand())
	templateCmd.AddCommand(newTemplateListCommand())

	return templateCmd
}

func newTemplateRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <template-file>",
		Short: "Run a single template",
		Long: `Execute a vulnerability detection template and output the results.

The template can be specified as:
  - A file path (e.g., ./templates/my-template.yaml)
  - An absolute path (e.g., /path/to/template.yaml)

Examples:
  sirius-agent template run ./templates/cve-2024-001.yaml
  sirius-agent template run /etc/sirius/templates/ssh-check.yaml
  sirius-agent template run my-template.yaml --format table`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ValidateFormat(); err != nil {
				return err
			}

			templatePath := args[0]

			// Parse template
			template, err := parser.ParseTemplate(templatePath)
			if err != nil {
				return fmt.Errorf("failed to parse template: %w", err)
			}

			// Validate template
			if err := parser.ValidateTemplate(template); err != nil {
				return fmt.Errorf("template validation failed: %w", err)
			}

			// Execute template
			exec := executor.New()
			ctx := context.Background()
			startTime := time.Now()
			result, err := exec.ExecuteTemplate(ctx, template)
			if err != nil {
				return fmt.Errorf("template execution failed: %w", err)
			}
			executionTime := time.Since(startTime)

			// Create summary for single template
			results := []*types.Result{result}
			summary := output.NewScanSummary(results, executionTime, 1)

			// Format and output result
			formatter := GetFormatter()
			formatted, err := formatter.FormatScanResults(results, summary)
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			return WriteOutput(formatted)
		},
	}
}

func newTemplateRunAllCommand() *cobra.Command {
	var workers int
	var timeout int

	cmd := &cobra.Command{
		Use:   "run-all [directory]",
		Short: "Run all templates",
		Long: `Discover and execute all templates.

Without a directory argument, uses the template manager to discover templates from:
  1. Custom templates (highest priority)
  2. Server-synced templates
  3. Built-in templates (embedded in binary)

With a directory argument, scans only that specific directory.

The command will:
  1. Recursively find all .yaml/.yml files
  2. Parse and validate each template
  3. Execute all valid templates in parallel using a worker pool
  4. Output results in the specified format

Examples:
  sirius-agent template run-all                          # Use template manager
  sirius-agent template run-all ./templates/             # Specific directory
  sirius-agent template run-all --format table           # Table output for terminal
  sirius-agent template run-all --format csv             # CSV for spreadsheets
  sirius-agent template run-all --format markdown        # Markdown for reports
  sirius-agent template run-all ./templates/ --workers 10
  sirius-agent template run-all --workers 1 --timeout 300`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ValidateFormat(); err != nil {
				return err
			}

			// Validate worker count
			if workers > 0 {
				if err := executor.ValidateWorkerCount(workers); err != nil {
					return err
				}
			}

			// Discover templates
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle Ctrl+C for graceful shutdown
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
				<-sigChan
				fmt.Fprintf(os.Stderr, "\n⚠️  Interrupt received, stopping execution...\n")
				cancel()
			}()

			var templates []*types.Template
			var errors []error

			formatter := GetFormatter()
			isVerboseFormat := format == "text" || format == "table"

			if len(args) > 0 {
				// Specific directory provided
				directory := args[0]
				templates, errors = parser.DiscoverTemplatesWithContext(ctx, directory)

				if len(errors) > 0 && isVerboseFormat {
					fmt.Fprintf(os.Stderr, "⚠️  Discovery errors:\n")
					for _, err := range errors {
						fmt.Fprintf(os.Stderr, "  - %v\n", err)
					}
				}

				if len(templates) == 0 {
					return fmt.Errorf("no valid templates found in %s", directory)
				}
			} else {
				// Use template manager
				logger := zap.NewNop() // Use production logger if available
				manager, err := storage.NewManager(logger)
				if err != nil {
					return fmt.Errorf("failed to initialize template manager: %w", err)
				}

				templates, err = manager.DiscoverTemplates(ctx)
				if err != nil {
					return fmt.Errorf("failed to discover templates: %w", err)
				}

				if len(templates) == 0 {
					return fmt.Errorf("no templates available. Install templates in %s or specify a directory", manager.GetStoragePath())
				}

				if isVerboseFormat {
					fmt.Fprintf(os.Stderr, "📁 Using template manager (base: %s)\n", manager.GetStoragePath())
				}
			}

			// Configure worker pool
			config := executor.DefaultWorkerPoolConfig()
			config.Context = ctx
			config.Workers = workers
			if timeout > 0 {
				config.PerTemplateTimeout = time.Duration(timeout) * time.Second
			}

			// Execute all templates in parallel
			if isVerboseFormat {
				fmt.Fprintf(os.Stderr, "🚀 Executing %d template(s) with %d worker(s)...\n", len(templates), config.Workers)
			}

			startTime := time.Now()
			results, execErrors := executor.ExecuteTemplatesParallelWithConfig(templates, config)
			executionTime := time.Since(startTime)

			if len(execErrors) > 0 && isVerboseFormat {
				fmt.Fprintf(os.Stderr, "⚠️  Execution errors:\n")
				for _, err := range execErrors {
					fmt.Fprintf(os.Stderr, "  - %v\n", err)
				}
			}

			// Create summary
			summary := output.NewScanSummary(results, executionTime, config.Workers)

			// Format and output results
			formatted, err := formatter.FormatScanResults(results, summary)
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			return WriteOutput(formatted)
		},
	}

	cmd.Flags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "Number of parallel workers (0 = CPU count, max 50)")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 300, "Per-template timeout in seconds (default 300)")

	return cmd
}

func newTemplateValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <template-file>",
		Short: "Validate a template",
		Long: `Check if a template file has valid syntax and structure.

This command parses the template and validates:
  - YAML syntax
  - Required fields (id, info, detection)
  - Valid severity levels
  - Valid platform types
  - Detection step configuration

Examples:
  sirius-agent template validate ./my-template.yaml
  sirius-agent template validate /path/to/template.yaml --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ValidateFormat(); err != nil {
				return err
			}

			templatePath := args[0]

			// Parse template
			template, err := parser.ParseTemplate(templatePath)
			var validationResult *output.ValidationResult

			if err != nil {
				validationResult = &output.ValidationResult{
					Valid:  false,
					Errors: []string{fmt.Sprintf("parse error: %v", err)},
				}
			} else {
				// Validate template
				validationErr := parser.ValidateTemplate(template)
				validationResult = &output.ValidationResult{
					Valid:        validationErr == nil,
					TemplateID:   template.ID,
					TemplateName: template.Info.Name,
					Severity:     string(template.Info.Severity),
					StepCount:    len(template.Detection.Steps),
				}
				if validationErr != nil {
					validationResult.Errors = []string{validationErr.Error()}
				}
			}

			// Format and output
			formatter := GetFormatter()
			formatted, err := formatter.FormatValidation(validationResult)
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			return WriteOutput(formatted)
		},
	}
}

func newTemplateListCommand() *cobra.Command {
	var directory string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Long: `Discover and list all available templates.

By default, uses the template manager to discover templates from all sources:
  1. Custom templates (highest priority)
  2. Server-synced templates
  3. Built-in templates (embedded in binary)

Use --directory to search a specific directory instead.

Examples:
  sirius-agent template list                          # Use template manager
  sirius-agent template list --directory ./templates/ # Specific directory
  sirius-agent template list --format table           # Table output
  sirius-agent template list --format csv             # CSV for spreadsheets
  sirius-agent template list --format json            # JSON output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ValidateFormat(); err != nil {
				return err
			}

			ctx := context.Background()
			var templates []*types.Template
			var errors []error

			formatter := GetFormatter()
			isVerboseFormat := format == "text" || format == "table"

			if directory != "" {
				// Specific directory provided
				templates, errors = parser.DiscoverTemplates(directory)
			} else {
				// Use template manager (default)
				logger := zap.NewNop()
				manager, err := storage.NewManager(logger)
				if err != nil {
					return fmt.Errorf("failed to initialize template manager: %w", err)
				}

				templates, err = manager.DiscoverTemplates(ctx)
				if err != nil {
					return fmt.Errorf("failed to discover templates: %w", err)
				}

				if isVerboseFormat {
					fmt.Fprintf(os.Stderr, "📁 Using template manager (base: %s)\n\n", manager.GetStoragePath())
				}
			}

			if len(errors) > 0 && isVerboseFormat {
				fmt.Fprintf(os.Stderr, "⚠️  Discovery errors:\n")
				for _, err := range errors {
					fmt.Fprintf(os.Stderr, "  - %v\n", err)
				}
				fmt.Fprintln(os.Stderr)
			}

			if len(templates) == 0 {
				if isVerboseFormat {
					fmt.Printf("No templates found\n")
				}
				return nil
			}

			// Format and output template list
			formatted, err := formatter.FormatTemplateList(templates)
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			return WriteOutput(formatted)
		},
	}

	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory to search for templates (defaults to template manager)")

	return cmd
}
