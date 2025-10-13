package repository

import (
	"time"
)

// Manifest represents the repository manifest file
type Manifest struct {
	// Version manifest version
	Version string `json:"version"`

	// Updated when manifest was last updated
	Updated time.Time `json:"updated"`

	// Templates template file information
	Templates map[string]*FileInfo `json:"templates"`

	// Scripts script file information
	Scripts map[string]*FileInfo `json:"scripts"`

	// Signatures digital signatures for files
	Signatures map[string]string `json:"signatures,omitempty"`

	// Metadata additional manifest metadata
	Metadata *ManifestMetadata `json:"metadata,omitempty"`
}

// ManifestMetadata contains additional manifest information
type ManifestMetadata struct {
	// Publisher repository publisher
	Publisher string `json:"publisher,omitempty"`

	// Description repository description
	Description string `json:"description,omitempty"`

	// License repository license
	License string `json:"license,omitempty"`

	// URL repository URL
	URL string `json:"url,omitempty"`

	// MinAgentVersion minimum agent version required
	MinAgentVersion string `json:"min_agent_version,omitempty"`

	// Tags repository tags
	Tags []string `json:"tags,omitempty"`
}

// FileInfo contains information about a repository file
type FileInfo struct {
	// Path file path relative to repository root
	Path string `json:"path"`

	// Version file version
	Version string `json:"version"`

	// Checksum file checksum
	Checksum string `json:"checksum"`

	// ChecksumAlgorithm checksum algorithm used
	ChecksumAlgorithm HashAlgorithm `json:"checksum_algorithm"`

	// Size file size in bytes
	Size int64 `json:"size"`

	// Updated when file was last updated
	Updated time.Time `json:"updated"`

	// VulnerabilityIDs associated vulnerability IDs
	VulnerabilityIDs []string `json:"vulnerability_ids,omitempty"`

	// Platforms supported platforms
	Platforms []string `json:"platforms,omitempty"`

	// Dependencies file dependencies
	Dependencies []string `json:"dependencies,omitempty"`

	// Description file description
	Description string `json:"description,omitempty"`

	// Author file author
	Author string `json:"author,omitempty"`

	// Tags file tags
	Tags []string `json:"tags,omitempty"`
}

// TemplateInfo contains template-specific information
type TemplateInfo struct {
	// FileInfo base file information
	*FileInfo

	// TemplateID unique template identifier
	TemplateID string `json:"template_id"`

	// Severity vulnerability severity
	Severity string `json:"severity"`

	// CVE associated CVE
	CVE string `json:"cve,omitempty"`

	// DetectionType type of detection (file-hash, registry, config-file)
	DetectionType string `json:"detection_type"`

	// Confidence detection confidence level
	Confidence float64 `json:"confidence"`

	// Content template content (loaded on demand)
	Content []byte `json:"-"`

	// LoadedAt when template was loaded into memory
	LoadedAt time.Time `json:"loaded_at,omitempty"`
}

// ScriptInfo contains script-specific information
type ScriptInfo struct {
	// FileInfo base file information
	*FileInfo

	// ScriptName script name
	ScriptName string `json:"script_name"`

	// Language script language
	Language string `json:"language"`

	// VulnerabilityID associated vulnerability
	VulnerabilityID string `json:"vulnerability_id"`

	// Severity vulnerability severity
	Severity string `json:"severity"`

	// Timeout execution timeout
	Timeout time.Duration `json:"timeout"`

	// RequiredPrivileges required privileges
	RequiredPrivileges []string `json:"required_privileges,omitempty"`

	// Parameters script parameters
	Parameters []ScriptParameter `json:"parameters,omitempty"`

	// Content script content (loaded on demand)
	Content []byte `json:"-"`

	// LoadedAt when script was loaded into memory
	LoadedAt time.Time `json:"loaded_at,omitempty"`
}

// ScriptParameter represents a script parameter
type ScriptParameter struct {
	// Name parameter name
	Name string `json:"name"`

	// Type parameter type
	Type string `json:"type"`

	// Required whether parameter is required
	Required bool `json:"required"`

	// Default default value
	Default interface{} `json:"default,omitempty"`

	// Description parameter description
	Description string `json:"description,omitempty"`

	// Validation validation rules
	Validation *ParameterValidation `json:"validation,omitempty"`
}

