package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	siriusbootstrap "github.com/SiriusScan/app-agent/internal/family/sirius/bootstrap"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/executor"
	"github.com/SiriusScan/app-agent/internal/template/parser"
)

func main() {
	siriusbootstrap.LoadCompatibilityRuntime()
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "run":
		runTemplate()
	case "run-all":
		runAllTemplates()
	case "validate":
		validateTemplate()
	case "list-modules":
		listModules()
	case "module-info":
		moduleInfo()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Template System CLI - Test and run vulnerability detection templates")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  template-cli run <template-file>        Run a single template")
	fmt.Println("  template-cli run-all <directory>        Run all templates in directory")
	fmt.Println("  template-cli validate <template-file>   Validate a template")
	fmt.Println("  template-cli list-modules               List available detection modules")
	fmt.Println("  template-cli module-info <type>         Show module information")
	fmt.Println("  template-cli help                       Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  template-cli run testing/test-templates/01-file-hash.yaml")
	fmt.Println("  template-cli run-all testing/test-templates/")
	fmt.Println("  template-cli validate my-template.yaml")
	fmt.Println("  template-cli list-modules")
	fmt.Println("  template-cli module-info file_hash")
}

func runTemplate() {
	if len(os.Args) < 3 {
		fmt.Println("Error: template file required")
		fmt.Println("Usage: template-cli run <template-file>")
		os.Exit(1)
	}

	templatePath := os.Args[2]

	fmt.Printf("🔍 Running template: %s\n", templatePath)
	fmt.Println("=" + "=======================================")

	// Parse template
	template, err := parser.ParseTemplate(templatePath)
	if err != nil {
		fmt.Printf("\n❌ Failed to parse template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Parsed template: %s\n", template.ID)
	fmt.Printf("  Name: %s\n", template.Info.Name)
	fmt.Printf("  Severity: %s\n", template.Info.Severity)
	fmt.Printf("  Steps: %d\n", len(template.Detection.Steps))

	// Validate template
	if err := parser.ValidateTemplate(template); err != nil {
		fmt.Printf("\n❌ Template validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n✓ Template is valid")

	// Execute template
	fmt.Println("\n🚀 Executing template...")
	exec := executor.New()
	ctx := context.Background()

	result, err := exec.ExecuteTemplate(ctx, template)
	if err != nil {
		fmt.Printf("\n❌ Execution failed: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Println("\n📊 Results:")
	fmt.Println("-" + "----------")

	if result.Matched {
		fmt.Println("✅ MATCHED - Vulnerability detected!")
	} else {
		fmt.Println("❌ NOT MATCHED - System is safe")
	}

	fmt.Printf("\nConfidence: %.2f\n", result.Confidence)
	fmt.Printf("Executed Steps: %d\n", len(result.Steps))
	fmt.Printf("Errors: %d\n", len(result.Errors))
	fmt.Printf("Host: %s\n", result.Host)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))

	// Show step details
	if len(result.Steps) > 0 {
		fmt.Println("\n📋 Step Details:")
		for i, step := range result.Steps {
			fmt.Printf("  Step %d (%s):\n", i+1, step.Type)
			fmt.Printf("    Matched: %v\n", step.Matched)
			if step.Error != "" {
				fmt.Printf("    Error: %s\n", step.Error)
			}
			fmt.Printf("    Duration: %v\n", step.Duration)
		}
	}

	// Show errors if any
	if len(result.Errors) > 0 {
		fmt.Println("\n⚠️  Errors:")
		for i, errMsg := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, errMsg)
		}
	}

	// Output JSON
	fmt.Println("\n📄 JSON Output:")
	fmt.Println("-" + "--------------")
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}

