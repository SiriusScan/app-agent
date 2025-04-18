package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/SiriusScan/app-agent/internal/command"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/store"
	pb "github.com/SiriusScan/app-agent/proto/hello"

	// "github.com/SiriusScan/go-api/sirius/agent/models" // Commented out due to import error
	"github.com/SiriusScan/go-api/sirius/queue"
	// "github.com/sirius-project/app-agent/internal/valkey" // Commented out due to import error
)

// Define queue names (should match frontend)
const AGENT_COMMAND_QUEUE = "agent_commands"
const AGENT_RESPONSE_QUEUE = "agent_response"
const TERMINAL_RESPONSE_QUEUE = "terminal_response" // Keep for potential direct terminal responses if needed

// Command represents a command to be executed
type Command struct {
	Command       string `json:"command"`
	UserID        string `json:"user_id"`
	Timestamp     string `json:"timestamp"`
	ResponseQueue string `json:"response_queue"`
}

// CommandStatus represents the current status of a command
type CommandStatus struct {
	ID           string
	AgentID      string
	Command      string
	Status       string // "pending", "sent", "completed", "failed"
	StartTime    time.Time
	CompleteTime time.Time
	Result       *pb.CommandResult
	Error        error
}

// CommandMessage represents a message received from the queue
type CommandMessage struct {
	Action    string   `json:"action,omitempty"` // For special actions like list_agents, initialize_session
	Command   string   `json:"command,omitempty"`
	AgentID   string   `json:"agentId,omitempty"`
	UserID    string   `json:"userId"`
	Timestamp string   `json:"timestamp"`
	SessionID string   `json:"sessionId,omitempty"` // Used for initialize_session
	Target    struct { // Use inline struct temporarily
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"target,omitempty"`
	ResponseQueue string `json:"responseQueue,omitempty"`
}

// Response structure
type CommandResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"` // Generic message
}

// Agent structure (assuming basic details)
type Agent struct {
	ID        string
	Name      string
	Status    string
	LastSeen  time.Time
	SessionID string // Track active session
	// Add other necessary fields like connection details
}

// Server implements the HelloService gRPC server
type Server struct {
	pb.UnimplementedHelloServiceServer
	logger  *zap.Logger
	config  *config.ServerConfig
	server  *grpc.Server
	logFile *os.File

	// Track connected agents and their streams
	agentsMutex sync.RWMutex
	agents      map[string]pb.HelloService_ConnectStreamServer // Store the gRPC stream

	// Track command status
	commandsMutex sync.RWMutex
	commands      map[string]*CommandStatus

	// Queue processing
	queueCtx    context.Context
	queueCancel context.CancelFunc

	// Response storage
	responseStore store.ResponseStore

	// Map to correlate command string with unique command ID for queue commands
	pendingCommandsMutex sync.Mutex
	pendingCommands      map[string]string // Key: agentID:commandString, Value: agentID:timestampID

	// valkey *valkey.Valkey // Commented out due to import error
}

// NewServer creates a new HelloService server
func NewServer(cfg *config.ServerConfig, logger *zap.Logger) (*Server, error) {
	// Create a log file for command output
	logFile, err := os.OpenFile("server_commands.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("Failed to open log file", zap.Error(err))
		// Print to console as well
		fmt.Printf("Failed to open log file: %v\n", err)
	} else {
		fmt.Printf("Successfully opened log file: %s\n", "server_commands.log")
	}

	// Initialize response store
	responseStore, err := store.NewResponseStore()
	if err != nil {
		logger.Error("Failed to create response store", zap.Error(err))
		return nil, fmt.Errorf("failed to create response store: %w", err)
	}

	/* // Commented out due to import error
	valkeyInstance, err := valkey.NewValkey()
	if err != nil {
		logger.Error("Failed to create Valkey instance", zap.Error(err))
		return nil, fmt.Errorf("failed to create Valkey instance: %w", err)
	}
	*/

	return &Server{
		logger:        logger,
		config:        cfg,
		logFile:       logFile,
		agents:        make(map[string]pb.HelloService_ConnectStreamServer),
		commands:      make(map[string]*CommandStatus),
		responseStore: responseStore,
		// valkey:        valkeyInstance, // Commented out due to import error
		pendingCommands: make(map[string]string),
	}, nil
}

// Ping implements the Ping RPC method
func (s *Server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	agentID := req.AgentId
	s.agentsMutex.RLock()
	_, exists := s.agents[agentID] // Check if stream exists
	s.agentsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not connected", agentID)
	}

	// If stream exists, agent is considered connected.
	s.logger.Info("Received ping from agent", zap.String("agent_id", agentID))
	return &pb.PingResponse{Message: "Pong"}, nil
}

