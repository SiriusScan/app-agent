package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// GitHubRepositoryManager implements RepositoryManager for GitHub repositories
type GitHubRepositoryManager struct {
	logger     *zap.Logger
	config     *RepositoryConfiguration
	manifest   *Manifest
	localPath  string
	httpClient *http.Client
}

// NewGitHubRepositoryManager creates a new GitHub repository manager
func NewGitHubRepositoryManager(logger *zap.Logger) *GitHubRepositoryManager {
	return &GitHubRepositoryManager{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Increased from 30 to 60 seconds
		},
	}
}

// Initialize sets up the repository manager
func (m *GitHubRepositoryManager) Initialize(ctx context.Context) error {
	if m.config == nil {
		return fmt.Errorf("configuration not set")
	}

	m.localPath = m.config.LocalPath

	// Create local directory if it doesn't exist
	if err := os.MkdirAll(m.localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Load existing manifest if available
	manifestPath := filepath.Join(m.localPath, "repository-manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		if _, err := m.loadManifest(); err != nil {
			m.logger.Warn("Failed to load existing manifest", zap.Error(err))
		}
	} else {
		// Fallback to legacy manifest.json
		manifestPath = filepath.Join(m.localPath, "manifest.json")
		if _, err := os.Stat(manifestPath); err == nil {
			if _, err := m.loadManifest(); err != nil {
				m.logger.Warn("Failed to load existing manifest", zap.Error(err))
			}
		}
	}

	return nil
}

// UpdateRepository synchronizes with remote repository
func (m *GitHubRepositoryManager) UpdateRepository(ctx context.Context) (*UpdateResult, error) {
	m.logger.Info("Starting repository update",
		zap.String("url", m.config.RemoteURL))

	result := &UpdateResult{
		StartedAt:  time.Now(),
		UpdateType: m.config.UpdateStrategy,
	}

	// Check for updates
	updateCheck, err := m.checkForUpdates(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("update check failed: %v", err))
		return result, err
	}

	if !updateCheck.UpdatesAvailable {
		m.logger.Info("No updates available")
		result.Success = true
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result, nil
	}

	// Download updates
	updateData, err := m.downloadUpdates(ctx, updateCheck)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("download failed: %v", err))
		return result, err
	}

	// Apply updates
	if err := m.applyUpdates(ctx, updateData); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("apply failed: %v", err))
		return result, err
	}

	result.Success = true
	result.PreviousVersion = updateCheck.CurrentVersion
	result.NewVersion = updateCheck.LatestVersion
	result.FilesAdded = make([]string, 0, len(updateCheck.FilesToAdd))
	for _, file := range updateCheck.FilesToAdd {
		result.FilesAdded = append(result.FilesAdded, file.Path)
	}
	result.FilesUpdated = make([]string, 0, len(updateCheck.FilesToUpdate))
	for _, file := range updateCheck.FilesToUpdate {
		result.FilesUpdated = append(result.FilesUpdated, file.Path)
	}
	result.FilesRemoved = updateCheck.FilesToRemove
	result.BytesDownloaded = updateCheck.TotalDownloadSize
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	m.logger.Info("Repository update completed",
		zap.Int("files_added", len(result.FilesAdded)),
		zap.Int("files_updated", len(result.FilesUpdated)),
		zap.Int("files_removed", len(result.FilesRemoved)))

	return result, nil
}

// LoadManifest loads and validates the repository manifest
func (m *GitHubRepositoryManager) LoadManifest() (*Manifest, error) {
	if m.manifest != nil {
		return m.manifest, nil
	}

	// Try repository-manifest.json first (our format)
	manifestPath := filepath.Join(m.localPath, "repository-manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return m.loadManifest()
	}

	// Fallback to manifest.json (legacy format)
	manifestPath = filepath.Join(m.localPath, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, fmt.Errorf("manifest not found: %w", err)
	}

	return m.loadManifest()
}

// SaveManifest saves the repository manifest
func (m *GitHubRepositoryManager) SaveManifest(manifest *Manifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest is nil")
	}

	m.manifest = manifest
	return m.saveManifest()
}

// GetRepositoryInfo returns current repository information
func (m *GitHubRepositoryManager) GetRepositoryInfo() (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		LocalPath:     m.localPath,
		RemoteURL:     m.config.RemoteURL,
		Configuration: m.config,
		Status:        RepositoryStatusHealthy,
	}

	if m.manifest != nil {
		info.CurrentVersion = m.manifest.Version
		info.LastUpdate = m.manifest.Updated
		info.TemplateCount = len(m.manifest.Templates)
		info.ScriptCount = len(m.manifest.Scripts)

		// Calculate total size
		for _, template := range m.manifest.Templates {
			info.TotalSize += template.Size
		}
		for _, script := range m.manifest.Scripts {
			info.TotalSize += script.Size
		}
	}

	return info, nil
}

