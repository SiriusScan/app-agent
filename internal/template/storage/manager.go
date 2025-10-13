package storage

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/parser"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// TemplateSource represents the origin of a template
type TemplateSource string

const (
	SourceBuiltin TemplateSource = "builtin"
	SourceCustom  TemplateSource = "custom"
	SourceServer  TemplateSource = "server"
)

// Manager handles template discovery, loading, and precedence
type Manager struct {
	baseDir    string
	embeddedFS embed.FS // Set during initialization for built-in templates
	logger     *zap.Logger
}

// NewManager creates a new template manager
func NewManager(logger *zap.Logger) (*Manager, error) {
	baseDir, err := GetTemplateBaseDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get template base directory: %w", err)
	}

	// Ensure directory structure exists
	if err := EnsureDirectoryStructure(baseDir); err != nil {
		logger.Warn("Failed to create template directory structure",
			zap.String("base_dir", baseDir),
			zap.Error(err))
		// Don't fail - we can still use embedded templates
	}

	return &Manager{
		baseDir: baseDir,
		logger:  logger,
	}, nil
}

// NewManagerWithBaseDir creates a manager with a specific base directory (for testing)
func NewManagerWithBaseDir(baseDir string, logger *zap.Logger) (*Manager, error) {
	if err := EnsureDirectoryStructure(baseDir); err != nil {
		return nil, fmt.Errorf("failed to create directory structure: %w", err)
	}

	return &Manager{
		baseDir: baseDir,
		logger:  logger,
	}, nil
}

// SetEmbeddedFS sets the embedded filesystem for built-in templates
func (m *Manager) SetEmbeddedFS(fs embed.FS) {
	m.embeddedFS = fs
}

// DiscoverTemplates discovers templates from all sources with precedence.
// Precedence order: custom > server > builtin
// Templates with the same ID from higher precedence sources override lower ones.
func (m *Manager) DiscoverTemplates(ctx context.Context) ([]*types.Template, error) {
	templates := make(map[string]*types.Template) // key: template ID

	// 1. Load built-in templates (lowest priority)
	builtins, err := m.loadBuiltinTemplates(ctx)
	if err != nil {
		m.logger.Debug("Failed to load built-in templates", zap.Error(err))
	}
	for _, t := range builtins {
		templates[t.ID] = t
		m.logger.Debug("Loaded built-in template",
			zap.String("id", t.ID),
			zap.String("name", t.Info.Name))
	}

	// 2. Load server-synced templates (medium priority, overrides builtin)
	serverDir := filepath.Join(m.baseDir, "server")
	serverTemplates, serverErrors := parser.DiscoverTemplatesWithContext(ctx, serverDir)
	if len(serverErrors) > 0 {
		m.logger.Debug("Errors loading server templates",
			zap.Int("error_count", len(serverErrors)))
	}
	for _, t := range serverTemplates {
		if existing, exists := templates[t.ID]; exists {
			m.logger.Debug("Server template overrides built-in",
				zap.String("id", t.ID),
				zap.String("name", t.Info.Name),
				zap.String("previous_source", existing.FilePath))
		}
		templates[t.ID] = t
	}

	// 3. Load custom templates (highest priority, overrides all)
	customDir := filepath.Join(m.baseDir, "custom")
	customTemplates, customErrors := parser.DiscoverTemplatesWithContext(ctx, customDir)
	if len(customErrors) > 0 {
		m.logger.Debug("Errors loading custom templates",
			zap.Int("error_count", len(customErrors)))
	}
	for _, t := range customTemplates {
		if existing, exists := templates[t.ID]; exists {
			m.logger.Info("Custom template overrides other source",
				zap.String("id", t.ID),
				zap.String("name", t.Info.Name),
				zap.String("previous_source", existing.FilePath))
		}
		templates[t.ID] = t
	}

	// Convert map to slice
	result := make([]*types.Template, 0, len(templates))
	for _, t := range templates {
		result = append(result, t)
	}

	m.logger.Info("Template discovery complete",
		zap.Int("total_templates", len(result)),
		zap.Int("builtin", len(builtins)),
		zap.Int("server", len(serverTemplates)),
		zap.Int("custom", len(customTemplates)))

	return result, nil
}

// GetTemplate retrieves a specific template by ID
func (m *Manager) GetTemplate(ctx context.Context, id string) (*types.Template, error) {
	templates, err := m.DiscoverTemplates(ctx)
	if err != nil {
		return nil, err
	}

	for _, t := range templates {
		if t.ID == id {
			return t, nil
		}
	}

	return nil, fmt.Errorf("template %q not found", id)
}

// ListTemplates lists templates by source
func (m *Manager) ListTemplates(ctx context.Context, source TemplateSource) ([]*types.Template, error) {
	switch source {
	case SourceBuiltin:
		return m.loadBuiltinTemplates(ctx)
	case SourceCustom:
		templates, _ := parser.DiscoverTemplatesWithContext(ctx, filepath.Join(m.baseDir, "custom"))
		return templates, nil
	case SourceServer:
		templates, _ := parser.DiscoverTemplatesWithContext(ctx, filepath.Join(m.baseDir, "server"))
		return templates, nil
	default:
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

// GetStoragePath returns the base template storage directory
func (m *Manager) GetStoragePath() string {
	return m.baseDir
}

