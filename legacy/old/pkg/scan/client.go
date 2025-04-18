package scan

import (
	"context"
	"time"

	pb "github.com/SiriusScan/app-agent/proto/scan/v1"
	"google.golang.org/grpc"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
)

// Client represents a gRPC client for the agent service
type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
}

func NewClient(conn *grpc.ClientConn) (*Client, error) {
	return &Client{
		conn:   conn,
		client: pb.NewAgentServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Connect establishes a connection with the agent service
func (c *Client) Connect(ctx context.Context, agentID string) (pb.AgentService_ConnectClient, error) {
	req := &pb.ConnectRequest{
		AgentId: agentID,
	}
	return c.client.Connect(ctx, req)
}

// ReportResult sends command execution results back to the server
func (c *Client) ReportResult(ctx context.Context, result *pb.CommandResult) error {
	_, err := c.client.ReportResult(ctx, result)
	return err
}
