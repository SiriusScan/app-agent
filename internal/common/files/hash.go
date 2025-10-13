package files

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
)

// HashAlgorithm represents supported hash algorithms
type HashAlgorithm string

const (
	SHA256 HashAlgorithm = "sha256"
	SHA1   HashAlgorithm = "sha1"
	MD5    HashAlgorithm = "md5"
	SHA512 HashAlgorithm = "sha512"
)

// CalculateHash calculates the hash of a file using the specified algorithm.
// Supported algorithms: sha256, sha1, md5, sha512 (default: sha256)
func CalculateHash(path string, algorithm HashAlgorithm) (string, error) {
	// Read file contents
	data, err := ReadFile(path)
	if err != nil {
		return "", err
	}

	// Calculate hash based on algorithm
	var hasher hash.Hash

	switch algorithm {
	case SHA256, "":
		hasher = sha256.New()
	case SHA1:
		hasher = sha1.New()
	case MD5:
		hasher = md5.New()
	case SHA512:
		hasher = sha512.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	hasher.Write(data)
	hashBytes := hasher.Sum(nil)

	return hex.EncodeToString(hashBytes), nil
}

// CalculateSHA256 calculates the SHA256 hash of a file.
func CalculateSHA256(path string) (string, error) {
	return CalculateHash(path, SHA256)
}

// CalculateSHA1 calculates the SHA1 hash of a file.
func CalculateSHA1(path string) (string, error) {
	return CalculateHash(path, SHA1)
}

// CalculateMD5 calculates the MD5 hash of a file.
func CalculateMD5(path string) (string, error) {
	return CalculateHash(path, MD5)
}

// CalculateSHA512 calculates the SHA512 hash of a file.
func CalculateSHA512(path string) (string, error) {
	return CalculateHash(path, SHA512)
}

// HashesMatch compares two hash strings (case-insensitive).
func HashesMatch(hash1, hash2 string) bool {
	// Convert to lowercase for case-insensitive comparison
	return len(hash1) == len(hash2) && 
		   hex.EncodeToString([]byte(hash1)) == hex.EncodeToString([]byte(hash2)) ||
		   hash1 == hash2
}

