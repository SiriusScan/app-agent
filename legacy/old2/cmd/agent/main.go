package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/grpc"
	"github.com/SiriusScan/app-agent/proto"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	// Parse command-line flags
	serverAddr := flag.String("server", "", "Server address (overrides env config)")
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("🚀 Starting agent application", zap.String("component", "agent"))

	// Load configuration
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		logger.Fatal("❌ Failed to load configuration", zap.Error(err))
	}

	// Override server address if provided via flag
	if *serverAddr != "" {
		cfg.GRPCServerAddr = *serverAddr
	}

	// Generate agent ID if not provided
	agentID := cfg.AgentID
	if agentID == "" {
		agentID = uuid.New().String()
		logger.Info("Generated agent ID", zap.String("agent_id", agentID))
	}

	logger.Info("✅ Configuration loaded",
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
		zap.String("server_address", cfg.GRPCServerAddr),
		zap.String("agent_id", agentID))

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Define agent capabilities
	capabilities := map[string]string{
		"version":    cfg.Version,
		"scan_types": "port-scan,service-detection,vulnerability-scan",
		"max_scans":  "10",
		"system":     "macOS",
	}

	// Initialize processed command tracking
	processedCommands := make(map[string]bool)

	// Initialize gRPC client
	logger.Info("Connecting to gRPC server", zap.String("addr", cfg.GRPCServerAddr))
	client, err := grpc.NewClient(cfg.GRPCServerAddr, cfg.TLSEnabled, cfg.TLSCertFile, logger)
	if err != nil {
		logger.Fatal("❌ Failed to create gRPC client", zap.Error(err))
	}
	defer client.Close()

	// Run the agent
	if err := runAgent(ctx, client, logger, agentID, capabilities, processedCommands); err != nil {
		logger.Fatal("❌ Agent execution failed", zap.Error(err))
	}

	// Wait for graceful shutdown
	logger.Info("✅ Agent shutting down gracefully")
}

// runAgent runs the agent with a real gRPC connection
func runAgent(ctx context.Context, client *grpc.Client, logger *zap.Logger, agentID string, capabilities map[string]string, processedCommands map[string]bool) error {
	// Connect to the server and get bidirectional stream
	logger.Info("Establishing bidirectional stream with server", zap.String("agent_id", agentID))
	stream, err := client.Connect(ctx, agentID, capabilities)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	logger.Info("✅ Connected to gRPC server stream")

	// Process commands from the server
	for {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled, exiting agent loop")
			return nil
		default:
			// Receive command from server
			cmd, err := stream.Recv()
			if err != nil {
				logger.Error("Failed to receive command", zap.Error(err))
				// Attempt to reconnect after a delay
				time.Sleep(5 * time.Second)
				stream, err = client.Connect(ctx, agentID, capabilities)
				if err != nil {
					logger.Error("Failed to reconnect", zap.Error(err))
					continue
				}
				continue
			}

			// Process command
			logger.Info("📥 Received command",
				zap.String("command_id", cmd.CommandId),
				zap.String("command_type", cmd.Type),
				zap.Any("parameters", cmd.Parameters))

			// Check if this command has already been processed
			if _, exists := processedCommands[cmd.CommandId]; exists {
				logger.Info("🔄 Skipping already processed command",
					zap.String("command_id", cmd.CommandId),
					zap.String("command_type", cmd.Type))
				continue
			}

			// Execute command
			result, err := executeCommand(ctx, cmd, logger)
			if err != nil {
				logger.Error("❌ Failed to execute command",
					zap.String("command_id", cmd.CommandId),
					zap.Error(err))

				// Send error result
				result = &proto.ResultMessage{
					CommandId:    cmd.CommandId,
					AgentId:      agentID,
					Status:       proto.ResultStatus_RESULT_STATUS_ERROR,
					ErrorMessage: err.Error(),
					CompletedAt:  time.Now().UnixNano(),
				}
			}

			// Log the result details
			if cmd.Type == "exec" {
				if result.Status == proto.ResultStatus_RESULT_STATUS_ERROR {
					logger.Error("❌ Command execution failed",
						zap.String("command_id", cmd.CommandId),
						zap.String("command", cmd.Parameters["command"]),
						zap.String("error", result.ErrorMessage))
				} else {
					// Log the output directly to make it visible
					logger.Info("✅ Command execution result",
						zap.String("command_id", cmd.CommandId),
						zap.String("command", cmd.Parameters["command"]),
						zap.String("status", result.Status.String()))

					// Print the actual output to make it visible in logs
					fmt.Printf("\n----- COMMAND OUTPUT -----\n%s\n-------------------------\n\n", result.Output)
				}
			}

			// Send result back to server
			if err := stream.Send(&proto.AgentMessage{
				AgentId:  agentID,
				Version:  capabilities["version"],
				Status:   proto.AgentStatus_AGENT_STATUS_READY,
				Metadata: capabilities,
				Payload: &proto.AgentMessage_Result{
					Result: result,
				},
			}); err != nil {
				logger.Error("Failed to send result", zap.Error(err))
			} else {
				logger.Info("📤 Sent command result to server",
					zap.String("command_id", cmd.CommandId),
					zap.String("status", result.Status.String()))
			}

			// Mark command as processed
			processedCommands[cmd.CommandId] = true

			// Cleanup old commands if the map gets too large
			if len(processedCommands) > 1000 {
				cleanupOldCommands(processedCommands, 500)
				logger.Info("🧹 Cleaned up command history",
					zap.Int("remaining_commands", len(processedCommands)))
			}
		}
	}
}

