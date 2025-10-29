package storage

import (
	"context"
	"embed"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

//go:embed templates/builtin/*.yaml
var embeddedTemplates embed.FS

// loadBuiltinTemplates loads templates from the embedded filesystem
// NOTE: Currently disabled - returns empty list. Built-in templates can be re-enabled
// in the future by uncommenting the code below.
func (m *Manager) loadBuiltinTemplates(ctx context.Context) ([]*types.Template, error) {
	// Built-in templates are currently disabled
	// Return empty list to use only server and custom templates
	m.logger.Debug("Built-in templates disabled, using only server and custom templates")
	return []*types.Template{}, nil

	/* COMMENTED OUT - Re-enable when needed:
	// Use the package-level embedded FS
	fsToUse := embeddedTemplates

	// If manager has custom embedded FS (for testing), use that instead
	if m.embeddedFS != (embed.FS{}) {
		fsToUse = m.embeddedFS
	}

	templates := []*types.Template{}

	err := fs.WalkDir(fsToUse, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if d.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}

		// Read file from embedded FS
		data, err := fsToUse.ReadFile(path)
		if err != nil {
			m.logger.Warn("Failed to read embedded template",
				zap.String("path", path),
				zap.Error(err))
			return nil // Continue processing
		}

		// Parse template
		template, err := parser.ParseTemplateBytes(data)
		if err != nil {
			m.logger.Warn("Failed to parse embedded template",
				zap.String("path", path),
				zap.Error(err))
			return nil // Continue processing
		}

		// Validate template
		if err := parser.ValidateTemplate(template); err != nil {
			m.logger.Warn("Invalid embedded template",
				zap.String("path", path),
				zap.Error(err))
			return nil // Continue processing
		}

		// Set file path to embedded path
		template.FilePath = "builtin:" + path

		templates = append(templates, template)
		return nil
	})

	if err != nil {
		return templates, err
	}

	m.logger.Debug("Loaded built-in templates",
		zap.Int("count", len(templates)))

	return templates, nil
	*/
}
