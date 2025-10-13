package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect/template"
	"go.uber.org/zap"
)

// CustomContent represents a custom template or script
type CustomContent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "template" or "script"
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// CustomManifest represents the manifest file for custom content
type CustomManifest struct {
	Version        string    `json:"version"`
	LastUpdated    time.Time `json:"last_updated"`
	Templates      []string  `json:"templates"`
	Scripts        []string  `json:"scripts"`
	TotalTemplates int       `json:"total_templates"`
	TotalScripts   int       `json:"total_scripts"`
	CreatedBy      string    `json:"created_by"`
}

// CustomStorageManager handles custom template and script storage
type CustomStorageManager struct {
	logger        *zap.Logger
	templatesDir  string
	scriptsDir    string
	manifestFile  string
	contentMutex  sync.RWMutex
	manifestMutex sync.RWMutex
	manifest      *CustomManifest
	ValkeyStore   *template.ValKeyTemplateStore // Added ValKey integration (exported for server access)
}

// NewCustomStorageManager creates a new custom storage manager
func NewCustomStorageManager(logger *zap.Logger, templatesDir, scriptsDir, manifestFile string, valkeyStore *template.ValKeyTemplateStore) (*CustomStorageManager, error) {
	manager := &CustomStorageManager{
		logger:       logger,
		templatesDir: templatesDir,
		scriptsDir:   scriptsDir,
		manifestFile: manifestFile,
		ValkeyStore:  valkeyStore, // Initialize ValKey store
	}

	// Create directories if they don't exist
	if err := manager.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Load or create manifest
	if err := manager.loadOrCreateManifest(); err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	logger.Info("Custom storage manager initialized",
		zap.String("templates_dir", templatesDir),
		zap.String("scripts_dir", scriptsDir),
		zap.String("manifest_file", manifestFile),
		zap.Bool("valkey_enabled", valkeyStore != nil))

	return manager, nil
}

