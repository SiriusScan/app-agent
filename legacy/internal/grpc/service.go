package grpc

import (
	"context"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/SiriusScan/app-agent/internal/command"
	pb "github.com/SiriusScan/app-agent/proto/scan"
)

// agentServiceServer implements the pb.AgentServiceServer interface
type agentServiceServer struct {
	pb.UnimplementedAgentServiceServer
	logger       *zap.Logger
	server       *Server
	agentStreams map[string]pb.AgentService_ConnectServer
	streamsMutex sync.RWMutex
}

// NewAgentServiceServer creates a new agent service server
func NewAgentServiceServer(logger *zap.Logger, server *Server) pb.AgentServiceServer {
	return &agentServiceServer{
		logger:       logger,
		server:       server,
		agentStreams: make(map[string]pb.AgentService_ConnectServer),
		streamsMutex: sync.RWMutex{},
	}
}

// Ping is a simple RPC to test connectivity
func (s *agentServiceServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	s.logger.Info("Received ping", zap.String("agent_id", req.AgentId))

	return &pb.PingResponse{
		Message:   "Pong from server!",
		Timestamp: time.Now().Unix(),
	}, nil
}

// Connect establishes a bidirectional stream with the agent
func (s *agentServiceServer) Connect(stream pb.AgentService_ConnectServer) error {
	ctx := stream.Context()
	agentID := getAgentID(ctx)

	if agentID == "" {
		return status.Error(codes.Unauthenticated, "agent_id is required in metadata")
	}

	s.logger.Info("Agent stream started", zap.String("agent_id", agentID))

	// Register agent with server
	s.server.RegisterAgent(agentID)

	// Store the stream for sending commands
	s.registerAgentStream(agentID, stream)

	// Cleanup on exit
	defer func() {
		s.logger.Info("Agent stream ended", zap.String("agent_id", agentID))
		s.unregisterAgentStream(agentID)
		s.server.UnregisterAgent(agentID)
	}()

	// Process stream
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			s.logger.Error("Error receiving message", zap.String("agent_id", agentID), zap.Error(err))
			return err
		}

		s.logger.Info("Received message from agent",
			zap.String("agent_id", req.AgentId),
			zap.String("status", req.Status.String()),
		)

		// Process command results if present
		if req.GetCommandResult() != nil {
			s.processCommandResult(agentID, req.GetCommandResult())
		}
	}
}

// registerAgentStream registers an agent's stream for sending commands
func (s *agentServiceServer) registerAgentStream(agentID string, stream pb.AgentService_ConnectServer) {
	s.streamsMutex.Lock()
	defer s.streamsMutex.Unlock()

	s.agentStreams[agentID] = stream
	s.logger.Debug("Registered agent stream", zap.String("agent_id", agentID))
}

// unregisterAgentStream removes an agent's stream
func (s *agentServiceServer) unregisterAgentStream(agentID string) {
	s.streamsMutex.Lock()
	defer s.streamsMutex.Unlock()

	delete(s.agentStreams, agentID)
	s.logger.Debug("Unregistered agent stream", zap.String("agent_id", agentID))
}

// processCommandResult handles command result messages from agents
func (s *agentServiceServer) processCommandResult(agentID string, result *pb.CommandResult) {
	s.logger.Info("Processing command result",
		zap.String("agent_id", agentID),
		zap.String("command_id", result.CommandId),
		zap.String("status", result.Status))

	// TODO: Store command results in Valkey
}

// SendCommand sends a command to a specific agent
func (s *agentServiceServer) SendCommand(agentID string, cmd *pb.CommandMessage) error {
	s.streamsMutex.RLock()
	stream, ok := s.agentStreams[agentID]
	s.streamsMutex.RUnlock()

	if !ok {
		return status.Errorf(codes.NotFound, "agent %s is not connected", agentID)
	}

	s.logger.Debug("Sending command to agent",
		zap.String("agent_id", agentID),
		zap.String("command_id", cmd.Id))

	if err := stream.Send(cmd); err != nil {
		s.logger.Error("Failed to send command to agent",
			zap.String("agent_id", agentID),
			zap.String("command_id", cmd.Id),
			zap.Error(err))
		return err
	}

	return nil
}

// IsAgentConnected checks if an agent is currently connected
func (s *agentServiceServer) IsAgentConnected(agentID string) bool {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()

	_, ok := s.agentStreams[agentID]
	return ok
}

// GetAgentIDs returns a list of all connected agent IDs
func (s *agentServiceServer) GetAgentIDs() []string {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()

	agentIDs := make([]string, 0, len(s.agentStreams))
	for agentID := range s.agentStreams {
		agentIDs = append(agentIDs, agentID)
	}

	return agentIDs
}

// GetConnectedAgentIDs returns a list of all connected agent IDs
// Implements the command.AgentCommandSender interface
func (s *agentServiceServer) GetConnectedAgentIDs() []string {
	return s.GetAgentIDs()
}

// SendCommandToAgent sends a command to a specific agent using a Command struct
// Implements the command.AgentCommandSender interface
func (s *agentServiceServer) SendCommandToAgent(agentID string, cmd command.Command) error {
	// Convert to gRPC command message
	cmdMsg := &pb.CommandMessage{
		Id:             cmd.ID,
		Type:           cmd.Type,
		Params:         cmd.Params,
		TimeoutSeconds: 60, // Default timeout of 60 seconds
	}

	// Use the existing SendCommand method
	return s.SendCommand(agentID, cmdMsg)
}

// Helper function to extract agent ID from context metadata
func getAgentID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get("agent-id")
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
