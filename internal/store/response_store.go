package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/command"
	"github.com/SiriusScan/go-api/sirius/store"
)

// ResponseStore defines the interface for storing and retrieving command responses
type ResponseStore interface {
	// Store stores a command response
	Store(ctx context.Context, response *command.CommandResponse) error
	// Get retrieves a command response by its key
	Get(ctx context.Context, key string) (*command.CommandResponse, error)
	// List retrieves all command responses for an agent
	List(ctx context.Context, agentID string) ([]*command.CommandResponse, error)
	// Delete removes a command response
	Delete(ctx context.Context, key string) error
	// Close closes the store connection
	Close() error
}

// valkeyResponseStore implements ResponseStore using the KVStore interface
type valkeyResponseStore struct {
	store store.KVStore
}

// NewResponseStore creates a new ResponseStore using the Valkey KVStore
func NewResponseStore() (ResponseStore, error) {
	kvStore, err := store.NewValkeyStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey store: %w", err)
	}

	return &valkeyResponseStore{
		store: kvStore,
	}, nil
}

// Store stores a command response in Valkey
func (s *valkeyResponseStore) Store(ctx context.Context, response *command.CommandResponse) error {
	if err := response.Validate(); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to serialize response: %w", err)
	}

	key := fmt.Sprintf("cmd:response:%s:%s", response.AgentID, response.CommandID)
	err = s.store.SetValue(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to store response: %w", err)
	}

	return nil
}

// Get retrieves a command response from Valkey
func (s *valkeyResponseStore) Get(ctx context.Context, key string) (*command.CommandResponse, error) {
	resp, err := s.store.GetValue(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	// Check for empty value
	if resp.Message.Value == "" {
		return nil, nil
	}

	var response command.CommandResponse
	err = json.Unmarshal([]byte(resp.Message.Value), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return &response, nil
}

// List retrieves all command responses for an agent by scanning keys with pattern
func (s *valkeyResponseStore) List(ctx context.Context, agentID string) ([]*command.CommandResponse, error) {
	pattern := fmt.Sprintf("cmd:response:%s:*", agentID)
	keys, err := s.store.ListKeys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list response keys: %w", err)
	}

	var responses []*command.CommandResponse
	for _, key := range keys {
		response, err := s.Get(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get response for key %s: %w", key, err)
		}
		if response != nil {
			responses = append(responses, response)
		}
	}

	return responses, nil
}

// Delete removes a command response from Valkey
func (s *valkeyResponseStore) Delete(ctx context.Context, key string) error {
	err := s.store.DeleteValue(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete response: %w", err)
	}
	return nil
}

// Close closes the Valkey client connection
func (s *valkeyResponseStore) Close() error {
	return s.store.Close()
}
