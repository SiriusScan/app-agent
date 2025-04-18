package agent

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	// Import for Host struct

	"github.com/SiriusScan/app-agent/internal/commands"
	_ "github.com/SiriusScan/app-agent/internal/commands/scan"   // Import for side-effect (registration)
	_ "github.com/SiriusScan/app-agent/internal/commands/status" // Import for side-effect (registration)
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/shell"
	pb "github.com/SiriusScan/app-agent/proto/hello"
)

// Agent implements the client side of HelloService
type Agent struct {
	logger    *zap.Logger
	config    *config.AgentConfig
	conn      *grpc.ClientConn
	client    pb.HelloServiceClient
	stream    pb.HelloService_ConnectStreamClient
	startTime time.Time          // Time the agent was initialized
	agentInfo commands.AgentInfo // Dependencies to pass to commands

	// PowerShell related fields
	powerShellPath   string // Detected or configured path to PowerShell/pwsh
	scriptingEnabled bool   // Whether scripting is available and enabled
}

// NewAgent creates a new HelloService client (agent)
func NewAgent(cfg *config.AgentConfig, logger *zap.Logger) *Agent {
	psPath := cfg.PowerShellPath
	psFound := false
	var findErr error

	if psPath == "" {
		// If no override path, try to find PowerShell/pwsh
		logger.Debug("PowerShell path not configured, attempting to find...")
		psPath, psFound, findErr = shell.FindPowerShell()
		if findErr != nil {
			// Log the error but don't fail agent initialization
			logger.Warn("Error finding PowerShell", zap.String("executable", psPath), zap.Error(findErr))
		} else if psFound {
			logger.Info("Found PowerShell executable", zap.String("path", psPath))
		} else {
			logger.Info("PowerShell executable (pwsh/powershell.exe) not found in PATH")
		}
	} else {
		// If path is configured, assume it's usable (existence check might be added later if needed)
		logger.Info("Using configured PowerShell path", zap.String("path", psPath))
		psFound = true // Assume configured path is valid for capability reporting
	}

	// Determine final scripting capability
	scriptingIsEnabled := cfg.EnableScripting && psFound
	logger.Info("Scripting capability determined",
		zap.Bool("config_enabled", cfg.EnableScripting),
		zap.Bool("powershell_found", psFound),
		zap.Bool("scripting_active", scriptingIsEnabled))

	// Create API Client adapter
	apiAdapter := commands.NewAPIClientAdapter()

	// Create AgentInfo struct
	agentInfo := commands.AgentInfo{
		Logger:           logger,
		Config:           cfg,
		APIClient:        apiAdapter,
		StartTime:        time.Now(),         // Use current time for AgentInfo
		ScriptingEnabled: scriptingIsEnabled, // Pass the determined status
		PowerShellPath:   psPath,             // Pass the found/configured path
	}

	return &Agent{
		logger:           logger,
		config:           cfg,
		startTime:        agentInfo.StartTime, // Align agent startTime with AgentInfo
		powerShellPath:   psPath,              // Store the found or configured path
		scriptingEnabled: scriptingIsEnabled,
		agentInfo:        agentInfo, // Store the AgentInfo struct
	}
}

// Connect establishes a connection to the gRPC server and opens the stream
func (a *Agent) Connect(ctx context.Context) error {
	a.logger.Info("Connecting to server", zap.String("address", a.config.ServerAddress))

	conn, err := grpc.DialContext(
		ctx,
		a.config.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Wait for connection to be established
	)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	a.conn = conn
	a.client = pb.NewHelloServiceClient(conn)
	a.logger.Info("Connected to server successfully")

	// --- Start Stream with Metadata ---
	a.logger.Info("Opening bidirectional stream with server")

	// Prepare metadata
	md := metadata.New(map[string]string{
		"agent_id":          a.config.AgentID,
		"scripting_enabled": strconv.FormatBool(a.scriptingEnabled),
		// Add other relevant capabilities here if needed
	})
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	// Open the bidirectional stream with metadata
	stream, err := a.client.ConnectStream(streamCtx)
	if err != nil {
		// Close the underlying connection if stream fails
		a.conn.Close()
		return fmt.Errorf("failed to establish stream: %w", err)
	}
	a.stream = stream
	a.logger.Info("Stream established successfully")

	return nil
}

