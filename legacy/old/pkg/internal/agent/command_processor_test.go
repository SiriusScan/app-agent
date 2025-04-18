package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	scanpb "github.com/SiriusScan/app-agent/proto/scan/v1"
	"github.com/SiriusScan/go-api/sirius/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockStore is a mock implementation of store.KVStore
type MockStore struct {
	mock.Mock
}

func (m *MockStore) SetValue(ctx context.Context, key string, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockStore) GetValue(ctx context.Context, key string) (store.ValkeyResponse, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(store.ValkeyResponse), args.Error(1)
}

func (m *MockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockServer is a mock implementation of CommandQueueServer
type MockServer struct {
	mock.Mock
}

func (m *MockServer) QueueCommand(agentID string, cmd *scanpb.Command) error {
	args := m.Called(agentID, cmd)
	return args.Error(0)
}

func TestCommandProcessor_ProcessCommand(t *testing.T) {
	// Create test dependencies
	logger := zap.NewNop()
	mockStore := new(MockStore)
	mockServer := new(MockServer)

	// Create command processor
	processor := NewCommandProcessor(mockServer, mockStore, logger)

	// Test cases
	tests := []struct {
		name    string
		command map[string]interface{}
		setup   func()
		assert  func(error)
	}{
		{
			name: "successful scan command",
			command: map[string]interface{}{
				"command_id": "test-cmd-1",
				"type":       "scan",
				"agent_id":   "agent-1",
				"args": map[string]string{
					"scan_type": "vulnerability",
					"target":    "localhost",
				},
				"timeout": int64(300),
			},
			setup: func() {
				// Expect state to be stored
				mockStore.On("SetValue", mock.Anything, "command:test-cmd-1", mock.MatchedBy(func(value string) bool {
					var state CommandState
					if err := json.Unmarshal([]byte(value), &state); err != nil {
						return false
					}
					return state.Status == "pending" && state.CommandID == "test-cmd-1"
				})).Return(nil)

				// Expect command to be queued
				mockServer.On("QueueCommand", "agent-1", mock.MatchedBy(func(cmd *scanpb.Command) bool {
					return cmd.CommandId == "test-cmd-1" && cmd.Type == scanpb.CommandType_COMMAND_TYPE_SCAN
				})).Return(nil)
			},
			assert: func(err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "invalid command - missing ID",
			command: map[string]interface{}{
				"type":     "scan",
				"agent_id": "agent-1",
				"args":     map[string]string{},
			},
			setup: func() {},
			assert: func(err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "command ID is required")
			},
		},
		{
			name: "command queue failure",
			command: map[string]interface{}{
				"command_id": "test-cmd-2",
				"type":       "scan",
				"agent_id":   "agent-1",
				"args":       map[string]string{},
			},
			setup: func() {
				mockStore.On("SetValue", mock.Anything, "command:test-cmd-2", mock.Anything).Return(nil)
				mockServer.On("QueueCommand", "agent-1", mock.Anything).Return(assert.AnError)
				mockStore.On("SetValue", mock.Anything, "command:test-cmd-2", mock.MatchedBy(func(value string) bool {
					var state CommandState
					if err := json.Unmarshal([]byte(value), &state); err != nil {
						return false
					}
					return state.Status == "failed"
				})).Return(nil)
			},
			assert: func(err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to queue command")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			tt.setup()

			// Convert command to JSON
			cmdJSON, err := json.Marshal(tt.command)
			assert.NoError(t, err)

			// Process command
			err = processor.ProcessCommand(context.Background(), cmdJSON)

			// Assert result
			tt.assert(err)

			// Verify expectations
			mockStore.AssertExpectations(t)
			mockServer.AssertExpectations(t)
		})
	}
}

func TestCommandProcessor_HandleCommandResult(t *testing.T) {
	mockStore := new(MockStore)
	mockServer := new(MockServer)
	logger := zap.NewNop()

	processor := NewCommandProcessor(mockServer, mockStore, logger)

	// Test cases
	tests := []struct {
		name    string
		result  *scanpb.CommandResult
		setup   func()
		wantErr bool
	}{
		{
			name: "successful scan result",
			result: &scanpb.CommandResult{
				CommandId:       "test-cmd-1",
				AgentId:         "test-agent-1",
				ExecutionStatus: "success",
				Output:          "scan completed",
				Metadata: map[string]string{
					"scan_id": "scan-123",
				},
				ResultDetails: &scanpb.CommandResult_Scan{
					Scan: &scanpb.ScanResult{
						ScanId: "scan-123",
					},
				},
			},
			setup: func() {
				mockStore.On("GetValue", mock.Anything, "task:test-cmd-1").Return(
					store.ValkeyResponse{
						Message: store.ValkeyValue{
							Value: `{"command_id":"test-cmd-1","type":"SCAN","agent_id":"test-agent-1"}`,
						},
					}, nil)
				mockStore.On("SetValue", mock.Anything, "task:test-cmd-1", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed scan result",
			result: &scanpb.CommandResult{
				CommandId:       "test-cmd-2",
				AgentId:         "test-agent-1",
				ExecutionStatus: "error",
				Output:          "scan failed",
				Metadata: map[string]string{
					"error": "target unreachable",
				},
			},
			setup: func() {
				mockStore.On("GetValue", mock.Anything, "task:test-cmd-2").Return(
					store.ValkeyResponse{
						Message: store.ValkeyValue{
							Value: `{"command_id":"test-cmd-2","type":"SCAN","agent_id":"test-agent-1"}`,
						},
					}, nil)
				mockStore.On("SetValue", mock.Anything, "task:test-cmd-2", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "command not found",
			result: &scanpb.CommandResult{
				CommandId:       "non-existent",
				AgentId:         "test-agent-1",
				ExecutionStatus: "success",
			},
			setup: func() {
				mockStore.On("GetValue", mock.Anything, "task:non-existent").Return(
					store.ValkeyResponse{}, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			tt.setup()

			// Test the function
			err := processor.HandleCommandResult(context.Background(), tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCommandResult() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify expectations
			mockStore.AssertExpectations(t)
		})
	}
}

func TestCommandProcessor_validateCommand(t *testing.T) {
	mockStore := new(MockStore)
	mockServer := new(MockServer)
	logger := zap.NewNop()

	processor := NewCommandProcessor(mockServer, mockStore, logger)

	// Test cases
	tests := []struct {
		name    string
		task    *TaskMessage
		wantErr bool
	}{
		{
			name: "valid scan command",
			task: &TaskMessage{
				ID:      "test-cmd-1",
				Type:    "SCAN",
				AgentID: "test-agent-1",
				Args: map[string]string{
					"target": "localhost",
				},
			},
			wantErr: false,
		},
		{
			name: "missing agent ID",
			task: &TaskMessage{
				ID:   "test-cmd-2",
				Type: "SCAN",
				Args: map[string]string{
					"target": "localhost",
				},
			},
			wantErr: true,
		},
		{
			name: "missing command ID",
			task: &TaskMessage{
				Type:    "SCAN",
				AgentID: "test-agent-1",
				Args: map[string]string{
					"target": "localhost",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.validateCommand(tt.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandProcessor_Timeout(t *testing.T) {
	mockStore := new(MockStore)
	mockServer := new(MockServer)
	logger := zap.NewNop()

	processor := NewCommandProcessor(mockServer, mockStore, logger)

	// Add task to tasks map
	processor.mu.Lock()
	processor.tasks["test-cmd-1"] = &TaskMessage{
		ID:      "test-cmd-1",
		AgentID: "test-agent-1",
		Status:  "pending",
	}
	processor.mu.Unlock()

	// Test timeout
	ctx := context.Background()
	timeout := 100 * time.Millisecond
	processor.monitorTimeout(ctx, "test-cmd-1", timeout)

	// Wait for timeout
	time.Sleep(2 * timeout)

	// Verify task was removed and status updated
	processor.mu.RLock()
	task, exists := processor.tasks["test-cmd-1"]
	processor.mu.RUnlock()

	assert.False(t, exists, "task should be removed after timeout")
	if task != nil {
		assert.Equal(t, "timeout", task.Status, "task status should be timeout")
		assert.Equal(t, "command execution timed out", task.Error, "task error should be set")
	}

	// Verify store was updated
	mockStore.AssertExpectations(t)
}
