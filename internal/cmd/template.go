package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

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
  sirius-agent template run my-template.yaml --format text`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			result, err := exec.ExecuteTemplate(ctx, template)
			if err != nil {
				return fmt.Errorf("template execution failed: %w", err)
			}

			// Output result
			return outputResult(result)
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
  4. Output results (one per line in JSONL format, or summary in text format)

Examples:
  sirius-agent template run-all                          # Use template manager
  sirius-agent template run-all ./templates/              # Specific directory
  sirius-agent template run-all /etc/sirius/templates/ --format text
  sirius-agent template run-all ./templates/ --workers 10
  sirius-agent template run-all --workers 1 --timeout 300`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if len(args) > 0 {
				// Specific directory provided
				directory := args[0]
				templates, errors = parser.DiscoverTemplatesWithContext(ctx, directory)

				if len(errors) > 0 && format == "text" {
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

				if format == "text" {
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
			if format == "text" {
				fmt.Fprintf(os.Stderr, "🚀 Executing %d template(s) with %d worker(s)...\n", len(templates), workers)
			}

			results, execErrors := executor.ExecuteTemplatesParallelWithConfig(templates, config)

			if len(execErrors) > 0 && format == "text" {
				fmt.Fprintf(os.Stderr, "⚠️  Execution errors:\n")
				for _, err := range execErrors {
					fmt.Fprintf(os.Stderr, "  - %v\n", err)
				}
			}

			// Count matches
			matchCount := 0
			for _, result := range results {
				if result != nil && result.Matched {
					matchCount++
				}
			}

			// Output results
			if format == "jsonl" {
				return outputResultsJSONL(results)
			} else if format == "text" {
				return outputResultsText(results, matchCount)
			}

			// Default: JSON array
			return outputResultsJSON(results)
		},
	}

	cmd.Flags().IntVarP(&workers, "workers", "w", 0, "Number of parallel workers (0 = CPU count, max 50)")
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
  sirius-agent template validate /path/to/template.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templatePath := args[0]

			// Parse template
			template, err := parser.ParseTemplate(templatePath)
			if err != nil {
				return fmt.Errorf("parse error: %w", err)
			}

			// Validate template
			if err := parser.ValidateTemplate(template); err != nil {
				return fmt.Errorf("validation error: %w", err)
			}

			if format == "text" {
				fmt.Printf("✅ Template is valid: %s\n", template.ID)
				fmt.Printf("   Name: %s\n", template.Info.Name)
				fmt.Printf("   Severity: %s\n", template.Info.Severity)
				fmt.Printf("   Steps: %d\n", len(template.Detection.Steps))
			} else {
				// JSON output
				output := map[string]interface{}{
					"valid":    true,
					"id":       template.ID,
					"name":     template.Info.Name,
					"severity": template.Info.Severity,
					"steps":    len(template.Detection.Steps),
				}
				return outputJSON(output)
			}

			return nil
		},
	}
}

func newTemplateListCommand() *cobra.Command {
	var directory string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Long: `Discover and list all available templates in the specified directory.

By default, lists templates from the current directory.
Use --directory to specify a different location.

Examples:
  sirius-agent template list
  sirius-agent template list --directory ./templates/
  sirius-agent template list --format text`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Discover templates
			templates, errors := parser.DiscoverTemplates(directory)

			if len(errors) > 0 && format == "text" {
				fmt.Fprintf(os.Stderr, "⚠️  Discovery errors:\n")
				for _, err := range errors {
					fmt.Fprintf(os.Stderr, "  - %v\n", err)
				}
			}

			if len(templates) == 0 {
				if format == "text" {
					fmt.Printf("No templates found in %s\n", directory)
				}
				return nil
			}

			// Output template list
			if format == "text" {
				fmt.Printf("📋 Found %d template(s) in %s:\n\n", len(templates), directory)
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
			} else {
				// JSON output
				list := make([]map[string]interface{}, len(templates))
				for i, t := range templates {
					list[i] = map[string]interface{}{
						"id":        t.ID,
						"name":      t.Info.Name,
						"severity":  t.Info.Severity,
						"author":    t.Info.Author,
						"steps":     len(t.Detection.Steps),
						"file_path": t.FilePath,
					}
				}
				return outputJSON(list)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&directory, "directory", "d", ".", "Directory to search for templates")

	return cmd
}

// Output helper functions

func outputResult(result *types.Result) error {
	writer, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer func() {
		if writer != os.Stdout {
			writer.Close()
		}
	}()

	if format == "text" {
		return outputResultText(writer, result)
	}

	// Default: JSON
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputResultText(writer *os.File, result *types.Result) error {
	if result.Matched {
		fmt.Fprintf(writer, "✅ MATCHED - Vulnerability detected!\n")
	} else {
		fmt.Fprintf(writer, "❌ NOT MATCHED - System is safe\n")
	}

	fmt.Fprintf(writer, "\nTemplate: %s (%s)\n", result.TemplateName, result.TemplateID)
	fmt.Fprintf(writer, "Severity: %s\n", result.Severity)
	fmt.Fprintf(writer, "Confidence: %.2f\n", result.Confidence)
	fmt.Fprintf(writer, "Steps: %d\n", len(result.Steps))
	fmt.Fprintf(writer, "Host: %s\n", result.Host)
	fmt.Fprintf(writer, "Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))

	if len(result.Errors) > 0 {
		fmt.Fprintf(writer, "\n⚠️  Errors:\n")
		for _, err := range result.Errors {
			fmt.Fprintf(writer, "  - %s\n", err)
		}
	}

	return nil
}

func outputResultsJSON(results []*types.Result) error {
	writer, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer func() {
		if writer != os.Stdout {
			writer.Close()
		}
	}()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func outputResultsJSONL(results []*types.Result) error {
	writer, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer func() {
		if writer != os.Stdout {
			writer.Close()
		}
	}()

	encoder := json.NewEncoder(writer)
	for _, result := range results {
		if err := encoder.Encode(result); err != nil {
			return err
		}
	}
	return nil
}

func outputResultsText(results []*types.Result, matchCount int) error {
	writer, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer func() {
		if writer != os.Stdout {
			writer.Close()
		}
	}()

	fmt.Fprintf(writer, "📊 Execution Summary\n")
	fmt.Fprintf(writer, "="+strings.Repeat("=", 50)+"\n\n")

	for i, result := range results {
		if result.Matched {
			fmt.Fprintf(writer, "[%d] ✅ %s (Confidence: %.2f)\n", i+1, result.TemplateName, result.Confidence)
		} else {
			fmt.Fprintf(writer, "[%d] ❌ %s\n", i+1, result.TemplateName)
		}
	}

	fmt.Fprintf(writer, "\n"+strings.Repeat("=", 52)+"\n")
	fmt.Fprintf(writer, "Summary: %d/%d templates matched\n", matchCount, len(results))

	if matchCount > 0 {
		fmt.Fprintf(writer, "⚠️  Vulnerabilities detected!\n")
	} else {
		fmt.Fprintf(writer, "✅ System is clean\n")
	}

	return nil
}

func outputJSON(data interface{}) error {
	writer, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer func() {
		if writer != os.Stdout {
			writer.Close()
		}
	}()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