// ConnectStream implements the bidirectional streaming RPC
func (s *Server) ConnectStream(stream pb.HelloService_ConnectStreamServer) error {
	var agentID string

	// Wait for the first message to get the agent ID
	msg, err := stream.Recv()
	if err != nil {
		s.logger.Error("Failed to receive initial message", zap.Error(err))
		return fmt.Errorf("failed to receive initial message: %w", err)
	}

	// Extract the agent ID from the message
	agentID = msg.AgentId
	if agentID == "" {
		s.logger.Error("Agent ID missing from initial message")
		return fmt.Errorf("agent ID missing from initial message")
	}

	s.logger.Info("Agent connected to stream", zap.String("agent_id", agentID))
	fmt.Printf("Agent connected to stream: %s\n", agentID)

	// Register the agent
	s.agentsMutex.Lock()
	s.agents[agentID] = stream
	s.agentsMutex.Unlock()

	// Cleanup when the function exits
	defer func() {
		s.agentsMutex.Lock()
		delete(s.agents, agentID)
		s.agentsMutex.Unlock()
		s.logger.Info("Agent disconnected from stream", zap.String("agent_id", agentID))
	}()

	// Send welcome message - trigger initial status update/scan
	welcomeMsg := &pb.ServerMessage{
		Id:   "welcome-status", // Changed ID slightly for clarity
		Type: pb.MessageType_COMMAND,
		Payload: &pb.ServerMessage_Command{
			Command: &pb.CommandRequest{
				Command: "internal:status", // Send status command on connect
			},
		},
	}

	if err := stream.Send(welcomeMsg); err != nil {
		s.logger.Error("Failed to send initial status command", zap.String("agent_id", agentID), zap.Error(err))
		return fmt.Errorf("failed to send initial status command: %w", err)
	}
	s.logger.Info("Sent initial status command to agent", zap.String("agent_id", agentID))

	// Process incoming messages
	for {
		msg, err := stream.Recv()
		if err != nil {
			s.logger.Error("Error receiving message from agent",
				zap.String("agent_id", agentID),
				zap.Error(err))
			return fmt.Errorf("error receiving message from agent: %w", err)
		}

		// Process message based on type
		switch msg.Type {
		case pb.MessageType_HEARTBEAT:
			s.handleHeartbeat(agentID, msg.GetHeartbeat())

		case pb.MessageType_RESULT:
			s.handleCommandResult(agentID, msg.GetResult())

		default:
			s.logger.Warn("Received unknown message type",
				zap.String("agent_id", agentID),
				zap.Int32("message_type", int32(msg.Type)))
		}
	}
}

// handleHeartbeat processes heartbeat messages from agents
func (s *Server) handleHeartbeat(agentID string, heartbeat *pb.HeartbeatMessage) {
	if heartbeat == nil {
		s.logger.Warn("Received empty heartbeat", zap.String("agent_id", agentID))
		return
	}

	s.logger.Debug("Received heartbeat from agent",
		zap.String("agent_id", agentID),
		zap.Int64("timestamp", heartbeat.Timestamp),
		zap.Float64("cpu_usage", heartbeat.CpuUsage),
		zap.Float64("memory_usage", heartbeat.MemoryUsage))
}

