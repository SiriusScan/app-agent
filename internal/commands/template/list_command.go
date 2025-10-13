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

// ListCommand implements the template listing command
type ListCommand struct {
	discoveryService *template.TemplateDiscoveryService
}

// Ensures ListCommand implements the Command interface at compile time.
var _ commands.Command = (*ListCommand)(nil)

func init() {
	commands.Register("list-templates", &ListCommand{})
	commands.Register("internal:list-templates", &ListCommand{})
}

// Execute lists all available templates with their metadata
func (c *ListCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (output string, err error) {
	agentInfo.Logger.Info("Executing template listing command", zap.String("args", args))

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
	showSource := true
	showDetails := false

	for _, arg := range argsList {
		switch {
		case arg == "--json":
			format = "json"
		case arg == "--yaml":
			format = "yaml"
		case arg == "--no-source":
			showSource = false
		case arg == "--details":
			showDetails = true
		}
	}

	// Discover all templates
	templates, err := c.discoveryService.ListAllTemplates(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list templates: %w", err)
	}

	// Generate output based on format
	switch format {
	case "json":
		return c.generateJSONOutput(templates, showSource, showDetails)
	case "yaml":
		return c.generateYAMLOutput(templates, showSource, showDetails)
	default:
		return c.generateTextOutput(templates, showSource, showDetails)
	}
}

// generateTextOutput generates human-readable text output
func (c *ListCommand) generateTextOutput(templates []*template.DiscoveredTemplate, showSource, showDetails bool) (string, error) {
	if len(templates) == 0 {
		return "No templates found.\n", nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📋 Found %d templates:\n\n", len(templates)))

	// Group templates by source
	sourceGroups := make(map[string][]*template.DiscoveredTemplate)
	for _, template := range templates {
		source := template.Source.Type
		sourceGroups[source] = append(sourceGroups[source], template)
	}

	// Output templates by source
	for source, sourceTemplates := range sourceGroups {
		output.WriteString(fmt.Sprintf("🔹 %s Templates (%d):\n", strings.Title(source), len(sourceTemplates)))

		for i, template := range sourceTemplates {
			output.WriteString(fmt.Sprintf("  %d. %s\n", i+1, template.Name))
			output.WriteString(fmt.Sprintf("      ID: %s\n", template.ID))
			output.WriteString(fmt.Sprintf("      Description: %s\n", template.Description))
			output.WriteString(fmt.Sprintf("      Severity: %s\n", template.Severity))
			output.WriteString(fmt.Sprintf("      Type: %s\n", template.Type))

			if showSource {
				output.WriteString(fmt.Sprintf("      Source: %s (%s)\n", template.Source.Name, template.Source.Type))
				output.WriteString(fmt.Sprintf("      Priority: %d\n", template.Source.Priority))
			}

			if showDetails {
				output.WriteString(fmt.Sprintf("      Hash: %s\n", template.Hash))
				output.WriteString(fmt.Sprintf("      Size: %d bytes\n", template.Size))
				output.WriteString(fmt.Sprintf("      Created: %s\n", template.CreatedAt.Format("2006-01-02 15:04:05")))
				output.WriteString(fmt.Sprintf("      Updated: %s\n", template.UpdatedAt.Format("2006-01-02 15:04:05")))
			}

			output.WriteString("\n")
		}
	}

	return output.String(), nil
}

// generateJSONOutput generates JSON output
func (c *ListCommand) generateJSONOutput(templates []*template.DiscoveredTemplate, showSource, showDetails bool) (string, error) {
	type TemplateOutput struct {
		ID          string                   `json:"id"`
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Severity    string                   `json:"severity"`
		Type        string                   `json:"type"`
		Source      *template.TemplateSource `json:"source,omitempty"`
		Details     map[string]interface{}   `json:"details,omitempty"`
	}

	var output []TemplateOutput
	for _, template := range templates {
		templateOutput := TemplateOutput{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			Severity:    template.Severity,
			Type:        template.Type,
		}

		if showSource {
			templateOutput.Source = &template.Source
		}

		if showDetails {
			templateOutput.Details = map[string]interface{}{
				"hash":      template.Hash,
				"size":      template.Size,
				"created":   template.CreatedAt,
				"updated":   template.UpdatedAt,
				"file_path": template.FilePath,
			}
		}

		output = append(output, templateOutput)
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), nil
}

// generateYAMLOutput generates YAML output
func (c *ListCommand) generateYAMLOutput(templates []*template.DiscoveredTemplate, showSource, showDetails bool) (string, error) {
	// For now, we'll use JSON and convert to YAML-like format
	// In a real implementation, you'd use a YAML library
	jsonOutput, err := c.generateJSONOutput(templates, showSource, showDetails)
	if err != nil {
		return "", err
	}

	// Simple conversion for demo purposes
	// In production, use a proper YAML library
	return jsonOutput, nil
}