// ParameterValidation contains parameter validation rules
type ParameterValidation struct {
	// Pattern regex pattern for validation
	Pattern string `json:"pattern,omitempty"`

	// MinLength minimum string length
	MinLength int `json:"min_length,omitempty"`

	// MaxLength maximum string length
	MaxLength int `json:"max_length,omitempty"`

	// MinValue minimum numeric value
	MinValue float64 `json:"min_value,omitempty"`

	// MaxValue maximum numeric value
	MaxValue float64 `json:"max_value,omitempty"`

	// AllowedValues list of allowed values
	AllowedValues []interface{} `json:"allowed_values,omitempty"`
}

// RepositoryInfo contains current repository information
type RepositoryInfo struct {
	// LocalPath local repository path
	LocalPath string `json:"local_path"`

	// RemoteURL remote repository URL
	RemoteURL string `json:"remote_url,omitempty"`

	// CurrentVersion current repository version
	CurrentVersion string `json:"current_version"`

	// LastUpdate when repository was last updated
	LastUpdate time.Time `json:"last_update"`

	// TemplateCount number of templates
	TemplateCount int `json:"template_count"`

	// ScriptCount number of scripts
	ScriptCount int `json:"script_count"`

	// TotalSize total repository size in bytes
	TotalSize int64 `json:"total_size"`

	// Configuration repository configuration
	Configuration *RepositoryConfiguration `json:"configuration"`

	// Status repository status
	Status RepositoryStatus `json:"status"`
}

// RepositoryStatus represents repository status
type RepositoryStatus string

const (
	RepositoryStatusHealthy   RepositoryStatus = "healthy"
	RepositoryStatusOutdated  RepositoryStatus = "outdated"
	RepositoryStatusCorrupted RepositoryStatus = "corrupted"
	RepositoryStatusUpdating  RepositoryStatus = "updating"
	RepositoryStatusError     RepositoryStatus = "error"
	RepositoryStatusNotFound  RepositoryStatus = "not_found"
)

// UpdateResult contains the result of a repository update
type UpdateResult struct {
	// Success whether update was successful
	Success bool `json:"success"`

	// UpdateType type of update performed
	UpdateType UpdateStrategy `json:"update_type"`

	// PreviousVersion previous repository version
	PreviousVersion string `json:"previous_version"`

	// NewVersion new repository version
	NewVersion string `json:"new_version"`

	// FilesUpdated list of updated files
	FilesUpdated []string `json:"files_updated"`

	// FilesAdded list of added files
	FilesAdded []string `json:"files_added"`

	// FilesRemoved list of removed files
	FilesRemoved []string `json:"files_removed"`

	// BytesDownloaded total bytes downloaded
	BytesDownloaded int64 `json:"bytes_downloaded"`

	// Duration update duration
	Duration time.Duration `json:"duration"`

	// StartedAt when update started
	StartedAt time.Time `json:"started_at"`

	// CompletedAt when update completed
	CompletedAt time.Time `json:"completed_at"`

	// Errors any errors encountered
	Errors []string `json:"errors,omitempty"`

	// Warnings any warnings generated
	Warnings []string `json:"warnings,omitempty"`
}

// UpdateCheck contains information about available updates
type UpdateCheck struct {
	// UpdatesAvailable whether updates are available
	UpdatesAvailable bool `json:"updates_available"`

	// CurrentVersion current local version
	CurrentVersion string `json:"current_version"`

	// LatestVersion latest remote version
	LatestVersion string `json:"latest_version"`

	// FilesToUpdate files that need updating
	FilesToUpdate []*FileUpdateInfo `json:"files_to_update"`

	// FilesToAdd new files to add
	FilesToAdd []*FileInfo `json:"files_to_add"`

	// FilesToRemove files to remove
	FilesToRemove []string `json:"files_to_remove"`

	// TotalDownloadSize total download size in bytes
	TotalDownloadSize int64 `json:"total_download_size"`

	// CheckedAt when check was performed
	CheckedAt time.Time `json:"checked_at"`
}

