package agent

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/debugtrace"
	siriusruntime "github.com/SiriusScan/app-agent/internal/family/sirius/runtime"
	"github.com/SiriusScan/app-agent/internal/shell"
	templateagent "github.com/SiriusScan/app-agent/internal/template/agent"
	pb "github.com/SiriusScan/app-agent/proto/hello"
)

// Agent implements the client side of HelloService
type Agent struct {
	logger    *zap.Logger
	config    *config.AgentConfig
	transport *grpcTransport
	startTime time.Time          // Time the agent was initialized
	agentInfo commands.AgentInfo // Dependencies to pass to commands

	// PowerShell related fields
	powerShellPath   string // Detected or configured path to PowerShell/pwsh
	scriptingEnabled bool   // Whether scripting is available and enabled

	// Template sync manager
	syncManager *templateagent.AgentSyncManager

	// Auth token for gRPC authentication
	authToken string
}

// NewAgent creates a new HelloService client (agent)
func NewAgent(cfg *config.AgentConfig, logger *zap.Logger) *Agent {
	runtimeCtx := siriusruntime.NewContext(cfg, logger)

	return &Agent{
		logger:           logger,
		config:           cfg,
		transport:        newGRPCTransport(logger),
		startTime:        runtimeCtx.AgentInfo.StartTime,
		powerShellPath:   runtimeCtx.PowerShellPath,
		scriptingEnabled: runtimeCtx.ScriptingEnabled,
		agentInfo:        runtimeCtx.AgentInfo,
		authToken:        cfg.AuthToken,
	}
}

// Connect establishes a connection to the gRPC server and opens the stream
func (a *Agent) Connect(ctx context.Context) error {
	a.logger.Info("Connecting to server", zap.String("address", a.config.ServerAddress))

	if err := a.transport.Connect(a.config.ServerAddress); err != nil {
		return err
	}
	a.logger.Info("Connected to server successfully")

	// --- Start Stream with Metadata ---
	a.logger.Info("Opening bidirectional stream with server")

	stream, err := a.transport.OpenStream(ctx, map[string]string{
		"agent_id":          a.config.AgentID,
		"scripting_enabled": strconv.FormatBool(a.scriptingEnabled),
		// Add other relevant capabilities here if needed
	})
	if err != nil {
		return err
	}
	_ = stream
	a.logger.Info("Stream established successfully")

	return nil
}

