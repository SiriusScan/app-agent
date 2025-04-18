package agent

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	pb "github.com/SiriusScan/app-agent/internal/proto/scan"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Agent represents a client agent that connects to the server
type Agent struct {
	ID           string
	Version      string
	Status       pb.AgentStatus
	Capabilities map[string]string

	logger *zap.Logger
	config *config.Config
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	stream pb.AgentService_ConnectClient

	mu          sync.RWMutex
	isConnected bool
	stopCh      chan struct{}

	commandHandlers map[pb.CommandType]CommandHandler
}

// CommandHandler defines the interface for handling different command types
type CommandHandler interface {
	Execute(ctx context.Context, cmd *pb.CommandMessage) (*pb.ResultMessage, error)
}

// AgentConfig holds the agent configuration
type AgentConfig struct {
	ID           string
	Version      string
	ServerAddr   string
	AuthToken    string
	TLSEnabled   bool
	TLSCertFile  string
	Capabilities map[string]string
}

// NewAgent creates a new agent with the provided configuration
func NewAgent(logger *zap.Logger, cfg *config.Config, agentCfg *AgentConfig) *Agent {
	if agentCfg.ID == "" {
		agentCfg.ID = uuid.New().String()
	}

	agent := &Agent{
		ID:              agentCfg.ID,
		Version:         agentCfg.Version,
		Status:          pb.AgentStatus_AGENT_STATUS_READY,
		Capabilities:    agentCfg.Capabilities,
		logger:          logger.Named("agent").With(zap.String("agent_id", agentCfg.ID)),
		config:          cfg,
		isConnected:     false,
		stopCh:          make(chan struct{}),
		commandHandlers: make(map[pb.CommandType]CommandHandler),
	}

	// Register default command handlers
	agent.registerDefaultHandlers()

	return agent
}

// Start connects to the server and starts the command handling loop
func (a *Agent) Start(ctx context.Context) error {
	a.logger.Info("🚀 Starting agent",
		zap.String("version", a.Version),
		zap.String("server_addr", a.config.GRPCServerAddr))

	// Connect to the server
	if err := a.connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Start command processing loop
	go a.processCommands(ctx)

	// Send heartbeats
	go a.sendHeartbeats(ctx)

	// Wait for context cancellation or stop signal
	select {
	case <-ctx.Done():
		a.logger.Info("🛑 Context cancelled, stopping agent")
	case <-a.stopCh:
		a.logger.Info("🛑 Stop signal received, stopping agent")
	}

	return a.disconnect()
}

// Stop stops the agent and disconnects from the server
func (a *Agent) Stop() {
	a.logger.Info("🛑 Stopping agent")
	close(a.stopCh)
}

// connect establishes a connection to the gRPC server
func (a *Agent) connect(ctx context.Context) error {
	a.logger.Info("🔄 Connecting to server", zap.String("address", a.config.GRPCServerAddr))

	// Configure dial options
	dialOpts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Add credentials
	if a.config.TLSEnabled {
		creds, err := credentials.NewClientTLSFromFile(a.config.TLSCertFile, "")
		if err != nil {
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Connect to the server
	conn, err := grpc.DialContext(ctx, a.config.GRPCServerAddr, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	a.conn = conn
	a.client = pb.NewAgentServiceClient(conn)

	// Establish the bidirectional stream
	stream, err := a.client.Connect(ctx)
	if err != nil {
		a.conn.Close()
		return fmt.Errorf("failed to establish stream: %w", err)
	}

	a.stream = stream

	// Send authentication message
	authMsg := &pb.AgentMessage{
		AgentId:  a.ID,
		Version:  a.Version,
		Status:   a.Status,
		Metadata: a.Capabilities,
		Payload: &pb.AgentMessage_Auth{
			Auth: &pb.AuthRequest{
				AgentId:      a.ID,
				AuthToken:    "dummy-token", // Replace with actual auth token
				Capabilities: a.Capabilities,
			},
		},
	}

	if err := stream.Send(authMsg); err != nil {
		a.disconnect()
		return fmt.Errorf("failed to send authentication message: %w", err)
	}

	a.mu.Lock()
	a.isConnected = true
	a.mu.Unlock()

	a.logger.Info("✅ Connected to server successfully")

	return nil
}

// disconnect closes the connection to the server
func (a *Agent) disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isConnected {
		return nil
	}

	a.logger.Info("🔌 Disconnecting from server")

	if a.stream != nil {
		a.stream.CloseSend()
	}

	if a.conn != nil {
		a.conn.Close()
	}

	a.isConnected = false
	a.stream = nil
	a.conn = nil

	a.logger.Info("✅ Disconnected from server")

	return nil
}

// processCommands listens for commands from the server and processes them
func (a *Agent) processCommands(ctx context.Context) {
	a.logger.Info("🎧 Starting command processing loop")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("🛑 Context cancelled, stopping command processing")
			return
		case <-a.stopCh:
			a.logger.Info("🛑 Stop signal received, stopping command processing")
			return
		default:
			// Check if we're connected
			a.mu.RLock()
			if !a.isConnected {
				a.mu.RUnlock()
				a.logger.Warn("⚠️ Not connected to server, waiting...")
				time.Sleep(5 * time.Second)
				continue
			}

			stream := a.stream
			a.mu.RUnlock()

			// Receive command from server
			cmd, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					a.logger.Info("📡 Stream closed by server")
				} else {
					a.logger.Error("❌ Error receiving command", zap.Error(err))
				}

				// Attempt to reconnect
				a.reconnect(ctx)
				continue
			}

			// Process the command
			a.logger.Info("📥 Received command",
				zap.String("command_id", cmd.CommandId),
				zap.String("type", cmd.Type.String()))

			// Handle the command in a separate goroutine
			go a.handleCommand(ctx, cmd)
		}
	}
}