// handleCommandResult processes command result messages from agents
func (s *Server) handleCommandResult(agentID string, result *pb.CommandResult) {
	if result == nil {
		s.logger.Warn("Received empty command result", zap.String("agent_id", agentID))
		return
	}

	// Look up the original command ID (agentID:timestamp) using agentID and command string
	pendingKey := agentID + ":" + result.Command
	s.pendingCommandsMutex.Lock()
	commandID, found := s.pendingCommands[pendingKey]
	if found {
		delete(s.pendingCommands, pendingKey) // Remove entry once processed
	}
	s.pendingCommandsMutex.Unlock()

	if !found {
		// If not found, check if it's the initial status command
		if result.Command == "internal:status" {
			s.logger.Info("Received result for initial 'internal:status' command",
				zap.String("agent_id", agentID),
				zap.Int32("exit_code", result.ExitCode))
			// Optionally process/log the result.Output here if needed
			// Don't store in ResponseStore as there's no queue command ID
			s.sendCommandAcknowledgment(agentID, result)
			return // Handled, exit function
		}

		// If not found and not the initial status command, log the warning
		s.logger.Warn("Could not find pending command mapping for result",
			zap.String("lookup_key", pendingKey),
			zap.String("agent_id", agentID),
			zap.String("command", result.Command))
		// Optionally, could attempt the old s.commands lookup here as a fallback if needed
		// Send acknowledgment even if not found, agent did its job
		s.sendCommandAcknowledgment(agentID, result)
		return // Cannot proceed with storage
	}

	// Create and store command response using the retrieved commandID
	response := command.NewCommandResponse(commandID, agentID, result.Command)
	if result.ExitCode == 0 {
		response.SetCompleted(result.Output, int(result.ExitCode))
	} else {
		if result.Error != "" {
			response.SetFailed(fmt.Errorf("%s", result.Error))
		} else {
			response.SetFailed(fmt.Errorf("command failed with exit code %d", result.ExitCode))
		}
	}

	// Store the response
	ctx := context.Background()
	if err := s.responseStore.Store(ctx, response); err != nil {
		s.logger.Error("Failed to store command response",
			zap.String("command_id", commandID),
			zap.String("agent_id", agentID),
			zap.Error(err))
	} else {
		// Add explicit nil check before logging and calling GenerateKey
		if response == nil {
			s.logger.Error("BUG: CommandResponse object is nil before logging storage success",
				zap.String("command_id", commandID),
				zap.String("agent_id", agentID))
			return // Avoid panic
		}
		s.logger.Info("Stored command response",
			zap.String("command_id", commandID),
			zap.String("key", response.GenerateKey()), // Log the key used for storage
			zap.String("agent_id", agentID))
	}

	s.logger.Info("Received command result from agent",
		zap.String("agent_id", agentID),
		zap.String("command", result.Command),
		zap.Int32("exit_code", result.ExitCode),
		zap.Int64("execution_time", result.ExecutionTime))

	// Print to stdout
	fmt.Printf("\n===== COMMAND EXECUTION RESULT =====\n")
	fmt.Printf("Agent: %s\n", agentID)
	fmt.Printf("Command: %s\n", result.Command)
	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Execution Time: %dms\n", result.ExecutionTime)
	fmt.Printf("==================================\n\n")
	fmt.Printf("Output:\n%s\n", result.Output)
	if result.Error != "" {
		fmt.Printf("Error:\n%s\n", result.Error)
	}
	fmt.Printf("\n==================================\n")

	// Log to file
	if s.logFile != nil {
		timestamp := fmt.Sprintf("[%s] ", time.Now().Format(time.RFC3339))
		header := fmt.Sprintf("\n===== COMMAND EXECUTION RESULT =====\n")
		agentLine := fmt.Sprintf("Agent: %s\n", agentID)
		commandLine := fmt.Sprintf("Command: %s\n", result.Command)
		exitCodeLine := fmt.Sprintf("Exit Code: %d\n", result.ExitCode)
		executionTimeLine := fmt.Sprintf("Execution Time: %dms\n", result.ExecutionTime)
		separator := fmt.Sprintf("==================================\n\n")
		outputPrefix := "Output:\n"
		output := result.Output

		var errorSection string
		if result.Error != "" {
			errorSection = fmt.Sprintf("\nError:\n%s\n", result.Error)
		}

		footer := fmt.Sprintf("\n==================================\n")

		fmt.Fprintf(s.logFile, "%s%s%s%s%s%s%s%s%s%s%s%s",
			timestamp, header, agentLine, commandLine, exitCodeLine, executionTimeLine,
			separator, outputPrefix, output, errorSection, footer, "\n")

		fmt.Printf("Command output logged to %s\n", "server_commands.log")
	}

	// Send acknowledgment back to the agent
	s.sendCommandAcknowledgment(agentID, result)
}

