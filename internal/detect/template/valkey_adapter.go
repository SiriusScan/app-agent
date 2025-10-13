package template

import (
	"context"

	"github.com/SiriusScan/go-api/sirius/store"
)

// ValKeyAdapter adapts the go-api store.KVStore to the template.KVStore interface
type ValKeyAdapter struct {
	store store.KVStore
}

// NewValKeyAdapter creates a new ValKey adapter
func NewValKeyAdapter(store store.KVStore) *ValKeyAdapter {
	return &ValKeyAdapter{
		store: store,
	}
}

// GetValue retrieves a value from the store
func (vka *ValKeyAdapter) GetValue(ctx context.Context, key string) (string, error) {
	response, err := vka.store.GetValue(ctx, key)
	if err != nil {
		return "", err
	}
	return response.Message.Value, nil
}

// SetValue stores a value in the store
func (vka *ValKeyAdapter) SetValue(ctx context.Context, key, value string) error {
	return vka.store.SetValue(ctx, key, value)
}

// ListKeys lists keys matching a pattern
func (vka *ValKeyAdapter) ListKeys(ctx context.Context, pattern string) ([]string, error) {
	return vka.store.ListKeys(ctx, pattern)
}

// DeleteValue deletes a value from the store
func (vka *ValKeyAdapter) DeleteValue(ctx context.Context, key string) error {
	return vka.store.DeleteValue(ctx, key)
}

// Close closes the store connection
func (vka *ValKeyAdapter) Close() error {
	return vka.store.Close()
}
