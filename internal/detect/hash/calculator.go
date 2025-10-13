package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
)

// HashCalculator provides file hash calculation capabilities
type HashCalculator struct {
	logger *zap.Logger
}

// NewHashCalculator creates a new hash calculator instance
func NewHashCalculator(logger *zap.Logger) *HashCalculator {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HashCalculator{
		logger: logger,
	}
}

// CalculateFileHash computes hash for a file using the specified algorithm
func (hc *HashCalculator) CalculateFileHash(filePath string, algorithm detect.HashAlgorithm) (string, error) {
	hc.logger.Debug("Calculating file hash",
		zap.String("file", filePath),
		zap.String("algorithm", string(algorithm)))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Create appropriate hasher
	hasher, err := hc.createHasher(algorithm)
	if err != nil {
		return "", fmt.Errorf("failed to create hasher for %s: %w", algorithm, err)
	}

	// Calculate hash using streaming approach for memory efficiency
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash for %s: %w", filePath, err)
	}

	// Get hash as hex string
	hashBytes := hasher.Sum(nil)
	hashString := fmt.Sprintf("%x", hashBytes)

	hc.logger.Debug("Hash calculation completed",
		zap.String("file", filePath),
		zap.String("algorithm", string(algorithm)),
		zap.String("hash", hashString))

	return hashString, nil
}

// VerifyFileHash checks if file matches expected hash
func (hc *HashCalculator) VerifyFileHash(filePath string, expectedHash string, algorithm detect.HashAlgorithm) (bool, error) {
	hc.logger.Debug("Verifying file hash",
		zap.String("file", filePath),
		zap.String("algorithm", string(algorithm)),
		zap.String("expected_hash", expectedHash))

	actualHash, err := hc.CalculateFileHash(filePath, algorithm)
	if err != nil {
		return false, err
	}

	// Compare hashes (case insensitive)
	matches := strings.EqualFold(actualHash, expectedHash)

	hc.logger.Debug("Hash verification completed",
		zap.String("file", filePath),
		zap.Bool("matches", matches),
		zap.String("actual_hash", actualHash),
		zap.String("expected_hash", expectedHash))

	return matches, nil
}

// BatchHashCheck verifies multiple files against hash targets
func (hc *HashCalculator) BatchHashCheck(targets []detect.HashTarget) ([]*detect.HashResult, error) {
	hc.logger.Info("Starting batch hash verification",
		zap.Int("targets", len(targets)))

	results := make([]*detect.HashResult, len(targets))

	for i, target := range targets {
		result := &detect.HashResult{
			Target:     target,
			CheckedAt:  now(),
			FileExists: false,
			Matches:    false,
		}

		// Check if file exists
		if _, err := os.Stat(target.Path); os.IsNotExist(err) {
			result.Error = fmt.Sprintf("file does not exist: %s", target.Path)
			results[i] = result
			continue
		}
		result.FileExists = true

		// Calculate actual hash
		actualHash, err := hc.CalculateFileHash(target.Path, target.Algorithm)
		if err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}
		result.ActualHash = actualHash

		// Compare with expected hash
		result.Matches = strings.EqualFold(actualHash, target.ExpectedHash)

		results[i] = result
	}

	hc.logger.Info("Batch hash verification completed",
		zap.Int("total", len(results)),
		zap.Int("matches", countMatches(results)),
		zap.Int("errors", countErrors(results)))

	return results, nil
}

// GetFileInfo returns file information including existence and basic properties
func (hc *HashCalculator) GetFileInfo(filePath string) (*FileInfo, error) {
	stat, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return &FileInfo{
			Path:   filePath,
			Exists: false,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	fileInfo := &FileInfo{
		Path:        filePath,
		Exists:      true,
		Size:        stat.Size(),
		ModTime:     stat.ModTime(),
		IsDirectory: stat.IsDir(),
		Mode:        stat.Mode(),
	}

	// Check if file is executable
	fileInfo.IsExecutable = hc.isExecutable(stat.Mode())

	hc.logger.Debug("Retrieved file info",
		zap.String("path", filePath),
		zap.Bool("exists", fileInfo.Exists),
		zap.Int64("size", fileInfo.Size),
		zap.Bool("executable", fileInfo.IsExecutable))

	return fileInfo, nil
}

// createHasher creates a hash.Hash instance for the specified algorithm
func (hc *HashCalculator) createHasher(algorithm detect.HashAlgorithm) (hash.Hash, error) {
	switch algorithm {
	case detect.HashAlgorithmSHA256:
		return sha256.New(), nil
	case detect.HashAlgorithmSHA1:
		return sha1.New(), nil
	case detect.HashAlgorithmMD5:
		return md5.New(), nil
	case detect.HashAlgorithmSHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}

// isExecutable checks if file has executable permissions
func (hc *HashCalculator) isExecutable(mode os.FileMode) bool {
	// Check if any execute bit is set
	return mode&0111 != 0
}

// ValidateHashString checks if a hash string is valid for the given algorithm
func (hc *HashCalculator) ValidateHashString(hashString string, algorithm detect.HashAlgorithm) error {
	// Remove any whitespace
	hashString = strings.TrimSpace(hashString)

	// Check length based on algorithm
	var expectedLength int
	switch algorithm {
	case detect.HashAlgorithmSHA256:
		expectedLength = 64
	case detect.HashAlgorithmSHA1:
		expectedLength = 40
	case detect.HashAlgorithmMD5:
		expectedLength = 32
	case detect.HashAlgorithmSHA512:
		expectedLength = 128
	default:
		return fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	if len(hashString) != expectedLength {
		return fmt.Errorf("invalid hash length for %s: expected %d, got %d",
			algorithm, expectedLength, len(hashString))
	}

	// Check if all characters are valid hex
	for _, char := range hashString {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'f') ||
			(char >= 'A' && char <= 'F')) {
			return fmt.Errorf("invalid hex character in hash: %c", char)
		}
	}

	return nil
}

// GetSupportedAlgorithms returns list of supported hash algorithms
func (hc *HashCalculator) GetSupportedAlgorithms() []detect.HashAlgorithm {
	return []detect.HashAlgorithm{
		detect.HashAlgorithmSHA256,
		detect.HashAlgorithmSHA1,
		detect.HashAlgorithmMD5,
		detect.HashAlgorithmSHA512,
	}
}

// Helper functions

func countMatches(results []*detect.HashResult) int {
	count := 0
	for _, result := range results {
		if result.Matches {
			count++
		}
	}
	return count
}

func countErrors(results []*detect.HashResult) int {
	count := 0
	for _, result := range results {
		if result.Error != "" {
			count++
		}
	}
	return count
}

// For testing, we can override this
var now = func() time.Time {
	return time.Now()
}
