package template

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ValKey key constants for template management
const (
	// Global template manifests
	ValKeyTemplateManifestKey       = "agent:template:manifest"
	ValKeyTemplateRepoManifestKey   = "agent:template:repo-manifest"
	ValKeyTemplateCustomManifestKey = "agent:template:custom-manifest"

	// Individual template storage
	ValKeyTemplatePrefix     = "agent:template:"
	ValKeyTemplateMetaPrefix = "agent:template:meta:"

	// Template management settings
	ValKeyTemplateSettingsKey = "agent:template:settings"

	// Template source categories
	TemplateSourceCustom     = "custom"
	TemplateSourceRepository = "repository"
	TemplateSourceLocal      = "local"
)

// TemplateSource represents the source of a template
type TemplateSource struct {
	Type        string    `json:"type"`          // "custom", "repository", "local"
	Name        string    `json:"name"`          // Repository name or "custom"
	URL         string    `json:"url,omitempty"` // Repository URL if applicable
	Priority    int       `json:"priority"`      // Priority for conflict resolution (higher = more important)
	LastUpdated time.Time `json:"last_updated"`
}

// TemplateMetadata represents metadata for a template
type TemplateMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Version     string            `json:"version"`
	Severity    string            `json:"severity"`
	Tags        []string          `json:"tags"`
	Category    string            `json:"category"`
	Source      TemplateSource    `json:"source"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	UsageCount  int               `json:"usage_count"`
	LastUsed    *time.Time        `json:"last_used,omitempty"`
	CustomData  map[string]string `json:"custom_data,omitempty"`
}

// TemplateContent represents the actual template content
type TemplateContent struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"` // YAML content
	Hash      string    `json:"hash"`    // Content hash for validation
	Size      int64     `json:"size"`    // Content size in bytes
	UpdatedAt time.Time `json:"updated_at"`
}

// TemplateManifest represents the global template manifest
type TemplateManifest struct {
	Name        string                      `json:"name"`
	Version     string                      `json:"version"`
	Description string                      `json:"description"`
	LastUpdated time.Time                   `json:"last_updated"`
	Templates   map[string]TemplateMetadata `json:"templates"`
	Sources     map[string]TemplateSource   `json:"sources"`
	Statistics  TemplateStatistics          `json:"statistics"`
}

// TemplateStatistics represents usage statistics
type TemplateStatistics struct {
	TotalTemplates      int       `json:"total_templates"`
	CustomTemplates     int       `json:"custom_templates"`
	RepositoryTemplates int       `json:"repository_templates"`
	LocalTemplates      int       `json:"local_templates"`
	ActiveTemplates     int       `json:"active_templates"`
	LastSyncTime        time.Time `json:"last_sync_time"`
}

// RepositoryList represents the list of template repositories
type RepositoryList struct {
	Repositories []Repository `json:"repositories"`
	LastUpdated  time.Time    `json:"last_updated"`
}

// Repository represents a single template repository
type Repository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// CustomTemplateManifest represents custom templates
type CustomTemplateManifest struct {
	Version     string                      `json:"version"`
	LastUpdated time.Time                   `json:"last_updated"`
	Templates   map[string]TemplateMetadata `json:"templates"`
	Statistics  CustomTemplateStatistics    `json:"statistics"`
}

// CustomTemplateStatistics represents custom template statistics
type CustomTemplateStatistics struct {
	TotalTemplates int       `json:"total_templates"`
	LastCreated    time.Time `json:"last_created"`
	LastModified   time.Time `json:"last_modified"`
}

// TemplateSettings represents template management settings
type TemplateSettings struct {
	AutoSyncEnabled    bool          `json:"auto_sync_enabled"`
	SyncInterval       time.Duration `json:"sync_interval"`
	MaxTemplates       int           `json:"max_templates"`
	DefaultPriority    int           `json:"default_priority"`
	ConflictResolution string        `json:"conflict_resolution"` // "custom", "repository", "newest"
	LastSettingsUpdate time.Time     `json:"last_settings_update"`
}

// ValKeyTemplateStore provides methods for template storage in ValKey
type ValKeyTemplateStore struct {
	store KVStore
}

// KVStore interface for ValKey operations
type KVStore interface {
	GetValue(ctx context.Context, key string) (string, error)
	SetValue(ctx context.Context, key, value string) error
	ListKeys(ctx context.Context, pattern string) ([]string, error)
	DeleteValue(ctx context.Context, key string) error
	Close() error
}

// NewValKeyTemplateStore creates a new ValKey template store
func NewValKeyTemplateStore(store KVStore) *ValKeyTemplateStore {
	return &ValKeyTemplateStore{
		store: store,
	}
}

// GetTemplateManifest retrieves the global template manifest
func (vts *ValKeyTemplateStore) GetTemplateManifest(ctx context.Context) (*TemplateManifest, error) {
	data, err := vts.store.GetValue(ctx, ValKeyTemplateManifestKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get template manifest: %w", err)
	}

	var manifest TemplateManifest
	if err := json.Unmarshal([]byte(data), &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template manifest: %w", err)
	}

	return &manifest, nil
}

// SetTemplateManifest stores the global template manifest
func (vts *ValKeyTemplateStore) SetTemplateManifest(ctx context.Context, manifest *TemplateManifest) error {
	manifest.LastUpdated = time.Now()

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal template manifest: %w", err)
	}

	return vts.store.SetValue(ctx, ValKeyTemplateManifestKey, string(data))
}

