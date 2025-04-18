package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// MockKVStore provides a mock implementation of the KVStore interface for testing
type MockKVStore struct {
	data map[string]string
}

// NewMockKVStore creates a new mock store
func NewMockKVStore() *MockKVStore {
	return &MockKVStore{
		data: make(map[string]string),
	}
}

// SetValue implements KVStore.SetValue
func (m *MockKVStore) SetValue(ctx context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

// GetValue implements KVStore.GetValue
func (m *MockKVStore) GetValue(ctx context.Context, key string) (ValkeyResponse, error) {
	if value, ok := m.data[key]; ok {
		return ValkeyResponse{
			Message: Message{
				Value: value,
			},
		}, nil
	}
	return ValkeyResponse{}, nil
}

// Close implements KVStore.Close
func (m *MockKVStore) Close() error {
	return nil
}

func TestResultStorage_Store(t *testing.T) {
	// Create a mock store
	mockStore := NewMockKVStore()

	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create the result storage
	storage := NewResultStorage(mockStore, logger)

	// Create a test result
	result := &CommandResult{
		CommandID:   "cmd-123",
		AgentID:     "agent-456",
		Type:        "shell",
		Status:      "success",
		ExecutedAt:  time.Now().Add(-5 * time.Second),
		CompletedAt: time.Now(),
		Duration:    5000,
		ExitCode:    0,
		Output:      "Command output",
		Error:       "",
		Metadata:    map[string]string{"key": "value"},
		Data:        map[string]interface{}{"data_key": "data_value"},
	}

	// Store the result
	err := storage.Store(context.Background(), result)
	require.NoError(t, err)

	// Verify the result was stored
	key := CommandResultPrefix + result.CommandID
	resp, err := mockStore.GetValue(context.Background(), key)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Message.Value)

	// Unmarshal the stored result
	var storedResult CommandResult
	err = json.Unmarshal([]byte(resp.Message.Value), &storedResult)
	require.NoError(t, err)

	// Verify the result matches
	assert.Equal(t, result.CommandID, storedResult.CommandID)
	assert.Equal(t, result.AgentID, storedResult.AgentID)
	assert.Equal(t, result.Type, storedResult.Type)
	assert.Equal(t, result.Status, storedResult.Status)
	assert.Equal(t, result.ExitCode, storedResult.ExitCode)
	assert.Equal(t, result.Output, storedResult.Output)
	assert.Equal(t, result.Error, storedResult.Error)

	// Verify agent index was created
	agentKey := AgentResultPrefix + result.AgentID + ":" + result.CommandID
	resp, err = mockStore.GetValue(context.Background(), agentKey)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Message.Value)

	// Verify time index was created
	timeIndexPrefix := ResultIndexPrefix + result.CompletedAt.Unix()
	found := false
	for k := range mockStore.data {
		if k == timeIndexPrefix+":"+result.CommandID {
			found = true
			break
		}
	}
	assert.True(t, found, "Time index key not found")
}

func TestResultStorage_Get(t *testing.T) {
	// Create a mock store
	mockStore := NewMockKVStore()

	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create the result storage
	storage := NewResultStorage(mockStore, logger)

	// Create a test result
	result := &CommandResult{
		CommandID:   "cmd-123",
		AgentID:     "agent-456",
		Type:        "shell",
		Status:      "success",
		ExecutedAt:  time.Now().Add(-5 * time.Second),
		CompletedAt: time.Now(),
		Duration:    5000,
		ExitCode:    0,
		Output:      "Command output",
		Error:       "",
		Metadata:    map[string]string{"key": "value"},
		Data:        map[string]interface{}{"data_key": "data_value"},
	}

	// Store the result
	err := storage.Store(context.Background(), result)
	require.NoError(t, err)

	// Retrieve the result
	retrievedResult, err := storage.Get(context.Background(), result.CommandID)
	require.NoError(t, err)

	// Verify the result matches
	assert.Equal(t, result.CommandID, retrievedResult.CommandID)
	assert.Equal(t, result.AgentID, retrievedResult.AgentID)
	assert.Equal(t, result.Type, retrievedResult.Type)
	assert.Equal(t, result.Status, retrievedResult.Status)
	assert.Equal(t, result.ExitCode, retrievedResult.ExitCode)
	assert.Equal(t, result.Output, retrievedResult.Output)
	assert.Equal(t, result.Error, retrievedResult.Error)
}

func TestResultStorage_StoreBatch(t *testing.T) {
	// Create a mock store
	mockStore := NewMockKVStore()

	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create the result storage
	storage := NewResultStorage(mockStore, logger)

	// Create test results
	results := []*CommandResult{
		{
			CommandID:   "cmd-123",
			AgentID:     "agent-456",
			Type:        "shell",
			Status:      "success",
			ExecutedAt:  time.Now().Add(-5 * time.Second),
			CompletedAt: time.Now(),
			ExitCode:    0,
			Output:      "Command output 1",
		},
		{
			CommandID:   "cmd-456",
			AgentID:     "agent-456",
			Type:        "shell",
			Status:      "error",
			ExecutedAt:  time.Now().Add(-10 * time.Second),
			CompletedAt: time.Now(),
			ExitCode:    1,
			Output:      "Command output 2",
			Error:       "Command error",
		},
	}

	// Store the results
	err := storage.StoreBatch(context.Background(), results)
	require.NoError(t, err)

	// Verify both results were stored
	for _, result := range results {
		retrievedResult, err := storage.Get(context.Background(), result.CommandID)
		require.NoError(t, err)
		assert.Equal(t, result.CommandID, retrievedResult.CommandID)
		assert.Equal(t, result.Status, retrievedResult.Status)
		assert.Equal(t, result.Output, retrievedResult.Output)
	}
}

func TestResultStorage_ValidationErrors(t *testing.T) {
	// Create a mock store
	mockStore := NewMockKVStore()

	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create the result storage
	storage := NewResultStorage(mockStore, logger)

	// Test nil result
	err := storage.Store(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil result")

	// Test empty command ID
	err = storage.Store(context.Background(), &CommandResult{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command ID is required")

	// Test Get with empty command ID
	_, err = storage.Get(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command ID is required")

	// Test GetByAgent with empty agent ID
	_, err = storage.GetByAgent(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")

	// Test PurgeByCommandID with empty command ID
	err = storage.PurgeByCommandID(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command ID is required")
}
