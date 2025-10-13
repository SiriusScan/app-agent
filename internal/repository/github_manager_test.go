package repository

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestGitHubRepositoryManager_Initialize(t *testing.T) {
	logger := zap.NewNop()
	manager := NewGitHubRepositoryManager(logger)

	config := &RepositoryConfiguration{
		RemoteURL:      "https://github.com/test/repo",
		LocalPath:      "/tmp/test-repo",
		UpdateInterval: 24 * time.Hour,
		UpdateStrategy: UpdateStrategyIncremental,
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		UserAgent:      "Sirius-Agent/1.0",
	}

	manager.SetConfiguration(config)

	ctx := context.Background()
	err := manager.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
}

func TestGitHubRepositoryManager_GetRepositoryInfo(t *testing.T) {
	logger := zap.NewNop()
	manager := NewGitHubRepositoryManager(logger)

	config := &RepositoryConfiguration{
		RemoteURL:      "https://github.com/test/repo",
		LocalPath:      "/tmp/test-repo",
		UpdateInterval: 24 * time.Hour,
		UpdateStrategy: UpdateStrategyIncremental,
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		UserAgent:      "Sirius-Agent/1.0",
	}

	manager.SetConfiguration(config)

	info, err := manager.GetRepositoryInfo()
	if err != nil {
		t.Errorf("GetRepositoryInfo failed: %v", err)
	}

	if info.LocalPath != config.LocalPath {
		t.Errorf("Expected local path %s, got %s", config.LocalPath, info.LocalPath)
	}

	if info.RemoteURL != config.RemoteURL {
		t.Errorf("Expected remote URL %s, got %s", config.RemoteURL, info.RemoteURL)
	}
}

func TestManifest_Validation(t *testing.T) {
	manifest := &Manifest{
		Version: "1.0.0",
		Updated: time.Now(),
		Templates: map[string]*FileInfo{
			"template1.yaml": {
				Path:        "template1.yaml",
				Version:     "1.0.0",
				Checksum:    "abc123",
				Size:        1024,
				Updated:     time.Now(),
				Description: "Test template",
				Author:      "test@example.com",
			},
		},
		Scripts: map[string]*FileInfo{
			"script1.sh": {
				Path:        "script1.sh",
				Version:     "1.0.0",
				Checksum:    "def456",
				Size:        512,
				Updated:     time.Now(),
				Description: "Test script",
				Author:      "test@example.com",
			},
		},
		Metadata: &ManifestMetadata{
			Publisher:   "Test Publisher",
			Description: "Test repository",
			License:     "MIT",
			URL:         "https://github.com/test/repo",
		},
	}

	if manifest.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", manifest.Version)
	}

	if len(manifest.Templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(manifest.Templates))
	}

	if len(manifest.Scripts) != 1 {
		t.Errorf("Expected 1 script, got %d", len(manifest.Scripts))
	}

	if manifest.Metadata.Publisher != "Test Publisher" {
		t.Errorf("Expected publisher 'Test Publisher', got %s", manifest.Metadata.Publisher)
	}
}