// Close closes the gRPC connection
func (a *Agent) Close() error {
	if a.transport != nil {
		a.logger.Info("Closing connection to server")
		return a.transport.Close()
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
	resp, err := a.transport.Ping(ctx, req)
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
	if a.transport.Stream() == nil {
		return fmt.Errorf("stream not established; call Connect first")
	}

	a.logger.Info("Starting to wait for commands on established stream")
	// #region agent log
	debugtrace.Log("pre-fix", "H1,H2,H4", "internal/agent/agent.go:125", "wait_for_commands_enter", map[string]interface{}{
		"agentId":       a.config.AgentID,
		"serverAddress": a.config.ServerAddress,
		"tokenFilePath": a.config.TokenFilePath,
		"hasToken":      a.authToken != "",
	})
	// #endregion

	// ── Send initial heartbeat with auth token ────────────────────────────
	// The server expects the very first message to contain the agent_id
	// and (optionally) the auth_token for authentication.
	if err := a.sendAuthenticatedHeartbeat(ctx); err != nil {
		// #region agent log
		debugtrace.Log("pre-fix", "H1,H2,H4", "internal/agent/agent.go:136", "initial_heartbeat_failed", map[string]interface{}{
			"agentId": a.config.AgentID,
			"error":   err.Error(),
		})
		// #endregion
		return fmt.Errorf("failed to send initial authenticated heartbeat: %w", err)
	}

	// ── Receive and process the welcome message ───────────────────────────
	// The server responds with a welcome/status command that may carry a
	// newly-issued auth_token for brand-new agents.
	welcomeMsg, err := a.transport.Stream().Recv()
	if err != nil {
		a.logger.Error("Error receiving welcome message from server", zap.Error(err))
		// #region agent log
		debugtrace.Log("pre-fix", "H1,H3,H4", "internal/agent/agent.go:147", "welcome_receive_failed", map[string]interface{}{
			"agentId":       a.config.AgentID,
			"tokenFilePath": a.config.TokenFilePath,
			"hasToken":      a.authToken != "",
			"error":         err.Error(),
		})
		// #endregion
		if strings.Contains(err.Error(), "agent token validation failed") {
			a.logger.Error(
				"Server rejected the persisted auth token — common after switching Sirius engines, resetting Valkey, or using Docker Sirius. Clear the token and reconnect so the engine can issue a new one.",
				zap.String("token_file", a.config.TokenFilePath),
				zap.String("unset_env", "AGENT_AUTH_TOKEN"),
			)
		}
		return fmt.Errorf("error receiving welcome message: %w", err)
	}

	// Capture newly issued token (non-empty only when server generated one).
	if newToken := welcomeMsg.GetAuthToken(); newToken != "" {
		a.authToken = newToken
		a.logger.Info("Received new auth token from server — persisting")
		if err := a.config.SaveAuthToken(newToken); err != nil {
			a.logger.Warn("Failed to persist auth token to file (will retry next connect)",
				zap.Error(err))
		}
	}
	// #region agent log
	debugtrace.Log("pre-fix", "H1,H3,H5", "internal/agent/agent.go:170", "welcome_received", map[string]interface{}{
		"agentId":         a.config.AgentID,
		"messageType":     int32(welcomeMsg.GetType()),
		"command":         welcomeMsg.GetCommand().GetCommand(),
		"authTokenIssued": welcomeMsg.GetAuthToken() != "",
		"tokenFilePath":   a.config.TokenFilePath,
	})
	// #endregion

	// Process the welcome message content (usually an internal:status command).
	a.processServerMessage(ctx, welcomeMsg)

	// Start background heartbeat routine
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	go a.heartbeatRoutine(heartbeatCtx)

	// Listen for messages from the server
	for {
		msg, err := a.transport.Stream().Recv()
		if err != nil {
			a.logger.Error("Error receiving message from server stream", zap.Error(err))
			return fmt.Errorf("error receiving message from server stream: %w", err)
		}

		a.processServerMessage(ctx, msg)

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

// processServerMessage dispatches a single ServerMessage to the appropriate handler.
func (a *Agent) processServerMessage(ctx context.Context, msg *pb.ServerMessage) {
	switch msg.Type {
	case pb.MessageType_COMMAND:
		a.handleCommand(ctx, msg.GetCommand())
	case pb.MessageType_ACKNOWLEDGMENT:
		a.handleAcknowledgment(msg.GetAcknowledgment())
	case pb.MessageType_TEMPLATE_UPDATE:
		a.handleTemplateUpdate(ctx, msg.GetTemplateUpdate())
	default:
		a.logger.Warn("Received unknown message type", zap.Int32("type", int32(msg.Type)))
	}
}

// sendAuthenticatedHeartbeat sends the initial heartbeat including the auth
// token so the server can authenticate (or provision) this agent.
func (a *Agent) sendAuthenticatedHeartbeat(ctx context.Context) error {
	if a.transport.Stream() == nil {
		return fmt.Errorf("no active stream")
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	heartbeat := &pb.HeartbeatMessage{
		Timestamp:   time.Now().Unix(),
		CpuUsage:    0.0,
		MemoryUsage: float64(memStats.Alloc) / 1024 / 1024,
	}

	msg := &pb.AgentMessage{
		AgentId: a.config.AgentID,
		Type:    pb.MessageType_HEARTBEAT,
		Payload: &pb.AgentMessage_Heartbeat{
			Heartbeat: heartbeat,
		},
		AuthToken: a.authToken, // Empty on first-ever connection; populated on reconnects.
	}

	a.logger.Info("Sending authenticated initial heartbeat",
		zap.String("agent_id", a.config.AgentID),
		zap.Bool("has_token", a.authToken != ""))

	return a.StreamSend(msg)
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
	if a.transport.Stream() == nil {
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

	return a.StreamSend(msg)
}

// StreamSend sends a message on the gRPC stream with mutex protection.
// gRPC stream Send() is not safe for concurrent calls from multiple goroutines.
func (a *Agent) StreamSend(msg *pb.AgentMessage) error {
	return a.transport.Send(msg)
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
	if a.transport.Stream() == nil {
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

	if err := a.StreamSend(msg); err != nil {
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

// handleTemplateUpdate processes template update messages from the server
func (a *Agent) handleTemplateUpdate(ctx context.Context, update *pb.TemplateUpdate) {
	if update == nil {
		a.logger.Warn("Received nil template update")
		return
	}

	if a.syncManager == nil {
		a.logger.Warn("Sync manager not initialized, cannot process template update")
		return
	}

	a.logger.Info("Processing template update from server",
		zap.String("template_id", update.TemplateId),
		zap.Bool("is_custom", update.IsCustom))

	if err := a.syncManager.HandleTemplateUpdate(ctx, update); err != nil {
		a.logger.Error("Failed to process template update",
			zap.String("template_id", update.TemplateId),
			zap.Error(err))
	} else {
		a.logger.Info("Template update processed successfully",
			zap.String("template_id", update.TemplateId))
	}
}

// SetSyncManager sets the template sync manager for this agent
func (a *Agent) SetSyncManager(syncManager *templateagent.AgentSyncManager) {
	a.syncManager = syncManager
	a.agentInfo.TemplateSync = &templateSyncAdapter{manager: syncManager}
	a.logger.Info("Template sync manager set for agent")
}

// GetStream returns the gRPC stream for this agent
func (a *Agent) GetStream() pb.HelloService_ConnectStreamClient {
	return a.transport.Stream()
}

// Helper function (add if not already present or import strings)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
