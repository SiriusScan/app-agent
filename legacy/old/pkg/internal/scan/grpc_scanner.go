package scan

import (
	"context"
	"fmt"
	"io"

	pb "github.com/SiriusScan/app-agent/proto/scan/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// GRPCScanner implements the Scanner interface using gRPC
type GRPCScanner struct {
	client pb.AgentServiceClient
	logger *zap.Logger
}

// NewGRPCScanner creates a new GRPC scanner instance
func NewGRPCScanner(conn *grpc.ClientConn, logger *zap.Logger) *GRPCScanner {
	return &GRPCScanner{
		client: pb.NewAgentServiceClient(conn),
		logger: logger,
	}
}

// Connect establishes a connection to the gRPC server and starts receiving commands
func (s *GRPCScanner) Connect(ctx context.Context) error {
	stream, err := s.client.Connect(ctx, &pb.ConnectRequest{})
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	// Handle command stream
	go func() {
		for {
			cmd, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				s.logger.Error("Error receiving command", zap.Error(err))
				return
			}

			// Process command
			s.handleCommand(ctx, cmd)
		}
	}()

	return nil
}

// ReportResult sends the result of a command execution back to the server
func (s *GRPCScanner) ReportResult(ctx context.Context, result *pb.CommandResult) error {
	_, err := s.client.ReportResult(ctx, result)
	if err != nil {
		return fmt.Errorf("failed to report result: %v", err)
	}
	return nil
}

// handleCommand processes a received command
func (s *GRPCScanner) handleCommand(ctx context.Context, cmd *pb.Command) {
	// TODO: Implement command handling logic
	s.logger.Info("Received command", zap.String("command_id", cmd.CommandId))
}
