package internal

import (
	"context"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/grpc"
	"github.com/SiriusScan/app-agent/internal/store"
	pb "github.com/SiriusScan/go-api/sirius/agent/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/metadata"
)

func TestCommandResultStorage(t *testing.T) {
	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create a mock store
	mockStore := store.NewMockKVStore()

	// Create a result storage
	resultStorage := store.NewResultStorage(mockStore, logger)

	// Create a server config
	cfg := &config.Config{
		GRPC: config.GRPCConfig{
			Port: 50000, // Use a high port for testing
		},
	}

	// Create a gRPC server with the result storage
	server := grpc.NewServer(logger, cfg, mockStore)

	// Start the server in a separate goroutine
	go func() {
		if err := server.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Clean up when we're done
	defer server.Stop()

	// Create a test client stream
	stream := &mockAgentStream{
		ctx: metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs("agent-id", "test-agent-1"),
		),
		t: t,
	}

	// Connect the agent
	err := server.Connect(stream)
	require.NoError(t, err)

	// Report a command result
	cmdResult := &pb.ResultMessage{
		CommandId: "test-cmd-001",
		Status:    "success",
		ExitCode:  0,
		Output:    "Test command output",
		Type:      "shell",
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		CompletedAt: time.Now().UnixNano(),
	}

	// Add the result to the stream
	stream.results = append(stream.results, cmdResult)

	// Wait for result processing
	time.Sleep(200 * time.Millisecond)

	// Verify the result was stored in Valkey
	storedResult, err := resultStorage.Get(context.Background(), "test-cmd-001")
	require.NoError(t, err)

	// Check the stored result
	assert.Equal(t, "test-cmd-001", storedResult.CommandID)
	assert.Equal(t, "test-agent-1", storedResult.AgentID)
	assert.Equal(t, "success", storedResult.Status)
	assert.Equal(t, int32(0), storedResult.ExitCode)
	assert.Equal(t, "Test command output", storedResult.Output)
	assert.Equal(t, "shell", storedResult.Type)
	assert.Contains(t, storedResult.Metadata, "key1")
	assert.Equal(t, "value1", storedResult.Metadata["key1"])
}

// Mock agent stream for testing
type mockAgentStream struct {
	pb.Agent_ConnectServer
	ctx     context.Context
	t       *testing.T
	results []*pb.ResultMessage
}

func (m *mockAgentStream) Context() context.Context {
	return m.ctx
}

func (m *mockAgentStream) Send(cmd *pb.CommandMessage) error {
	m.t.Logf("Received command: %s", cmd.CommandId)
	return nil
}

func (m *mockAgentStream) Recv() (*pb.AgentMessage, error) {
	// If there are results to report, send them
	if len(m.results) > 0 {
		result := m.results[0]
		m.results = m.results[1:]

		return &pb.AgentMessage{
			Message: &pb.AgentMessage_Result{
				Result: result,
			},
		}, nil
	}

	// Otherwise, just send a heartbeat
	time.Sleep(100 * time.Millisecond)
	return &pb.AgentMessage{
		Message: &pb.AgentMessage_Heartbeat{
			Heartbeat: &pb.HeartbeatMessage{
				Timestamp: time.Now().UnixNano(),
			},
		},
	}, nil
}
