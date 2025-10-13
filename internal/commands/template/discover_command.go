package template

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/detect/template"
	"github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"
)

// DiscoverCommand implements the template discovery command
type DiscoverCommand struct {
	discoveryService *template.TemplateDiscoveryService
}

// Ensures DiscoverCommand implements the Command interface at compile time.
var _ commands.Command = (*DiscoverCommand)(nil)

func init() {
	commands.Register("discover-templates", &DiscoverCommand{})
	commands.Register("internal:discover-templates", &DiscoverCommand{})
}

// Execute performs comprehensive template discovery
func (c *DiscoverCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (output string, err error) {
	agentInfo.Logger.Info("Executing template discovery command", zap.String("args", args))

	// Initialize ValKey store
	kvStore, err := store.NewValkeyStore()
	if err != nil {
		return "", fmt.Errorf("failed to initialize ValKey store: %w", err)
	}
	defer kvStore.Close()

	// Create ValKey adapter and template store
	valkeyAdapter := template.NewValKeyAdapter(kvStore)
	valkeyStore := template.NewValKeyTemplateStore(valkeyAdapter)

	// Create template parser
	parser := template.NewTemplateParser(agentInfo.Logger, []string{"/app-agent/templates"})

	// Create discovery service
	c.discoveryService = template.NewTemplateDiscoveryService(agentInfo.Logger, valkeyStore, parser)

	// Parse arguments
	argsList := strings.Fields(args)
	format := "text"

	for _, arg := range argsList {
		switch {
		case arg == "--json":
			format = "json"
		case arg == "--yaml":
			format = "yaml"
		}
	}

	// Perform discovery
	templateDirs := []string{"/app-agent/templates", "/app-agent/custom-templates"}
	result, err := c.discoveryService.DiscoverAllTemplates(ctx, templateDirs)
	if err != nil {
		return "", fmt.Errorf("failed to discover templates: %w", err)
	}

	// Generate output based on format
	switch format {
	case "json":
		return c.generateJSONOutput(result)
	case "yaml":
		return c.generateYAMLOutput(result)
	default:
		return c.generateTextOutput(result)
	}
}

// generateTextOutput generates human-readable text output
func (c *DiscoverCommand) generateTextOutput(result *template.DiscoveryResult) (string, error) {
	var output strings.Builder
	output.WriteString("🔍 Template Discovery Results\n")
	output.WriteString("============================\n\n")

	// Statistics
	output.WriteString(fmt.Sprintf("📊 Statistics:\n"))
	output.WriteString(fmt.Sprintf("  Total Templates: %d\n", result.Statistics.TotalTemplates))
	output.WriteString(fmt.Sprintf("  Custom Templates: %d\n", result.Statistics.CustomTemplates))
	output.WriteString(fmt.Sprintf("  Repository Templates: %d\n", result.Statistics.RepositoryTemplates))
	output.WriteString(fmt.Sprintf("  Local Templates: %d\n", result.Statistics.LocalTemplates))
	output.WriteString(fmt.Sprintf("  Active Templates: %d\n", result.Statistics.ActiveTemplates))
	output.WriteString(fmt.Sprintf("  Last Sync: %s\n", result.Statistics.LastSyncTime.Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("  Discovery Time: %s\n", result.LastDiscovery.Format("2006-01-02 15:04:05")))
	output.WriteString("\n")

	// Sources
	if len(result.Sources) > 0 {
		output.WriteString("📚 Sources:\n")
		for name, source := range result.Sources {
			output.WriteString(fmt.Sprintf("  %s (%s): Priority %d\n", name, source.Type, source.Priority))
		}
		output.WriteString("\n")
	}

	// Templates by source
	sourceGroups := make(map[string][]*template.DiscoveredTemplate)
	for _, template := range result.Templates {
		source := template.Source.Type
		sourceGroups[source] = append(sourceGroups[source], template)
	}

	for source, templates := range sourceGroups {
		output.WriteString(fmt.Sprintf("🔹 %s Templates (%d):\n", strings.Title(source), len(templates)))
		for i, template := range templates {
			output.WriteString(fmt.Sprintf("  %d. %s (%s)\n", i+1, template.Name, template.ID))
			output.WriteString(fmt.Sprintf("      Severity: %s, Type: %s\n", template.Severity, template.Type))
			if template.Description != "" {
				output.WriteString(fmt.Sprintf("      Description: %s\n", template.Description))
			}
		}
		output.WriteString("\n")
	}

	// Errors
	if len(result.Errors) > 0 {
		output.WriteString("⚠️  Errors:\n")
		for i, err := range result.Errors {
			output.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))
		}
		output.WriteString("\n")
	}

	output.WriteString("✅ Template discovery completed successfully!\n")
	return output.String(), nil
}

// generateJSONOutput generates JSON output
func (c *DiscoverCommand) generateJSONOutput(result *template.DiscoveryResult) (string, error) {
	type DiscoveryOutput struct {
		Statistics    template.TemplateStatistics        `json:"statistics"`
		Sources       map[string]template.TemplateSource `json:"sources"`
		Templates     []*template.DiscoveredTemplate     `json:"templates"`
		Errors        []string                           `json:"errors"`
		LastDiscovery string                             `json:"last_discovery"`
	}

	output := DiscoveryOutput{
		Statistics:    result.Statistics,
		Sources:       result.Sources,
		Templates:     result.Templates,
		Errors:        result.Errors,
		LastDiscovery: result.LastDiscovery.Format("2006-01-02T15:04:05Z"),
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), nil
}

// generateYAMLOutput generates YAML output
func (c *DiscoverCommand) generateYAMLOutput(result *template.DiscoveryResult) (string, error) {
	// For now, we'll use JSON and convert to YAML-like format
	// In a real implementation, you'd use a YAML library
	jsonOutput, err := c.generateJSONOutput(result)
	if err != nil {
		return "", err
	}

	// Simple conversion for demo purposes
	// In production, use a proper YAML library
	return jsonOutput, nil
}
