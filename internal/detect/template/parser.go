package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// TemplateParser handles YAML template parsing and validation
type TemplateParser struct {
	logger          *zap.Logger
	templateDirs    []string
	loadedTemplates map[string]*detect.VulnTemplate
}

// NewTemplateParser creates a new template parser instance
func NewTemplateParser(logger *zap.Logger, templateDirs []string) *TemplateParser {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &TemplateParser{
		logger:          logger,
		templateDirs:    templateDirs,
		loadedTemplates: make(map[string]*detect.VulnTemplate),
	}
}

// LoadTemplates discovers and loads all YAML templates from configured directories
func (p *TemplateParser) LoadTemplates(ctx context.Context) ([]*detect.VulnTemplate, error) {
	p.logger.Info("Starting template discovery and loading",
		zap.Strings("template_dirs", p.templateDirs))

	var allTemplates []*detect.VulnTemplate
	loadErrors := make([]string, 0)

	for _, templateDir := range p.templateDirs {
		templates, errors := p.loadTemplatesFromDirectory(ctx, templateDir)
		allTemplates = append(allTemplates, templates...)
		loadErrors = append(loadErrors, errors...)
	}

	p.logger.Info("Template loading completed",
		zap.Int("total_templates", len(allTemplates)),
		zap.Int("load_errors", len(loadErrors)))

	if len(loadErrors) > 0 {
		p.logger.Warn("Some templates failed to load", zap.Strings("errors", loadErrors))
	}

	// Store loaded templates for quick access
	for _, template := range allTemplates {
		p.loadedTemplates[template.ID] = template
	}

	return allTemplates, nil
}

// LoadTemplate loads and validates a single YAML template file
func (p *TemplateParser) LoadTemplate(templatePath string) (*detect.VulnTemplate, error) {
	p.logger.Debug("Loading template file", zap.String("path", templatePath))

	// Read template file
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Parse YAML content
	var template detect.VulnTemplate
	if err := yaml.Unmarshal(templateData, &template); err != nil {
		return nil, fmt.Errorf("failed to parse YAML template %s: %w", templatePath, err)
	}

	// Set metadata
	template.FilePath = templatePath
	template.LoadedAt = time.Now()

	// Validate template structure
	if err := p.ValidateTemplate(&template); err != nil {
		return nil, fmt.Errorf("template validation failed for %s: %w", templatePath, err)
	}

	// Normalize platform-specific paths
	if err := p.normalizePlatformPaths(&template); err != nil {
		return nil, fmt.Errorf("path normalization failed for %s: %w", templatePath, err)
	}

	p.logger.Debug("Successfully loaded template",
		zap.String("id", template.ID),
		zap.String("name", template.Info.Name),
		zap.String("path", templatePath))

	return &template, nil
}

// ValidateTemplate checks template syntax and structure
func (p *TemplateParser) ValidateTemplate(template *detect.VulnTemplate) error {
	if template == nil {
		return fmt.Errorf("template is nil")
	}

	// Validate required fields
	if template.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	if template.Info.Name == "" {
		return fmt.Errorf("template info.name is required")
	}

	if template.Info.Severity == "" {
		return fmt.Errorf("template info.severity is required")
	}

	// Validate severity level
	validSeverities := map[detect.SeverityLevel]bool{
		detect.SeverityLevelCritical: true,
		detect.SeverityLevelHigh:     true,
		detect.SeverityLevelMedium:   true,
		detect.SeverityLevelLow:      true,
		detect.SeverityLevelInfo:     true,
	}
	if !validSeverities[template.Info.Severity] {
		return fmt.Errorf("invalid severity level: %s", template.Info.Severity)
	}

	// Validate detection configuration
	if err := p.validateDetectionConfig(&template.Detection); err != nil {
		return fmt.Errorf("detection config validation failed: %w", err)
	}

	return nil
}