// ensureDirectories creates the custom storage directories if they don't exist
func (csm *CustomStorageManager) ensureDirectories() error {
	dirs := []string{csm.templatesDir, csm.scriptsDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// loadOrCreateManifest loads the existing manifest or creates a new one
func (csm *CustomStorageManager) loadOrCreateManifest() error {
	csm.manifestMutex.Lock()
	defer csm.manifestMutex.Unlock()

	// Check if manifest file exists
	if _, err := os.Stat(csm.manifestFile); os.IsNotExist(err) {
		// Create new manifest
		csm.logger.Info("Custom manifest file not found, creating new manifest",
			zap.String("file", csm.manifestFile))
		csm.manifest = &CustomManifest{
			Version:        "1.0",
			LastUpdated:    time.Now(),
			Templates:      []string{},
			Scripts:        []string{},
			TotalTemplates: 0,
			TotalScripts:   0,
			CreatedBy:      "sirius-agent-server",
		}
		return csm.saveManifest()
	}

	// Load existing manifest
	data, err := os.ReadFile(csm.manifestFile)
	if err != nil {
		csm.logger.Error("Failed to read manifest file, creating new manifest",
			zap.String("file", csm.manifestFile),
			zap.Error(err))
		// Create new manifest on read error
		csm.manifest = &CustomManifest{
			Version:        "1.0",
			LastUpdated:    time.Now(),
			Templates:      []string{},
			Scripts:        []string{},
			TotalTemplates: 0,
			TotalScripts:   0,
			CreatedBy:      "sirius-agent-server",
		}
		return csm.saveManifest()
	}

	// Try to parse the manifest
	var manifest CustomManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		// Backup the corrupted file
		backupName := fmt.Sprintf("%s.bak-%d", csm.manifestFile, time.Now().Unix())
		if backupErr := os.WriteFile(backupName, data, 0644); backupErr != nil {
			csm.logger.Error("Failed to backup corrupted manifest file",
				zap.String("file", csm.manifestFile),
				zap.String("backup", backupName),
				zap.Error(backupErr))
		} else {
			csm.logger.Warn("Backed up corrupted manifest file",
				zap.String("file", csm.manifestFile),
				zap.String("backup", backupName))
		}

		csm.logger.Error("Failed to parse manifest file, creating new manifest",
			zap.String("file", csm.manifestFile),
			zap.Error(err))

		// Create new manifest on parse error
		csm.manifest = &CustomManifest{
			Version:        "1.0",
			LastUpdated:    time.Now(),
			Templates:      []string{},
			Scripts:        []string{},
			TotalTemplates: 0,
			TotalScripts:   0,
			CreatedBy:      "sirius-agent-server",
		}
		return csm.saveManifest()
	}

	// Validate the parsed manifest structure
	if manifest.Templates == nil {
		manifest.Templates = []string{}
	}
	if manifest.Scripts == nil {
		manifest.Scripts = []string{}
	}
	if manifest.Version == "" {
		manifest.Version = "1.0"
	}
	if manifest.CreatedBy == "" {
		manifest.CreatedBy = "sirius-agent-server"
	}

	// Update counts to match actual arrays
	manifest.TotalTemplates = len(manifest.Templates)
	manifest.TotalScripts = len(manifest.Scripts)

	csm.manifest = &manifest
	csm.logger.Info("Successfully loaded custom manifest",
		zap.String("file", csm.manifestFile),
		zap.Int("templates", manifest.TotalTemplates),
		zap.Int("scripts", manifest.TotalScripts))

	return nil
}

// saveManifest saves the manifest to disk
func (csm *CustomStorageManager) saveManifest() error {
	csm.manifest.LastUpdated = time.Now()

	data, err := json.MarshalIndent(csm.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(csm.manifestFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return nil
}

// SaveTemplate saves a custom template
func (csm *CustomStorageManager) SaveTemplate(content *CustomContent) error {
	csm.contentMutex.Lock()
	defer csm.contentMutex.Unlock()

	// Validate template content
	if err := csm.validateTemplate(content); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Set metadata
	content.Type = "template"
	content.UpdatedAt = time.Now()
	if content.CreatedAt.IsZero() {
		content.CreatedAt = time.Now()
	}

	// Save template file
	templateFile := filepath.Join(csm.templatesDir, content.ID+".yaml")
	if err := os.WriteFile(templateFile, []byte(content.Content), 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	// Update manifest
	csm.manifestMutex.Lock()
	defer csm.manifestMutex.Unlock()

	// Add to templates list if not already present
	found := false
	for _, id := range csm.manifest.Templates {
		if id == content.ID {
			found = true
			break
		}
	}
	if !found {
		csm.manifest.Templates = append(csm.manifest.Templates, content.ID)
		csm.manifest.TotalTemplates = len(csm.manifest.Templates)
	}

	// Sync to ValKey for template discovery
	if err := csm.syncTemplateToValKey(context.Background(), content); err != nil {
		csm.logger.Warn("Failed to sync template to ValKey", zap.String("template_id", content.ID), zap.Error(err))
	}

	return csm.saveManifest()
}

// syncTemplateToValKey syncs a custom template to ValKey for template discovery
func (csm *CustomStorageManager) syncTemplateToValKey(ctx context.Context, content *CustomContent) error {
	if csm.ValkeyStore == nil {
		return fmt.Errorf("ValKey store not initialized")
	}

	// Calculate content hash
	hash := sha256.Sum256([]byte(content.Content))
	hashStr := fmt.Sprintf("%x", hash)

	// Create template metadata for ValKey
	metadata := &template.TemplateMetadata{
		ID:          content.ID,
		Name:        content.Name,
		Description: content.Description,
		Author:      content.CreatedBy,
		Version:     content.Version,
		Severity:    "medium", // Default severity
		Tags:        content.Tags,
		Category:    "custom",
		Source: template.TemplateSource{
			Type:        template.TemplateSourceCustom,
			Name:        "custom",
			Priority:    100, // High priority for custom templates
			LastUpdated: content.UpdatedAt,
		},
		CreatedAt:  content.CreatedAt,
		UpdatedAt:  content.UpdatedAt,
		UsageCount: 0,
	}

	// Create template content for ValKey
	valkeyContent := &template.TemplateContent{
		ID:        content.ID,
		Content:   content.Content,
		Hash:      hashStr,
		Size:      int64(len(content.Content)),
		UpdatedAt: content.UpdatedAt,
	}

	// Store template metadata and content in ValKey
	if err := csm.ValkeyStore.SetTemplateMetadata(ctx, content.ID, metadata); err != nil {
		return fmt.Errorf("failed to store template metadata in ValKey: %w", err)
	}

	if err := csm.ValkeyStore.SetTemplateContent(ctx, content.ID, valkeyContent); err != nil {
		return fmt.Errorf("failed to store template content in ValKey: %w", err)
	}

	// Update custom template manifest in ValKey
	customManifest, err := csm.ValkeyStore.GetCustomTemplateManifest(ctx)
	if err != nil {
		// Create new manifest if it doesn't exist
		customManifest = &template.CustomTemplateManifest{
			Version:     "1.0.0",
			LastUpdated: time.Now(),
			Templates:   make(map[string]template.TemplateMetadata),
			Statistics: template.CustomTemplateStatistics{
				TotalTemplates: 0,
				LastCreated:    time.Now(),
				LastModified:   time.Now(),
			},
		}
	}

	// Add template to manifest
	customManifest.Templates[content.ID] = *metadata
	customManifest.LastUpdated = time.Now()
	customManifest.Statistics.TotalTemplates = len(customManifest.Templates)
	customManifest.Statistics.LastModified = time.Now()
	if _, exists := customManifest.Templates[content.ID]; !exists {
		customManifest.Statistics.LastCreated = time.Now()
	}

	// Save updated manifest
	if err := csm.ValkeyStore.SetCustomTemplateManifest(ctx, customManifest); err != nil {
		return fmt.Errorf("failed to update custom template manifest in ValKey: %w", err)
	}

	csm.logger.Info("Successfully synced template to ValKey",
		zap.String("template_id", content.ID),
		zap.String("template_name", content.Name))

	return nil
}

// deleteTemplateFromValKey removes a custom template from ValKey
func (csm *CustomStorageManager) deleteTemplateFromValKey(ctx context.Context, templateID string) error {
	if csm.ValkeyStore == nil {
		return fmt.Errorf("ValKey store not initialized")
	}

	// Delete template content and metadata from ValKey
	if err := csm.ValkeyStore.DeleteTemplate(ctx, templateID); err != nil {
		return fmt.Errorf("failed to delete template from ValKey: %w", err)
	}

	// Update custom template manifest in ValKey
	customManifest, err := csm.ValkeyStore.GetCustomTemplateManifest(ctx)
	if err != nil {
		// If manifest doesn't exist, nothing to update
		return nil
	}

	// Remove template from manifest
	delete(customManifest.Templates, templateID)
	customManifest.LastUpdated = time.Now()
	customManifest.Statistics.TotalTemplates = len(customManifest.Templates)
	customManifest.Statistics.LastModified = time.Now()

	// Save updated manifest
	if err := csm.ValkeyStore.SetCustomTemplateManifest(ctx, customManifest); err != nil {
		return fmt.Errorf("failed to update custom template manifest in ValKey: %w", err)
	}

	csm.logger.Info("Successfully deleted template from ValKey",
		zap.String("template_id", templateID))

	return nil
}

// SaveScript saves a custom script
func (csm *CustomStorageManager) SaveScript(content *CustomContent) error {
	csm.contentMutex.Lock()
	defer csm.contentMutex.Unlock()

	// Validate script content
	if err := csm.validateScript(content); err != nil {
		return fmt.Errorf("script validation failed: %w", err)
	}

	// Set metadata
	content.Type = "script"
	content.UpdatedAt = time.Now()
	if content.CreatedAt.IsZero() {
		content.CreatedAt = time.Now()
	}

	// Determine file extension based on script type
	extension := ".sh" // Default to bash
	if strings.Contains(content.Content, "powershell") || strings.Contains(content.Content, "Get-") {
		extension = ".ps1"
	} else if strings.Contains(content.Content, "python") || strings.Contains(content.Content, "import ") {
		extension = ".py"
	}

	// Save script file
	scriptFile := filepath.Join(csm.scriptsDir, content.ID+extension)
	if err := os.WriteFile(scriptFile, []byte(content.Content), 0755); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	// Update manifest
	csm.manifestMutex.Lock()
	defer csm.manifestMutex.Unlock()

	// Add to scripts list if not already present
	found := false
	for _, id := range csm.manifest.Scripts {
		if id == content.ID {
			found = true
			break
		}
	}
	if !found {
		csm.manifest.Scripts = append(csm.manifest.Scripts, content.ID)
		csm.manifest.TotalScripts = len(csm.manifest.Scripts)
	}

	return csm.saveManifest()
}

// GetTemplate retrieves a custom template
func (csm *CustomStorageManager) GetTemplate(id string) (*CustomContent, error) {
	csm.contentMutex.RLock()
	defer csm.contentMutex.RUnlock()

	templateFile := filepath.Join(csm.templatesDir, id+".yaml")

	content, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Get file info for timestamps
	fileInfo, err := os.Stat(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &CustomContent{
		ID:        id,
		Type:      "template",
		Content:   string(content),
		CreatedAt: fileInfo.ModTime(),
		UpdatedAt: fileInfo.ModTime(),
	}, nil
}

// GetScript retrieves a custom script
func (csm *CustomStorageManager) GetScript(id string) (*CustomContent, error) {
	csm.contentMutex.RLock()
	defer csm.contentMutex.RUnlock()

	// Try different extensions
	extensions := []string{".sh", ".ps1", ".py"}

	for _, ext := range extensions {
		scriptFile := filepath.Join(csm.scriptsDir, id+ext)

		content, err := os.ReadFile(scriptFile)
		if err == nil {
			// Get file info for timestamps
			fileInfo, err := os.Stat(scriptFile)
			if err != nil {
				return nil, fmt.Errorf("failed to get file info: %w", err)
			}

			return &CustomContent{
				ID:        id,
				Type:      "script",
				Content:   string(content),
				CreatedAt: fileInfo.ModTime(),
				UpdatedAt: fileInfo.ModTime(),
			}, nil
		}
	}

	return nil, fmt.Errorf("script not found")
}

// ListTemplates returns all custom template IDs
func (csm *CustomStorageManager) ListTemplates() ([]string, error) {
	csm.manifestMutex.RLock()
	defer csm.manifestMutex.RUnlock()

	return append([]string{}, csm.manifest.Templates...), nil
}

// ListScripts returns all custom script IDs
func (csm *CustomStorageManager) ListScripts() ([]string, error) {
	csm.manifestMutex.RLock()
	defer csm.manifestMutex.RUnlock()

	return append([]string{}, csm.manifest.Scripts...), nil
}

// DeleteTemplate deletes a custom template
func (csm *CustomStorageManager) DeleteTemplate(id string) error {
	csm.contentMutex.Lock()
	defer csm.contentMutex.Unlock()

	templateFile := filepath.Join(csm.templatesDir, id+".yaml")

	if err := os.Remove(templateFile); err != nil {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	// Update manifest
	csm.manifestMutex.Lock()
	defer csm.manifestMutex.Unlock()

	// Remove from templates list
	for i, templateID := range csm.manifest.Templates {
		if templateID == id {
			csm.manifest.Templates = append(csm.manifest.Templates[:i], csm.manifest.Templates[i+1:]...)
			csm.manifest.TotalTemplates = len(csm.manifest.Templates)
			break
		}
	}

	// Sync deletion to ValKey
	if err := csm.deleteTemplateFromValKey(context.Background(), id); err != nil {
		csm.logger.Warn("Failed to delete template from ValKey", zap.String("template_id", id), zap.Error(err))
	}

	return csm.saveManifest()
}

// DeleteScript deletes a custom script
func (csm *CustomStorageManager) DeleteScript(id string) error {
	csm.contentMutex.Lock()
	defer csm.contentMutex.Unlock()

	// Try different extensions
	extensions := []string{".sh", ".ps1", ".py"}

	for _, ext := range extensions {
		scriptFile := filepath.Join(csm.scriptsDir, id+ext)

		if err := os.Remove(scriptFile); err == nil {
			// Update manifest
			csm.manifestMutex.Lock()
			defer csm.manifestMutex.Unlock()

			// Remove from scripts list
			for i, scriptID := range csm.manifest.Scripts {
				if scriptID == id {
					csm.manifest.Scripts = append(csm.manifest.Scripts[:i], csm.manifest.Scripts[i+1:]...)
					csm.manifest.TotalScripts = len(csm.manifest.Scripts)
					break
				}
			}

			return csm.saveManifest()
		}
	}

	return fmt.Errorf("script not found")
}

// GetManifest returns the current manifest
func (csm *CustomStorageManager) GetManifest() *CustomManifest {
	csm.manifestMutex.RLock()
	defer csm.manifestMutex.RUnlock()

	return csm.manifest
}

// validateTemplate validates template content
func (csm *CustomStorageManager) validateTemplate(content *CustomContent) error {
	if content.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	if content.Content == "" {
		return fmt.Errorf("template content is required")
	}

	// Basic YAML validation (could be enhanced with proper YAML parsing)
	if !strings.Contains(content.Content, "id:") {
		return fmt.Errorf("template must contain 'id' field")
	}

	return nil
}

// validateScript validates script content
func (csm *CustomStorageManager) validateScript(content *CustomContent) error {
	if content.ID == "" {
		return fmt.Errorf("script ID is required")
	}

	if content.Content == "" {
		return fmt.Errorf("script content is required")
	}

	return nil
}
