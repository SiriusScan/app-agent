package valkey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	valkey "github.com/valkey-io/valkey-go"
	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

// ValKeyTemplateStorage manages template storage in ValKey
type ValKeyTemplateStorage struct {
	client valkey.Client
	logger *zap.Logger
}

// NewValKeyTemplateStorage creates a new ValKey template storage instance
func NewValKeyTemplateStorage(client valkey.Client, logger *zap.Logger) *ValKeyTemplateStorage {
	return &ValKeyTemplateStorage{
		client: client,
		logger: logger,
	}
}

// Template storage key constants
const (
	TemplateManifestKey     = "template:manifest"
	TemplateRepoManifestKey = "template:repo-manifest"
	TemplateStandardPrefix  = "template:standard:"
	TemplateCustomPrefix    = "template:custom:"
	TemplateMetaPrefix      = "template:meta:"
	TemplateVersionPrefix   = "template:version:"
)

// TemplateManifest represents the global template manifest
type TemplateManifest struct {
	Version     string                 `json:"version"`
	Updated     time.Time              `json:"updated"`
	Statistics  TemplateStatistics     `json:"statistics"`
	Templates   map[string]*TemplateInfo `json:"templates"`
	LastSync    time.Time              `json:"last_sync"`
}

// TemplateStatistics contains template statistics
type TemplateStatistics struct {
	TotalTemplates     int `json:"total_templates"`
	StandardTemplates  int `json:"standard_templates"`
	CustomTemplates    int `json:"custom_templates"`
	ByType             map[string]int `json:"by_type"`
	ByPlatform         map[string]int `json:"by_platform"`
	BySeverity         map[string]int `json:"by_severity"`
}

