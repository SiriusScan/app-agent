package store

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrNotFound   = errors.New("key not found")
	ErrEmptyValue = errors.New("empty value")
)

// Client defines the interface for interacting with the key-value store
type Client interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) (string, error)

	// List retrieves all values matching a pattern
	List(ctx context.Context, pattern string) (map[string]string, error)

	// Close closes the store connection
	Close() error
}