// ValidateRepository checks repository integrity
func (m *GitHubRepositoryManager) ValidateRepository(ctx context.Context) (*ValidationResult, error) {
	result := &ValidationResult{
		ValidatedAt:    time.Now(),
		ValidationType: ValidationTypeStructure,
		Valid:          true,
	}

	if m.manifest == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:     "manifest_missing",
			Message:  "Repository manifest not found",
			Severity: ErrorSeverityCritical,
		})
		return result, nil
	}

	// Validate all files in manifest
	for path, fileInfo := range m.manifest.Templates {
		if err := m.validateFile(path, fileInfo); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:     "template_validation",
				Message:  err.Error(),
				Location: path,
				Severity: ErrorSeverityHigh,
			})
		}
	}

	for path, fileInfo := range m.manifest.Scripts {
		if err := m.validateFile(path, fileInfo); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:     "script_validation",
				Message:  err.Error(),
				Location: path,
				Severity: ErrorSeverityHigh,
			})
		}
	}

	return result, nil
}

// checkForUpdates checks if updates are available
func (m *GitHubRepositoryManager) checkForUpdates(ctx context.Context) (*UpdateCheck, error) {
	// Download latest manifest
	remoteManifest, err := m.downloadManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to download remote manifest: %w", err)
	}

	check := &UpdateCheck{
		CheckedAt: time.Now(),
	}

	if m.manifest == nil {
		check.UpdatesAvailable = true
		check.LatestVersion = remoteManifest.Version
		check.CurrentVersion = "none"

		// All files are new
		for _, fileInfo := range remoteManifest.Templates {
			check.FilesToAdd = append(check.FilesToAdd, fileInfo)
			check.TotalDownloadSize += fileInfo.Size
		}
		for _, fileInfo := range remoteManifest.Scripts {
			check.FilesToAdd = append(check.FilesToAdd, fileInfo)
			check.TotalDownloadSize += fileInfo.Size
		}
		return check, nil
	}

	check.CurrentVersion = m.manifest.Version
	check.LatestVersion = remoteManifest.Version

	// Compare manifests
	if remoteManifest.Version != m.manifest.Version {
		check.UpdatesAvailable = true

		// Find new and updated files
		for path, remoteFile := range remoteManifest.Templates {
			if localFile, exists := m.manifest.Templates[path]; !exists {
				check.FilesToAdd = append(check.FilesToAdd, remoteFile)
				check.TotalDownloadSize += remoteFile.Size
			} else if localFile.Checksum != remoteFile.Checksum {
				check.FilesToUpdate = append(check.FilesToUpdate, &FileUpdateInfo{
					Path:            path,
					CurrentVersion:  localFile.Version,
					NewVersion:      remoteFile.Version,
					CurrentChecksum: localFile.Checksum,
					NewChecksum:     remoteFile.Checksum,
					Size:            remoteFile.Size,
					ChangeType:      FileChangeTypeModified,
				})
				check.TotalDownloadSize += remoteFile.Size
			}
		}

		for scriptPath, remoteFile := range remoteManifest.Scripts {
			if localFile, exists := m.manifest.Scripts[scriptPath]; !exists {
				check.FilesToAdd = append(check.FilesToAdd, remoteFile)
				check.TotalDownloadSize += remoteFile.Size
			} else if localFile.Checksum != remoteFile.Checksum {
				check.FilesToUpdate = append(check.FilesToUpdate, &FileUpdateInfo{
					Path:            scriptPath,
					CurrentVersion:  localFile.Version,
					NewVersion:      remoteFile.Version,
					CurrentChecksum: localFile.Checksum,
					NewChecksum:     remoteFile.Checksum,
					Size:            remoteFile.Size,
					ChangeType:      FileChangeTypeModified,
				})
				check.TotalDownloadSize += remoteFile.Size
			}
		}

		// Find removed files
		for path := range m.manifest.Templates {
			if _, exists := remoteManifest.Templates[path]; !exists {
				check.FilesToRemove = append(check.FilesToRemove, path)
			}
		}
		for path := range m.manifest.Scripts {
			if _, exists := remoteManifest.Scripts[path]; !exists {
				check.FilesToRemove = append(check.FilesToRemove, path)
			}
		}
	}

	return check, nil
}

