package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/SiriusScan/app-agent/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Client represents a gRPC client for communicating with the server
type Client struct {
	logger *zap.Logger
	conn   *grpc.ClientConn
	client proto.AgentServiceClient
}

// NewClient creates a new gRPC client
func NewClient(serverAddr string, enableTLS bool, certFile string, logger *zap.Logger) (*Client, error) {
	var opts []grpc.DialOption

	// Configure TLS if enabled
	if enableTLS {
		if certFile == "" {
			return nil, fmt.Errorf("TLS enabled but no certificate file provided")
		}

		// Load certificate
		// TODO: Update to use os.ReadFile instead of deprecated ioutil.ReadFile
		certData, err := ioutil.ReadFile(certFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate file: %w", err)
		}

		// Create certificate pool
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(certData) {
			return nil, fmt.Errorf("failed to add certificate to pool")
		}

		// Create TLS config
		creds := credentials.NewTLS(&tls.Config{
			RootCAs: certPool,
		})

		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		// Use insecure connection if TLS is disabled
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add keepalive options
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}))

	// Connect to server
	logger.Info("Connecting to gRPC server", zap.String("addr", serverAddr))
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	// Create client
	client := proto.NewAgentServiceClient(conn)

	return &Client{
		logger: logger,
		conn:   conn,
		client: client,
	}, nil
}

// Connect establishes a bidirectional connection with the server
func (c *Client) Connect(ctx context.Context, agentID string, capabilities map[string]string) (proto.AgentService_ConnectClient, error) {
	// Create stream
	stream, err := c.client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Create authentication message as first message - this is required by the server
	authRequest := &proto.AuthRequest{
		AgentId:      agentID,
		AuthToken:    "", // Empty for now, could be configured later
		Capabilities: capabilities,
	}

	// Send authentication message
	authMessage := &proto.AgentMessage{
		AgentId:  agentID,
		Version:  capabilities["version"], // Version from capabilities
		Status:   proto.AgentStatus_AGENT_STATUS_READY,
		Metadata: capabilities,
		Payload: &proto.AgentMessage_Auth{
			Auth: authRequest,
		},
	}

	if err := stream.Send(authMessage); err != nil {
		return nil, fmt.Errorf("failed to send authentication message: %w", err)
	}

	// Start heartbeat goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Create heartbeat message
				metrics := map[string]string{
					"cpu":    "0.0",
					"memory": "0.0",
				}

				heartbeat := &proto.HeartbeatMessage{
					Timestamp: time.Now().UnixNano(),
					Metrics:   metrics,
				}

				// Send heartbeat message
				heartbeatMsg := &proto.AgentMessage{
					AgentId:  agentID,
					Version:  capabilities["version"],
					Status:   proto.AgentStatus_AGENT_STATUS_READY,
					Metadata: capabilities,
					Payload: &proto.AgentMessage_Heartbeat{
						Heartbeat: heartbeat,
					},
				}

				if err := stream.Send(heartbeatMsg); err != nil {
					c.logger.Error("Failed to send heartbeat", zap.Error(err))
					return
				}

				c.logger.Debug("Sent heartbeat", zap.Int64("timestamp", heartbeat.Timestamp))
			}
		}
	}()

	return stream, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ReportResult sends a result message to the server
func (c *Client) ReportResult(ctx context.Context, result *proto.ResultMessage) (*proto.ResultResponse, error) {
	return c.client.ReportResult(ctx, result)
}
