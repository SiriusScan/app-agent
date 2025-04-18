package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	scanpb "github.com/SiriusScan/app-agent/proto/scan/v1"
	"github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ServerConfig holds the configuration for the gRPC server
type ServerConfig struct {
	TLSConfig      *tls.Config
	RateLimit      float64 // Requests per second
	RateBurst      int     // Maximum burst size
	KeepAliveTime  time.Duration
	KeepAliveIntvl time.Duration
}

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		RateLimit:      100, // 100 requests per second
		RateBurst:      20,  // Allow bursts of up to 20 requests
		KeepAliveTime:  30 * time.Second,
		KeepAliveIntvl: 10 * time.Second,
	}
}

// CommandQueue represents a queue of commands for an agent
type CommandQueue struct {
	mu       sync.Mutex
	commands []*scanpb.Command
}

// Server implements the AgentService gRPC server
type Server struct {
	scanpb.UnimplementedAgentServiceServer
	logger       *zap.Logger
	auth         *Authenticator
	store        store.KVStore
	commandMu    sync.RWMutex
	commandQueue map[string]*CommandQueue // map[agentID]*CommandQueue
	limiter      *rate.Limiter
	config       *ServerConfig
}

// NewServer creates a new AgentService server
func NewServer(logger *zap.Logger, store store.KVStore, config *ServerConfig) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	return &Server{
		logger:       logger,
		auth:         NewAuthenticator(),
		store:        store,
		commandQueue: make(map[string]*CommandQueue),
		limiter:      rate.NewLimiter(rate.Limit(config.RateLimit), config.RateBurst),
		config:       config,
	}
}

// GetGRPCServerOptions returns the configured gRPC server options
func (s *Server) GetGRPCServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    s.config.KeepAliveTime,
			Timeout: s.config.KeepAliveIntvl,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             s.config.KeepAliveTime / 2,
			PermitWithoutStream: true,
		}),
	}

	if s.config.TLSConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.config.TLSConfig)))
	}

	return opts
}

// Connect handles agent connections and streams commands to them
func (s *Server) Connect(req *scanpb.ConnectRequest, stream scanpb.AgentService_ConnectServer) error {
	// Apply rate limiting
	if err := s.limiter.Wait(stream.Context()); err != nil {
		return status.Error(codes.ResourceExhausted, "too many requests")
	}

	// Authenticate the agent and get metadata
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization token")
	}

	// Get token metadata for additional validation
	metadata, err := s.auth.GetTokenMetadata(tokens[0])
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid token")
	}

	// Verify agent ID matches token metadata if present
	if expectedID, ok := metadata["agent_id"]; ok && expectedID != req.AgentId {
		return status.Error(codes.PermissionDenied, "agent ID mismatch")
	}

	agentID := req.AgentId
	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent ID is required")
	}

	// Initialize command queue for this agent
	s.commandMu.Lock()
	if _, exists := s.commandQueue[agentID]; !exists {
		s.commandQueue[agentID] = &CommandQueue{
			commands: make([]*scanpb.Command, 0),
		}
	}
	s.commandMu.Unlock()

	s.logger.Info("Agent connected",
		zap.String("agent_id", agentID),
		zap.String("hostname", req.Hostname),
		zap.String("platform", req.Platform))

	// Store agent connection status and metadata in Valkey
	connectionData := map[string]interface{}{
		"agent_id":  agentID,
		"status":    "connected",
		"hostname":  req.Hostname,
		"platform":  req.Platform,
		"metadata":  req.Metadata,
		"timestamp": time.Now().Unix(),
	}

	if err := s.storeAgentConnection(stream.Context(), agentID, connectionData); err != nil {
		s.logger.Error("Failed to store agent connection data",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}

	// Start command processing for this agent
	go s.processCommands(agentID, stream)

	// Start keep-alive monitoring
	keepAliveTicker := time.NewTicker(s.config.KeepAliveIntvl)
	defer keepAliveTicker.Stop()

	// Create a context with timeout for disconnection handling
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// Start a goroutine to handle disconnection cleanup
	go func() {
		<-ctx.Done()
		s.handleAgentDisconnection(agentID)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-keepAliveTicker.C:
			// Send empty command as keep-alive
			if err := stream.Send(&scanpb.Command{
				CommandId: "keep-alive",
				Type:      scanpb.CommandType_COMMAND_TYPE_STATUS,
				CommandDetails: &scanpb.Command_Status{
					Status: &scanpb.StatusCommand{
						Metrics: []string{"ping"},
					},
				},
			}); err != nil {
				s.logger.Error("Keep-alive failed",
					zap.String("agent_id", agentID),
					zap.Error(err))
				return err
			}
		}
	}
}

