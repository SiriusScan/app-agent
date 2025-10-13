package parser

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

// ParseTemplate parses a template from a file path.
// It reads the file, unmarshals the YAML content, and sets metadata fields.
func ParseTemplate(path string) (*types.Template, error) {
	if path == "" {
		return nil, fmt.Errorf("template path cannot be empty")
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("template file not found: %s", path)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading template: %s", path)
		}
		return nil, fmt.Errorf("failed to read template file %s: %w", path, err)
	}

	// Parse the bytes
	template, err := ParseTemplateBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", path, err)
	}

	// Set the file path metadata
	template.FilePath = path

	return template, nil
}

// ParseTemplateBytes parses a template from raw bytes.
// It unmarshals the YAML content and sets the LoadedAt metadata field.
func ParseTemplateBytes(data []byte) (*types.Template, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("template data cannot be empty")
	}

	var template types.Template

	// Unmarshal the YAML
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Set metadata
	template.LoadedAt = time.Now()

	// Set default detection logic if not specified
	if template.Detection.Logic == "" {
		template.Detection.Logic = types.LogicAll
	}

	// Set default weights if not specified
	for i := range template.Detection.Steps {
		if template.Detection.Steps[i].Weight == 0 {
			template.Detection.Steps[i].Weight = 1.0
		}
	}

	return &template, nil
}

