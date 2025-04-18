package agent

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	pb "github.com/SiriusScan/app-agent/proto/scan/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func setupTest(t *testing.T) (*Server, pb.AgentServiceClient, func()) {
	t.Helper()

	// Create a new server
	grpcServer := grpc.NewServer()
	agentServer := NewServer(slog.Default())
	pb.RegisterAgentServiceServer(grpcServer, agentServer)

	// Start serving on the listener in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Errorf("error serving server: %v", err)
		}
	}()

	// Create a client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := pb.NewAgentServiceClient(conn)

	cleanup := func() {
		conn.Close()
		grpcServer.Stop()
	}

	return agentServer, client, cleanup
}

func TestServer_Connect(t *testing.T) {
	_, client, cleanup := setupTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test successful connection
	t.Run("successful connection", func(t *testing.T) {
		req := &pb.ConnectRequest{
			AgentId:  "test-agent-1",
			Hostname: "test-host",
			Platform: "linux",
			Metadata: map[string]string{"version": "1.0.0"},
		}

		stream, err := client.Connect(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, stream)

		// Cleanup
		stream.CloseSend()
	})

	// Test invalid request
	t.Run("invalid request", func(t *testing.T) {
		req := &pb.ConnectRequest{
			// Missing agent_id
			Hostname: "test-host",
			Platform: "linux",
		}

		stream, err := client.Connect(ctx, req)
		require.NoError(t, err)

		// Wait for error response
		_, err = stream.Recv()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent_id is required")
	})
}

func TestServer_ReportResult(t *testing.T) {
	_, client, cleanup := setupTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connect an agent
	connectReq := &pb.ConnectRequest{
		AgentId:  "test-agent-1",
		Hostname: "test-host",
		Platform: "linux",
	}

	stream, err := client.Connect(ctx, connectReq)
	require.NoError(t, err)
	defer stream.CloseSend()

	// Test successful result reporting
	t.Run("successful result", func(t *testing.T) {
		result := &pb.CommandResult{
			CommandId: "cmd-1",
			AgentId:   "test-agent-1",
			Status:    "success",
			Output:    "test output",
			Metadata:  map[string]string{"duration": "1s"},
			Timestamp: time.Now().Unix(),
		}

		ack, err := client.ReportResult(ctx, result)
		require.NoError(t, err)
		assert.Equal(t, result.CommandId, ack.CommandId)
		assert.Equal(t, "accepted", ack.Status)
	})

	// Test invalid result
	t.Run("invalid result", func(t *testing.T) {
		result := &pb.CommandResult{
			// Missing command_id and agent_id
			Status: "success",
			Output: "test output",
		}

		ack, err := client.ReportResult(ctx, result)
		assert.Error(t, err)
		assert.Nil(t, ack)
		assert.Contains(t, err.Error(), "command_id is required")
	})

	// Test non-existent agent
	t.Run("non-existent agent", func(t *testing.T) {
		result := &pb.CommandResult{
			CommandId: "cmd-1",
			AgentId:   "non-existent",
			Status:    "success",
			Output:    "test output",
		}

		ack, err := client.ReportResult(ctx, result)
		assert.Error(t, err)
		assert.Nil(t, ack)
		assert.Contains(t, err.Error(), "agent not found")
	})
}

func TestServer_SendCommand(t *testing.T) {
	agentServer, client, cleanup := setupTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect an agent
	connectReq := &pb.ConnectRequest{
		AgentId:  "test-agent-1",
		Hostname: "test-host",
		Platform: "linux",
	}

	stream, err := client.Connect(ctx, connectReq)
	require.NoError(t, err)
	defer stream.CloseSend()

	// Test sending command
	t.Run("send command", func(t *testing.T) {
		cmd := &pb.Command{
			CommandId: "cmd-1",
			Type:      "scan",
			Args:      map[string]string{"target": "localhost"},
			Timeout:   60,
		}

		err := agentServer.SendCommand("test-agent-1", cmd)
		require.NoError(t, err)

		// Verify command received by agent
		received, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, cmd.CommandId, received.CommandId)
		assert.Equal(t, cmd.Type, received.Type)
		assert.Equal(t, cmd.Args, received.Args)
		assert.Equal(t, cmd.Timeout, received.Timeout)
	})

	// Test sending to non-existent agent
	t.Run("non-existent agent", func(t *testing.T) {
		cmd := &pb.Command{
			CommandId: "cmd-2",
			Type:      "scan",
		}

		err := agentServer.SendCommand("non-existent", cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found")
	})
}

func TestServer_ListAgents(t *testing.T) {
	agentServer, client, cleanup := setupTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initially no agents
	agents := agentServer.ListAgents()
	assert.Empty(t, agents)

	// Connect two agents
	for i, id := range []string{"agent-1", "agent-2"} {
		req := &pb.ConnectRequest{
			AgentId:  id,
			Hostname: "host-" + id,
			Platform: "linux",
		}

		stream, err := client.Connect(ctx, req)
		require.NoError(t, err)
		defer stream.CloseSend()

		// Verify agent count
		agents = agentServer.ListAgents()
		assert.Len(t, agents, i+1)
	}

	// Verify agent details
	agents = agentServer.ListAgents()
	assert.Len(t, agents, 2)
	assert.Contains(t, []string{agents[0].ID, agents[1].ID}, "agent-1")
	assert.Contains(t, []string{agents[0].ID, agents[1].ID}, "agent-2")
}

func TestServer_GetAgent(t *testing.T) {
	agentServer, client, cleanup := setupTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect an agent
	req := &pb.ConnectRequest{
		AgentId:  "test-agent",
		Hostname: "test-host",
		Platform: "linux",
		Metadata: map[string]string{"version": "1.0.0"},
	}

	stream, err := client.Connect(ctx, req)
	require.NoError(t, err)
	defer stream.CloseSend()

	// Test getting existing agent
	t.Run("existing agent", func(t *testing.T) {
		agent, exists := agentServer.GetAgent("test-agent")
		assert.True(t, exists)
		assert.NotNil(t, agent)
		assert.Equal(t, "test-agent", agent.ID)
		assert.Equal(t, "test-host", agent.Hostname)
		assert.Equal(t, "linux", agent.Platform)
		assert.Equal(t, map[string]string{"version": "1.0.0"}, agent.Metadata)
	})

	// Test getting non-existent agent
	t.Run("non-existent agent", func(t *testing.T) {
		agent, exists := agentServer.GetAgent("non-existent")
		assert.False(t, exists)
		assert.Nil(t, agent)
	})
}
