package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test successful read
	data, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Content mismatch: got %q, want %q", string(data), testContent)
	}

	t.Log("✅ File reading working")
}

func TestReadFileString(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Test content"

	os.WriteFile(testFile, []byte(testContent), 0644)

	content, err := ReadFileString(testFile)
	if err != nil {
		t.Fatalf("Failed to read file as string: %v", err)
	}

	if content != testContent {
		t.Errorf("Content mismatch: got %q, want %q", content, testContent)
	}

	t.Log("✅ ReadFileString working")
}

func TestReadFileNotFound(t *testing.T) {
	_, err := ReadFile("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}

	if _, ok := err.(*FileNotFoundError); !ok {
		t.Errorf("Expected FileNotFoundError, got %T", err)
	}

	t.Log("✅ File not found error handling working")
}

func TestReadFileTooLarge(t *testing.T) {
	tempDir := t.TempDir()
	largeFile := filepath.Join(tempDir, "large.txt")

	// Create a file larger than the limit
	largeContent := make([]byte, 1024)
	os.WriteFile(largeFile, largeContent, 0644)

	// Try to read with small size limit
	opts := ReadOptions{
		MaxSize: 512, // Smaller than file size
		Timeout: DefaultReadTimeout,
	}

	_, err := ReadFileWithOptions(largeFile, opts)
	if err == nil {
		t.Fatal("Expected error for file too large")
	}

	if _, ok := err.(*FileTooLargeError); !ok {
		t.Errorf("Expected FileTooLargeError, got %T", err)
	}

	t.Log("✅ File size limit enforcement working")
}

func TestReadFileWithTimeout(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Very short timeout should still work for small files
	opts := ReadOptions{
		MaxSize: DefaultMaxFileSize,
		Timeout: 1 * time.Millisecond,
	}

	// This might or might not timeout depending on system speed
	// Just verify we get either success or timeout error
	_, err := ReadFileWithOptions(testFile, opts)
	if err != nil {
		if _, ok := err.(*TimeoutError); !ok {
			t.Errorf("If error occurs, expected TimeoutError, got %T", err)
		}
	}

	t.Log("✅ Timeout handling working")
}

func TestCalculateHash(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	os.WriteFile(testFile, []byte(testContent), 0644)

	// Known hashes for "Hello, World!"
	knownHashes := map[HashAlgorithm]string{
		SHA256: "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f",
		SHA1:   "0a0a9f2a6772942557ab5355d76af442f8f65e01",
		MD5:    "65a8e27d8879283831b664bd8b7f0ad4",
	}

	for algo, expectedHash := range knownHashes {
		hash, err := CalculateHash(testFile, algo)
		if err != nil {
			t.Fatalf("Failed to calculate %s hash: %v", algo, err)
		}

		if hash != expectedHash {
			t.Errorf("%s hash mismatch:\n  got:  %s\n  want: %s", algo, hash, expectedHash)
		}
	}

	t.Log("✅ Hash calculation working (SHA256, SHA1, MD5)")
}

func TestCalculateHashHelpers(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "test"
	os.WriteFile(testFile, []byte(testContent), 0644)

	// Test helper functions
	_, err := CalculateSHA256(testFile)
	if err != nil {
		t.Errorf("CalculateSHA256 failed: %v", err)
	}

	_, err = CalculateSHA1(testFile)
	if err != nil {
		t.Errorf("CalculateSHA1 failed: %v", err)
	}

	_, err = CalculateMD5(testFile)
	if err != nil {
		t.Errorf("CalculateMD5 failed: %v", err)
	}

	_, err = CalculateSHA512(testFile)
	if err != nil {
		t.Errorf("CalculateSHA512 failed: %v", err)
	}

	t.Log("✅ Hash helper functions working")
}

func TestCalculateHashInvalidAlgorithm(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	_, err := CalculateHash(testFile, "invalid")
	if err == nil {
		t.Fatal("Expected error for invalid algorithm")
	}

	t.Log("✅ Invalid algorithm handling working")
}

func TestExists(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Test existing file
	if !Exists(testFile) {
		t.Error("Exists() returned false for existing file")
	}

	// Test non-existing file
	if Exists(filepath.Join(tempDir, "nonexistent.txt")) {
		t.Error("Exists() returned true for non-existing file")
	}

	t.Log("✅ Exists() working")
}

func TestIsFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Test file
	if !IsFile(testFile) {
		t.Error("IsFile() returned false for file")
	}

	// Test directory
	if IsFile(tempDir) {
		t.Error("IsFile() returned true for directory")
	}

	// Test non-existent
	if IsFile(filepath.Join(tempDir, "nonexistent.txt")) {
		t.Error("IsFile() returned true for non-existent path")
	}

	t.Log("✅ IsFile() working")
}

func TestIsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Test directory
	if !IsDirectory(tempDir) {
		t.Error("IsDirectory() returned false for directory")
	}

	// Test file
	if IsDirectory(testFile) {
		t.Error("IsDirectory() returned true for file")
	}

	t.Log("✅ IsDirectory() working")
}

func TestIsReadable(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Test readable file
	if !IsReadable(testFile) {
		t.Error("IsReadable() returned false for readable file")
	}

	// Test non-existent file
	if IsReadable(filepath.Join(tempDir, "nonexistent.txt")) {
		t.Error("IsReadable() returned true for non-existent file")
	}

	t.Log("✅ IsReadable() working")
}

func TestGetFileSize(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	os.WriteFile(testFile, []byte(testContent), 0644)

	size := GetFileSize(testFile)
	expectedSize := int64(len(testContent))

	if size != expectedSize {
		t.Errorf("GetFileSize() = %d, want %d", size, expectedSize)
	}

	// Test non-existent file
	if GetFileSize(filepath.Join(tempDir, "nonexistent.txt")) != 0 {
		t.Error("GetFileSize() should return 0 for non-existent file")
	}

	t.Log("✅ GetFileSize() working")
}

