package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/store"
	"github.com/SiriusScan/app-agent/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Server represents the gRPC server implementation
type Server struct {
	proto.UnimplementedAgentServiceServer
	logger        *zap.Logger
	config        *config.Config
	grpcServer    *grpc.Server
	agents        map[string]*AgentConnection
	mu            sync.RWMutex
	resultStorage store.ResultStorageInterface
	processedCmds map[string]bool // Track all processed command IDs
}

// AgentConnection represents a connected agent
type AgentConnection struct {
	AgentID   string
	Status    proto.AgentStatus
	Stream    proto.AgentService_ConnectServer
	LastSeen  time.Time
	Metadata  map[string]string
	Connected bool
}

// NewServer creates a new gRPC server instance
func NewServer(logger *zap.Logger, cfg *config.Config, kvStore store.KVStore) (*Server, error) {
	// Create the result storage
	resultStorage := store.NewResultStorage(kvStore, logger)

	s := &Server{
		logger:        logger.Named("grpc_server"),
		config:        cfg,
		agents:        make(map[string]*AgentConnection),
		resultStorage: resultStorage,
		processedCmds: make(map[string]bool),
	}

	// Configure server options
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              1 * time.Minute,
			Timeout:           20 * time.Second,
		}),
	}

	// Add TLS if configured
	if cfg.TLSEnabled {
		creds, err := loadTLSCredentials(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Create gRPC server
	s.grpcServer = grpc.NewServer(opts...)
	proto.RegisterAgentServiceServer(s.grpcServer, s)

	return s, nil
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.config.GRPCServerAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("🚀 Starting gRPC server",
		zap.String("address", s.config.GRPCServerAddr),
		zap.Bool("tls_enabled", s.config.TLSEnabled),
	)

	// Start server in a goroutine
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Error("Failed to serve gRPC",
				zap.Error(err),
			)
		}
	}()

	// Monitor context for shutdown
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("🛑 Stopping gRPC server")
	s.grpcServer.GracefulStop()
}

// Connect implements the Connect RPC method
func (s *Server) Connect(stream proto.AgentService_ConnectServer) error {
	// First message must be authentication
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive initial message: %w", err)
	}

	// Verify authentication
	auth := msg.GetAuth()
	if auth == nil {
		return fmt.Errorf("first message must be authentication")
	}

	// Register agent connection
	conn := &AgentConnection{
		AgentID:   auth.AgentId,
		Status:    proto.AgentStatus_AGENT_STATUS_READY,
		Stream:    stream,
		LastSeen:  time.Now(),
		Metadata:  auth.Capabilities,
		Connected: true,
	}

	s.mu.Lock()
	s.agents[auth.AgentId] = conn
	s.mu.Unlock()

	s.logger.Info("✅ Agent connected",
		zap.String("agent_id", auth.AgentId),
		zap.Any("capabilities", auth.Capabilities),
	)

	// Message handling loop
	for {
		msg, err := stream.Recv()
		if err != nil {
			s.mu.Lock()
			conn.Connected = false
			s.mu.Unlock()
			s.logger.Warn("❌ Agent disconnected",
				zap.String("agent_id", conn.AgentID),
				zap.Error(err),
			)
			return err
		}

		conn.LastSeen = time.Now()

		// Handle different message types
		switch payload := msg.GetPayload().(type) {
		case *proto.AgentMessage_Heartbeat:
			s.handleHeartbeat(conn, payload.Heartbeat)
		case *proto.AgentMessage_Result:
			s.handleStreamResult(conn, payload.Result)
		}
	}
}

