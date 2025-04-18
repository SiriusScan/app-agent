package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/grpc"
	pb "github.com/SiriusScan/app-agent/proto/scan"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Info("Starting Test Agent (Client)")

	// Load configuration
	cfg := config.LoadAgentConfig()
	logger.Info("Configuration loaded",
		zap.String("agent_id", cfg.AgentID),
		zap.String("server_address", cfg.ServerAddr))

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the gRPC client
	client, err := grpc.NewClient(cfg.ServerAddr, logger)
	if err != nil {
		logger.Fatal("Failed to create gRPC client", zap.Error(err))
	}
	defer client.Close()

	// Connect to server
	if err := client.Connect(ctx); err != nil {
		logger.Fatal("Failed to connect to server", zap.Error(err))
	}

	// Run agent with the client
	if err := runAgent(ctx, client, logger, cfg); err != nil {
		logger.Fatal("Agent failed", zap.Error(err))
	}
}

// runAgent runs the main agent loop
func runAgent(ctx context.Context, client *grpc.Client, logger *zap.Logger, cfg *config.AgentConfig) error {
	// Try a simple ping to test connectivity
	logger.Info("Connected to server, sending ping")
	message, err := client.Ping(ctx, cfg.AgentID)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	logger.Info("Ping successful", zap.String("response", message))

	// Establish bidirectional stream connection
	stream, err := client.ConnectStream(ctx, cfg.AgentID)
	if err != nil {
		return fmt.Errorf("failed to connect stream: %w", err)
	}

	// After connecting to the server, set up a goroutine to handle incoming commands
	go func() {
		for {
			// Check if the stream is still active
			if stream == nil {
				logger.Error("Stream is nil, cannot receive commands")
				return
			}

			// Receive a command from the server
			cmd, err := stream.Recv()
			if err == io.EOF {
				logger.Info("Command stream ended")
				return
			}
			if err != nil {
				logger.Error("Error receiving command", zap.Error(err))
				return
			}

			// Log the received command
			logger.Info("Received command from server",
				zap.String("command_id", cmd.Id),
				zap.String("type", cmd.Type),
				zap.Any("params", cmd.Params))

			// Execute the command in a separate goroutine
			go func(cmdMsg *pb.CommandMessage) {
				// Simulate command execution
				logger.Info("Executing command", zap.String("command_id", cmdMsg.Id))

				// Prepare a result
				result := &pb.CommandResult{
					CommandId:   cmdMsg.Id,
					Status:      "completed",
					Data:        map[string]string{"result": "Command executed successfully"},
					CompletedAt: time.Now().Unix(),
				}

				// If it's a scan command, add more specific data
				if cmdMsg.Type == "scan" {
					target, ok := cmdMsg.Params["target"]
					if !ok {
						target = "unknown"
					}

					// Simulate scan results
					result.Data["target"] = target
					result.Data["vulnerabilities_found"] = "5"
					result.Data["scan_duration"] = "10.5"
				}

				// Send the result back to the server
				response := &pb.AgentMessage{
					AgentId: cfg.AgentID,
					Status:  pb.AgentStatus_IDLE,
					Payload: &pb.AgentMessage_CommandResult{
						CommandResult: result,
					},
				}

				// Send the result
				if err := stream.Send(response); err != nil {
					logger.Error("Failed to send command result", zap.Error(err))
					return
				}

				logger.Info("Sent command result to server",
					zap.String("command_id", cmdMsg.Id),
					zap.String("status", result.Status))
			}(cmd)
		}
	}()

	// Main loop - for now just keep the agent alive
	logger.Info("Agent running. Press Ctrl+C to exit")

	// Wait for interrupt signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	logger.Info("Shutting down agent...")
	return nil
}