// downloadUpdates downloads available updates
func (m *GitHubRepositoryManager) downloadUpdates(ctx context.Context, check *UpdateCheck) (*UpdateData, error) {
	// Download new manifest
	remoteManifest, err := m.downloadManifest(ctx)
	if err != nil {
		return nil, err
	}

	updateData := &UpdateData{
		Manifest:   remoteManifest,
		Files:      make(map[string][]byte),
		Signatures: make(map[string][]byte),
		UpdateType: m.config.UpdateStrategy,
	}

	// Download new and updated files
	for _, fileInfo := range check.FilesToAdd {
		content, err := m.downloadFile(ctx, fileInfo.Path)
		if err != nil {
			m.logger.Warn("Failed to download file, skipping",
				zap.String("path", fileInfo.Path),
				zap.Error(err))
			continue
		}
		updateData.Files[fileInfo.Path] = content
	}

	for _, fileUpdate := range check.FilesToUpdate {
		content, err := m.downloadFile(ctx, fileUpdate.Path)
		if err != nil {
			m.logger.Warn("Failed to download file, skipping",
				zap.String("path", fileUpdate.Path),
				zap.Error(err))
			continue
		}
		updateData.Files[fileUpdate.Path] = content
	}

	return updateData, nil
}

// applyUpdates applies downloaded updates
func (m *GitHubRepositoryManager) applyUpdates(ctx context.Context, updateData *UpdateData) error {
	// Create backup
	backupPath := filepath.Join(m.localPath, fmt.Sprintf("backup-%d", time.Now().Unix()))
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Backup current manifest
	if m.manifest != nil {
		backupManifestPath := filepath.Join(backupPath, "manifest.json")
		if err := m.saveManifestToPath(backupManifestPath); err != nil {
			return fmt.Errorf("failed to backup manifest: %w", err)
		}
	}

	// Save new files
	for path, content := range updateData.Files {
		filePath := filepath.Join(m.localPath, path)

		// Create directory if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}

		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return fmt.Errorf("failed to save file %s: %w", path, err)
		}
	}

	// Remove deleted files (only if we have an existing manifest)
	if m.manifest != nil {
		for path := range m.manifest.Templates {
			if _, exists := updateData.Manifest.Templates[path]; !exists {
				filePath := filepath.Join(m.localPath, path)
				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					m.logger.Warn("Failed to remove file", zap.String("path", path), zap.Error(err))
				}
			}
		}
		for path := range m.manifest.Scripts {
			if _, exists := updateData.Manifest.Scripts[path]; !exists {
				filePath := filepath.Join(m.localPath, path)
				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					m.logger.Warn("Failed to remove file", zap.String("path", path), zap.Error(err))
				}
			}
		}
	}

	// Update manifest
	m.manifest = updateData.Manifest
	return m.saveManifest()
}