// Close closes the gRPC connection
func (a *Agent) Close() error {
	if a.conn != nil {
		a.logger.Info("Closing connection to server")
		return a.conn.Close()
	}
	return nil
}

// Ping sends a ping request to the server
func (a *Agent) Ping(ctx context.Context) error {
	a.logger.Info("Sending ping request to server", zap.String("agent_id", a.config.AgentID))

	// Create ping request
	req := &pb.PingRequest{
		AgentId: a.config.AgentID,
	}

	// Send the request
	resp, err := a.client.Ping(ctx, req)
	if err != nil {
		a.logger.Error("Failed to ping server", zap.Error(err))
		return fmt.Errorf("failed to ping server: %w", err)
	}

	a.logger.Info("Ping successful",
		zap.String("message", resp.Message),
		zap.Int64("timestamp", resp.Timestamp))

	return nil
}

// WaitForCommands listens for commands from the server using the established stream
func (a *Agent) WaitForCommands(ctx context.Context) error {
	if a.stream == nil {
		return fmt.Errorf("stream not established; call Connect first")
	}

	a.logger.Info("Starting to wait for commands on established stream")

	// Send initial heartbeat (optional, could be done after receiving first server message)
	if err := a.sendHeartbeat(ctx); err != nil {
		a.logger.Warn("Failed to send initial heartbeat", zap.Error(err))
		// Decide if this is fatal; for now, just log it.
	}

	// Start background heartbeat routine
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	go a.heartbeatRoutine(heartbeatCtx)

	// Listen for messages from the server
	for {
		msg, err := a.stream.Recv()
		if err != nil {
			a.logger.Error("Error receiving message from server stream", zap.Error(err))
			return fmt.Errorf("error receiving message from server stream: %w", err)
		}

		// Process the message based on its type
		switch msg.Type {
		case pb.MessageType_COMMAND:
			a.handleCommand(ctx, msg.GetCommand())
		case pb.MessageType_ACKNOWLEDGMENT:
			a.handleAcknowledgment(msg.GetAcknowledgment())
		default:
			a.logger.Warn("Received unknown message type", zap.Int32("type", int32(msg.Type)))
		}

		// Check if context is done
		select {
		case <-ctx.Done():
			a.logger.Info("Context cancelled, stopping command listener")
			return ctx.Err()
		default:
			// Continue processing
		}
	}
}

// heartbeatRoutine sends periodic heartbeats to the server
func (a *Agent) heartbeatRoutine(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.sendHeartbeat(ctx); err != nil {
				a.logger.Error("Failed to send heartbeat", zap.Error(err))
			}
		case <-ctx.Done():
			a.logger.Info("Heartbeat routine stopping due to context cancellation")
			return
		}
	}
}

// sendHeartbeat sends a heartbeat message to the server
func (a *Agent) sendHeartbeat(ctx context.Context) error {
	// Only send if we have a stream
	if a.stream == nil {
		return fmt.Errorf("no active stream")
	}

	// Get system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Create heartbeat message
	heartbeat := &pb.HeartbeatMessage{
		Timestamp:   time.Now().Unix(),
		CpuUsage:    0.0,                                   // This should be replaced with actual CPU usage
		MemoryUsage: float64(memStats.Alloc) / 1024 / 1024, // MB
	}

	// Create agent message
	msg := &pb.AgentMessage{
		AgentId: a.config.AgentID,
		Type:    pb.MessageType_HEARTBEAT,
		Payload: &pb.AgentMessage_Heartbeat{
			Heartbeat: heartbeat,
		},
	}

	// Send the message
	a.logger.Debug("Sending heartbeat to server",
		zap.Int64("timestamp", heartbeat.Timestamp),
		zap.Float64("memory_usage_mb", heartbeat.MemoryUsage))

	return a.stream.Send(msg)
}

// handleCommand processes a command received from the server
func (a *Agent) handleCommand(ctx context.Context, cmdReq *pb.CommandRequest) {
	if cmdReq == nil {
		a.logger.Warn("Received nil command request")
		return
	}

	commandString := cmdReq.Command // The raw command string
	a.logger.Info("Received command string from server", zap.String("command_string", commandString))

	// Process the command string
	a.processCommandString(ctx, commandString)
}

