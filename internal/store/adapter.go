package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SiriusScan/go-api/sirius/store"
)

// adapter implements the store.Client interface using Sirius KVStore
type adapter struct {
	store store.KVStore
}

// NewClient creates a new store client
func NewClient(cfg *Config) (Client, error) {
	if cfg == nil {
		return nil, errors.New("store configuration is required")
	}

	kvStore, err := store.NewValkeyStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey store: %w", err)
	}

	return &adapter{
		store: kvStore,
	}, nil
}

// Get retrieves a value by key
func (a *adapter) Get(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New("key cannot be empty")
	}

	resp, err := a.store.GetValue(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get value: %w", err)
	}

	// Check for nil response or nil Message
	if resp.Message.Value == "" {
		// If the key exists but has no value, return empty value error
		return "", ErrEmptyValue
	}

	return resp.Message.Value, nil
}

// List retrieves all values matching a pattern
func (a *adapter) List(ctx context.Context, pattern string) (map[string]string, error) {
	if pattern == "" {
		return nil, errors.New("pattern cannot be empty")
	}

	// Get matching keys using ListKeys method
	keys, err := a.store.ListKeys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	// Get values for each key
	results := make(map[string]string)
	for _, key := range keys {
		resp, err := a.store.GetValue(ctx, key)
		if err != nil || resp.Message.Value == "" {
			continue // Skip keys that can't be retrieved or have empty values
		}
		results[key] = resp.Message.Value
	}

	return results, nil
}

// Close closes the store connection
func (a *adapter) Close() error {
	if a.store == nil {
		return nil
	}
	return a.store.Close()
}
