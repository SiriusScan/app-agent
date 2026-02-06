package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/template/parser"
	"github.com/SiriusScan/app-agent/internal/template/types"
	pb "github.com/SiriusScan/app-agent/proto/hello"
)

// StreamSendFunc is a function that sends an AgentMessage on the gRPC stream.
// This allows the sync manager to use the agent's mutex-protected send.
type StreamSendFunc func(msg *pb.AgentMessage) error

// AgentSyncManager manages template synchronization for agents
type AgentSyncManager struct {
	cacheDir  string
	logger    *zap.Logger
	serverURL string
	agentID   string

	// gRPC stream for communication with server
	grpcStream pb.HelloService_ConnectStreamClient
	// streamSendFunc is a mutex-protected send function provided by the agent.
	// When set, it is used instead of grpcStream.Send() to avoid data races.
	streamSendFunc StreamSendFunc
}

// CacheManifest represents the local cache manifest
type CacheManifest struct {
	Version    string                        `json:"version"`
	LastSync   time.Time                     `json:"last_sync"`
	ServerURL  string                        `json:"server_url"`
	Templates  map[string]*CacheTemplateInfo `json:"templates"`
	Statistics CacheStatistics               `json:"statistics"`
}

// CacheTemplateInfo represents template information in the cache
type CacheTemplateInfo struct {
	ID            string    `json:"id"`
	Version       string    `json:"version"`
	Checksum      string    `json:"checksum"`
	Size          int64     `json:"size"`
	Severity      string    `json:"severity"`
	Platforms     []string  `json:"platforms"`
	DetectionType string    `json:"detection_type"`
	Author        string    `json:"author"`
	Created       time.Time `json:"created"`
	Updated       time.Time `json:"updated"`
	FilePath      string    `json:"file_path"`
	IsCustom      bool      `json:"is_custom"`
}

// CacheStatistics contains cache statistics
type CacheStatistics struct {
	TotalTemplates    int           `json:"total_templates"`
	StandardTemplates int           `json:"standard_templates"`
	CustomTemplates   int           `json:"custom_templates"`
	LastSyncDuration  time.Duration `json:"last_sync_duration"`
	CacheSize         int64         `json:"cache_size"`
}

// NewAgentSyncManager creates a new agent sync manager
func NewAgentSyncManager(logger *zap.Logger, serverURL string, agentID string) (*AgentSyncManager, error) {
	cacheDir := GetAgentTemplateCacheDir()

	// Ensure cache directory structure exists
	if err := EnsureCacheDirectoryStructure(); err != nil {
		return nil, fmt.Errorf("failed to create cache directory structure: %w", err)
	}

	return &AgentSyncManager{
		cacheDir:  cacheDir,
		logger:    logger,
		serverURL: serverURL,
		agentID:   agentID,
	}, nil
}

// SetGRPCStream sets the gRPC stream reference for communication with server
func (asm *AgentSyncManager) SetGRPCStream(stream pb.HelloService_ConnectStreamClient) {
	asm.grpcStream = stream
	asm.logger.Info("gRPC stream reference set for template sync manager")
}

// SetStreamSendFunc sets a mutex-protected send function for the gRPC stream.
// When set, this is used instead of grpcStream.Send() to avoid concurrent send races.
func (asm *AgentSyncManager) SetStreamSendFunc(fn StreamSendFunc) {
	asm.streamSendFunc = fn
	asm.logger.Info("Stream send function set for template sync manager")
}

// sendMessage sends an AgentMessage using the thread-safe send function if available,
// falling back to the raw stream Send() otherwise.
func (asm *AgentSyncManager) sendMessage(msg *pb.AgentMessage) error {
	if asm.streamSendFunc != nil {
		return asm.streamSendFunc(msg)
	}
	if asm.grpcStream == nil {
		return fmt.Errorf("gRPC stream not set; cannot send message")
	}
	return asm.grpcStream.Send(msg)
}

