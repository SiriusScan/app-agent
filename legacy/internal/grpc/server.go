package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/SiriusScan/app-agent/internal/config"
	pb "github.com/SiriusScan/app-agent/proto/scan"
)

// Server represents the gRPC server for the Agent Manager
type Server struct {
	logger     *zap.Logger
	config     *config.ServerConfig
	grpcServer *grpc.Server
	agents     map[string]struct{} // Simple agent tracking for now
	mu         sync.RWMutex
	service    pb.AgentServiceServer // Store the service instance
}

// NewServer creates a new gRPC server instance
func NewServer(cfg *config.ServerConfig, logger *zap.Logger) *Server {
	grpcServer := grpc.NewServer()

	server := &Server{
		logger:     logger,
		config:     cfg,
		grpcServer: grpcServer,
		agents:     make(map[string]struct{}),
	}

	// Create and register the service with the server
	service := NewAgentServiceServer(logger, server)
	pb.RegisterAgentServiceServer(grpcServer, service)

	// Store the service instance
	server.service = service

	return server
}

// GetAgentService returns the service instance for command distribution
func (s *Server) GetAgentService() *agentServiceServer {
	return s.service.(*agentServiceServer)
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context) error {
	addr := s.config.GRPCServerAddr
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Starting gRPC server", zap.String("address", addr))

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Error("Failed to serve", zap.Error(err))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return s.Stop()
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() error {
	s.logger.Info("Stopping gRPC server")

	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	timeout := time.NewTimer(5 * time.Second)
	select {
	case <-stopped:
		return nil
	case <-timeout.C:
		s.logger.Warn("Server stop timeout, forcing shutdown")
		s.grpcServer.Stop()
		return nil
	}
}

// RegisterAgent registers a new agent connection
func (s *Server) RegisterAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agents[agentID] = struct{}{}

	// Log connected agents
	agents := make([]string, 0, len(s.agents))
	for agent := range s.agents {
		agents = append(agents, agent)
	}

	s.logger.Info("Agent registered",
		zap.String("agent_id", agentID),
		zap.Strings("connected_agents", agents),
		zap.Int("total_agents", len(agents)),
	)
}

// UnregisterAgent removes an agent connection
func (s *Server) UnregisterAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.agents, agentID)

	// Log remaining agents
	agents := make([]string, 0, len(s.agents))
	for agent := range s.agents {
		agents = append(agents, agent)
	}

	s.logger.Info("Agent unregistered",
		zap.String("agent_id", agentID),
		zap.Strings("remaining_agents", agents),
		zap.Int("total_agents", len(agents)),
	)
}

// GetAgents returns the list of connected agent IDs
func (s *Server) GetAgents() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]string, 0, len(s.agents))
	for agentID := range s.agents {
		agents = append(agents, agentID)
	}

	return agents
}