// sendCommandAcknowledgment sends an acknowledgment message back to the agent
func (s *Server) sendCommandAcknowledgment(agentID string, result *pb.CommandResult) {
	s.agentsMutex.RLock()
	stream, ok := s.agents[agentID]
	s.agentsMutex.RUnlock()

	if !ok {
		s.logger.Error("Cannot send acknowledgment - agent not connected",
			zap.String("agent_id", agentID))
		return
	}

	ackMsg := &pb.ServerMessage{
		Id:   fmt.Sprintf("ack-%d", time.Now().Unix()),
		Type: pb.MessageType_ACKNOWLEDGMENT,
		Payload: &pb.ServerMessage_Acknowledgment{
			Acknowledgment: &pb.Acknowledgment{
				CommandId: result.Command,
				Status:    "completed",
			},
		},
	}

	if err := stream.Send(ackMsg); err != nil {
		s.logger.Error("Failed to send command acknowledgment",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}
}

// StartQueueProcessor starts listening for commands on the agent queue
func (s *Server) StartQueueProcessor(ctx context.Context) {
	s.logger.Info("Starting agent queue processor")
	s.queueCtx, s.queueCancel = context.WithCancel(ctx)

	go func() {
		processor := func(msg string) {
			s.logger.Info("Received message on agent queue", zap.String("message", msg))
			var cmdMsg CommandMessage
			if err := json.Unmarshal([]byte(msg), &cmdMsg); err != nil {
				s.logger.Error("Failed to unmarshal command message", zap.Error(err), zap.String("message", msg))
				s.sendErrorResponse(AGENT_RESPONSE_QUEUE, "Invalid command format")
				return
			}

			// Added log to check parsed agent ID
			s.logger.Debug("Parsed command message", zap.String("agent_id", cmdMsg.AgentID), zap.String("command", cmdMsg.Command), zap.String("action", cmdMsg.Action))

			switch cmdMsg.Action {
			case "list_agents":
				s.handleListAgents()
			case "initialize_session":
				s.handleInitializeSession(cmdMsg)
			default:
				// Assume regular command if Action is empty and Command is present
				if cmdMsg.Command != "" {
					s.handleExecuteCommand(cmdMsg)
				} else {
					s.logger.Warn("Unknown or empty action/command received", zap.Any("command", cmdMsg))
					s.sendErrorResponse(AGENT_RESPONSE_QUEUE, "Unknown or empty command")
				}
			}
		}

		queue.Listen(AGENT_COMMAND_QUEUE, processor) // Use imported queue.Listen
		s.logger.Info("Agent queue processor stopped")
	}()
}

// handleListAgents sends the list of connected agents to AGENT_RESPONSE_QUEUE
func (s *Server) handleListAgents() {
	s.agentsMutex.RLock()
	agentIDs := make([]string, 0, len(s.agents))
	for id := range s.agents {
		agentIDs = append(agentIDs, id)
	}
	s.agentsMutex.RUnlock()

	responseBytes, err := json.Marshal(agentIDs) // Send back only the array of IDs
	if err != nil {
		s.logger.Error("Failed to marshal agent list", zap.Error(err))
		s.sendErrorResponse(AGENT_RESPONSE_QUEUE, "Failed to create agent list response")
		return
	}

	// Send response to the dedicated agent response queue
	if err := queue.Send(AGENT_RESPONSE_QUEUE, string(responseBytes)); err != nil {
		s.logger.Error("Failed to send agent list response", zap.Error(err))
	}
	s.logger.Info("Sent agent list response", zap.Strings("agents", agentIDs))
}

// handleInitializeSession validates and confirms session start, responds to AGENT_RESPONSE_QUEUE
func (s *Server) handleInitializeSession(cmd CommandMessage) {
	s.agentsMutex.RLock()
	_, exists := s.agents[cmd.AgentID]
	s.agentsMutex.RUnlock()

	if !exists {
		s.logger.Warn("Initialize session failed: Agent not found", zap.String("agentId", cmd.AgentID))
		s.sendErrorResponse(AGENT_RESPONSE_QUEUE, "Agent "+cmd.AgentID+" not found")
		return
	}

	s.logger.Info("Initialized session for agent", zap.String("sessionId", cmd.SessionID), zap.String("agentId", cmd.AgentID))

	// Send success confirmation to the dedicated agent response queue
	s.sendSuccessResponse(AGENT_RESPONSE_QUEUE, "Session initialized successfully")
}

// handleExecuteCommand sends command to agent via stream and acknowledges to AGENT_RESPONSE_QUEUE
func (s *Server) handleExecuteCommand(cmd CommandMessage) {
	if cmd.Target.Type != "agent" || cmd.Target.ID == "" {
		s.logger.Error("Invalid target for agent command execution", zap.Any("target", cmd.Target))
		s.sendErrorResponse(AGENT_RESPONSE_QUEUE, "Invalid target for agent command")
		return
	}

	targetAgentID := cmd.Target.ID

	s.agentsMutex.RLock()
	stream, exists := s.agents[targetAgentID]
	s.agentsMutex.RUnlock()

	if !exists {
		s.logger.Warn("Execute command failed: Agent not connected via stream", zap.String("agentId", targetAgentID))
		s.sendErrorResponse(AGENT_RESPONSE_QUEUE, "Agent "+targetAgentID+" not connected")
		return
	}

	// 1. Send acknowledgement back to the frontend immediately via AGENT_RESPONSE_QUEUE
	s.sendSuccessResponse(AGENT_RESPONSE_QUEUE, "Command received, forwarding to agent")

	// 2. Generate the unique command ID
	commandID := fmt.Sprintf("%s:%s", targetAgentID, cmd.Timestamp)

	// 3. Store the mapping before sending
	pendingKey := targetAgentID + ":" + cmd.Command
	s.pendingCommandsMutex.Lock()
	s.pendingCommands[pendingKey] = commandID
	s.pendingCommandsMutex.Unlock()

	// 4. Forward command to the actual agent via gRPC stream
	commandToSend := &pb.ServerMessage{
		Id:   commandID, // Use the generated unique ID
		Type: pb.MessageType_COMMAND,
		Payload: &pb.ServerMessage_Command{
			Command: &pb.CommandRequest{
				Command: cmd.Command,
			},
		},
	}

	s.logger.Info("Forwarding command to agent stream", zap.String("agentId", targetAgentID), zap.String("commandId", commandID), zap.String("command", cmd.Command))
	if err := stream.Send(commandToSend); err != nil {
		s.logger.Error("Failed to send command to agent stream", zap.String("agentId", targetAgentID), zap.Error(err))
		// Remove from pending map if send fails
		s.pendingCommandsMutex.Lock()
		delete(s.pendingCommands, pendingKey)
		s.pendingCommandsMutex.Unlock()
		// Note: No response sent back here on send failure, frontend will time out polling Valkey
		s.agentsMutex.Lock()
		delete(s.agents, targetAgentID)
		s.agentsMutex.Unlock()
	}
}

// sendSuccessResponse sends a generic success message to the specified queue
func (s *Server) sendSuccessResponse(queueName string, message string) {
	response := CommandResponse{
		Success: true,
		Message: message,
	}
	responseBytes, _ := json.Marshal(response)
	if err := queue.Send(queueName, string(responseBytes)); err != nil {
		s.logger.Error("Failed to send success response", zap.String("queue", queueName), zap.Error(err))
	}
}

// sendErrorResponse sends an error message to the specified queue
func (s *Server) sendErrorResponse(queueName string, errorMsg string) {
	response := CommandResponse{
		Success: false,
		Error:   errorMsg,
	}
	responseBytes, _ := json.Marshal(response)
	if err := queue.Send(queueName, string(responseBytes)); err != nil {
		s.logger.Error("Failed to send error response", zap.String("queue", queueName), zap.Error(err))
	}
}

// Start starts the gRPC server and command processors
func (s *Server) Start() error {
	// Create a new gRPC server
	s.server = grpc.NewServer()
	pb.RegisterHelloServiceServer(s.server, s)
	reflection.Register(s.server)

	// Create a context for the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the queue processor
	s.StartQueueProcessor(ctx)

	// Start command input processing
	go s.commandInputLoop()

	// Create listener
	lis, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Server listening",
		zap.String("address", s.config.Address))

	// Serve gRPC
	return s.server.Serve(lis)
}