// storeAgentConnection stores the agent's connection data in Valkey
func (s *Server) storeAgentConnection(ctx context.Context, agentID string, data map[string]interface{}) error {
	connectionJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal connection data: %w", err)
	}

	key := fmt.Sprintf("agent:%s:connection", agentID)
	return s.store.SetValue(ctx, key, string(connectionJSON))
}

// handleAgentDisconnection performs cleanup when an agent disconnects
func (s *Server) handleAgentDisconnection(agentID string) {
	// Update agent status to disconnected
	disconnectData := map[string]interface{}{
		"agent_id":  agentID,
		"status":    "disconnected",
		"timestamp": time.Now().Unix(),
	}

	if err := s.storeAgentConnection(context.Background(), agentID, disconnectData); err != nil {
		s.logger.Error("Failed to store agent disconnection",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}

	s.logger.Info("Agent disconnected", zap.String("agent_id", agentID))

	// Clean up command queue
	s.commandMu.Lock()
	delete(s.commandQueue, agentID)
	s.commandMu.Unlock()
}

// validateCommand validates a command based on its type
func (s *Server) validateCommand(cmd *scanpb.Command) error {
	if cmd.CommandId == "" {
		return status.Error(codes.InvalidArgument, "command ID is required")
	}

	switch cmd.Type {
	case scanpb.CommandType_COMMAND_TYPE_SCAN:
		if cmd.GetCommandDetails() == nil || cmd.GetCommandDetails().(*scanpb.Command_Scan) == nil {
			return status.Error(codes.InvalidArgument, "scan command details required")
		}
		if cmd.GetCommandDetails().(*scanpb.Command_Scan).Scan.ScanType == "" {
			return status.Error(codes.InvalidArgument, "scan type is required")
		}
	case scanpb.CommandType_COMMAND_TYPE_EXECUTE:
		if cmd.GetCommandDetails() == nil || cmd.GetCommandDetails().(*scanpb.Command_Execute) == nil {
			return status.Error(codes.InvalidArgument, "execute command details required")
		}
		if cmd.GetCommandDetails().(*scanpb.Command_Execute).Execute.Command == "" {
			return status.Error(codes.InvalidArgument, "command string is required")
		}
	case scanpb.CommandType_COMMAND_TYPE_STATUS:
		if cmd.GetCommandDetails() == nil || cmd.GetCommandDetails().(*scanpb.Command_Status) == nil {
			return status.Error(codes.InvalidArgument, "status command details required")
		}
	case scanpb.CommandType_COMMAND_TYPE_UPDATE:
		if cmd.GetCommandDetails() == nil || cmd.GetCommandDetails().(*scanpb.Command_Update) == nil {
			return status.Error(codes.InvalidArgument, "update command details required")
		}
		if len(cmd.GetCommandDetails().(*scanpb.Command_Update).Update.Config) == 0 {
			return status.Error(codes.InvalidArgument, "update configuration is required")
		}
	default:
		return status.Error(codes.InvalidArgument, "invalid command type")
	}

	return nil
}

// QueueCommand adds a command to an agent's queue
func (s *Server) QueueCommand(agentID string, cmd *scanpb.Command) error {
	// Validate command
	if err := s.validateCommand(cmd); err != nil {
		return err
	}

	s.commandMu.RLock()
	queue, exists := s.commandQueue[agentID]
	s.commandMu.RUnlock()

	if !exists {
		return status.Error(codes.NotFound, "agent not found")
	}

	queue.mu.Lock()
	queue.commands = append(queue.commands, cmd)
	queue.mu.Unlock()

	s.logger.Info("Command queued",
		zap.String("agent_id", agentID),
		zap.String("command_id", cmd.CommandId),
		zap.String("type", cmd.Type.String()))

	// Store command in Valkey
	if err := s.storeCommand(context.Background(), agentID, cmd); err != nil {
		s.logger.Error("Failed to store command",
			zap.String("agent_id", agentID),
			zap.String("command_id", cmd.CommandId),
			zap.Error(err))
	}

	return nil
}

// storeCommand stores a command in Valkey
func (s *Server) storeCommand(ctx context.Context, agentID string, cmd *scanpb.Command) error {
	cmdData := map[string]interface{}{
		"command_id": cmd.CommandId,
		"agent_id":   agentID,
		"type":       cmd.Type.String(),
		"timestamp":  time.Now().Unix(),
	}

	cmdJSON, err := json.Marshal(cmdData)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	key := fmt.Sprintf("command:%s", cmd.CommandId)
	return s.store.SetValue(ctx, key, string(cmdJSON))
}

// processCommands processes queued commands for an agent
func (s *Server) processCommands(agentID string, stream scanpb.AgentService_ConnectServer) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return
		case <-ticker.C:
			s.commandMu.RLock()
			queue := s.commandQueue[agentID]
			s.commandMu.RUnlock()

			queue.mu.Lock()
			if len(queue.commands) > 0 {
				cmd := queue.commands[0]
				queue.commands = queue.commands[1:]
				queue.mu.Unlock()

				if err := stream.Send(cmd); err != nil {
					s.logger.Error("Failed to send command",
						zap.String("agent_id", agentID),
						zap.String("command_id", cmd.CommandId),
						zap.Error(err))
					continue
				}

				// Update command status in Valkey
				if err := s.updateCommandStatus(context.Background(), cmd.CommandId, "sent"); err != nil {
					s.logger.Error("Failed to update command status",
						zap.String("command_id", cmd.CommandId),
						zap.Error(err))
				}

				s.logger.Info("Command sent",
					zap.String("agent_id", agentID),
					zap.String("command_id", cmd.CommandId))
			} else {
				queue.mu.Unlock()
			}
		}
	}
}