// validateDetectionConfig validates the detection configuration section
func (p *TemplateParser) validateDetectionConfig(config *detect.DetectionConfig) error {
	if config == nil {
		return fmt.Errorf("detection config is required")
	}

	// Validate detection type
	validTypes := map[detect.DetectionType]bool{
		detect.DetectionTypeFileHash:   true,
		detect.DetectionTypeRegistry:   true,
		detect.DetectionTypeConfigFile: true,
		detect.DetectionTypeProcess:    true,
		detect.DetectionTypeService:    true,
		detect.DetectionTypeNetwork:    true,
	}
	if !validTypes[config.Type] {
		return fmt.Errorf("invalid detection type: %s", config.Type)
	}

	// Validate type-specific requirements
	switch config.Type {
	case detect.DetectionTypeFileHash:
		if len(config.Targets) == 0 {
			return fmt.Errorf("file-hash detection requires at least one target")
		}
		if config.Method == "" {
			config.Method = "sha256" // Default hash method
		}
		// Validate hash method
		validMethods := map[string]bool{
			"sha256": true, "sha1": true, "md5": true, "sha512": true,
		}
		if !validMethods[config.Method] {
			return fmt.Errorf("invalid hash method: %s", config.Method)
		}

	case detect.DetectionTypeRegistry:
		if len(config.Keys) == 0 {
			return fmt.Errorf("registry detection requires at least one key")
		}

	case detect.DetectionTypeConfigFile:
		if len(config.Files) == 0 {
			return fmt.Errorf("config-file detection requires at least one file")
		}
	}

	// Validate conditions
	if len(config.Conditions) == 0 {
		return fmt.Errorf("detection requires at least one condition")
	}

	for i, condition := range config.Conditions {
		if err := p.validateCondition(&condition); err != nil {
			return fmt.Errorf("condition %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateCondition validates a single detection condition
func (p *TemplateParser) validateCondition(condition *detect.DetectionCondition) error {
	if condition == nil {
		return fmt.Errorf("condition is nil")
	}

	validConditionTypes := map[detect.ConditionType]bool{
		detect.ConditionTypeFileExists:          true,
		detect.ConditionTypeHashMatch:           true,
		detect.ConditionTypeFileExecutable:      true,
		detect.ConditionTypeKeyExists:           true,
		detect.ConditionTypeValueMatchesPattern: true,
		detect.ConditionTypePatternFound:        true,
	}

	if !validConditionTypes[condition.Type] {
		return fmt.Errorf("invalid condition type: %s", condition.Type)
	}

	return nil
}

// normalizePlatformPaths normalizes file paths for cross-platform compatibility
func (p *TemplateParser) normalizePlatformPaths(template *detect.VulnTemplate) error {
	currentPlatform := runtime.GOOS

	// Normalize detection targets
	for i := range template.Detection.Targets {
		target := &template.Detection.Targets[i]

		// Check if target is applicable to current platform
		if len(target.Platform) > 0 {
			applicable := false
			for _, platform := range target.Platform {
				if platform == currentPlatform {
					applicable = true
					break
				}
			}
			if !applicable {
				continue // Skip this target on current platform
			}
		}

		// Normalize path separators
		target.Path = filepath.Clean(target.Path)
	}

	// Normalize file paths
	for i := range template.Detection.Files {
		file := &template.Detection.Files[i]
		file.Path = filepath.Clean(file.Path)
	}

	return nil
}

// loadTemplatesFromDirectory recursively loads templates from a directory
func (p *TemplateParser) loadTemplatesFromDirectory(ctx context.Context, templateDir string) ([]*detect.VulnTemplate, []string) {
	var templates []*detect.VulnTemplate
	var errors []string

	// Check if directory exists with timeout
	dirCheckCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use a goroutine to check directory existence with timeout
	dirExists := make(chan bool, 1)
	go func() {
		if _, err := os.Stat(templateDir); os.IsNotExist(err) {
			dirExists <- false
		} else {
			dirExists <- true
		}
	}()

	// Wait for directory check with timeout
	select {
	case exists := <-dirExists:
		if !exists {
			p.logger.Warn("Template directory does not exist", zap.String("dir", templateDir))
			return templates, errors
		}
	case <-dirCheckCtx.Done():
		p.logger.Warn("Template directory check timed out", zap.String("dir", templateDir))
		return templates, errors
	}

	// Walk through directory with overall timeout protection
	walkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Use a channel to handle walk results
	walkDone := make(chan struct{})
	walkError := make(chan error, 1)

	go func() {
		defer close(walkDone)

		err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				errors = append(errors, fmt.Sprintf("error accessing path %s: %v", path, err))
				return nil // Continue walking
			}

			// Only process YAML files
			if !isYAMLFile(path) {
				return nil
			}

			// Check context cancellation
			select {
			case <-walkCtx.Done():
				return walkCtx.Err()
			default:
			}

			// Load template with individual timeout
			templateCtx, templateCancel := context.WithTimeout(walkCtx, 10*time.Second)
			defer templateCancel()

			// Load template in goroutine with timeout
			templateDone := make(chan struct{})
			var template *detect.VulnTemplate
			var loadErr error

			go func() {
				defer close(templateDone)
				template, loadErr = p.LoadTemplate(path)
			}()

			// Wait for template loading with timeout
			select {
			case <-templateDone:
				if loadErr != nil {
					errors = append(errors, fmt.Sprintf("failed to load template %s: %v", path, loadErr))
				} else if template != nil {
					templates = append(templates, template)
				}
			case <-templateCtx.Done():
				errors = append(errors, fmt.Sprintf("template loading timed out for %s", path))
			}

			return nil
		})

		if err != nil {
			walkError <- err
		}
	}()

	// Wait for walk completion or timeout
	select {
	case <-walkDone:
		// Walk completed normally
	case err := <-walkError:
		errors = append(errors, fmt.Sprintf("error walking directory %s: %v", templateDir, err))
	case <-walkCtx.Done():
		errors = append(errors, fmt.Sprintf("template directory walk timed out for %s", templateDir))
	}

	p.logger.Debug("Loaded templates from directory",
		zap.String("dir", templateDir),
		zap.Int("templates", len(templates)),
		zap.Int("errors", len(errors)))

	return templates, errors
}

// GetLoadedTemplate retrieves a previously loaded template by ID
func (p *TemplateParser) GetLoadedTemplate(templateID string) (*detect.VulnTemplate, bool) {
	template, exists := p.loadedTemplates[templateID]
	return template, exists
}

// ListLoadedTemplates returns all currently loaded templates
func (p *TemplateParser) ListLoadedTemplates() []*detect.VulnTemplate {
	templates := make([]*detect.VulnTemplate, 0, len(p.loadedTemplates))
	for _, template := range p.loadedTemplates {
		templates = append(templates, template)
	}
	return templates
}

// FilterTemplatesByPlatform filters templates applicable to the current platform
func (p *TemplateParser) FilterTemplatesByPlatform(templates []*detect.VulnTemplate, platform string) []*detect.VulnTemplate {
	if platform == "" {
		platform = runtime.GOOS
	}

	var filtered []*detect.VulnTemplate
	for _, template := range templates {
		if p.isTemplateApplicableToPlatform(template, platform) {
			filtered = append(filtered, template)
		}
	}

	p.logger.Debug("Filtered templates by platform",
		zap.String("platform", platform),
		zap.Int("total_templates", len(templates)),
		zap.Int("applicable_templates", len(filtered)))

	return filtered
}

// isTemplateApplicableToPlatform checks if template applies to given platform
func (p *TemplateParser) isTemplateApplicableToPlatform(template *detect.VulnTemplate, platform string) bool {
	// Check if any targets apply to this platform
	for _, target := range template.Detection.Targets {
		if len(target.Platform) == 0 {
			return true // No platform restriction means universal
		}
		for _, targetPlatform := range target.Platform {
			if targetPlatform == platform {
				return true
			}
		}
	}

	// Check files (config-file detection)
	if len(template.Detection.Files) > 0 {
		return true // Config files are generally platform-agnostic
	}

	// Check registry keys (Windows-specific)
	if len(template.Detection.Keys) > 0 {
		return platform == "windows"
	}

	return false
}

// isYAMLFile checks if a file has a YAML extension
func isYAMLFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".yaml" || ext == ".yml"
}

// GetTemplateStats returns statistics about loaded templates
func (p *TemplateParser) GetTemplateStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalTemplates := len(p.loadedTemplates)
	stats["total_templates"] = totalTemplates

	// Count by detection type
	typeCount := make(map[detect.DetectionType]int)
	severityCount := make(map[detect.SeverityLevel]int)

	for _, template := range p.loadedTemplates {
		typeCount[template.Detection.Type]++
		severityCount[template.Info.Severity]++
	}

	stats["by_detection_type"] = typeCount
	stats["by_severity"] = severityCount
	stats["platform_current"] = runtime.GOOS

	return stats
}
