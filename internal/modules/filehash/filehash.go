package filehash

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/common/files"
	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
)

const (
	// DefaultTimeout is the maximum time for hash calculation
	DefaultTimeout = 30 * time.Second
)

// FileHashModule detects vulnerable file versions by comparing file hashes
type FileHashModule struct{}

// Execute implements the Module interface.
// It calculates the hash of a file and compares it to an expected hash value.
//
// Config fields:
//   - path (string, required): Path to the file to check
//   - hash (string, required): Expected hash value to compare against
//   - algorithm (string, optional): Hash algorithm to use (sha256, sha1, md5, sha512). Defaults to sha256.
func (m *FileHashModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	// Extract config values
	path := config.GetString("path")
	expectedHash := config.GetString("hash")
	algorithm := config.GetString("algorithm")

	// Validate required fields
	if path == "" {
		return &modules.Result{
			Matched: false,
			Error:   "config field 'path' is required",
		}, nil
	}

	if expectedHash == "" {
		return &modules.Result{
			Matched: false,
			Error:   "config field 'hash' is required",
		}, nil
	}

	// Default algorithm
	if algorithm == "" {
		algorithm = "sha256"
	}

	// Normalize algorithm name
	algorithm = strings.ToLower(algorithm)

	// Apply timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	// Calculate the actual hash
	actualHash, err := m.calculateHash(ctx, path, algorithm)
	if err != nil {
		return &modules.Result{
			Matched: false,
			Error:   fmt.Sprintf("failed to calculate hash: %v", err),
		}, nil
	}

	// Normalize hashes for comparison (case-insensitive)
	expectedHash = strings.ToLower(strings.TrimSpace(expectedHash))
	actualHash = strings.ToLower(strings.TrimSpace(actualHash))

	// Compare hashes
	matched := expectedHash == actualHash

	// Build evidence
	evidence := map[string]interface{}{
		"path":          path,
		"algorithm":     algorithm,
		"expected_hash": expectedHash,
		"actual_hash":   actualHash,
		"matched":       matched,
	}

	return &modules.Result{
		Matched:  matched,
		Evidence: evidence,
	}, nil
}

// calculateHash calculates the hash of a file using the specified algorithm
func (m *FileHashModule) calculateHash(ctx context.Context, path string, algorithm string) (string, error) {
	// Note: Context is reserved for future use (e.g., cancellation of long-running hash operations)
	// Current implementation doesn't use context, but signature includes it for future compatibility
	_ = ctx

	switch algorithm {
	case "sha256":
		return files.CalculateSHA256(path)
	case "sha1":
		return files.CalculateSHA1(path)
	case "md5":
		return files.CalculateMD5(path)
	case "sha512":
		return files.CalculateSHA512(path)
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s (supported: sha256, sha1, md5, sha512)", algorithm)
	}
}

// init registers the FileHash module in the global registry
func init() {
	descriptor := modules.Descriptor{
		Type:        "file_hash",
		Name:        "File Hash Validator",
		Description: "Compares file cryptographic hashes to detect specific vulnerable file versions",
		Version:     "1.0.0",
		Author:      "Sirius Scan",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs: map[string]string{
			"path":      "Path to the file to check",
			"hash":      "Expected hash value to compare against",
			"algorithm": "Hash algorithm to use (sha256, sha1, md5, sha512). Defaults to sha256.",
		},
	}

	// Register module
	if err := registry.Register(&FileHashModule{}, descriptor); err != nil {
		panic(fmt.Sprintf("failed to register file_hash module: %v", err))
	}
}