func runAllTemplates() {
	if len(os.Args) < 3 {
		fmt.Println("Error: directory required")
		fmt.Println("Usage: template-cli run-all <directory>")
		os.Exit(1)
	}

	directory := os.Args[2]

	fmt.Printf("🔍 Discovering templates in: %s\n", directory)
	fmt.Println("=" + "========================================")

	// Discover templates
	templates, errors := parser.DiscoverTemplates(directory)

	fmt.Printf("\n📊 Discovery Results:\n")
	fmt.Printf("  Valid templates: %d\n", len(templates))
	fmt.Printf("  Errors: %d\n", len(errors))

	if len(errors) > 0 {
		fmt.Println("\n⚠️  Discovery Errors:")
		for i, err := range errors {
			fmt.Printf("  %d. %v\n", i+1, err)
		}
	}

	if len(templates) == 0 {
		fmt.Println("\n❌ No valid templates found")
		os.Exit(0)
	}

	// Execute each template
	exec := executor.New()
	ctx := context.Background()

	fmt.Println("\n🚀 Executing templates...")
	fmt.Println("-" + "------------------------")

	matchedCount := 0
	for i, template := range templates {
		fmt.Printf("\n[%d/%d] %s (%s)\n", i+1, len(templates), template.Info.Name, template.ID)

		result, err := exec.ExecuteTemplate(ctx, template)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		if result.Matched {
			fmt.Printf("  ✅ MATCHED (Confidence: %.2f)\n", result.Confidence)
			matchedCount++
		} else {
			fmt.Printf("  ❌ NOT MATCHED\n")
		}
	}

	// Summary
	fmt.Println("\n" + "========================================")
	fmt.Printf("Summary: %d/%d templates matched\n", matchedCount, len(templates))
	if matchedCount > 0 {
		fmt.Println("⚠️  Vulnerabilities detected!")
	} else {
		fmt.Println("✅ System is clean")
	}
}

func validateTemplate() {
	if len(os.Args) < 3 {
		fmt.Println("Error: template file required")
		fmt.Println("Usage: template-cli validate <template-file>")
		os.Exit(1)
	}

	templatePath := os.Args[2]

	fmt.Printf("🔍 Validating template: %s\n", templatePath)
	fmt.Println("=" + "================================")

	// Parse template
	template, err := parser.ParseTemplate(templatePath)
	if err != nil {
		fmt.Printf("\n❌ Parse error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Parsed successfully\n")
	fmt.Printf("  ID: %s\n", template.ID)
	fmt.Printf("  Name: %s\n", template.Info.Name)
	fmt.Printf("  Severity: %s\n", template.Info.Severity)

	// Validate template
	if err := parser.ValidateTemplate(template); err != nil {
		fmt.Printf("\n❌ Validation failed:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Template is valid!")
	fmt.Printf("  Steps: %d\n", len(template.Detection.Steps))
	fmt.Printf("  Logic: %s\n", template.Detection.Logic)
}

func listModules() {
	fmt.Println("📦 Available Detection Modules")
	fmt.Println("=" + "==============================")

	modules := registry.List()

	if len(modules) == 0 {
		fmt.Println("\nNo modules registered")
		return
	}

	fmt.Printf("\nFound %d module(s):\n\n", len(modules))

	for i, moduleType := range modules {
		descriptor := registry.GetDescriptor(moduleType)
		if descriptor == nil {
			continue
		}

		fmt.Printf("%d. %s\n", i+1, descriptor.Type)
		fmt.Printf("   Name: %s\n", descriptor.Name)
		fmt.Printf("   Description: %s\n", descriptor.Description)
		fmt.Printf("   Supported OS: %v\n", descriptor.SupportedOS)
		if i < len(modules)-1 {
			fmt.Println()
		}
	}

	fmt.Println("\nUse 'template-cli module-info <type>' for detailed information")
}

func moduleInfo() {
	if len(os.Args) < 3 {
		fmt.Println("Error: module type required")
		fmt.Println("Usage: template-cli module-info <type>")
		os.Exit(1)
	}

	moduleType := os.Args[2]

	descriptor := registry.GetDescriptor(moduleType)
	if descriptor == nil {
		fmt.Printf("❌ Module '%s' not found\n", moduleType)
		fmt.Println("\nUse 'template-cli list-modules' to see available modules")
		os.Exit(1)
	}

	fmt.Printf("📦 Module Information: %s\n", moduleType)
	fmt.Println("=" + "====================================")

	fmt.Printf("\nType: %s\n", descriptor.Type)
	fmt.Printf("Name: %s\n", descriptor.Name)
	fmt.Printf("Version: %s\n", descriptor.Version)
	fmt.Printf("Author: %s\n", descriptor.Author)
	fmt.Printf("\nDescription:\n  %s\n", descriptor.Description)

	fmt.Printf("\nSupported OS:\n")
	for _, os := range descriptor.SupportedOS {
		fmt.Printf("  - %s\n", os)
	}

	if len(descriptor.ConfigDocs) > 0 {
		fmt.Printf("\nConfiguration Fields:\n")
		for field, desc := range descriptor.ConfigDocs {
			fmt.Printf("  - %s: %s\n", field, desc)
		}
	}
}