// ReportResult implements the ReportResult RPC method
func (s *Server) ReportResult(ctx context.Context, result *proto.ResultMessage) (*proto.ResultResponse, error) {
	s.logger.Info("📥 Received result via ReportResult method",
		zap.String("command_id", result.CommandId),
		zap.String("agent_id", result.AgentId),
		zap.String("status", result.Status.String()),
	)

	// Create a CommandResult from the ResultMessage
	cmdResult := &store.CommandResult{
		CommandID:   result.CommandId,
		AgentID:     result.AgentId,
		Type:        result.Type,
		Status:      result.Status.String(),
		CompletedAt: time.Unix(0, result.CompletedAt),
		ExitCode:    result.ExitCode,
		Output:      result.Output,
		Error:       result.ErrorMessage,
		Metadata:    result.Data,
	}

	// Store the result
	if err := s.resultStorage.Store(ctx, cmdResult); err != nil {
		s.logger.Error("❌ Failed to store result",
			zap.String("command_id", result.CommandId),
			zap.String("agent_id", result.AgentId),
			zap.Error(err),
		)

		return &proto.ResultResponse{
			CommandId: result.CommandId,
			Accepted:  false,
			Message:   fmt.Sprintf("Failed to store result: %v", err),
		}, nil
	}

	s.logger.Info("✅ Result stored successfully",
		zap.String("command_id", result.CommandId),
		zap.String("agent_id", result.AgentId),
		zap.String("status", result.Status.String()),
	)

	// Mark command as processed
	s.mu.Lock()
	s.processedCmds[result.CommandId] = true
	s.mu.Unlock()

	return &proto.ResultResponse{
		CommandId: result.CommandId,
		Accepted:  true,
		Message:   "Result received and stored successfully",
	}, nil
}

// handleHeartbeat processes agent heartbeat messages
func (s *Server) handleHeartbeat(conn *AgentConnection, heartbeat *proto.HeartbeatMessage) {
	s.logger.Debug("💓 Received heartbeat",
		zap.String("agent_id", conn.AgentID),
		zap.Time("timestamp", time.Unix(0, heartbeat.Timestamp)),
		zap.Any("metrics", heartbeat.Metrics),
	)
}

// handleStreamResult processes results received through the stream
func (s *Server) handleStreamResult(conn *AgentConnection, result *proto.ResultMessage) {
	s.logger.Info("📊 Received stream result",
		zap.String("command_id", result.CommandId),
		zap.String("agent_id", conn.AgentID),
		zap.String("status", result.Status.String()),
	)

	// Check if we've already processed this command
	s.mu.Lock()
	alreadyProcessed := s.processedCmds[result.CommandId]
	s.mu.Unlock()

	if alreadyProcessed {
		s.logger.Info("Skipping already processed command result",
			zap.String("command_id", result.CommandId),
			zap.String("agent_id", conn.AgentID))
		return
	}

	// Create a context for processing this result
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Reuse the ReportResult logic to store the result
	resp, err := s.ReportResult(ctx, result)
	if err != nil {
		s.logger.Error("❌ Failed to process stream result",
			zap.String("command_id", result.CommandId),
			zap.String("agent_id", conn.AgentID),
			zap.Error(err),
		)
		return
	}

	if !resp.Accepted {
		s.logger.Warn("⚠️ Stream result not accepted",
			zap.String("command_id", result.CommandId),
			zap.String("agent_id", conn.AgentID),
			zap.String("message", resp.Message),
		)
		return
	}
}

// loadTLSCredentials loads TLS credentials from the configuration
func loadTLSCredentials(cfg *config.Config) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}), nil
}

// SendCommandToAgent sends a command to a specific agent
func (s *Server) SendCommandToAgent(agentID string, cmd *proto.CommandMessage) error {
	s.mu.RLock()
	agent, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists || !agent.Connected {
		return fmt.Errorf("agent not connected: %s", agentID)
	}

	if err := agent.Stream.Send(cmd); err != nil {
		s.logger.Error("Failed to send command to agent",
			zap.String("agent_id", agentID),
			zap.String("command_id", cmd.CommandId),
			zap.Error(err))
		return err
	}

	s.logger.Info("✅ Command sent to agent",
		zap.String("agent_id", agentID),
		zap.String("command_id", cmd.CommandId),
		zap.String("type", cmd.Type))

	return nil
}