// cleanupOldCommands removes the oldest entries from the command history map
// to prevent unbounded growth
func cleanupOldCommands(processedCommands map[string]bool, targetSize int) {
	// If we're already below the target size, don't do anything
	if len(processedCommands) <= targetSize {
		return
	}

	// Convert map keys to a slice and sort them
	// This is a simple approach - for production, we'd use a time-based approach
	// or a proper LRU cache
	keys := make([]string, 0, len(processedCommands))
	for k := range processedCommands {
		keys = append(keys, k)
	}

	// Remove oldest entries (we're assuming command IDs are somewhat time-ordered)
	// In practice, command IDs should include timestamps or sequence numbers
	toRemove := len(processedCommands) - targetSize
	for i := 0; i < toRemove; i++ {
		delete(processedCommands, keys[i])
	}
}

// executeCommand executes a command based on its type
func executeCommand(ctx context.Context, cmd *proto.CommandMessage, logger *zap.Logger) (*proto.ResultMessage, error) {
	// Process command based on type
	switch cmd.Type {
	case "scan":
		// Simulate scan execution
		time.Sleep(2 * time.Second)
		return &proto.ResultMessage{
			CommandId: cmd.CommandId,
			AgentId:   cmd.AgentId,
			Status:    proto.ResultStatus_RESULT_STATUS_SUCCESS,
			Data: map[string]string{
				"scan_result": "success",
				"findings":    "5 vulnerabilities",
				"scan_time":   "2s",
			},
			Type:        "scan",
			CompletedAt: time.Now().UnixNano(),
		}, nil

	case "status":
		// Simulate status check
		return &proto.ResultMessage{
			CommandId: cmd.CommandId,
			AgentId:   cmd.AgentId,
			Status:    proto.ResultStatus_RESULT_STATUS_SUCCESS,
			Data: map[string]string{
				"agent_status": "healthy",
				"uptime":       "3600s",
				"load":         "0.5",
			},
			Type:        "status",
			CompletedAt: time.Now().UnixNano(),
		}, nil

	case "exec":
		// Handle shell command execution
		logger.Info("Executing shell command", zap.Any("params", cmd.Parameters))

		// Extract command
		shellCmd, ok := cmd.Parameters["command"]
		if !ok || shellCmd == "" {
			return nil, fmt.Errorf("missing required parameter: command")
		}

		// Create command execution context with timeout
		timeoutSec := 30 // Default timeout
		if timeoutStr, ok := cmd.Parameters["timeout"]; ok {
			if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
				timeoutSec = parsed
			}
		}

		execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()

		// Execute command
		output, err := executeShellCommand(execCtx, shellCmd)

		// Prepare result
		result := &proto.ResultMessage{
			CommandId:   cmd.CommandId,
			AgentId:     cmd.AgentId,
			Status:      proto.ResultStatus_RESULT_STATUS_SUCCESS,
			Data:        map[string]string{"command": shellCmd},
			Type:        "exec",
			Output:      output,
			CompletedAt: time.Now().UnixNano(),
		}

		if err != nil {
			result.Status = proto.ResultStatus_RESULT_STATUS_ERROR
			result.ErrorMessage = err.Error()
			logger.Error("Command execution failed",
				zap.String("command", shellCmd),
				zap.Error(err))
		} else {
			logger.Info("Command execution successful",
				zap.String("command", shellCmd),
				zap.Int("output_length", len(output)))
		}

		return result, nil

	default:
		return nil, fmt.Errorf("unsupported command type: %s", cmd.Type)
	}
}

// executeShellCommand executes a shell command and returns its output
func executeShellCommand(ctx context.Context, command string) (string, error) {
	// Create the command
	cmd := exec.Command("sh", "-c", command)

	// Create a buffer to capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Create a channel to signal command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for command to complete or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, attempt to kill the process
		if err := cmd.Process.Kill(); err != nil {
			return "", fmt.Errorf("command timed out but couldn't kill process: %w", err)
		}
		return "", fmt.Errorf("command timed out after %s", ctx.Err())
	case err := <-done:
		// Command completed
		if err != nil {
			return stdout.String() + stderr.String(), fmt.Errorf("command failed: %w: %s", err, stderr.String())
		}
		return stdout.String(), nil
	}
}
