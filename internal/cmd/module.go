package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/spf13/cobra"
)

// NewModuleCommand creates the module command group.
func NewModuleCommand() *cobra.Command {
	moduleCmd := &cobra.Command{
		Use:   "module",
		Short: "Module introspection",
		Long:  "Commands for listing and inspecting available detection modules",
	}

	moduleCmd.AddCommand(newModuleListCommand())
	moduleCmd.AddCommand(newModuleInfoCommand())

	return moduleCmd
}

func newModuleListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered modules",
		Long: `Display all available detection modules registered in the agent.

Each module represents a type of detection step that can be used in templates.

Examples:
  sirius-agent module list
  sirius-agent module list --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleTypes := registry.List()

			if len(moduleTypes) == 0 {
				if format == "text" || format == "table" {
					fmt.Println("No modules registered")
				}
				return nil
			}

			// Sort for consistent output
			sort.Strings(moduleTypes)

			if format == "text" || format == "table" {
				return outputModuleListText(moduleTypes)
			}

			// JSON output
			return outputModuleListJSON(moduleTypes)
		},
	}
}

func newModuleInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <module-type>",
		Short: "Show detailed module information",
		Long: `Display detailed information about a specific detection module.

Shows:
  - Module type and name
  - Description
  - Version and author
  - Supported operating systems
  - Configuration fields and their descriptions

Examples:
  sirius-agent module info file_hash
  sirius-agent module info file_content
  sirius-agent module info file_hash --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleType := args[0]

			descriptor := registry.GetDescriptor(moduleType)
			if descriptor == nil {
				return fmt.Errorf("module '%s' not found", moduleType)
			}

			if format == "text" || format == "table" {
				return outputModuleInfoText(descriptor)
			}

			// JSON output
			return outputModuleJSON(descriptor)
		},
	}
}

// Output helper functions

func outputModuleListText(moduleTypes []string) error {
	fmt.Printf("📦 Registered Detection Modules\n")
	fmt.Print("=" + strings.Repeat("=", 50) + "\n\n")

	for i, moduleType := range moduleTypes {
		descriptor := registry.GetDescriptor(moduleType)
		if descriptor == nil {
			continue
		}

		fmt.Printf("%d. %s\n", i+1, descriptor.Type)
		fmt.Printf("   Name: %s\n", descriptor.Name)
		fmt.Printf("   Description: %s\n", descriptor.Description)
		fmt.Printf("   Supported OS: %v\n", descriptor.SupportedOS)

		if i < len(moduleTypes)-1 {
			fmt.Println()
		}
	}

	fmt.Print("\n" + strings.Repeat("=", 52) + "\n")
	fmt.Printf("Total: %d module(s)\n", len(moduleTypes))
	fmt.Printf("\nUse 'sirius-agent module info <type>' for detailed information\n")

	return nil
}

func outputModuleListJSON(moduleTypes []string) error {
	modules := make([]map[string]interface{}, 0, len(moduleTypes))

	for _, moduleType := range moduleTypes {
		descriptor := registry.GetDescriptor(moduleType)
		if descriptor == nil {
			continue
		}

		modules = append(modules, map[string]interface{}{
			"type":         descriptor.Type,
			"name":         descriptor.Name,
			"description":  descriptor.Description,
			"version":      descriptor.Version,
			"author":       descriptor.Author,
			"supported_os": descriptor.SupportedOS,
		})
	}

	return outputModuleJSON(modules)
}

func outputModuleInfoText(descriptor *modules.Descriptor) error {
	fmt.Printf("📦 Module Information: %s\n", descriptor.Type)
	fmt.Print("=" + strings.Repeat("=", 50) + "\n\n")

	fmt.Printf("Type: %s\n", descriptor.Type)
	fmt.Printf("Name: %s\n", descriptor.Name)
	fmt.Printf("Version: %s\n", descriptor.Version)
	fmt.Printf("Author: %s\n", descriptor.Author)

	fmt.Printf("\nDescription:\n")
	fmt.Printf("  %s\n", descriptor.Description)

	fmt.Printf("\nSupported Operating Systems:\n")
	for _, os := range descriptor.SupportedOS {
		fmt.Printf("  - %s\n", os)
	}

	if len(descriptor.ConfigDocs) > 0 {
		fmt.Printf("\nConfiguration Fields:\n")
		
		// Sort config fields for consistent output
		configKeys := make([]string, 0, len(descriptor.ConfigDocs))
		for key := range descriptor.ConfigDocs {
			configKeys = append(configKeys, key)
		}
		sort.Strings(configKeys)

		for _, key := range configKeys {
			fmt.Printf("  - %s: %s\n", key, descriptor.ConfigDocs[key])
		}
	}

	return nil
}

// outputModuleJSON outputs data as JSON
func outputModuleJSON(data interface{}) error {
	writer, err := getOutputWriter()
	if err != nil {
		return err
	}
	defer func() {
		if writer != nil {
			writer.Close()
		}
	}()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