// commandInputLoop reads commands from stdin and sends them to connected agents
func (s *Server) commandInputLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nAvailable commands:")
	fmt.Println("  <agent_id> <command>  - Send command to specific agent")
	fmt.Println("  list                  - Show connected agents")
	fmt.Println("  status [cmd_id]       - Show status of all or specific command")
	fmt.Println("  pending               - Show pending commands")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Print("> ")

	for scanner.Scan() {
		input := scanner.Text()
		if input == "" {
			fmt.Print("> ")
			continue
		}

		parts := strings.Fields(input)
		if len(parts) == 0 {
			fmt.Print("> ")
			continue
		}

		switch parts[0] {
		case "list":
			s.listConnectedAgents()

		case "status":
			if len(parts) > 1 {
				// Show specific command status
				if cmd := s.GetCommandStatus(parts[1]); cmd != nil {
					fmt.Printf("Command ID: %s\n", cmd.ID)
					fmt.Printf("Agent: %s\n", cmd.AgentID)
					fmt.Printf("Command: %s\n", cmd.Command)
					fmt.Printf("Status: %s\n", cmd.Status)
					fmt.Printf("Start Time: %s\n", cmd.StartTime.Format(time.RFC3339))
					if cmd.CompleteTime != (time.Time{}) {
						fmt.Printf("Complete Time: %s\n", cmd.CompleteTime.Format(time.RFC3339))
					}
					if cmd.Error != nil {
						fmt.Printf("Error: %s\n", cmd.Error)
					}
				} else {
					fmt.Printf("Command %s not found\n", parts[1])
				}
			} else {
				// Show all command statuses
				s.commandsMutex.RLock()
				for _, cmd := range s.commands {
					fmt.Printf("Command ID: %s, Agent: %s, Status: %s\n",
						cmd.ID, cmd.AgentID, cmd.Status)
				}
				s.commandsMutex.RUnlock()
			}

		case "pending":
			pending := s.ListPendingCommands()
			if len(pending) == 0 {
				fmt.Println("No pending commands")
			} else {
				fmt.Println("Pending commands:")
				for _, cmd := range pending {
					fmt.Printf("ID: %s, Agent: %s, Command: %s, Status: %s\n",
						cmd.ID, cmd.AgentID, cmd.Command, cmd.Status)
				}
			}

		default:
			// Assume it's a command for an agent
			if len(parts) < 2 {
				fmt.Println("Invalid format. Use: <agent_id> <command>")
			} else {
				agentID := parts[0]
				command := strings.Join(parts[1:], " ")
				if err := s.SendCommandToAgent(agentID, command); err != nil {
					fmt.Printf("Error: %s\n", err)
				}
			}
		}

		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		s.logger.Error("Error reading user input", zap.Error(err))
	}
}