// handleCommand processes a single command
func (a *Agent) handleCommand(ctx context.Context, cmd *pb.CommandMessage) {
	a.logger.Debug("🔄 Processing command",
		zap.String("command_id", cmd.CommandId),
		zap.String("type", cmd.Type.String()),
		zap.Any("parameters", cmd.Parameters))

	// Find handler for command type
	handler, ok := a.commandHandlers[cmd.Type]
	if !ok {
		a.logger.Error("❌ No handler for command type",
			zap.String("command_id", cmd.CommandId),
			zap.String("type", cmd.Type.String()))

		// Report error
		a.reportResult(ctx, &pb.ResultMessage{
			CommandId:    cmd.CommandId,
			AgentId:      a.ID,
			Status:       pb.ResultStatus_RESULT_STATUS_ERROR,
			ErrorMessage: fmt.Sprintf("unsupported command type: %s", cmd.Type.String()),
		})
		return
	}

	// Set status to busy
	a.updateStatus(pb.AgentStatus_AGENT_STATUS_BUSY)

	// Execute the command
	result, err := handler.Execute(ctx, cmd)
	if err != nil {
		a.logger.Error("❌ Command execution failed",
			zap.String("command_id", cmd.CommandId),
			zap.Error(err))

		// Report error
		a.reportResult(ctx, &pb.ResultMessage{
			CommandId:    cmd.CommandId,
			AgentId:      a.ID,
			Status:       pb.ResultStatus_RESULT_STATUS_ERROR,
			ErrorMessage: err.Error(),
		})
	} else {
		// Report success
		a.reportResult(ctx, result)
	}

	// Set status back to ready
	a.updateStatus(pb.AgentStatus_AGENT_STATUS_READY)
}

// reportResult sends the command result back to the server
func (a *Agent) reportResult(ctx context.Context, result *pb.ResultMessage) {
	a.logger.Debug("📤 Reporting result",
		zap.String("command_id", result.CommandId),
		zap.String("status", result.Status.String()))

	// First try to send through stream
	a.mu.RLock()
	if a.isConnected && a.stream != nil {
		resultMsg := &pb.AgentMessage{
			AgentId: a.ID,
			Version: a.Version,
			Status:  a.Status,
			Payload: &pb.AgentMessage_Result{
				Result: result,
			},
		}

		err := a.stream.Send(resultMsg)
		a.mu.RUnlock()

		if err == nil {
			a.logger.Info("✅ Result sent via stream",
				zap.String("command_id", result.CommandId))
			return
		}

		a.logger.Warn("⚠️ Failed to send result via stream, falling back to ReportResult RPC",
			zap.Error(err))
	} else {
		a.mu.RUnlock()
	}

	// Fall back to direct RPC
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()

	if client != nil {
		response, err := client.ReportResult(ctx, result)
		if err != nil {
			a.logger.Error("❌ Failed to report result",
				zap.String("command_id", result.CommandId),
				zap.Error(err))
		} else {
			a.logger.Info("✅ Result reported successfully",
				zap.String("command_id", result.CommandId),
				zap.Bool("accepted", response.Accepted),
				zap.String("message", response.Message))
		}
	} else {
		a.logger.Error("❌ Cannot report result: no client connection",
			zap.String("command_id", result.CommandId))
	}
}

