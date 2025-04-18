package store

import (
	"context"
	"fmt"

	goAPIStore "github.com/SiriusScan/go-api/sirius/store"
)

// StoreAdapter adapts the Go API's KVStore to our internal KVStore interface
type StoreAdapter struct {
	goStore goAPIStore.KVStore
}

// NewStoreAdapter creates a new adapter for the Go API's KVStore
func NewStoreAdapter(goStore goAPIStore.KVStore) KVStore {
	return &StoreAdapter{
		goStore: goStore,
	}
}

// SetValue implements KVStore.SetValue by delegating to the Go API's KVStore
func (a *StoreAdapter) SetValue(ctx context.Context, key, value string) error {
	return a.goStore.SetValue(ctx, key, value)
}

// GetValue implements KVStore.GetValue by delegating to the Go API's KVStore and converting the result
func (a *StoreAdapter) GetValue(ctx context.Context, key string) (ValkeyResponse, error) {
	goResp, err := a.goStore.GetValue(ctx, key)
	if err != nil {
		return ValkeyResponse{}, err
	}

	// Convert the response
	resp := ValkeyResponse{
		Message: Message{
			Value: goResp.Message.Value,
		},
	}

	return resp, nil
}

// Close implements KVStore.Close by delegating to the Go API's KVStore
func (a *StoreAdapter) Close() error {
	return a.goStore.Close()
}

// Keys implements KVStore.Keys by delegating to the Go API's KVStore or providing a fallback
func (a *StoreAdapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	// Check if the underlying store implements a Keys method via type assertion
	if keyser, ok := a.goStore.(interface {
		Keys(ctx context.Context, pattern string) ([]string, error)
	}); ok {
		return keyser.Keys(ctx, pattern)
	}

	// Fallback: Log a warning that this operation is not supported
	return nil, fmt.Errorf("Keys operation not supported by underlying store")
}

// Del implements KVStore.Del by delegating to the Go API's KVStore or providing a fallback
func (a *StoreAdapter) Del(ctx context.Context, key string) error {
	// Check if the underlying store implements a Del method via type assertion
	if deler, ok := a.goStore.(interface {
		Del(ctx context.Context, key string) error
	}); ok {
		return deler.Del(ctx, key)
	}

	// Fallback: Log a warning that this operation is not supported
	return fmt.Errorf("Del operation not supported by underlying store")
}
