package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	pb "github.com/SiriusScan/app-agent/internal/proto/scan"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ScanCommandHandler implements the handler for scan commands
type ScanCommandHandler struct {
	logger *zap.Logger
}

// Execute processes a scan command
func (h *ScanCommandHandler) Execute(ctx context.Context, cmd *pb.CommandMessage) (*pb.ResultMessage, error) {
	h.logger.Info("🔍 Executing scan command",
		zap.String("command_id", cmd.CommandId),
		zap.Any("parameters", cmd.Parameters))

	// Extract parameters
	target := cmd.Parameters["target"]
	scanType := cmd.Parameters["scan_type"]

	if target == "" {
		return nil, fmt.Errorf("missing required parameter: target")
	}

	// Log scan start
	h.logger.Info("🚀 Starting scan",
		zap.String("target", target),
		zap.String("scan_type", scanType))

	// Simulate scan operation
	scanDuration := simulateScanOperation(ctx, h.logger, target, scanType)

	// Check if context was cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Context not cancelled, continue
	}

	// Create result data
	resultData := map[string]string{
		"target":          target,
		"scan_type":       scanType,
		"scan_id":         uuid.New().String(),
		"duration_ms":     fmt.Sprintf("%d", scanDuration.Milliseconds()),
		"vulnerabilities": "7", // Simulated number of findings
		"timestamp":       fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Log scan completion
	h.logger.Info("✅ Scan completed",
		zap.String("target", target),
		zap.Duration("duration", scanDuration),
		zap.Any("result", resultData))

	// Return result
	return &pb.ResultMessage{
		CommandId: cmd.CommandId,
		AgentId:   cmd.AgentId,
		Status:    pb.ResultStatus_RESULT_STATUS_SUCCESS,
		Data:      resultData,
	}, nil
}

// UpdateCommandHandler implements the handler for update commands
type UpdateCommandHandler struct {
	logger *zap.Logger
}

// Execute processes an update command
func (h *UpdateCommandHandler) Execute(ctx context.Context, cmd *pb.CommandMessage) (*pb.ResultMessage, error) {
	h.logger.Info("🔄 Executing update command",
		zap.String("command_id", cmd.CommandId),
		zap.Any("parameters", cmd.Parameters))

	// Extract parameters
	updateType := cmd.Parameters["update_type"]
	version := cmd.Parameters["version"]

	if updateType == "" {
		return nil, fmt.Errorf("missing required parameter: update_type")
	}

	// Log update start
	h.logger.Info("🚀 Starting update",
		zap.String("update_type", updateType),
		zap.String("version", version))

	// Simulate update operation
	updateDuration := simulateUpdateOperation(ctx, h.logger, updateType, version)

	// Check if context was cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Context not cancelled, continue
	}

	// Create result data
	resultData := map[string]string{
		"update_type": updateType,
		"version":     version,
		"update_id":   uuid.New().String(),
		"duration_ms": fmt.Sprintf("%d", updateDuration.Milliseconds()),
		"status":      "success",
		"timestamp":   fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Log update completion
	h.logger.Info("✅ Update completed",
		zap.String("update_type", updateType),
		zap.Duration("duration", updateDuration),
		zap.Any("result", resultData))

	// Return result
	return &pb.ResultMessage{
		CommandId: cmd.CommandId,
		AgentId:   cmd.AgentId,
		Status:    pb.ResultStatus_RESULT_STATUS_SUCCESS,
		Data:      resultData,
	}, nil
}

// StopCommandHandler implements the handler for stop commands
type StopCommandHandler struct {
	logger *zap.Logger
	agent  *Agent
}