// processCommandString parses the command string and dispatches to the appropriate handler.
func (a *Agent) processCommandString(ctx context.Context, commandString string) {
	output, err := commands.Dispatch(ctx, a.agentInfo, commandString)

	if err != nil {
		if errors.Is(err, commands.ErrUnknownCommand) {
			// If it's not a known internal command, try executing it as a script
			a.logger.Debug("Command not found in internal registry, attempting script execution", zap.String("command", commandString))
			a.executeScriptCommand(ctx, commandString)
		} else {
			// Handle error from internal command execution
			a.logger.Error("Internal command execution failed", zap.String("command", commandString), zap.Error(err))
			a.sendCommandResult(ctx, commandString, "", err.Error(), 1, 0) // Assuming exit code 1 for internal errors
		}
	} else {
		// Internal command executed successfully
		a.sendCommandResult(ctx, commandString, output, "", 0, 0) // Assuming 0 execution time for now
	}
}

// executeScriptCommand attempts to execute the command string as a script via PowerShell.
func (a *Agent) executeScriptCommand(ctx context.Context, scriptContent string) {
	a.logger.Info("Attempting to execute command as script", zap.String("script_content_preview", scriptContent[:min(len(scriptContent), 50)])) // Log preview

	if !a.scriptingEnabled {
		a.logger.Warn("Script execution requested but scripting is disabled or PowerShell not found")
		a.sendCommandResult(ctx, scriptContent, "", "Scripting is disabled on this agent", 1, 0)
		return
	}

	startTime := time.Now()
	stdout, stderr, exitCode, err := shell.ExecuteScript(ctx, a.powerShellPath, scriptContent)
	executionTime := time.Since(startTime).Milliseconds()

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		a.logger.Error("Script execution failed",
			zap.Error(err),
			zap.Int("exit_code", exitCode),
			zap.String("stderr", stderr))
	} else {
		a.logger.Info("Script executed successfully", zap.Int("exit_code", exitCode))
	}

	a.sendCommandResult(ctx, scriptContent, stdout, errMsg, int32(exitCode), executionTime)
}

// sendCommandResult sends the result of a command execution back to the server.
func (a *Agent) sendCommandResult(ctx context.Context, originalCommand, output, errorMsg string, exitCode int32, executionTimeMs int64) {
	if a.stream == nil {
		a.logger.Error("Cannot send command result: no active stream")
		return
	}

	cmdResult := &pb.CommandResult{
		Command:       originalCommand,
		Output:        output,
		Error:         errorMsg,
		ExitCode:      exitCode,
		ExecutionTime: executionTimeMs,
	}

	msg := &pb.AgentMessage{
		AgentId: a.config.AgentID,
		Type:    pb.MessageType_RESULT,
		Payload: &pb.AgentMessage_Result{
			Result: cmdResult,
		},
	}

	if err := a.stream.Send(msg); err != nil {
		a.logger.Error("Failed to send command result to server", zap.Error(err))
		// Consider how to handle send failures - maybe retry?
		return
	}

	a.logger.Info("Command result sent to server", zap.String("original_command", originalCommand[:min(len(originalCommand), 50)]))
}

// ExecuteCommand runs a shell command and sends the result to the server
// DEPRECATED: Use processCommandString instead.
func (a *Agent) ExecuteCommand(ctx context.Context, command string) error {
	a.logger.Warn("ExecuteCommand function is deprecated, use processCommandString")
	// We keep the old implementation here for reference during transition, but call the new one.
	a.processCommandString(ctx, command)
	return nil // The new function handles sending results/errors
}

// handleAcknowledgment processes an acknowledgment received from the server
func (a *Agent) handleAcknowledgment(ack *pb.Acknowledgment) {
	if ack == nil {
		a.logger.Warn("Received nil acknowledgment")
		return
	}

	a.logger.Debug("Received command acknowledgment",
		zap.String("command_id", ack.CommandId),
		zap.String("status", ack.Status))
}

// Helper function (add if not already present or import strings)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
