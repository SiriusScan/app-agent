package filehash

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
)

// Known hashes for test content "Hello, World!\n"
const (
	testContent = "Hello, World!\n"
	testSHA256  = "c98c24b677eff44860afea6f493bbaec5bb1c4cbb209c6fc2bbb47f66ff2ad31"
	testSHA1    = "60fde9c2310b0d4cad4dab8d126b04387efba289"
	testMD5     = "bea8252ff4e80f41719ea13cdf007273"
	testSHA512  = "921618bc6d9f8059437c5e0397b13f973ab7c7a7b81f0ca31b70bf448fd800a460b67efda0020088bc97bf7d9da97a9e2ce7b20d46e066462ec44cf60284f9a7"
	wrongHash   = "0000000000000000000000000000000000000000000000000000000000000000"
)

func TestFileHashModule_Execute(t *testing.T) {
	t.Log("\n🔍 Testing FileHashModule.Execute()...")

	module := &FileHashModule{}
	ctx := context.Background()

	t.Run("valid SHA256 hash match", func(t *testing.T) {
		t.Log("\n  Testing SHA256 hash match...")

		// Create test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := modules.StepConfig{
			"path":      testFile,
			"hash":      testSHA256,
			"algorithm": "sha256",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if !result.Matched {
			t.Errorf("Expected matched=true, got false")
		}

		if result.Error != "" {
			t.Errorf("Expected no error, got: %s", result.Error)
		}

		// Verify evidence
		if result.Evidence["path"] != testFile {
			t.Errorf("Expected path in evidence")
		}

		if result.Evidence["algorithm"] != "sha256" {
			t.Errorf("Expected algorithm in evidence")
		}

		if result.Evidence["expected_hash"] != testSHA256 {
			t.Errorf("Expected expected_hash in evidence")
		}

		if result.Evidence["actual_hash"] != testSHA256 {
			t.Errorf("Expected actual_hash in evidence")
		}

		t.Log("  ✅ SHA256 hash match successful")
	})

	t.Run("valid SHA1 hash match", func(t *testing.T) {
		t.Log("\n  Testing SHA1 hash match...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := modules.StepConfig{
			"path":      testFile,
			"hash":      testSHA1,
			"algorithm": "sha1",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if !result.Matched {
			t.Errorf("Expected matched=true, got false")
		}

		t.Log("  ✅ SHA1 hash match successful")
	})

	t.Run("valid MD5 hash match", func(t *testing.T) {
		t.Log("\n  Testing MD5 hash match...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := modules.StepConfig{
			"path":      testFile,
			"hash":      testMD5,
			"algorithm": "md5",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if !result.Matched {
			t.Errorf("Expected matched=true, got false")
		}

		t.Log("  ✅ MD5 hash match successful")
	})

	t.Run("valid SHA512 hash match", func(t *testing.T) {
		t.Log("\n  Testing SHA512 hash match...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Only need first 128 characters of SHA512
		config := modules.StepConfig{
			"path":      testFile,
			"hash":      testSHA512,
			"algorithm": "sha512",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if !result.Matched {
			t.Errorf("Expected matched=true, got false")
		}

		t.Log("  ✅ SHA512 hash match successful")
	})

	t.Run("hash mismatch", func(t *testing.T) {
		t.Log("\n  Testing hash mismatch...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := modules.StepConfig{
			"path":      testFile,
			"hash":      wrongHash,
			"algorithm": "sha256",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if result.Matched {
			t.Errorf("Expected matched=false for wrong hash, got true")
		}

		if result.Error != "" {
			t.Errorf("Expected no error for mismatch, got: %s", result.Error)
		}

		// Evidence should still contain both hashes
		if result.Evidence["expected_hash"] != wrongHash {
			t.Error("Expected expected_hash in evidence")
		}

		if result.Evidence["actual_hash"] != testSHA256 {
			t.Error("Expected actual_hash in evidence")
		}

		t.Log("  ✅ Hash mismatch detected correctly")
	})

	t.Run("default algorithm", func(t *testing.T) {
		t.Log("\n  Testing default algorithm (SHA256)...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Don't specify algorithm - should default to sha256
		config := modules.StepConfig{
			"path": testFile,
			"hash": testSHA256,
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if !result.Matched {
			t.Errorf("Expected matched=true with default algorithm, got false")
		}

		if result.Evidence["algorithm"] != "sha256" {
			t.Errorf("Expected default algorithm to be sha256, got %v", result.Evidence["algorithm"])
		}

		t.Log("  ✅ Default algorithm (SHA256) works correctly")
	})

	t.Run("case insensitive hash comparison", func(t *testing.T) {
		t.Log("\n  Testing case insensitive hash comparison...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Use uppercase hash
		config := modules.StepConfig{
			"path":      testFile,
			"hash":      strings.ToUpper(testSHA256),
			"algorithm": "sha256",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if !result.Matched {
			t.Errorf("Expected matched=true with uppercase hash, got false")
		}

		t.Log("  ✅ Case insensitive comparison works")
	})

	t.Run("missing path", func(t *testing.T) {
		t.Log("\n  Testing missing path config...")

		config := modules.StepConfig{
			"hash":      testSHA256,
			"algorithm": "sha256",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if result.Matched {
			t.Error("Expected matched=false for missing path")
		}

		if !strings.Contains(result.Error, "path") || !strings.Contains(result.Error, "required") {
			t.Errorf("Expected error about missing path, got: %s", result.Error)
		}

		t.Logf("  ✅ Missing path error: %s", result.Error)
	})

	t.Run("missing hash", func(t *testing.T) {
		t.Log("\n  Testing missing hash config...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte(testContent), 0644)

		config := modules.StepConfig{
			"path":      testFile,
			"algorithm": "sha256",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if result.Matched {
			t.Error("Expected matched=false for missing hash")
		}

		if !strings.Contains(result.Error, "hash") || !strings.Contains(result.Error, "required") {
			t.Errorf("Expected error about missing hash, got: %s", result.Error)
		}

		t.Logf("  ✅ Missing hash error: %s", result.Error)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Log("\n  Testing file not found...")

		config := modules.StepConfig{
			"path":      "/nonexistent/file/path.txt",
			"hash":      testSHA256,
			"algorithm": "sha256",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if result.Matched {
			t.Error("Expected matched=false for missing file")
		}

		if result.Error == "" {
			t.Error("Expected error for missing file")
		}

		t.Logf("  ✅ File not found error: %s", result.Error)
	})

	t.Run("unsupported algorithm", func(t *testing.T) {
		t.Log("\n  Testing unsupported algorithm...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte(testContent), 0644)

		config := modules.StepConfig{
			"path":      testFile,
			"hash":      testSHA256,
			"algorithm": "sha9000",
		}

		result, err := module.Execute(ctx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		if result.Matched {
			t.Error("Expected matched=false for unsupported algorithm")
		}

		if !strings.Contains(result.Error, "unsupported") {
			t.Errorf("Expected error about unsupported algorithm, got: %s", result.Error)
		}

		t.Logf("  ✅ Unsupported algorithm error: %s", result.Error)
	})

	t.Run("context with deadline", func(t *testing.T) {
		t.Log("\n  Testing context with deadline...")

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte(testContent), 0644)

		// Create context with deadline (but generous enough to complete)
		deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		config := modules.StepConfig{
			"path":      testFile,
			"hash":      testSHA256,
			"algorithm": "sha256",
		}

		result, err := module.Execute(deadlineCtx, config)
		if err != nil {
			t.Fatalf("❌ Execute() returned error: %v", err)
		}

		// Should complete successfully with generous timeout
		if !result.Matched {
			t.Error("Expected matched=true with generous timeout")
		}

		if result.Error != "" {
			t.Errorf("Expected no error, got: %s", result.Error)
		}

		t.Log("  ✅ Context with deadline works correctly")
	})

	t.Log("\n✅ FileHashModule.Execute() tests completed")
}

func TestFileHashModule_Integration(t *testing.T) {
	t.Log("\n🔍 Testing FileHashModule integration with registry...")

	// The module should be registered via init()
	// Try to retrieve it from the registry
	module := registry.Get("file_hash")
	if module == nil {
		t.Fatal("❌ FileHash module not found in registry")
	}

	// Get descriptor separately
	descriptor := registry.GetDescriptor("file_hash")
	if descriptor == nil {
		t.Fatal("❌ FileHash descriptor is nil")
	}

	// Verify descriptor
	if descriptor.Type != "file_hash" {
		t.Errorf("Expected type 'file_hash', got '%s'", descriptor.Type)
	}

	if descriptor.Name == "" {
		t.Error("Descriptor name is empty")
	}

	if descriptor.Description == "" {
		t.Error("Descriptor description is empty")
	}

	if len(descriptor.SupportedOS) == 0 {
		t.Error("Descriptor has no supported OS")
	}

	if len(descriptor.ConfigDocs) == 0 {
		t.Error("Descriptor has no config docs")
	}

	t.Log("  ✅ FileHash module registered correctly")
	t.Logf("    Type: %s", descriptor.Type)
	t.Logf("    Name: %s", descriptor.Name)
	t.Logf("    Version: %s", descriptor.Version)
	t.Logf("    Supported OS: %v", descriptor.SupportedOS)
}