// sendHeartbeats periodically sends heartbeat messages to the server
func (a *Agent) sendHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	a.logger.Info("💓 Starting heartbeat loop")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("🛑 Context cancelled, stopping heartbeats")
			return
		case <-a.stopCh:
			a.logger.Info("🛑 Stop signal received, stopping heartbeats")
			return
		case <-ticker.C:
			a.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends a single heartbeat message to the server
func (a *Agent) sendHeartbeat() {
	a.mu.RLock()
	if !a.isConnected || a.stream == nil {
		a.mu.RUnlock()
		a.logger.Debug("⚠️ Not connected, skipping heartbeat")
		return
	}

	stream := a.stream
	a.mu.RUnlock()

	// Create heartbeat message
	heartbeatMsg := &pb.AgentMessage{
		AgentId:  a.ID,
		Version:  a.Version,
		Status:   a.Status,
		Metadata: a.Capabilities,
		Payload: &pb.AgentMessage_Heartbeat{
			Heartbeat: &pb.HeartbeatMessage{
				Timestamp: time.Now().UnixNano(),
				Metrics: map[string]string{
					"uptime": fmt.Sprintf("%d", time.Now().Unix()),
					"memory": "128MB", // Sample data
					"cpu":    "2%",    // Sample data
				},
			},
		},
	}

	// Send heartbeat
	if err := stream.Send(heartbeatMsg); err != nil {
		a.logger.Warn("⚠️ Failed to send heartbeat", zap.Error(err))
		// Attempt to reconnect on next heartbeat
	} else {
		a.logger.Debug("💓 Heartbeat sent successfully")
	}
}

// updateStatus updates the agent's status
func (a *Agent) updateStatus(status pb.AgentStatus) {
	a.mu.Lock()
	a.Status = status
	a.mu.Unlock()

	a.logger.Info("📊 Agent status updated", zap.String("status", status.String()))
}

// reconnect attempts to reconnect to the server
func (a *Agent) reconnect(ctx context.Context) {
	a.logger.Info("🔄 Attempting to reconnect to server")

	// Disconnect first
	a.disconnect()

	// Try to reconnect
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			a.logger.Info("🛑 Context cancelled during reconnect")
			return
		case <-a.stopCh:
			a.logger.Info("🛑 Stop signal received during reconnect")
			return
		default:
		}

		a.logger.Info("🔄 Reconnect attempt", zap.Int("attempt", i+1))

		if err := a.connect(ctx); err == nil {
			a.logger.Info("✅ Reconnected successfully")
			return
		} else {
			a.logger.Error("❌ Reconnect failed", zap.Error(err))
		}

		// Wait before retrying
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}

	a.logger.Error("❌ Failed to reconnect after multiple attempts")
}

// RegisterCommandHandler registers a handler for a specific command type
func (a *Agent) RegisterCommandHandler(cmdType pb.CommandType, handler CommandHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.commandHandlers[cmdType] = handler
	a.logger.Info("✅ Registered command handler", zap.String("type", cmdType.String()))
}

// registerDefaultHandlers registers the default command handlers
func (a *Agent) registerDefaultHandlers() {
	// Register scan command handler
	a.RegisterCommandHandler(pb.CommandType_COMMAND_TYPE_SCAN, &ScanCommandHandler{
		logger: a.logger.Named("scan_handler"),
	})

	// Register exec command handler (using COMMAND_TYPE_STOP)
	a.RegisterCommandHandler(pb.CommandType_COMMAND_TYPE_STOP, &ExecCommandHandler{
		logger: a.logger.Named("exec_handler"),
	})

	// Register update command handler
	a.RegisterCommandHandler(pb.CommandType_COMMAND_TYPE_UPDATE, &UpdateCommandHandler{
		logger: a.logger.Named("update_handler"),
	})
}

// IsConnected returns whether the agent is connected to the server
func (a *Agent) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isConnected
}
