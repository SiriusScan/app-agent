package store

import (
	"context"
	"strings"
)

// Message represents a stored message
type Message struct {
	Value string
}

// ValkeyResponse represents a response from the key-value store
type ValkeyResponse struct {
	Message Message
}

// KVStore defines the key/value operations
type KVStore interface {
	SetValue(ctx context.Context, key, value string) error
	GetValue(ctx context.Context, key string) (ValkeyResponse, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	Del(ctx context.Context, key string) error
	Close() error
}

// MockStore provides a mock implementation for testing
type MockStore struct {
	data map[string]string
}

// NewMockStore creates a new mock store
func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]string),
	}
}

// SetValue implements KVStore
func (m *MockStore) SetValue(ctx context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

// GetValue implements KVStore
func (m *MockStore) GetValue(ctx context.Context, key string) (ValkeyResponse, error) {
	if value, ok := m.data[key]; ok {
		return ValkeyResponse{
			Message: Message{
				Value: value,
			},
		}, nil
	}
	return ValkeyResponse{}, nil
}

// Keys implements KVStore
func (m *MockStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	
	// Simple pattern matching with * wildcard at the end
	if pattern == "*" {
		for k := range m.data {
			keys = append(keys, k)
		}
	} else if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		for k := range m.data {
			if strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}
	} else {
		// Exact match
		if _, ok := m.data[pattern]; ok {
			keys = append(keys, pattern)
		}
	}
	
	return keys, nil
}

// Del implements KVStore
func (m *MockStore) Del(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

// Close implements KVStore
func (m *MockStore) Close() error {
	return nil
}