// listConnectedAgents prints a list of all connected agents
func (s *Server) listConnectedAgents() {
	s.agentsMutex.RLock()
	defer s.agentsMutex.RUnlock()

	if len(s.agents) == 0 {
		fmt.Println("No agents connected")
		return
	}

	fmt.Println("Connected agents:")
	for agentID := range s.agents {
		fmt.Printf("- %s\n", agentID)
	}
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	s.logger.Info("Stopping server")

	// Cancel queue processing
	if s.queueCancel != nil {
		s.queueCancel()
	}

	// Stop the gRPC server
	if s.server != nil {
		s.server.GracefulStop()
	}

	// Close the log file
	if s.logFile != nil {
		if err := s.logFile.Close(); err != nil {
			s.logger.Error("Failed to close log file", zap.Error(err))
		}
	}

	// Close the response store
	if s.responseStore != nil {
		if err := s.responseStore.Close(); err != nil {
			s.logger.Error("Failed to close response store", zap.Error(err))
		}
	}

	s.logger.Info("Server stopped")
}

// SendCommandToAgent sends a command to a specific agent
func (s *Server) SendCommandToAgent(agentID, command string) error {
	s.agentsMutex.RLock()
	stream, ok := s.agents[agentID]
	s.agentsMutex.RUnlock()

	if !ok {
		s.logger.Error("Agent not connected", zap.String("agent_id", agentID))
		return fmt.Errorf("agent %s not connected", agentID)
	}

	// Generate command ID
	cmdID := fmt.Sprintf("cmd-%d", time.Now().Unix())

	// Create command status
	cmdStatus := &CommandStatus{
		ID:        cmdID,
		AgentID:   agentID,
		Command:   command,
		Status:    "pending",
		StartTime: time.Now(),
	}

	// Store command status
	s.commandsMutex.Lock()
	s.commands[cmdID] = cmdStatus
	s.commandsMutex.Unlock()

	// Create command message
	cmdMsg := &pb.ServerMessage{
		Id:   cmdID,
		Type: pb.MessageType_COMMAND,
		Payload: &pb.ServerMessage_Command{
			Command: &pb.CommandRequest{
				Command: command,
			},
		},
	}

	// Send the command
	s.logger.Info("Sending command to agent",
		zap.String("command_id", cmdID),
		zap.String("agent_id", agentID),
		zap.String("command", command))

	if err := stream.Send(cmdMsg); err != nil {
		s.logger.Error("Failed to send command",
			zap.String("command_id", cmdID),
			zap.String("agent_id", agentID),
			zap.String("command", command),
			zap.Error(err))

		// Update command status
		s.commandsMutex.Lock()
		cmdStatus.Status = "failed"
		cmdStatus.Error = err
		s.commandsMutex.Unlock()

		return fmt.Errorf("failed to send command: %w", err)
	}

	// Update command status
	s.commandsMutex.Lock()
	cmdStatus.Status = "sent"
	s.commandsMutex.Unlock()

	// Log to file
	if s.logFile != nil {
		fmt.Fprintf(s.logFile, "[%s] Sending command '%s' (ID: %s) to agent %s\n",
			time.Now().Format(time.RFC3339), command, cmdID, agentID)
	}

	return nil
}

// GetCommandStatus returns the status of a command
func (s *Server) GetCommandStatus(cmdID string) *CommandStatus {
	s.commandsMutex.RLock()
	defer s.commandsMutex.RUnlock()
	return s.commands[cmdID]
}

// ListPendingCommands returns a list of pending commands
func (s *Server) ListPendingCommands() []*CommandStatus {
	s.commandsMutex.RLock()
	defer s.commandsMutex.RUnlock()

	var pending []*CommandStatus
	for _, cmd := range s.commands {
		if cmd.Status == "pending" || cmd.Status == "sent" {
			pending = append(pending, cmd)
		}
	}
	return pending
}
