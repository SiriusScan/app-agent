package grpc

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/SiriusScan/app-agent/proto/scan"
)

// Client represents the gRPC client for the Test Agent
type Client struct {
	logger     *zap.Logger
	conn       *grpc.ClientConn
	serverAddr string
	client     pb.AgentServiceClient
}

// NewClient creates a new gRPC client instance
func NewClient(serverAddr string, logger *zap.Logger) (*Client, error) {
	logger.Info("Creating gRPC client", zap.String("server_address", serverAddr))

	return &Client{
		logger:     logger,
		serverAddr: serverAddr,
	}, nil
}

// Connect establishes a connection to the gRPC server
func (c *Client) Connect(ctx context.Context) error {
	// Set up connection options - for now using insecure for simplicity
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	// Set a timeout for the dial
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Connect to the server
	var err error
	c.conn, err = grpc.DialContext(dialCtx, c.serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.logger.Info("Connected to gRPC server")

	// Create the client stub
	c.client = pb.NewAgentServiceClient(c.conn)

	return nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		c.logger.Info("Closing gRPC connection")
		return c.conn.Close()
	}
	return nil
}

// Ping sends a simple ping to the server to verify the connection
func (c *Client) Ping(ctx context.Context, agentID string) (string, error) {
	req := &pb.PingRequest{
		AgentId: agentID,
	}

	resp, err := c.client.Ping(ctx, req)
	if err != nil {
		return "", fmt.Errorf("ping failed: %w", err)
	}

	return resp.Message, nil
}

// ConnectStream establishes a bidirectional stream connection with the server
func (c *Client) ConnectStream(ctx context.Context, agentID string) (pb.AgentService_ConnectClient, error) {
	c.logger.Info("Establishing bidirectional stream", zap.String("agent_id", agentID))

	// Create metadata with agent ID
	md := metadata.New(map[string]string{
		"agent-id": agentID,
	})

	// Create context with metadata
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	// Connect to the stream
	stream, err := c.client.Connect(streamCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect stream: %w", err)
	}

	// Send initial message to establish connection
	initialMsg := &pb.AgentMessage{
		AgentId: agentID,
		Status:  pb.AgentStatus_IDLE,
		Payload: &pb.AgentMessage_Heartbeat{
			Heartbeat: &pb.HeartbeatMessage{
				Timestamp:      time.Now().Unix(),
				CpuUsage:       0.0,
				MemoryUsage:    0.0,
				ActiveCommands: 0,
			},
		},
	}

	if err := stream.Send(initialMsg); err != nil {
		return nil, fmt.Errorf("failed to send initial message: %w", err)
	}

	c.logger.Info("Stream connection established", zap.String("agent_id", agentID))
	return stream, nil
}