// FileUpdateInfo contains information about a file update
type FileUpdateInfo struct {
	// Path file path
	Path string `json:"path"`

	// CurrentVersion current file version
	CurrentVersion string `json:"current_version"`

	// NewVersion new file version
	NewVersion string `json:"new_version"`

	// CurrentChecksum current file checksum
	CurrentChecksum string `json:"current_checksum"`

	// NewChecksum new file checksum
	NewChecksum string `json:"new_checksum"`

	// Size new file size
	Size int64 `json:"size"`

	// ChangeType type of change
	ChangeType FileChangeType `json:"change_type"`
}

// FileChangeType represents the type of file change
type FileChangeType string

const (
	FileChangeTypeModified FileChangeType = "modified"
	FileChangeTypeAdded    FileChangeType = "added"
	FileChangeTypeRemoved  FileChangeType = "removed"
	FileChangeTypeRenamed  FileChangeType = "renamed"
)

// UpdateData contains data for applying updates
type UpdateData struct {
	// Manifest new manifest
	Manifest *Manifest `json:"manifest"`

	// Files file content data
	Files map[string][]byte `json:"files"`

	// Signatures file signatures
	Signatures map[string][]byte `json:"signatures,omitempty"`

	// BackupPath backup location for rollback
	BackupPath string `json:"backup_path"`

	// UpdateType type of update
	UpdateType UpdateStrategy `json:"update_type"`
}

// UpdateHistoryEntry represents an update history entry
type UpdateHistoryEntry struct {
	// ID unique update identifier
	ID string `json:"id"`

	// Timestamp when update occurred
	Timestamp time.Time `json:"timestamp"`

	// FromVersion previous version
	FromVersion string `json:"from_version"`

	// ToVersion updated version
	ToVersion string `json:"to_version"`

	// UpdateType type of update
	UpdateType UpdateStrategy `json:"update_type"`

	// Success whether update was successful
	Success bool `json:"success"`

	// Duration update duration
	Duration time.Duration `json:"duration"`

	// FilesModified number of files modified
	FilesModified int `json:"files_modified"`

	// BytesDownloaded bytes downloaded
	BytesDownloaded int64 `json:"bytes_downloaded"`

	// Error error message if update failed
	Error string `json:"error,omitempty"`
}

// ValidationResult contains validation results
type ValidationResult struct {
	// Valid whether validation passed
	Valid bool `json:"valid"`

	// Errors validation errors
	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings validation warnings
	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// ValidatedAt when validation was performed
	ValidatedAt time.Time `json:"validated_at"`

	// ValidationType type of validation performed
	ValidationType ValidationType `json:"validation_type"`
}

// ValidationType represents the type of validation
type ValidationType string

const (
	ValidationTypeTemplate  ValidationType = "template"
	ValidationTypeScript    ValidationType = "script"
	ValidationTypeManifest  ValidationType = "manifest"
	ValidationTypeSignature ValidationType = "signature"
	ValidationTypeChecksum  ValidationType = "checksum"
	ValidationTypeStructure ValidationType = "structure"
	ValidationTypeSecurity  ValidationType = "security"
)

