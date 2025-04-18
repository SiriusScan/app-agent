package store

import (
	"context"
)

// Key represents a key in the store
type Key struct {
	ID     string
	Value  []byte
	Status string
}

// Client defines the interface for store operations
type Client interface {
	// Connect establishes a connection to the store server
	Connect(ctx context.Context) error

	// Close closes the connection to the store server
	Close() error

	// Get retrieves a key from the store
	Get(ctx context.Context, id string) (*Key, error)

	// Put stores a key in the store
	Put(ctx context.Context, key *Key) error

	// Delete removes a key from the store
	Delete(ctx context.Context, id string) error

	// List retrieves all keys matching a prefix
	List(ctx context.Context, prefix string) ([]*Key, error)
}