// Execute processes a stop command
func (h *StopCommandHandler) Execute(ctx context.Context, cmd *pb.CommandMessage) (*pb.ResultMessage, error) {
	h.logger.Info("🛑 Executing stop command",
		zap.String("command_id", cmd.CommandId),
		zap.Any("parameters", cmd.Parameters))

	// Extract parameters
	reason := cmd.Parameters["reason"]

	// Log stop request
	h.logger.Info("⏹️ Processing stop request",
		zap.String("reason", reason))

	// Create result data before stopping
	resultData := map[string]string{
		"stop_reason": reason,
		"timestamp":   fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Prepare result message
	result := &pb.ResultMessage{
		CommandId: cmd.CommandId,
		AgentId:   cmd.AgentId,
		Status:    pb.ResultStatus_RESULT_STATUS_SUCCESS,
		Data:      resultData,
	}

	// Schedule agent shutdown in a separate goroutine
	// This allows us to return the result before actually stopping
	go func() {
		// Wait a moment to ensure the result is sent
		time.Sleep(500 * time.Millisecond)
		h.logger.Info("🛑 Initiating agent shutdown", zap.String("reason", reason))
		h.agent.Stop()
	}()

	// Return result immediately
	return result, nil
}

// simulateScanOperation simulates a scanning operation with realistic timing
func simulateScanOperation(ctx context.Context, logger *zap.Logger, target string, scanType string) time.Duration {
	startTime := time.Now()

	// Determine the duration based on scan type
	var duration time.Duration
	switch scanType {
	case "quick":
		duration = 2 * time.Second
	case "full":
		duration = 5 * time.Second
	default:
		duration = 3 * time.Second
	}

	// Log progress
	logger.Info("🔄 Scan in progress",
		zap.String("target", target),
		zap.String("scan_type", scanType),
		zap.Duration("estimated_duration", duration))

	// Simulate work with progress updates
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			logger.Warn("⚠️ Scan cancelled", zap.String("reason", ctx.Err().Error()))
			return time.Since(startTime)
		case <-time.After(duration / 5):
			progress := (i + 1) * 20
			logger.Debug("🔄 Scan progress",
				zap.Int("percent", progress),
				zap.String("target", target))
		}
	}

	// Return the actual time taken
	return time.Since(startTime)
}

// simulateUpdateOperation simulates an update operation with realistic timing
func simulateUpdateOperation(ctx context.Context, logger *zap.Logger, updateType string, version string) time.Duration {
	startTime := time.Now()

	// Determine the duration based on update type
	var duration time.Duration
	switch updateType {
	case "definitions":
		duration = 2 * time.Second
	case "software":
		duration = 4 * time.Second
	default:
		duration = 3 * time.Second
	}

	// Log progress
	logger.Info("🔄 Update in progress",
		zap.String("update_type", updateType),
		zap.String("version", version),
		zap.Duration("estimated_duration", duration))

	// Simulate work with progress updates
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			logger.Warn("⚠️ Update cancelled", zap.String("reason", ctx.Err().Error()))
			return time.Since(startTime)
		case <-time.After(duration / 5):
			progress := (i + 1) * 20
			logger.Debug("🔄 Update progress",
				zap.Int("percent", progress),
				zap.String("update_type", updateType))
		}
	}

	// Return the actual time taken
	return time.Since(startTime)
}

// ExecCommandHandler implements the handler for executing shell commands
type ExecCommandHandler struct {
	logger *zap.Logger
}

// Execute processes a shell command execution
func (h *ExecCommandHandler) Execute(ctx context.Context, cmd *pb.CommandMessage) (*pb.ResultMessage, error) {
	h.logger.Info("🔧 Executing shell command",
		zap.String("command_id", cmd.CommandId),
		zap.Any("parameters", cmd.Parameters))

	// Extract parameters
	shellCommand := cmd.Parameters["command"]
	timeoutStr := cmd.Parameters["timeout"]

	if shellCommand == "" {
		return nil, fmt.Errorf("missing required parameter: command")
	}

	// Parse timeout with default fallback
	timeoutSec := 30
	if timeoutStr != "" {
		if _, err := fmt.Sscanf(timeoutStr, "%d", &timeoutSec); err != nil {
			h.logger.Warn("Invalid timeout format, using default",
				zap.String("timeout", timeoutStr),
				zap.Int("default_seconds", timeoutSec))
		}
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Log command execution start
	h.logger.Info("🚀 Executing command",
		zap.String("command", shellCommand),
		zap.Int("timeout_seconds", timeoutSec))

	// Execute the command
	output, err := executeShellCommand(execCtx, shellCommand)

	// Create result data
	resultData := map[string]string{
		"command":   shellCommand,
		"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	}

	if err != nil {
		h.logger.Error("❌ Command execution failed",
			zap.String("command", shellCommand),
			zap.Error(err))

		resultData["status"] = "failed"
		resultData["error"] = err.Error()

		return &pb.ResultMessage{
			CommandId:    cmd.CommandId,
			AgentId:      cmd.AgentId,
			Status:       pb.ResultStatus_RESULT_STATUS_ERROR,
			Data:         resultData,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Add output to result data
	resultData["status"] = "success"
	resultData["output"] = output

	// Log command completion
	h.logger.Info("✅ Command executed successfully",
		zap.String("command", shellCommand),
		zap.String("output_length", fmt.Sprintf("%d bytes", len(output))))

	// Return result
	return &pb.ResultMessage{
		CommandId: cmd.CommandId,
		AgentId:   cmd.AgentId,
		Status:    pb.ResultStatus_RESULT_STATUS_SUCCESS,
		Data:      resultData,
	}, nil
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