// ValidationError represents a validation error
type ValidationError struct {
	// Type error type
	Type string `json:"type"`

	// Message error message
	Message string `json:"message"`

	// Location error location (line, field, etc.)
	Location string `json:"location,omitempty"`

	// Severity error severity
	Severity ErrorSeverity `json:"severity"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	// Type warning type
	Type string `json:"type"`

	// Message warning message
	Message string `json:"message"`

	// Location warning location
	Location string `json:"location,omitempty"`
}

// ErrorSeverity represents error severity levels
type ErrorSeverity string

const (
	ErrorSeverityCritical ErrorSeverity = "critical"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityInfo     ErrorSeverity = "info"
)

// SignatureResult contains signature verification results
type SignatureResult struct {
	// Valid whether signature is valid
	Valid bool `json:"valid"`

	// SignerInfo information about the signer
	SignerInfo *SignerInfo `json:"signer_info,omitempty"`

	// Error signature verification error
	Error string `json:"error,omitempty"`

	// VerifiedAt when signature was verified
	VerifiedAt time.Time `json:"verified_at"`
}

// SignerInfo contains information about signature signer
type SignerInfo struct {
	// KeyID key identifier
	KeyID string `json:"key_id"`

	// Name signer name
	Name string `json:"name,omitempty"`

	// Email signer email
	Email string `json:"email,omitempty"`

	// Fingerprint key fingerprint
	Fingerprint string `json:"fingerprint"`

	// CreatedAt key creation time
	CreatedAt time.Time `json:"created_at,omitempty"`

	// ExpiresAt key expiration time
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// PublicKeyInfo contains public key information
type PublicKeyInfo struct {
	// KeyID key identifier
	KeyID string `json:"key_id"`

	// Fingerprint key fingerprint
	Fingerprint string `json:"fingerprint"`

	// Algorithm key algorithm
	Algorithm string `json:"algorithm"`

	// Length key length in bits
	Length int `json:"length"`

	// CreatedAt key creation time
	CreatedAt time.Time `json:"created_at,omitempty"`

	// ExpiresAt key expiration time
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// UserIDs associated user IDs
	UserIDs []string `json:"user_ids,omitempty"`
}

// ChecksumResult contains checksum verification results
type ChecksumResult struct {
	// Valid whether checksum is valid
	Valid bool `json:"valid"`

	// ExpectedChecksum expected checksum value
	ExpectedChecksum string `json:"expected_checksum"`

	// ActualChecksum calculated checksum value
	ActualChecksum string `json:"actual_checksum"`

	// Algorithm checksum algorithm used
	Algorithm HashAlgorithm `json:"algorithm"`

	// Error checksum verification error
	Error string `json:"error,omitempty"`

	// VerifiedAt when checksum was verified
	VerifiedAt time.Time `json:"verified_at"`
}

// SecurityScanResult contains security scan results
type SecurityScanResult struct {
	// Safe whether content is considered safe
	Safe bool `json:"safe"`

	// Threats detected security threats
	Threats []*SecurityThreat `json:"threats,omitempty"`

	// Warnings security warnings
	Warnings []*SecurityWarning `json:"warnings,omitempty"`

	// ScannedAt when scan was performed
	ScannedAt time.Time `json:"scanned_at"`

	// ScanType type of security scan
	ScanType SecurityScanType `json:"scan_type"`
}

// SecurityScanType represents the type of security scan
type SecurityScanType string

const (
	SecurityScanTypeMalware   SecurityScanType = "malware"
	SecurityScanTypeSignature SecurityScanType = "signature"
	SecurityScanTypeStatic    SecurityScanType = "static"
	SecurityScanTypeBehavior  SecurityScanType = "behavior"
)

// SecurityThreat represents a detected security threat
type SecurityThreat struct {
	// Type threat type
	Type string `json:"type"`

	// Description threat description
	Description string `json:"description"`

	// Severity threat severity
	Severity ErrorSeverity `json:"severity"`

	// Location threat location in content
	Location string `json:"location,omitempty"`

	// Confidence detection confidence
	Confidence float64 `json:"confidence"`
}

// SecurityWarning represents a security warning
type SecurityWarning struct {
	// Type warning type
	Type string `json:"type"`

	// Message warning message
	Message string `json:"message"`

	// Location warning location in content
	Location string `json:"location,omitempty"`
}

// RepositoryMetadata contains remote repository metadata
type RepositoryMetadata struct {
	// Name repository name
	Name string `json:"name"`

	// Description repository description
	Description string `json:"description"`

	// Publisher repository publisher
	Publisher string `json:"publisher"`

	// Version current version
	Version string `json:"version"`

	// Updated last update time
	Updated time.Time `json:"updated"`

	// License repository license
	License string `json:"license,omitempty"`

	// URL repository URL
	URL string `json:"url,omitempty"`

	// Size total repository size
	Size int64 `json:"size"`

	// FileCount total file count
	FileCount int `json:"file_count"`

	// Tags repository tags
	Tags []string `json:"tags,omitempty"`
}

// CacheInfo contains cache statistics
type CacheInfo struct {
	// Enabled whether cache is enabled
	Enabled bool `json:"enabled"`

	// SizeBytes current cache size in bytes
	SizeBytes int64 `json:"size_bytes"`

	// MaxSizeBytes maximum cache size in bytes
	MaxSizeBytes int64 `json:"max_size_bytes"`

	// FileCount number of cached files
	FileCount int `json:"file_count"`

	// HitRate cache hit rate
	HitRate float64 `json:"hit_rate"`

	// LastCleanup when cache was last cleaned
	LastCleanup time.Time `json:"last_cleanup,omitempty"`
}