// updateCommandStatus updates the status of a command in Valkey
func (s *Server) updateCommandStatus(ctx context.Context, commandID, status string) error {
	key := fmt.Sprintf("command:%s", commandID)
	resp, err := s.store.GetValue(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get command: %w", err)
	}

	var cmdData map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Message.Value), &cmdData); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	cmdData["status"] = status
	cmdData["updated_at"] = time.Now().Unix()

	cmdJSON, err := json.Marshal(cmdData)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	return s.store.SetValue(ctx, key, string(cmdJSON))
}

// ReportResult handles command execution results from agents
func (s *Server) ReportResult(ctx context.Context, result *scanpb.CommandResult) (*scanpb.CommandAck, error) {
	// Apply rate limiting
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, status.Error(codes.ResourceExhausted, "too many requests")
	}

	// Authenticate the agent
	if err := s.auth.AuthenticateContext(ctx); err != nil {
		return nil, err
	}

	// Validate result
	if result.CommandId == "" {
		return nil, status.Error(codes.InvalidArgument, "command ID is required")
	}
	if result.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	// Process result based on type
	var resultStatus string
	switch result.GetResultDetails().(type) {
	case *scanpb.CommandResult_Scan:
		// Process scan results
		if err := s.processScanResult(ctx, result); err != nil {
			resultStatus = "error"
			s.logger.Error("Failed to process scan result",
				zap.String("command_id", result.CommandId),
				zap.Error(err))
		} else {
			resultStatus = "success"
		}
	case *scanpb.CommandResult_Execute:
		// Process execute results
		if err := s.processExecuteResult(ctx, result); err != nil {
			resultStatus = "error"
			s.logger.Error("Failed to process execute result",
				zap.String("command_id", result.CommandId),
				zap.Error(err))
		} else {
			resultStatus = "success"
		}
	case *scanpb.CommandResult_Status:
		// Process status results
		if err := s.processStatusResult(ctx, result); err != nil {
			resultStatus = "error"
			s.logger.Error("Failed to process status result",
				zap.String("command_id", result.CommandId),
				zap.Error(err))
		} else {
			resultStatus = "success"
		}
	case *scanpb.CommandResult_Update:
		// Process update results
		if err := s.processUpdateResult(ctx, result); err != nil {
			resultStatus = "error"
			s.logger.Error("Failed to process update result",
				zap.String("command_id", result.CommandId),
				zap.Error(err))
		} else {
			resultStatus = "success"
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid result type")
	}

	// Store result in Valkey
	if err := s.storeResult(ctx, result); err != nil {
		s.logger.Error("Failed to store result",
			zap.String("command_id", result.CommandId),
			zap.Error(err))
	}

	return &scanpb.CommandAck{
		CommandId: result.CommandId,
		Status:    resultStatus,
	}, nil
}