// SyncFromServer sends a template sync request to the server via gRPC stream
func (asm *AgentSyncManager) SyncFromServer(ctx context.Context) error {
	asm.logger.Info("Starting template sync from server via gRPC",
		zap.String("server_url", asm.serverURL))

	if asm.grpcStream == nil && asm.streamSendFunc == nil {
		return fmt.Errorf("gRPC stream not set; cannot sync templates")
	}

	// Load local manifest to get last sync time
	localManifest, err := asm.loadCacheManifest()
	if err != nil {
		asm.logger.Warn("Failed to load local cache manifest, using zero timestamp", zap.Error(err))
		localManifest = &CacheManifest{
			Version:    "1.0.0",
			Templates:  make(map[string]*CacheTemplateInfo),
			Statistics: CacheStatistics{},
		}
	}

	// Create sync request
	syncRequest := &pb.TemplateSyncRequest{
		AgentId:  asm.agentID,
		LastSync: localManifest.LastSync.Unix(),
	}

	// Send sync request via gRPC stream (uses mutex-protected send if available)
	msg := &pb.AgentMessage{
		AgentId: asm.agentID,
		Type:    pb.MessageType_TEMPLATE_SYNC_REQUEST,
		Payload: &pb.AgentMessage_SyncRequest{
			SyncRequest: syncRequest,
		},
	}

	if err := asm.sendMessage(msg); err != nil {
		return fmt.Errorf("failed to send template sync request: %w", err)
	}

	asm.logger.Info("Template sync request sent to server, waiting for response",
		zap.Int64("last_sync", syncRequest.LastSync))

	// Note: Template updates will be received asynchronously via HandleTemplateUpdate
	// called from the agent's message processing loop

	return nil
}

// HandleManifestUpdate processes manifest received from server
func (asm *AgentSyncManager) HandleManifestUpdate(ctx context.Context, manifest *pb.TemplateManifest) error {
	asm.logger.Info("Received template manifest from server",
		zap.String("version", manifest.Version),
		zap.Int("templates", len(manifest.Templates)))

	// Update local manifest with server manifest data
	localManifest, err := asm.loadCacheManifest()
	if err != nil {
		localManifest = &CacheManifest{
			Version:    manifest.Version,
			Templates:  make(map[string]*CacheTemplateInfo),
			Statistics: CacheStatistics{},
		}
	}

	// Update version and server info
	localManifest.Version = manifest.Version
	localManifest.ServerURL = asm.serverURL
	localManifest.Statistics.TotalTemplates = int(manifest.Statistics.TotalTemplates)
	localManifest.Statistics.StandardTemplates = int(manifest.Statistics.StandardTemplates)
	localManifest.Statistics.CustomTemplates = int(manifest.Statistics.CustomTemplates)

	// Save updated manifest
	if err := asm.saveCacheManifest(localManifest); err != nil {
		asm.logger.Error("Failed to save cache manifest", zap.Error(err))
		return fmt.Errorf("failed to save cache manifest: %w", err)
	}

	asm.logger.Info("Template manifest processed and saved",
		zap.Int("total_templates", localManifest.Statistics.TotalTemplates))

	return nil
}

// HandleTemplateUpdate processes a template update received from the server
func (asm *AgentSyncManager) HandleTemplateUpdate(ctx context.Context, update *pb.TemplateUpdate) error {
	// Check if this is a manifest-only message
	if update.Manifest != nil && update.TemplateId == "" {
		asm.logger.Info("Received manifest-only update, processing manifest")
		return asm.HandleManifestUpdate(ctx, update.Manifest)
	}

	templateID := update.TemplateId
	asm.logger.Info("Received template update from server",
		zap.String("id", templateID),
		zap.String("version", update.Version),
		zap.Bool("is_custom", update.IsCustom))

	// Determine cache path
	var cachePath string
	if update.IsCustom {
		cachePath = filepath.Join(GetCustomTemplatesPath(), templateID+".yaml")
	} else {
		cachePath = filepath.Join(GetStandardTemplatesPath(), templateID+".yaml")
	}

	// Write template file atomically
	tempPath := cachePath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(update.Content), 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	if err := os.Rename(tempPath, cachePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename template file: %w", err)
	}

	// Verify checksum
	hash := sha256.Sum256([]byte(update.Content))
	actualChecksum := "sha256:" + hex.EncodeToString(hash[:])

	if actualChecksum != update.Checksum {
		asm.logger.Warn("Template checksum mismatch",
			zap.String("id", templateID),
			zap.String("expected", update.Checksum),
			zap.String("actual", actualChecksum))
	}

	// Update local manifest
	localManifest, err := asm.loadCacheManifest()
	if err != nil {
		asm.logger.Warn("Failed to load cache manifest, creating new one", zap.Error(err))
		localManifest = &CacheManifest{
			Version:    "1.0.0",
			Templates:  make(map[string]*CacheTemplateInfo),
			Statistics: CacheStatistics{},
		}
	}

	// Add/update template info in manifest
	localManifest.Templates[templateID] = &CacheTemplateInfo{
		ID:       templateID,
		Version:  update.Version,
		Checksum: update.Checksum,
		Size:     int64(len(update.Content)),
		FilePath: cachePath,
		IsCustom: update.IsCustom,
		Updated:  time.Now(),
	}

	// Update statistics
	localManifest.Statistics.TotalTemplates = len(localManifest.Templates)
	localManifest.Statistics.StandardTemplates = 0
	localManifest.Statistics.CustomTemplates = 0
	for _, template := range localManifest.Templates {
		if template.IsCustom {
			localManifest.Statistics.CustomTemplates++
		} else {
			localManifest.Statistics.StandardTemplates++
		}
	}

	// Update last sync time
	localManifest.LastSync = time.Now()

	// Save updated manifest
	if err := asm.saveCacheManifest(localManifest); err != nil {
		asm.logger.Error("Failed to save cache manifest", zap.Error(err))
		return fmt.Errorf("failed to save cache manifest: %w", err)
	}

	asm.logger.Info("Template update processed successfully",
		zap.String("id", templateID),
		zap.String("path", cachePath),
		zap.Int64("size", int64(len(update.Content))))

	return nil
}