// TemplateInfo represents template information stored in ValKey
type TemplateInfo struct {
	ID               string            `json:"id"`
	Version          string            `json:"version"`
	Checksum         string            `json:"checksum"`
	Size             int64             `json:"size"`
	Severity         string            `json:"severity"`
	Platforms        []string          `json:"platforms"`
	DetectionType    string            `json:"detection_type"`
	Author           string            `json:"author"`
	Created          time.Time         `json:"created"`
	Updated          time.Time         `json:"updated"`
	VulnerabilityIDs []string          `json:"vulnerability_ids"`
	Content          []byte            `json:"content,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// StoreTemplate stores a template in ValKey
func (s *ValKeyTemplateStorage) StoreTemplate(ctx context.Context, template *types.Template, content []byte, isCustom bool) error {
	// Skip if client is nil (graceful degradation)
	if s.client == nil {
		s.logger.Warn("Cannot store template - ValKey client is nil")
		return nil
	}
	
	// Calculate checksum
	checksum := s.calculateChecksum(content)

	// Create template info
	templateInfo := &TemplateInfo{
		ID:               template.ID,
		Version:          template.Info.Version,
		Checksum:         checksum,
		Size:             int64(len(content)),
		Severity:         string(template.Info.Severity),
		Platforms:        getPlatformsFromDetection(template.Detection),
		DetectionType:    getDetectionType(template.Detection),
		Author:           template.Info.Author,
		Created:          time.Now(), // Use current time as fallback
		Updated:          time.Now(), // Use current time as fallback
		VulnerabilityIDs: template.Info.CVE, // Use CVE field instead
		Content:          content,
		Metadata:         make(map[string]string),
	}

	// Determine key prefix
	var keyPrefix string
	if isCustom {
		keyPrefix = TemplateCustomPrefix
	} else {
		keyPrefix = TemplateStandardPrefix
	}

	// Store template content
	templateKey := keyPrefix + template.ID
	templateData, err := json.Marshal(templateInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	cmd := s.client.B().Set().Key(templateKey).Value(string(templateData)).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to store template: %w", err)
	}

	// Store template metadata (without content)
	metaInfo := *templateInfo
	metaInfo.Content = nil
	metaKey := TemplateMetaPrefix + template.ID
	metaData, err := json.Marshal(metaInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal template metadata: %w", err)
	}

	cmd = s.client.B().Set().Key(metaKey).Value(string(metaData)).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to store template metadata: %w", err)
	}

	// Store version/checksum for cache invalidation
	versionKey := TemplateVersionPrefix + template.ID
	cmd = s.client.B().Set().Key(versionKey).Value(checksum).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to store template version: %w", err)
	}

	s.logger.Info("Template stored successfully",
		zap.String("id", template.ID),
		zap.Bool("custom", isCustom),
		zap.String("checksum", checksum))

	return nil
}

// GetTemplate retrieves a template from ValKey
func (s *ValKeyTemplateStorage) GetTemplate(ctx context.Context, templateID string, isCustom bool) (*types.Template, []byte, error) {
	var keyPrefix string
	if isCustom {
		keyPrefix = TemplateCustomPrefix
	} else {
		keyPrefix = TemplateStandardPrefix
	}

	templateKey := keyPrefix + templateID
	cmd := s.client.B().Get().Key(templateKey).Build()
	resp := s.client.Do(ctx, cmd)
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil, fmt.Errorf("template not found: %s", templateID)
		}
		return nil, nil, fmt.Errorf("failed to get template: %w", err)
	}
	
	templateData, err := resp.ToString()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert template data to string: %w", err)
	}

	var templateInfo TemplateInfo
	if err := json.Unmarshal([]byte(templateData), &templateInfo); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	// Convert to types.Template
	template := &types.Template{
		ID: templateInfo.ID,
		Info: types.TemplateInfo{
			Name:        templateInfo.ID, // Use ID as name if not stored separately
			Author:      templateInfo.Author,
			Severity:    types.Severity(templateInfo.Severity),
			Version:     templateInfo.Version,
			CVE:         templateInfo.VulnerabilityIDs,
		},
		Detection: types.DetectionConfig{
			Steps: []types.DetectionStep{
				{
					Type: templateInfo.DetectionType,
				},
			},
		},
	}

	return template, templateInfo.Content, nil
}

// GetTemplateManifest retrieves the global template manifest
func (s *ValKeyTemplateStorage) GetTemplateManifest(ctx context.Context) (*TemplateManifest, error) {
	// Return empty manifest if client is nil
	if s.client == nil {
		return &TemplateManifest{
			Version:    "1.0.0",
			Updated:    time.Now(),
			Statistics: TemplateStatistics{},
			Templates:  make(map[string]*TemplateInfo),
		}, nil
	}
	
	cmd := s.client.B().Get().Key(TemplateManifestKey).Build()
	resp := s.client.Do(ctx, cmd)
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return &TemplateManifest{
				Version:    "1.0.0",
				Updated:    time.Now(),
				Statistics: TemplateStatistics{},
				Templates:  make(map[string]*TemplateInfo),
			}, nil
		}
		return nil, fmt.Errorf("failed to get template manifest: %w", err)
	}
	
	manifestData, err := resp.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest data to string: %w", err)
	}

	var manifest TemplateManifest
	if err := json.Unmarshal([]byte(manifestData), &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template manifest: %w", err)
	}

	return &manifest, nil
}

// UpdateTemplateManifest updates the global template manifest
func (s *ValKeyTemplateStorage) UpdateTemplateManifest(ctx context.Context, manifest *TemplateManifest) error {
	// Skip if client is nil (graceful degradation)
	if s.client == nil {
		s.logger.Warn("Cannot update manifest - ValKey client is nil")
		return nil
	}
	
	manifest.Updated = time.Now()
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal template manifest: %w", err)
	}

	cmd := s.client.B().Set().Key(TemplateManifestKey).Value(string(manifestData)).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to store template manifest: %w", err)
	}

	s.logger.Info("Template manifest updated",
		zap.String("version", manifest.Version),
		zap.Int("total_templates", manifest.Statistics.TotalTemplates))

	return nil
}

// ListTemplates lists all templates with their metadata
func (s *ValKeyTemplateStorage) ListTemplates(ctx context.Context, isCustom bool) ([]*TemplateInfo, error) {
	var keyPrefix string
	if isCustom {
		keyPrefix = TemplateCustomPrefix
	} else {
		keyPrefix = TemplateStandardPrefix
	}

	// Get all template keys
	cmd := s.client.B().Keys().Pattern(keyPrefix + "*").Build()
	resp := s.client.Do(ctx, cmd)
	if err := resp.Error(); err != nil {
		return nil, fmt.Errorf("failed to list template keys: %w", err)
	}
	
	keyMessages, err := resp.ToArray()
	if err != nil {
		return nil, fmt.Errorf("failed to convert keys response to array: %w", err)
	}
	
	keys := make([]string, len(keyMessages))
	for i, keyMsg := range keyMessages {
		key, err := keyMsg.ToString()
		if err != nil {
			return nil, fmt.Errorf("failed to convert key message to string: %w", err)
		}
		keys[i] = key
	}

	var templates []*TemplateInfo
	for _, key := range keys {
		cmd := s.client.B().Get().Key(key).Build()
		resp := s.client.Do(ctx, cmd)
		if err := resp.Error(); err != nil {
			s.logger.Warn("Failed to get template data", zap.String("key", key), zap.Error(err))
			continue
		}
		
		templateData, err := resp.ToString()
		if err != nil {
			s.logger.Warn("Failed to convert template data to string", zap.String("key", key), zap.Error(err))
			continue
		}

		var templateInfo TemplateInfo
		if err := json.Unmarshal([]byte(templateData), &templateInfo); err != nil {
			s.logger.Warn("Failed to unmarshal template", zap.String("key", key), zap.Error(err))
			continue
		}

		// Remove content for list operation
		templateInfo.Content = nil
		templates = append(templates, &templateInfo)
	}

	return templates, nil
}

// DeleteTemplate deletes a template from ValKey
func (s *ValKeyTemplateStorage) DeleteTemplate(ctx context.Context, templateID string, isCustom bool) error {
	var keyPrefix string
	if isCustom {
		keyPrefix = TemplateCustomPrefix
	} else {
		keyPrefix = TemplateStandardPrefix
	}

	// Delete template content
	templateKey := keyPrefix + templateID
	cmd := s.client.B().Del().Key(templateKey).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Delete template metadata
	metaKey := TemplateMetaPrefix + templateID
	cmd = s.client.B().Del().Key(metaKey).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to delete template metadata: %w", err)
	}

	// Delete version/checksum
	versionKey := TemplateVersionPrefix + templateID
	cmd = s.client.B().Del().Key(versionKey).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to delete template version: %w", err)
	}

	s.logger.Info("Template deleted successfully",
		zap.String("id", templateID),
		zap.Bool("custom", isCustom))

	return nil
}

// GetTemplateChecksum retrieves the checksum for a template
func (s *ValKeyTemplateStorage) GetTemplateChecksum(ctx context.Context, templateID string) (string, error) {
	versionKey := TemplateVersionPrefix + templateID
	cmd := s.client.B().Get().Key(versionKey).Build()
	resp := s.client.Do(ctx, cmd)
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return "", fmt.Errorf("template version not found: %s", templateID)
		}
		return "", fmt.Errorf("failed to get template checksum: %w", err)
	}
	
	checksum, err := resp.ToString()
	if err != nil {
		return "", fmt.Errorf("failed to convert checksum to string: %w", err)
	}

	return checksum, nil
}

// calculateChecksum calculates SHA256 checksum of template content
func (s *ValKeyTemplateStorage) calculateChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(hash[:])
}

// convertPlatformsToStrings converts Platform slice to string slice
func convertPlatformsToStrings(platforms []types.Platform) []string {
	result := make([]string, len(platforms))
	for i, platform := range platforms {
		result[i] = string(platform)
	}
	return result
}

// getDetectionType extracts detection type from DetectionConfig
func getDetectionType(detection types.DetectionConfig) string {
	if len(detection.Steps) > 0 {
		return detection.Steps[0].Type
	}
	return "unknown"
}

// getPlatformsFromDetection extracts platforms from DetectionConfig
func getPlatformsFromDetection(detection types.DetectionConfig) []string {
	var platforms []string
	platformSet := make(map[string]bool)
	
	for _, step := range detection.Steps {
		for _, platform := range step.Platforms {
			if !platformSet[string(platform)] {
				platforms = append(platforms, string(platform))
				platformSet[string(platform)] = true
			}
		}
	}
	
	// If no platforms specified, assume all platforms
	if len(platforms) == 0 {
		platforms = []string{"linux", "windows", "darwin"}
	}
	
	return platforms
}

// ValidateTemplate validates template content and structure
func (s *ValKeyTemplateStorage) ValidateTemplate(template *types.Template, content []byte) error {
	// Basic validation
	if template.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	if len(content) == 0 {
		return fmt.Errorf("template content is required")
	}

	if len(content) > 1024*1024 { // 1MB limit
		return fmt.Errorf("template content exceeds 1MB limit")
	}

	if template.Info.Severity == "" {
		return fmt.Errorf("template severity is required")
	}

	// Validate severity
	validSeverities := map[string]bool{
		"critical": true,
		"high":     true,
		"medium":   true,
		"low":      true,
	}
	if !validSeverities[string(template.Info.Severity)] {
		return fmt.Errorf("invalid severity: %s", template.Info.Severity)
	}

	// Validate detection type
	validTypes := map[string]bool{
		"file-hash":    true,
		"file-content": true,
		"config-file":  true,
		"registry":     true,
		"version-cmd":  true,
	}
	detectionType := getDetectionType(template.Detection)
	if !validTypes[detectionType] {
		return fmt.Errorf("invalid detection type: %s", detectionType)
	}

	return nil
}