// processScanResult processes and stores scan results
func (s *Server) processScanResult(ctx context.Context, result *scanpb.CommandResult) error {
	scanResult := result.GetScan()
	if scanResult == nil {
		return fmt.Errorf("scan result is nil")
	}

	// Store findings in Valkey
	key := fmt.Sprintf("scan:%s:findings", result.CommandId)
	findingsJSON, err := json.Marshal(scanResult.Findings)
	if err != nil {
		return fmt.Errorf("failed to marshal findings: %w", err)
	}

	return s.store.SetValue(ctx, key, string(findingsJSON))
}

// processExecuteResult processes and stores command execution results
func (s *Server) processExecuteResult(ctx context.Context, result *scanpb.CommandResult) error {
	execResult := result.GetExecute()
	if execResult == nil {
		return fmt.Errorf("execute result is nil")
	}

	// Store execution details in Valkey
	key := fmt.Sprintf("exec:%s:result", result.CommandId)
	execJSON, err := json.Marshal(execResult)
	if err != nil {
		return fmt.Errorf("failed to marshal execution result: %w", err)
	}

	return s.store.SetValue(ctx, key, string(execJSON))
}

// processStatusResult processes and stores agent status results
func (s *Server) processStatusResult(ctx context.Context, result *scanpb.CommandResult) error {
	statusResult := result.GetStatus()
	if statusResult == nil {
		return fmt.Errorf("status result is nil")
	}

	// Store status details in Valkey
	key := fmt.Sprintf("agent:%s:metrics", result.AgentId)
	statusJSON, err := json.Marshal(statusResult)
	if err != nil {
		return fmt.Errorf("failed to marshal status result: %w", err)
	}

	return s.store.SetValue(ctx, key, string(statusJSON))
}

// processUpdateResult processes and stores configuration update results
func (s *Server) processUpdateResult(ctx context.Context, result *scanpb.CommandResult) error {
	updateResult := result.GetUpdate()
	if updateResult == nil {
		return fmt.Errorf("update result is nil")
	}

	// Store update status in Valkey
	key := fmt.Sprintf("agent:%s:config_update", result.AgentId)
	updateJSON, err := json.Marshal(updateResult)
	if err != nil {
		return fmt.Errorf("failed to marshal update result: %w", err)
	}

	return s.store.SetValue(ctx, key, string(updateJSON))
}

// storeResult stores the complete command result in Valkey
func (s *Server) storeResult(ctx context.Context, result *scanpb.CommandResult) error {
	key := fmt.Sprintf("command:%s:result", result.CommandId)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return s.store.SetValue(ctx, key, string(resultJSON))
}