// downloadManifest downloads the repository manifest
func (m *GitHubRepositoryManager) downloadManifest(ctx context.Context) (*Manifest, error) {
	// Convert GitHub URL to raw.githubusercontent.com URL
	rawURL := strings.Replace(m.config.RemoteURL, "https://github.com/", "https://raw.githubusercontent.com/", 1)
	manifestURL := fmt.Sprintf("%s/main/repository-manifest.json", rawURL)

	req, err := http.NewRequestWithContext(ctx, "GET", manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	// Parse the repository manifest structure
	var repoManifest struct {
		Version    string `json:"version"`
		Updated    string `json:"updated"`
		Components struct {
			Templates struct {
				Manifest string `json:"manifest"`
			} `json:"templates"`
			Scripts struct {
				Manifest string `json:"manifest"`
			} `json:"scripts"`
		} `json:"components"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoManifest); err != nil {
		return nil, fmt.Errorf("failed to decode repository manifest: %w", err)
	}

	// Download templates manifest
	templatesData, err := m.downloadFile(ctx, repoManifest.Components.Templates.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to download templates manifest: %w", err)
	}

	var templatesManifest struct {
		Templates map[string]*FileInfo `json:"templates"`
	}
	if err := json.Unmarshal(templatesData, &templatesManifest); err != nil {
		return nil, fmt.Errorf("failed to decode templates manifest: %w", err)
	}

	// Download scripts manifest
	scriptsData, err := m.downloadFile(ctx, repoManifest.Components.Scripts.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to download scripts manifest: %w", err)
	}

	var scriptsManifest struct {
		Scripts map[string]*FileInfo `json:"scripts"`
	}
	if err := json.Unmarshal(scriptsData, &scriptsManifest); err != nil {
		return nil, fmt.Errorf("failed to decode scripts manifest: %w", err)
	}

	// Parse the updated timestamp
	updatedTime, err := time.Parse(time.RFC3339, repoManifest.Updated)
	if err != nil {
		updatedTime = time.Now() // Fallback to current time
	}

	// Construct full paths for templates and scripts with proper size information
	fullTemplates := make(map[string]*FileInfo)
	for relativePath, fileInfo := range templatesManifest.Templates {
		fullPath := filepath.Join("templates", relativePath)
		fileInfo.Path = fullPath

		// Try to get actual file size by downloading a small portion
		if fileInfo.Size == 0 {
			// Download the file to get its actual size
			content, err := m.downloadFile(ctx, fullPath)
			if err != nil {
				m.logger.Warn("Failed to get file size, using 0",
					zap.String("path", fullPath),
					zap.Error(err))
				fileInfo.Size = 0
			} else {
				fileInfo.Size = int64(len(content))
				// Calculate checksum
				hash := sha256.Sum256(content)
				fileInfo.Checksum = hex.EncodeToString(hash[:])
				fileInfo.ChecksumAlgorithm = HashAlgorithmSHA256
			}
		}

		fullTemplates[fullPath] = fileInfo
	}

	fullScripts := make(map[string]*FileInfo)
	for relativePath, fileInfo := range scriptsManifest.Scripts {
		fullPath := filepath.Join("scripts", relativePath)
		fileInfo.Path = fullPath

		// Try to get actual file size by downloading a small portion
		if fileInfo.Size == 0 {
			// Download the file to get its actual size
			content, err := m.downloadFile(ctx, fullPath)
			if err != nil {
				m.logger.Warn("Failed to get file size, using 0",
					zap.String("path", fullPath),
					zap.Error(err))
				fileInfo.Size = 0
			} else {
				fileInfo.Size = int64(len(content))
				// Calculate checksum
				hash := sha256.Sum256(content)
				fileInfo.Checksum = hex.EncodeToString(hash[:])
				fileInfo.ChecksumAlgorithm = HashAlgorithmSHA256
			}
		}

		fullScripts[fullPath] = fileInfo
	}

	// Combine into our expected manifest structure
	manifest := &Manifest{
		Version:   repoManifest.Version,
		Updated:   updatedTime,
		Templates: fullTemplates,
		Scripts:   fullScripts,
	}

	return manifest, nil
}

// downloadFile downloads a specific file from the repository
func (m *GitHubRepositoryManager) downloadFile(ctx context.Context, path string) ([]byte, error) {
	// Convert GitHub URL to raw.githubusercontent.com URL
	rawURL := strings.Replace(m.config.RemoteURL, "https://github.com/", "https://raw.githubusercontent.com/", 1)
	fileURL := fmt.Sprintf("%s/main/%s", rawURL, path)

	m.logger.Debug("Downloading file", zap.String("url", fileURL), zap.String("path", path))

	req, err := http.NewRequestWithContext(ctx, "GET", fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// loadManifest loads the local manifest file
func (m *GitHubRepositoryManager) loadManifest() (*Manifest, error) {
	// Try repository-manifest.json first (our format)
	manifestPath := filepath.Join(m.localPath, "repository-manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err == nil {
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to decode repository manifest: %w", err)
		}
		return &manifest, nil
	}

	// Fallback to manifest.json (legacy format)
	manifestPath = filepath.Join(m.localPath, "manifest.json")
	data, err = os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	return &manifest, nil
}

// saveManifest saves the manifest to local storage
func (m *GitHubRepositoryManager) saveManifest() error {
	return m.saveManifestToPath(filepath.Join(m.localPath, "manifest.json"))
}

// saveManifestToPath saves the manifest to a specific path
func (m *GitHubRepositoryManager) saveManifestToPath(path string) error {
	if m.manifest == nil {
		return fmt.Errorf("no manifest to save")
	}

	data, err := json.MarshalIndent(m.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// validateFile validates a file's integrity
func (m *GitHubRepositoryManager) validateFile(path string, fileInfo *FileInfo) error {
	filePath := filepath.Join(m.localPath, path)

	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Validate checksum
	actualChecksum, err := m.calculateFileChecksum(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if actualChecksum != fileInfo.Checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", fileInfo.Checksum, actualChecksum)
	}

	return nil
}

// calculateFileChecksum calculates SHA256 checksum of a file
func (m *GitHubRepositoryManager) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SetConfiguration sets the repository configuration
func (m *GitHubRepositoryManager) SetConfiguration(config *RepositoryConfiguration) {
	m.config = config
}
