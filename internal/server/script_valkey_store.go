package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SiriusScan/go-api/sirius/store"
)

// ScriptValKeyStoreImpl implements the ScriptValKeyStore interface
type ScriptValKeyStoreImpl struct {
	valkeyStore store.KVStore
}

// NewScriptValKeyStore creates a new script ValKey store implementation
func NewScriptValKeyStore(valkeyStore store.KVStore) *ScriptValKeyStoreImpl {
	return &ScriptValKeyStoreImpl{
		valkeyStore: valkeyStore,
	}
}

// SetScriptMetadata stores script metadata in ValKey
func (s *ScriptValKeyStoreImpl) SetScriptMetadata(ctx context.Context, scriptID string, metadata *ScriptMetadata) error {
	key := fmt.Sprintf("agent:script:meta:%s", scriptID)

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal script metadata: %w", err)
	}

	if err := s.valkeyStore.SetValue(ctx, key, string(data)); err != nil {
		return fmt.Errorf("failed to store script metadata in ValKey: %w", err)
	}

	return nil
}

// SetScriptContent stores script content in ValKey
func (s *ScriptValKeyStoreImpl) SetScriptContent(ctx context.Context, scriptID string, content *ScriptContent) error {
	key := fmt.Sprintf("agent:script:%s", scriptID)

	data, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal script content: %w", err)
	}

	if err := s.valkeyStore.SetValue(ctx, key, string(data)); err != nil {
		return fmt.Errorf("failed to store script content in ValKey: %w", err)
	}

	return nil
}

// ListScriptMetaKeys returns all script metadata keys
func (s *ScriptValKeyStoreImpl) ListScriptMetaKeys(ctx context.Context) ([]string, error) {
	pattern := "agent:script:meta:*"

	keys, err := s.valkeyStore.ListKeys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list script metadata keys: %w", err)
	}

	return keys, nil
}

// GetScriptManifest retrieves the global script manifest
func (s *ScriptValKeyStoreImpl) GetScriptManifest(ctx context.Context) (*ScriptManifest, error) {
	key := "agent:script:manifest"

	response, err := s.valkeyStore.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get script manifest: %w", err)
	}

	var manifest ScriptManifest
	if err := json.Unmarshal([]byte(response.Message.Value), &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal script manifest: %w", err)
	}

	return &manifest, nil
}

// SetScriptManifest stores the global script manifest
func (s *ScriptValKeyStoreImpl) SetScriptManifest(ctx context.Context, manifest *ScriptManifest) error {
	key := "agent:script:manifest"

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal script manifest: %w", err)
	}

	if err := s.valkeyStore.SetValue(ctx, key, string(data)); err != nil {
		return fmt.Errorf("failed to store script manifest in ValKey: %w", err)
	}

	return nil
}

// GetScriptMetadata retrieves script metadata by ID
func (s *ScriptValKeyStoreImpl) GetScriptMetadata(ctx context.Context, scriptID string) (*ScriptMetadata, error) {
	key := fmt.Sprintf("agent:script:meta:%s", scriptID)

	response, err := s.valkeyStore.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get script metadata: %w", err)
	}

	var metadata ScriptMetadata
	if err := json.Unmarshal([]byte(response.Message.Value), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal script metadata: %w", err)
	}

	return &metadata, nil
}

// GetScriptContent retrieves script content by ID
func (s *ScriptValKeyStoreImpl) GetScriptContent(ctx context.Context, scriptID string) (*ScriptContent, error) {
	key := fmt.Sprintf("agent:script:%s", scriptID)

	response, err := s.valkeyStore.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get script content: %w", err)
	}

	var content ScriptContent
	if err := json.Unmarshal([]byte(response.Message.Value), &content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal script content: %w", err)
	}

	return &content, nil
}

// ListAllScripts returns all scripts with their metadata
func (s *ScriptValKeyStoreImpl) ListAllScripts(ctx context.Context) (map[string]ScriptMetadata, error) {
	keys, err := s.ListScriptMetaKeys(ctx)
	if err != nil {
		return nil, err
	}

	scripts := make(map[string]ScriptMetadata)

	for _, key := range keys {
		scriptID := strings.TrimPrefix(key, "agent:script:meta:")

		metadata, err := s.GetScriptMetadata(ctx, scriptID)
		if err != nil {
			// Log warning but continue with other scripts
			continue
		}

		scripts[scriptID] = *metadata
	}

	return scripts, nil
}
