package repository

import (
	"context"
	"time"
)

// RepositoryManager manages vulnerability detection templates and scripts
type RepositoryManager interface {
	// Initialize sets up the repository structure
	Initialize(ctx context.Context) error

	// UpdateRepository synchronizes with remote repository
	UpdateRepository(ctx context.Context) (*UpdateResult, error)

	// LoadManifest loads and validates the repository manifest
	LoadManifest() (*Manifest, error)

	// SaveManifest saves the repository manifest
	SaveManifest(manifest *Manifest) error

	// GetRepositoryInfo returns current repository information
	GetRepositoryInfo() (*RepositoryInfo, error)

	// ValidateRepository checks repository integrity
	ValidateRepository(ctx context.Context) (*ValidationResult, error)
}

// TemplateManager manages vulnerability detection templates
type TemplateManager interface {
	// LoadTemplates loads all templates from repository
	LoadTemplates(ctx context.Context) ([]*TemplateInfo, error)

	// LoadTemplate loads a specific template by ID
	LoadTemplate(templateID string) (*TemplateInfo, error)

	// ValidateTemplate validates template format and content
	ValidateTemplate(templatePath string) (*ValidationResult, error)

	// ListTemplates returns list of available templates
	ListTemplates() ([]*TemplateInfo, error)

	// GetTemplateContent reads template content
	GetTemplateContent(templateID string) ([]byte, error)
}

// ScriptManager manages vulnerability detection scripts
type ScriptManager interface {
	// LoadScripts loads all scripts from repository
	LoadScripts(ctx context.Context) ([]*ScriptInfo, error)

	// LoadScript loads a specific script by name
	LoadScript(scriptName string) (*ScriptInfo, error)

	// ValidateScript validates script format and security
	ValidateScript(scriptPath string) (*ValidationResult, error)

	// ListScripts returns list of available scripts
	ListScripts() ([]*ScriptInfo, error)

	// GetScriptContent reads script content
	GetScriptContent(scriptName string) ([]byte, error)
}

// UpdateManager handles repository updates and synchronization
type UpdateManager interface {
	// CheckForUpdates checks if updates are available
	CheckForUpdates(ctx context.Context) (*UpdateCheck, error)

	// DownloadUpdates downloads available updates
	DownloadUpdates(ctx context.Context, updates *UpdateCheck) (*UpdateResult, error)

	// ApplyUpdates applies downloaded updates atomically
	ApplyUpdates(ctx context.Context, updateData *UpdateData) error

	// RollbackUpdate rolls back to previous version
	RollbackUpdate(ctx context.Context) error

	// ScheduleUpdates configures automatic update scheduling
	ScheduleUpdates(interval time.Duration) error

	// GetUpdateHistory returns update history
	GetUpdateHistory() ([]*UpdateHistoryEntry, error)
}

// SignatureValidator handles cryptographic signature verification
type SignatureValidator interface {
	// ValidateSignature verifies GPG signature of content
	ValidateSignature(content []byte, signature []byte) (*SignatureResult, error)

	// ValidateManifestSignature verifies manifest signature
	ValidateManifestSignature(manifest *Manifest) (*SignatureResult, error)

	// ValidateFileSignature verifies file signature using manifest
	ValidateFileSignature(filePath string, expectedSignature string) (*SignatureResult, error)

	// LoadPublicKey loads public key for verification
	LoadPublicKey(keyData []byte) error

	// GetPublicKeyInfo returns information about loaded key
	GetPublicKeyInfo() (*PublicKeyInfo, error)
}

// ContentValidator validates repository content integrity
type ContentValidator interface {
	// ValidateChecksum verifies file checksum
	ValidateChecksum(filePath string, expectedChecksum string, algorithm HashAlgorithm) (*ChecksumResult, error)

	// ValidateContent validates template/script content
	ValidateContent(contentType ContentType, content []byte) (*ValidationResult, error)

	// ValidateStructure validates repository directory structure
	ValidateStructure(repositoryPath string) (*ValidationResult, error)

	// ScanForMalware scans content for potential security issues
	ScanForMalware(content []byte, contentType ContentType) (*SecurityScanResult, error)
}

// RemoteRepository handles remote repository communication
type RemoteRepository interface {
	// GetManifest downloads remote manifest
	GetManifest(ctx context.Context) (*Manifest, error)

	// DownloadFile downloads a specific file
	DownloadFile(ctx context.Context, fileInfo *FileInfo) ([]byte, error)

	// ListFiles lists available files in remote repository
	ListFiles(ctx context.Context) ([]*FileInfo, error)

	// GetRepositoryMetadata gets remote repository metadata
	GetRepositoryMetadata(ctx context.Context) (*RepositoryMetadata, error)

	// TestConnection tests connectivity to remote repository
	TestConnection(ctx context.Context) error
}

// CacheManager handles local repository caching
type CacheManager interface {
	// GetCachedFile retrieves file from cache
	GetCachedFile(fileID string) ([]byte, error)

	// StoreCachedFile stores file in cache
	StoreCachedFile(fileID string, content []byte) error

	// InvalidateCache invalidates cached content
	InvalidateCache(fileID string) error

	// ClearCache clears all cached content
	ClearCache() error

	// GetCacheInfo returns cache statistics
	GetCacheInfo() (*CacheInfo, error)
}

// ContentType represents the type of repository content
type ContentType string

const (
	ContentTypeTemplate      ContentType = "template"
	ContentTypeScript        ContentType = "script"
	ContentTypeManifest      ContentType = "manifest"
	ContentTypeSignature     ContentType = "signature"
	ContentTypeDocumentation ContentType = "documentation"
)

// HashAlgorithm represents supported hash algorithms
type HashAlgorithm string

const (
	HashAlgorithmSHA256 HashAlgorithm = "sha256"
	HashAlgorithmSHA1   HashAlgorithm = "sha1"
	HashAlgorithmMD5    HashAlgorithm = "md5"
	HashAlgorithmSHA512 HashAlgorithm = "sha512"
)

// UpdateStrategy represents different update strategies
type UpdateStrategy string

const (
	UpdateStrategyIncremental UpdateStrategy = "incremental"
	UpdateStrategyFull        UpdateStrategy = "full"
	UpdateStrategyDelta       UpdateStrategy = "delta"
)

// RepositoryConfiguration contains repository settings
type RepositoryConfiguration struct {
	// RemoteURL remote repository URL
	RemoteURL string `json:"remote_url"`

	// LocalPath local repository path
	LocalPath string `json:"local_path"`

	// UpdateInterval automatic update interval
	UpdateInterval time.Duration `json:"update_interval"`

	// UpdateStrategy update strategy to use
	UpdateStrategy UpdateStrategy `json:"update_strategy"`

	// VerifySignatures whether to verify GPG signatures
	VerifySignatures bool `json:"verify_signatures"`

	// PublicKeyPath path to public key file
	PublicKeyPath string `json:"public_key_path,omitempty"`

	// CacheEnabled whether to enable local caching
	CacheEnabled bool `json:"cache_enabled"`

	// CacheSize maximum cache size in MB
	CacheSize int64 `json:"cache_size_mb"`

	// Timeout network operation timeout
	Timeout time.Duration `json:"timeout"`

	// RetryAttempts number of retry attempts
	RetryAttempts int `json:"retry_attempts"`

	// ProxyURL proxy server URL
	ProxyURL string `json:"proxy_url,omitempty"`

	// UserAgent HTTP user agent
	UserAgent string `json:"user_agent"`

	// Headers additional HTTP headers
	Headers map[string]string `json:"headers,omitempty"`
}
