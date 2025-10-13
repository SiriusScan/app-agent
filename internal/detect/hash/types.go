package hash

import (
	"os"
	"time"
)

// FileInfo contains information about a file
type FileInfo struct {
	// Path file path
	Path string `json:"path"`

	// Exists whether file exists
	Exists bool `json:"exists"`

	// Size file size in bytes
	Size int64 `json:"size"`

	// ModTime last modification time
	ModTime time.Time `json:"mod_time"`

	// IsDirectory whether this is a directory
	IsDirectory bool `json:"is_directory"`

	// IsExecutable whether file has execute permissions
	IsExecutable bool `json:"is_executable"`

	// Mode file mode/permissions
	Mode os.FileMode `json:"mode"`
}

// HashVerificationRequest represents a request to verify file hashes
type HashVerificationRequest struct {
	// Files list of files to verify
	Files []FileHashPair `json:"files"`

	// Algorithm hash algorithm to use
	Algorithm string `json:"algorithm"`

	// ConcurrentJobs number of concurrent hash calculations
	ConcurrentJobs int `json:"concurrent_jobs,omitempty"`
}

// FileHashPair represents a file path and expected hash
type FileHashPair struct {
	// Path file path
	Path string `json:"path"`

	// ExpectedHash expected hash value
	ExpectedHash string `json:"expected_hash"`

	// Description optional description
	Description string `json:"description,omitempty"`
}

// HashVerificationResponse contains results of hash verification
type HashVerificationResponse struct {
	// Results verification results for each file
	Results []HashVerificationResult `json:"results"`

	// Summary overall verification summary
	Summary HashVerificationSummary `json:"summary"`

	// ExecutionTime total time taken for verification
	ExecutionTime time.Duration `json:"execution_time"`
}

// HashVerificationResult represents the result of verifying a single file
type HashVerificationResult struct {
	// Path file path that was checked
	Path string `json:"path"`

	// ExpectedHash hash that was expected
	ExpectedHash string `json:"expected_hash"`

	// ActualHash hash that was calculated
	ActualHash string `json:"actual_hash"`

	// Matches whether hashes match
	Matches bool `json:"matches"`

	// FileExists whether file exists
	FileExists bool `json:"file_exists"`

	// FileSize size of the file in bytes
	FileSize int64 `json:"file_size,omitempty"`

	// IsExecutable whether file is executable
	IsExecutable bool `json:"is_executable"`

	// Error any error that occurred
	Error string `json:"error,omitempty"`

	// CheckedAt when verification was performed
	CheckedAt time.Time `json:"checked_at"`

	// Description optional description
	Description string `json:"description,omitempty"`
}

// HashVerificationSummary provides overall verification statistics
type HashVerificationSummary struct {
	// TotalFiles total number of files checked
	TotalFiles int `json:"total_files"`

	// MatchingFiles number of files with matching hashes
	MatchingFiles int `json:"matching_files"`

	// MissingFiles number of files that don't exist
	MissingFiles int `json:"missing_files"`

	// ErrorFiles number of files that had errors during verification
	ErrorFiles int `json:"error_files"`

	// SuccessRate percentage of successful verifications
	SuccessRate float64 `json:"success_rate"`

	// MatchRate percentage of files with matching hashes
	MatchRate float64 `json:"match_rate"`
}