// GetTemplateContent retrieves template content by ID
func (vts *ValKeyTemplateStore) GetTemplateContent(ctx context.Context, templateID string) (*TemplateContent, error) {
	key := ValKeyTemplatePrefix + templateID

	data, err := vts.store.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get template content for %s: %w", templateID, err)
	}

	var content TemplateContent
	if err := json.Unmarshal([]byte(data), &content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template content: %w", err)
	}

	return &content, nil
}

// SetTemplateContent stores template content
func (vts *ValKeyTemplateStore) SetTemplateContent(ctx context.Context, templateID string, content *TemplateContent) error {
	key := ValKeyTemplatePrefix + templateID
	content.UpdatedAt = time.Now()

	data, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal template content: %w", err)
	}

	return vts.store.SetValue(ctx, key, string(data))
}

// GetTemplateMetadata retrieves template metadata by ID
func (vts *ValKeyTemplateStore) GetTemplateMetadata(ctx context.Context, templateID string) (*TemplateMetadata, error) {
	key := ValKeyTemplateMetaPrefix + templateID

	data, err := vts.store.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get template metadata for %s: %w", templateID, err)
	}

	var metadata TemplateMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template metadata: %w", err)
	}

	return &metadata, nil
}

// SetTemplateMetadata stores template metadata
func (vts *ValKeyTemplateStore) SetTemplateMetadata(ctx context.Context, templateID string, metadata *TemplateMetadata) error {
	key := ValKeyTemplateMetaPrefix + templateID
	metadata.UpdatedAt = time.Now()

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal template metadata: %w", err)
	}

	return vts.store.SetValue(ctx, key, string(data))
}

// GetRepositoryList retrieves the repository list
func (vts *ValKeyTemplateStore) GetRepositoryList(ctx context.Context) (*RepositoryList, error) {
	data, err := vts.store.GetValue(ctx, ValKeyTemplateRepoManifestKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository list: %w", err)
	}

	var repoList RepositoryList
	if err := json.Unmarshal([]byte(data), &repoList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repository list: %w", err)
	}

	return &repoList, nil
}

// SetRepositoryList stores the repository list
func (vts *ValKeyTemplateStore) SetRepositoryList(ctx context.Context, repoList *RepositoryList) error {
	repoList.LastUpdated = time.Now()

	data, err := json.Marshal(repoList)
	if err != nil {
		return fmt.Errorf("failed to marshal repository list: %w", err)
	}

	return vts.store.SetValue(ctx, ValKeyTemplateRepoManifestKey, string(data))
}

// GetCustomTemplateManifest retrieves the custom template manifest
func (vts *ValKeyTemplateStore) GetCustomTemplateManifest(ctx context.Context) (*CustomTemplateManifest, error) {
	data, err := vts.store.GetValue(ctx, ValKeyTemplateCustomManifestKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom template manifest: %w", err)
	}

	var manifest CustomTemplateManifest
	if err := json.Unmarshal([]byte(data), &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom template manifest: %w", err)
	}

	return &manifest, nil
}

// SetCustomTemplateManifest stores the custom template manifest
func (vts *ValKeyTemplateStore) SetCustomTemplateManifest(ctx context.Context, manifest *CustomTemplateManifest) error {
	manifest.LastUpdated = time.Now()

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal custom template manifest: %w", err)
	}

	return vts.store.SetValue(ctx, ValKeyTemplateCustomManifestKey, string(data))
}

// GetTemplateSettings retrieves template management settings
func (vts *ValKeyTemplateStore) GetTemplateSettings(ctx context.Context) (*TemplateSettings, error) {
	data, err := vts.store.GetValue(ctx, ValKeyTemplateSettingsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get template settings: %w", err)
	}

	var settings TemplateSettings
	if err := json.Unmarshal([]byte(data), &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template settings: %w", err)
	}

	return &settings, nil
}

// SetTemplateSettings stores template management settings
func (vts *ValKeyTemplateStore) SetTemplateSettings(ctx context.Context, settings *TemplateSettings) error {
	settings.LastSettingsUpdate = time.Now()

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal template settings: %w", err)
	}

	return vts.store.SetValue(ctx, ValKeyTemplateSettingsKey, string(data))
}

// ListTemplateKeys retrieves all template keys
func (vts *ValKeyTemplateStore) ListTemplateKeys(ctx context.Context) ([]string, error) {
	return vts.store.ListKeys(ctx, ValKeyTemplatePrefix+"*")
}

// ListTemplateMetaKeys retrieves all template metadata keys
func (vts *ValKeyTemplateStore) ListTemplateMetaKeys(ctx context.Context) ([]string, error) {
	return vts.store.ListKeys(ctx, ValKeyTemplateMetaPrefix+"*")
}

// DeleteTemplate removes a template and its metadata
func (vts *ValKeyTemplateStore) DeleteTemplate(ctx context.Context, templateID string) error {
	// Delete template content
	contentKey := ValKeyTemplatePrefix + templateID
	if err := vts.store.DeleteValue(ctx, contentKey); err != nil {
		return fmt.Errorf("failed to delete template content: %w", err)
	}

	// Delete template metadata
	metaKey := ValKeyTemplateMetaPrefix + templateID
	if err := vts.store.DeleteValue(ctx, metaKey); err != nil {
		return fmt.Errorf("failed to delete template metadata: %w", err)
	}

	return nil
}

// Close closes the ValKey store connection
func (vts *ValKeyTemplateStore) Close() error {
	return vts.store.Close()
}