// ValidateCache verifies checksums of cached templates
func (asm *AgentSyncManager) ValidateCache(ctx context.Context) error {
	asm.logger.Info("Validating template cache")

	manifest, err := asm.loadCacheManifest()
	if err != nil {
		return fmt.Errorf("failed to load cache manifest: %w", err)
	}

	var validCount int
	var invalidCount int

	for templateID, templateInfo := range manifest.Templates {
		// Read template file
		content, err := os.ReadFile(templateInfo.FilePath)
		if err != nil {
			asm.logger.Error("Failed to read template file",
				zap.String("id", templateID),
				zap.String("path", templateInfo.FilePath),
				zap.Error(err))
			invalidCount++
			continue
		}

		// Calculate checksum
		hash := sha256.Sum256(content)
		actualChecksum := "sha256:" + hex.EncodeToString(hash[:])

		// Compare checksums
		if actualChecksum != templateInfo.Checksum {
			asm.logger.Error("Template checksum mismatch",
				zap.String("id", templateID),
				zap.String("expected", templateInfo.Checksum),
				zap.String("actual", actualChecksum))
			invalidCount++
		} else {
			validCount++
		}
	}

	asm.logger.Info("Cache validation completed",
		zap.Int("valid", validCount),
		zap.Int("invalid", invalidCount))

	if invalidCount > 0 {
		return fmt.Errorf("cache validation failed: %d invalid templates", invalidCount)
	}

	return nil
}

// LoadTemplates loads cached templates for scanning
func (asm *AgentSyncManager) LoadTemplates(ctx context.Context) ([]*types.Template, error) {
	asm.logger.Info("Loading cached templates")

	manifest, err := asm.loadCacheManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load cache manifest: %w", err)
	}

	var templates []*types.Template
	var errorCount int

	for templateID, templateInfo := range manifest.Templates {
		// Load template using parser
		template, err := parser.ParseTemplate(templateInfo.FilePath)
		if err != nil {
			asm.logger.Error("Failed to parse template",
				zap.String("id", templateID),
				zap.String("path", templateInfo.FilePath),
				zap.Error(err))
			errorCount++
			continue
		}

		templates = append(templates, template)
	}

	asm.logger.Info("Template loading completed",
		zap.Int("loaded", len(templates)),
		zap.Int("errors", errorCount))

	return templates, nil
}

// ClearCache removes all cached templates
func (asm *AgentSyncManager) ClearCache() error {
	asm.logger.Info("Clearing template cache")

	// Remove cache directory
	if err := os.RemoveAll(asm.cacheDir); err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Recreate directory structure
	if err := EnsureCacheDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to recreate cache directory structure: %w", err)
	}

	asm.logger.Info("Template cache cleared successfully")
	return nil
}

// GetCacheStatus returns the current cache status
func (asm *AgentSyncManager) GetCacheStatus() (*CacheManifest, error) {
	return asm.loadCacheManifest()
}

// loadCacheManifest loads the local cache manifest
func (asm *AgentSyncManager) loadCacheManifest() (*CacheManifest, error) {
	manifestPath := GetCacheManifestPath()

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &CacheManifest{
				Version:    "1.0.0",
				Templates:  make(map[string]*CacheTemplateInfo),
				Statistics: CacheStatistics{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read cache manifest: %w", err)
	}

	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse cache manifest: %w", err)
	}

	return &manifest, nil
}

// saveCacheManifest saves the local cache manifest
func (asm *AgentSyncManager) saveCacheManifest(manifest *CacheManifest) error {
	manifestPath := GetCacheManifestPath()

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache manifest: %w", err)
	}

	// Write manifest file atomically
	tempPath := manifestPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache manifest: %w", err)
	}

	if err := os.Rename(tempPath, manifestPath); err != nil {
		return fmt.Errorf("failed to rename cache manifest: %w", err)
	}

	return nil
}

// calculateCacheSize calculates the total size of the cache directory
func (asm *AgentSyncManager) calculateCacheSize() (int64, error) {
	var totalSize int64

	err := filepath.Walk(asm.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}
