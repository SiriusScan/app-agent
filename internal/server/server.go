package server

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	valkey "github.com/valkey-io/valkey-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/SiriusScan/app-agent/internal/command"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/store"
	pb "github.com/SiriusScan/app-agent/proto/hello"
	goapistore "github.com/SiriusScan/go-api/sirius/store"
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

	// Template management
	templateManager    *ServerTemplateManager
	repositoryManager  *RepositoryManager
	syncQueueProcessor *TemplateSyncQueueProcessor
	valkeyClient       valkey.Client

	// KVStore for agent token authentication
	kvStore goapistore.KVStore
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

	// Initialize ValKey client for template storage
	valkeyClient, err := NewValkeyClient(logger)
	if err != nil {
		logger.Warn("Failed to initialize ValKey client, templates will not be stored",
			zap.Error(err))
		valkeyClient = nil
	}

	// Initialize go-api KVStore for agent token auth
	kvStore, err := goapistore.NewValkeyStore()
	if err != nil {
		logger.Warn("Failed to create KVStore for agent auth, token auth will be unavailable",
			zap.Error(err))
	}

	// Create server instance first
	server := &Server{
		logger:          logger,
		config:          cfg,
		logFile:         logFile,
		agents:          make(map[string]pb.HelloService_ConnectStreamServer),
		commands:        make(map[string]*CommandStatus),
		responseStore:   responseStore,
		valkeyClient:    valkeyClient,
		pendingCommands: make(map[string]string),
		kvStore:         kvStore,
	}

	// Initialize template manager with server reference
	// Note: RepoURL is now deprecated in favor of RepositoryManager
	templateConfig := &TemplateConfig{
		RepoURL:         "", // Deprecated: Now managed by RepositoryManager
		RepoPath:        "/var/sirius/template-repos/sirius-agent-modules",
		RepoBranch:      "main",
		RepoID:          "sirius-agent-modules",
		SyncInterval:    24 * time.Hour,
		MaxTemplateSize: 1024 * 1024, // 1MB
	}

	templateManager := NewServerTemplateManager(valkeyClient, logger, templateConfig, server)
	server.templateManager = templateManager
	logger.Info("Template manager initialized (legacy - will be replaced by RepositoryManager)")

	// Initialize repository manager for multi-repo support
	if valkeyClient != nil {
		repositoryManager := NewRepositoryManager(
			valkeyClient,
			logger,
			"/opt/sirius/agent-templates/repos",
			server,
		)
		server.repositoryManager = repositoryManager

		// Initialize default repository if none exist
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := repositoryManager.InitializeDefaultRepository(ctx); err != nil {
			logger.Warn("Failed to initialize default repository", zap.Error(err))
		}
		cancel()

		// Create and start sync queue processor
		syncQueueProcessor := NewTemplateSyncQueueProcessor(repositoryManager, logger)
		server.syncQueueProcessor = syncQueueProcessor
		if err := syncQueueProcessor.StartListening(); err != nil {
			logger.Error("Failed to start sync queue processor", zap.Error(err))
		} else {
			logger.Info("Template sync queue processor started")
		}

		// Perform initial sync of all repositories
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			logger.Info("Performing initial repository sync")
			if err := repositoryManager.SyncAllRepositories(ctx); err != nil {
				logger.Error("Initial repository sync failed", zap.Error(err))
			} else {
				logger.Info("Initial repository sync completed successfully")
			}
		}()
	} else {
		logger.Warn("ValKey client not available, repository management disabled")
	}

	return server, nil
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

	// ── Agent token authentication ───────────────────────────────────────────
	var issuedToken string
	if s.kvStore != nil {
		suppliedToken := msg.GetAuthToken()
		if suppliedToken == "" {
			// First connection — check if a token already exists for this agent.
			if goapistore.HasAgentToken(context.Background(), s.kvStore, agentID) {
				// Token exists but agent didn't supply it — reject.
				s.logger.Warn("Agent connected without token but a token exists – rejecting",
					zap.String("agent_id", agentID))
				return fmt.Errorf("agent %s must present auth_token", agentID)
			}
			// Brand-new agent — generate and store a token.
			newToken, err := goapistore.GenerateAgentToken()
			if err != nil {
				s.logger.Error("Failed to generate agent token", zap.Error(err))
				return fmt.Errorf("failed to generate agent token: %w", err)
			}
			if err := goapistore.StoreAgentToken(context.Background(), s.kvStore, agentID, newToken); err != nil {
				s.logger.Error("Failed to store agent token", zap.Error(err))
				return fmt.Errorf("failed to store agent token: %w", err)
			}
			issuedToken = newToken
			s.logger.Info("Issued new auth token to agent", zap.String("agent_id", agentID))
		} else {
			// Returning agent — validate the supplied token.
			if _, err := goapistore.ValidateAgentToken(context.Background(), s.kvStore, agentID, suppliedToken); err != nil {
				s.logger.Warn("Agent token validation failed",
					zap.String("agent_id", agentID), zap.Error(err))
				return fmt.Errorf("agent token validation failed: %w", err)
			}
			s.logger.Info("Agent authenticated successfully", zap.String("agent_id", agentID))
		}
	}

	// Register the agent
	s.agentsMutex.Lock()
	s.agents[agentID] = stream
	s.agentsMutex.Unlock()
	s.syncConnectedAgentsToValKey()

	// Cleanup when the function exits - only remove if THIS stream is still registered
	defer func() {
		s.agentsMutex.Lock()
		if s.agents[agentID] == stream {
			delete(s.agents, agentID)
			s.logger.Info("Agent disconnected from stream", zap.String("agent_id", agentID))
		} else {
			s.logger.Info("Agent stream replaced by newer connection, skipping cleanup",
				zap.String("agent_id", agentID))
		}
		s.agentsMutex.Unlock()
		s.syncConnectedAgentsToValKey()
	}()

	// Send welcome message - trigger initial status update/scan.
	// If a token was issued, include it so the agent can persist it.
	welcomeMsg := &pb.ServerMessage{
		Id:   "welcome-status",
		Type: pb.MessageType_COMMAND,
		Payload: &pb.ServerMessage_Command{
			Command: &pb.CommandRequest{
				Command: "internal:status",
			},
		},
		AuthToken: issuedToken, // Non-empty only when a new token was generated.
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

		case pb.MessageType_TEMPLATE_SYNC_REQUEST:
			s.handleTemplateSyncRequest(agentID, msg.GetSyncRequest(), stream)

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

	// Refresh connected_agents key TTL on heartbeat
	s.syncConnectedAgentsToValKey()
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

	// Check if this is a coordinated scan result (contains --scan-id)
	if scanID := extractScanID(result.Command); scanID != "" {
		s.logger.Info("Coordinated scan result detected, merging into currentScan",
			zap.String("scan_id", scanID),
			zap.String("agent_id", agentID))
		go s.mergeAgentScanResults(agentID, scanID, result)
	}

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
		header := "\n===== COMMAND EXECUTION RESULT =====\n"
		agentLine := fmt.Sprintf("Agent: %s\n", agentID)
		commandLine := fmt.Sprintf("Command: %s\n", result.Command)
		exitCodeLine := fmt.Sprintf("Exit Code: %d\n", result.ExitCode)
		executionTimeLine := fmt.Sprintf("Execution Time: %dms\n", result.ExecutionTime)
		separator := "==================================\n\n"
		outputPrefix := "Output:\n"
		output := result.Output

		var errorSection string
		if result.Error != "" {
			errorSection = "\nError:\n" + result.Error + "\n"
		}

		footer := "\n==================================\n"

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

// handleTemplateSyncRequest processes template sync requests from agents
func (s *Server) handleTemplateSyncRequest(agentID string, syncReq *pb.TemplateSyncRequest, stream pb.HelloService_ConnectStreamServer) {
	if syncReq == nil {
		s.logger.Warn("Received nil template sync request", zap.String("agent_id", agentID))
		return
	}

	s.logger.Info("Processing template sync request",
		zap.String("agent_id", agentID),
		zap.Int64("last_sync", syncReq.LastSync))

	ctx := context.Background()

	// Get templates for sync from template manager
	manifest, templates, err := s.templateManager.GetTemplatesForSync(ctx, syncReq.LastSync)
	if err != nil {
		s.logger.Error("Failed to get templates for sync",
			zap.String("agent_id", agentID),
			zap.Error(err))

		// Send an empty manifest so the agent doesn't hang waiting for a response
		errorManifest := &pb.ServerMessage{
			Id:   fmt.Sprintf("template-manifest-error-%d", time.Now().Unix()),
			Type: pb.MessageType_TEMPLATE_UPDATE,
			Payload: &pb.ServerMessage_TemplateUpdate{
				TemplateUpdate: &pb.TemplateUpdate{
					Manifest: &pb.TemplateManifest{
						Version:   "1.0.0",
						Updated:   time.Now().Unix(),
						Templates: make(map[string]*pb.TemplateMetadata),
						Statistics: &pb.TemplateStatistics{
							TotalTemplates:    0,
							StandardTemplates: 0,
							CustomTemplates:   0,
						},
					},
				},
			},
		}
		if sendErr := stream.Send(errorManifest); sendErr != nil {
			s.logger.Error("Failed to send error manifest to agent",
				zap.String("agent_id", agentID),
				zap.Error(sendErr))
		}
		return
	}

	s.logger.Info("Retrieved templates for sync",
		zap.String("agent_id", agentID),
		zap.Int("template_count", len(templates)))

	// Send manifest as first message (with empty template content)
	manifestMsg := &pb.ServerMessage{
		Id:   fmt.Sprintf("template-manifest-%d", time.Now().Unix()),
		Type: pb.MessageType_TEMPLATE_UPDATE,
		Payload: &pb.ServerMessage_TemplateUpdate{
			TemplateUpdate: &pb.TemplateUpdate{
				Manifest: manifest,
			},
		},
	}

	if err := stream.Send(manifestMsg); err != nil {
		s.logger.Error("Failed to send template manifest",
			zap.String("agent_id", agentID),
			zap.Error(err))
		return
	}

	s.logger.Info("Sent template manifest to agent",
		zap.String("agent_id", agentID))

	// Stream templates one-by-one
	successCount := 0
	errorCount := 0

	for _, template := range templates {
		templateMsg := &pb.ServerMessage{
			Id:   fmt.Sprintf("template-%s-%d", template.TemplateId, time.Now().Unix()),
			Type: pb.MessageType_TEMPLATE_UPDATE,
			Payload: &pb.ServerMessage_TemplateUpdate{
				TemplateUpdate: template,
			},
		}

		if err := stream.Send(templateMsg); err != nil {
			s.logger.Error("Failed to send template",
				zap.String("agent_id", agentID),
				zap.String("template_id", template.TemplateId),
				zap.Error(err))
			errorCount++
			continue
		}

		successCount++
		s.logger.Debug("Sent template to agent",
			zap.String("agent_id", agentID),
			zap.String("template_id", template.TemplateId))
	}

	s.logger.Info("Template sync completed",
		zap.String("agent_id", agentID),
		zap.Int("success", successCount),
		zap.Int("errors", errorCount))
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

// agentAuthUnaryInterceptor validates that the calling agent has an active,
// authenticated stream before allowing unary RPCs such as Ping.  If the
// KVStore is unavailable, the interceptor falls back to only checking
// that the agent has a registered stream (backward-compatible behaviour).
func (s *Server) agentAuthUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Only intercept HelloService methods; allow reflection etc. through.
	if !strings.Contains(info.FullMethod, "HelloService") {
		return handler(ctx, req)
	}

	// Extract agent_id from PingRequest (currently the only unary RPC).
	type agentIDGetter interface{ GetAgentId() string }
	getter, ok := req.(agentIDGetter)
	if !ok {
		return handler(ctx, req) // Unknown request type — let the handler decide.
	}
	agentID := getter.GetAgentId()
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	// Check that the agent has an active stream (implies they already passed
	// stream-level token authentication).
	s.agentsMutex.RLock()
	_, connected := s.agents[agentID]
	s.agentsMutex.RUnlock()
	if !connected {
		return nil, fmt.Errorf("agent %s has no active stream — authenticate via ConnectStream first", agentID)
	}

	return handler(ctx, req)
}

// Start starts the gRPC server and command processors
func (s *Server) Start() error {
	// Create a new gRPC server with the auth interceptor
	s.server = grpc.NewServer(
		grpc.UnaryInterceptor(s.agentAuthUnaryInterceptor),
	)
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

	// Stop sync queue processor
	if s.syncQueueProcessor != nil {
		s.syncQueueProcessor.Stop()
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

// syncConnectedAgentsToValKey writes the current list of connected agent IDs to ValKey
// so that the frontend can discover agents without going through RabbitMQ.
func (s *Server) syncConnectedAgentsToValKey() {
	if s.valkeyClient == nil {
		return
	}
	s.agentsMutex.RLock()
	agentIDs := make([]string, 0, len(s.agents))
	for id := range s.agents {
		agentIDs = append(agentIDs, id)
	}
	s.agentsMutex.RUnlock()

	agentsJSON, _ := json.Marshal(agentIDs)
	ctx := context.Background()
	cmd := s.valkeyClient.B().Set().Key("connected_agents").Value(string(agentsJSON)).Ex(120 * time.Second).Build()
	if err := s.valkeyClient.Do(ctx, cmd).Error(); err != nil {
		s.logger.Warn("Failed to sync connected agents to ValKey", zap.Error(err))
	}
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

// extractScanID extracts the --scan-id value from a command string
func extractScanID(command string) string {
	parts := strings.Fields(command)
	for _, part := range parts {
		if strings.HasPrefix(part, "--scan-id=") {
			return strings.TrimPrefix(part, "--scan-id=")
		}
	}
	return ""
}

// AgentSubScanMetadata is the agent-specific metadata stored in SubScan.Metadata
type AgentSubScanMetadata struct {
	Mode             string             `json:"mode,omitempty"`
	DispatchedAgents []string           `json:"dispatched_agents"`
	AgentStatuses    []AgentStatusEntry `json:"agent_statuses"`
}

type AgentStatusEntry struct {
	AgentID              string `json:"agent_id"`
	Status               string `json:"status"`
	StartedAt            string `json:"started_at,omitempty"`
	CompletedAt          string `json:"completed_at,omitempty"`
	HostsFound           int    `json:"hosts_found"`
	VulnerabilitiesFound int    `json:"vulnerabilities_found"`
	Error                string `json:"error,omitempty"`
}

// AgentScanOutputEntry represents the JSON output from an agent template scan
type AgentScanOutputEntry struct {
	Summary *AgentScanSummaryEntry `json:"summary,omitempty"`
	Results []AgentTemplateResult  `json:"results,omitempty"`
}

// AgentScanSummaryEntry is the scan summary from agent output
type AgentScanSummaryEntry struct {
	TotalTemplates  int    `json:"total_templates"`
	Matched         int    `json:"matched"`
	NotMatched      int    `json:"not_matched"`
	Errors          int    `json:"errors"`
	ExecutionTimeMs int64  `json:"execution_time_ms"`
	Host            string `json:"host,omitempty"`
	PrimaryIP       string `json:"primary_ip,omitempty"`
}

// AgentTemplateResult is a single template result from agent output
type AgentTemplateResult struct {
	TemplateID      string   `json:"template_id"`
	TemplateName    string   `json:"template_name,omitempty"`
	VulnerabilityID string   `json:"vulnerability_id,omitempty"`
	Description     string   `json:"description,omitempty"`
	Severity        string   `json:"severity,omitempty"`
	RiskScore       float64  `json:"risk_score"`
	CVE             []string `json:"cve,omitempty"`
	Matched         bool     `json:"matched"`
	Host            string   `json:"host,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}

// isConcreteHostIP returns true only for a single IPv4/IPv6 address string (not CIDR, range, or hostname).
func isConcreteHostIP(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "/") || strings.Contains(s, "-") {
		return false
	}
	return net.ParseIP(s) != nil
}

// primaryIPMatchesTarget reports whether a concrete primary IP falls under a scan target
// (exact match for single-IP targets, or membership for CIDR targets).
func primaryIPMatchesTarget(primary, target string) bool {
	primary = strings.TrimSpace(primary)
	target = strings.TrimSpace(target)
	if primary == "" || target == "" || !isConcreteHostIP(primary) {
		return false
	}
	if isConcreteHostIP(target) {
		return primary == target
	}
	if _, ipNet, err := net.ParseCIDR(target); err == nil {
		ip := net.ParseIP(primary)
		return ip != nil && ipNet.Contains(ip)
	}
	return false
}

// agentIPMatchesTarget checks whether a resolved IP string matches a target (exact or CIDR).
func agentIPMatchesTarget(ipStr, target string) bool {
	ipStr = strings.TrimSpace(ipStr)
	target = strings.TrimSpace(target)
	if ipStr == "" || target == "" || !isConcreteHostIP(ipStr) {
		return false
	}
	return primaryIPMatchesTarget(ipStr, target)
}

func appendUniqueString(slice []string, s string) []string {
	if s == "" {
		return slice
	}
	for _, x := range slice {
		if x == s {
			return slice
		}
	}
	return append(slice, s)
}

func buildAgentClaimedIPs(scanResult *goapistore.ScanResult) map[string]bool {
	claimedIPs := make(map[string]bool)
	for _, h := range scanResult.Hosts {
		for _, src := range h.Sources {
			if src == "agent" && isConcreteHostIP(h.IP) {
				claimedIPs[h.IP] = true
				break
			}
		}
	}
	return claimedIPs
}

// resolveAgentHostIP determines the correct target IP for an agent that reported scan results.
// It never returns CIDR/range strings — only a concrete IP or empty string if unresolved.
// Priority order:
//  1. Primary IP matches a scan target (exact or CIDR membership)
//  2. Hostname matches an existing host entry with a concrete IP
//  3. Agent ID is a concrete IP and matches a target
//  4–5. DNS-resolve hostname / agent ID and match targets
//  6. First existing scan host with concrete IP not yet claimed by another agent
//  7. First concrete single-IP scan target not yet claimed
//  8. Primary IP if concrete (last resort)
//  9. Empty string — caller must not create a fake host row
func (s *Server) resolveAgentHostIP(agentID, agentHostname, agentPrimaryIP string, scanResult *goapistore.ScanResult) string {
	claimedIPs := buildAgentClaimedIPs(scanResult)

	// Strategy 1: Primary IP vs targets (supports CIDR targets)
	if agentPrimaryIP != "" && isConcreteHostIP(agentPrimaryIP) {
		for _, target := range scanResult.Targets {
			if primaryIPMatchesTarget(agentPrimaryIP, target) {
				s.logger.Info("Matched agent primary IP to scan target",
					zap.String("agent_id", agentID),
					zap.String("primary_ip", agentPrimaryIP),
					zap.String("target", target))
				return agentPrimaryIP
			}
		}
	}

	// Strategy 2: Hostname against existing host entries (require concrete IP)
	if agentHostname != "" {
		for _, h := range scanResult.Hosts {
			if !isConcreteHostIP(h.IP) {
				continue
			}
			if h.Hostname != "" && h.Hostname == agentHostname {
				s.logger.Info("Matched agent to existing host by hostname",
					zap.String("agent_id", agentID),
					zap.String("hostname", agentHostname),
					zap.String("matched_ip", h.IP))
				return h.IP
			}
		}
	}

	// Strategy 3: Agent ID as concrete IP matching a target
	if isConcreteHostIP(agentID) {
		for _, target := range scanResult.Targets {
			if primaryIPMatchesTarget(agentID, target) {
				s.logger.Info("Agent ID matches a scan target as concrete IP",
					zap.String("agent_id", agentID))
				return agentID
			}
		}
	}

	// Strategy 4: DNS-resolve agentHostname and match against targets
	if agentHostname != "" {
		if ips, err := net.LookupHost(agentHostname); err == nil {
			for _, ip := range ips {
				for _, target := range scanResult.Targets {
					if agentIPMatchesTarget(ip, target) {
						s.logger.Info("Resolved agent hostname to target IP via DNS",
							zap.String("agent_id", agentID),
							zap.String("hostname", agentHostname),
							zap.String("resolved_ip", ip))
						return ip
					}
				}
			}
		}
	}

	// Strategy 5: DNS-resolve agentID if it differs from hostname
	if agentID != agentHostname && agentID != "" && !isConcreteHostIP(agentID) {
		if ips, err := net.LookupHost(agentID); err == nil {
			for _, ip := range ips {
				for _, target := range scanResult.Targets {
					if agentIPMatchesTarget(ip, target) {
						s.logger.Info("Resolved agent ID to target IP via DNS",
							zap.String("agent_id", agentID),
							zap.String("resolved_ip", ip))
						return ip
					}
				}
			}
		}
	}

	// Strategy 6: First existing host with concrete IP not yet claimed by another agent
	for _, h := range scanResult.Hosts {
		if !isConcreteHostIP(h.IP) || claimedIPs[h.IP] {
			continue
		}
		s.logger.Info("Using first unclaimed discovered host for agent",
			zap.String("agent_id", agentID),
			zap.String("host_ip", h.IP))
		return h.IP
	}

	// Strategy 7: First concrete single-IP target not yet claimed
	for _, target := range scanResult.Targets {
		if !isConcreteHostIP(target) || claimedIPs[target] {
			continue
		}
		s.logger.Info("Using first unclaimed concrete scan target for agent",
			zap.String("agent_id", agentID),
			zap.String("target_ip", target))
		return target
	}

	// Strategy 8: Last resort — primary IP if concrete (may be outside stated targets)
	if agentPrimaryIP != "" && isConcreteHostIP(agentPrimaryIP) {
		s.logger.Warn("Could not align primary IP to targets; using primary IP as host key",
			zap.String("agent_id", agentID),
			zap.String("primary_ip", agentPrimaryIP))
		return agentPrimaryIP
	}

	s.logger.Warn("Could not resolve agent to a concrete host IP; skipping host/vuln merge for this result",
		zap.String("agent_id", agentID),
		zap.String("hostname", agentHostname),
		zap.String("primary_ip", agentPrimaryIP))
	return ""
}

// appendAgentVulnerabilitiesDeduped appends agent vulnerabilities keyed by (host_id, vuln id, agent_id).
func appendAgentVulnerabilitiesDeduped(
	existing []goapistore.VulnerabilitySummary,
	newOnes []goapistore.VulnerabilitySummary,
	hostIP, agentID string,
) ([]goapistore.VulnerabilitySummary, int) {
	keys := make(map[string]struct{})
	for _, v := range existing {
		if v.ScanSource != "agent" {
			continue
		}
		keys[fmt.Sprintf("%s|%s|%s", v.HostID, v.ID, v.AgentID)] = struct{}{}
	}
	added := 0
	for i := range newOnes {
		newOnes[i].HostID = hostIP
		key := fmt.Sprintf("%s|%s|%s", hostIP, newOnes[i].ID, agentID)
		if _, dup := keys[key]; dup {
			continue
		}
		keys[key] = struct{}{}
		existing = append(existing, newOnes[i])
		added++
	}
	return existing, added
}

// mergeAgentScanResults merges agent scan results into the unified currentScan ValKey key
func (s *Server) mergeAgentScanResults(agentID, scanID string, result *pb.CommandResult) {
	if s.valkeyClient == nil {
		s.logger.Warn("Cannot merge agent scan results: ValKey client not available")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse the agent's scan output FIRST (before reading currentScan)
	// The output is JSON from the template scan command
	var agentOutput AgentScanOutputEntry
	var matchedVulns []goapistore.VulnerabilitySummary
	agentHostname := ""
	agentPrimaryIP := ""

	if result.Output != "" {
		// Debug: log first 500 chars of agent output for verification
		outputPreview := result.Output
		if len(outputPreview) > 500 {
			outputPreview = outputPreview[:500] + "...(truncated)"
		}
		s.logger.Info("Agent scan raw output preview",
			zap.String("agent_id", agentID),
			zap.String("scan_id", scanID),
			zap.Int("output_length", len(result.Output)),
			zap.String("output_preview", outputPreview))

		if err := json.Unmarshal([]byte(result.Output), &agentOutput); err != nil {
			s.logger.Warn("Failed to parse agent scan output as JSON, trying to extract embedded JSON",
				zap.Error(err),
				zap.Int("output_length", len(result.Output)))
			// Output might be wrapped or have extra text; best effort
		}

		// Extract the agent's hostname and primary IP from summary
		if agentOutput.Summary != nil {
			if agentOutput.Summary.Host != "" {
				agentHostname = agentOutput.Summary.Host
			}
			if agentOutput.Summary.PrimaryIP != "" {
				agentPrimaryIP = agentOutput.Summary.PrimaryIP
			}
		}

		// Convert matched template results to vulnerability entries
		for _, tmplResult := range agentOutput.Results {
			if !tmplResult.Matched {
				continue
			}

			// Use the host from the result if available
			if tmplResult.Host != "" && agentHostname == "" {
				agentHostname = tmplResult.Host
			}

			vulnID := tmplResult.VulnerabilityID
			if vulnID == "" {
				vulnID = tmplResult.TemplateID
			}

			title := tmplResult.TemplateName
			if title == "" {
				title = tmplResult.TemplateID
			}

			severity := tmplResult.Severity
			if severity == "" {
				severity = "info"
			}

			vuln := goapistore.VulnerabilitySummary{
				ID:          vulnID,
				Severity:    severity,
				Title:       title,
				Description: tmplResult.Description,
				CVSSScore:   tmplResult.RiskScore,
				RiskScore:   tmplResult.RiskScore,
				ScanSource:  "agent",
				AgentID:     agentID,
			}
			matchedVulns = append(matchedVulns, vuln)
		}

		s.logger.Info("Parsed agent scan output",
			zap.Int("total_results", len(agentOutput.Results)),
			zap.Int("matched_vulns", len(matchedVulns)),
			zap.String("agent_hostname", agentHostname))
	}

	// Read current scan state from ValKey
	getCmd := s.valkeyClient.B().Get().Key("currentScan").Build()
	resp := s.valkeyClient.Do(ctx, getCmd)

	if resp.Error() != nil {
		s.logger.Error("Failed to read currentScan from ValKey",
			zap.Error(resp.Error()))
		return
	}

	encoded, err := resp.ToString()
	if err != nil {
		s.logger.Error("Failed to convert currentScan to string",
			zap.Error(err))
		return
	}

	// The currentScan value is base64-encoded JSON
	// We need to decode, modify, and re-encode
	decodedBytes, decErr := base64Decode(encoded)
	if decErr != nil {
		s.logger.Error("Failed to decode currentScan base64",
			zap.Error(decErr))
		return
	}

	var scanResult goapistore.ScanResult
	if err := json.Unmarshal(decodedBytes, &scanResult); err != nil {
		s.logger.Error("Failed to unmarshal currentScan",
			zap.Error(err))
		return
	}

	// Only merge if scan IDs match
	if scanResult.ID != scanID {
		s.logger.Warn("Scan ID mismatch, skipping merge",
			zap.String("current_scan_id", scanResult.ID),
			zap.String("agent_scan_id", scanID))
		return
	}

	// Resolve concrete host IP before mutating sub-scan status or host rows
	hostIP := s.resolveAgentHostIP(agentID, agentHostname, agentPrimaryIP, &scanResult)
	hostResolved := hostIP != ""

	// Update agent sub-scan status via sub_scans registry
	if scanResult.SubScans == nil {
		scanResult.SubScans = make(map[string]goapistore.SubScan)
	}

	// Get or create the agent sub-scan entry
	agentSubScan, hasAgentSS := scanResult.SubScans["agent"]
	if !hasAgentSS {
		agentSubScan = goapistore.SubScan{
			Type:    "agent",
			Enabled: true,
			Status:  "running",
			Progress: goapistore.SubScanProgress{
				Label: "agents",
			},
		}
	}

	// Parse existing agent metadata (or start fresh)
	var agentMeta AgentSubScanMetadata
	if agentSubScan.Metadata != nil {
		_ = json.Unmarshal(agentSubScan.Metadata, &agentMeta)
	}

	// Ensure this agent is tracked in DispatchedAgents
	agentFound := false
	for _, id := range agentMeta.DispatchedAgents {
		if id == agentID {
			agentFound = true
			break
		}
	}
	if !agentFound {
		agentMeta.DispatchedAgents = append(agentMeta.DispatchedAgents, agentID)
	}

	// Update or create agent status entry
	statusFound := false
	for i, as := range agentMeta.AgentStatuses {
		if as.AgentID == agentID {
			statusFound = true
			now := time.Now().Format(time.RFC3339)
			if result.ExitCode == 0 {
				agentMeta.AgentStatuses[i].Status = "completed"
				agentMeta.AgentStatuses[i].CompletedAt = now
				if hostResolved {
					agentMeta.AgentStatuses[i].HostsFound = 1
					agentMeta.AgentStatuses[i].VulnerabilitiesFound = len(matchedVulns)
				} else {
					agentMeta.AgentStatuses[i].HostsFound = 0
					agentMeta.AgentStatuses[i].VulnerabilitiesFound = 0
				}
			} else {
				agentMeta.AgentStatuses[i].Status = "failed"
				agentMeta.AgentStatuses[i].CompletedAt = now
				agentMeta.AgentStatuses[i].Error = result.Error
			}
			break
		}
	}
	if !statusFound {
		now := time.Now().Format(time.RFC3339)
		newStatus := AgentStatusEntry{AgentID: agentID, CompletedAt: now}
		if result.ExitCode == 0 {
			newStatus.Status = "completed"
			if hostResolved {
				newStatus.HostsFound = 1
				newStatus.VulnerabilitiesFound = len(matchedVulns)
			} else {
				newStatus.HostsFound = 0
				newStatus.VulnerabilitiesFound = 0
			}
		} else {
			newStatus.Status = "failed"
			newStatus.Error = result.Error
		}
		agentMeta.AgentStatuses = append(agentMeta.AgentStatuses, newStatus)
	}

	// Count completed agents and update progress
	completed := 0
	for _, as := range agentMeta.AgentStatuses {
		if as.Status == "completed" || as.Status == "failed" {
			completed++
		}
	}
	totalAgents := len(agentMeta.DispatchedAgents)
	agentSubScan.Progress.Completed = completed
	agentSubScan.Progress.Total = totalAgents

	// Update agent sub-scan status
	if totalAgents > 0 && completed >= totalAgents {
		agentSubScan.Status = "completed"
	} else {
		agentSubScan.Status = "running"
	}

	// Serialize metadata back
	metaBytes, _ := json.Marshal(agentMeta)
	agentSubScan.Metadata = metaBytes
	scanResult.SubScans["agent"] = agentSubScan

	// Determine overall scan completion: all enabled sub-scans must be done
	allSubScansDone := true
	for _, ss := range scanResult.SubScans {
		if ss.Enabled && ss.Status != "completed" && ss.Status != "failed" {
			allSubScansDone = false
			break
		}
	}
	if allSubScansDone && len(scanResult.SubScans) > 0 {
		scanResult.Status = "completed"
		if scanResult.EndTime == "" {
			scanResult.EndTime = time.Now().Format(time.RFC3339)
		}
		s.logger.Info("All sub-scans completed, marking overall scan as completed",
			zap.String("scan_id", scanID))
	}

	vulnsAdded := 0
	if hostResolved {
		// Merge host: look up by IP, create or update
		hostIdx := -1
		for i, h := range scanResult.Hosts {
			if h.IP == hostIP {
				hostIdx = i
				break
			}
		}
		if hostIdx >= 0 {
			// Merge: add hostname and source
			h := &scanResult.Hosts[hostIdx]
			if agentHostname != "" {
				if h.Hostname == "" {
					h.Hostname = agentHostname
				} else if h.Hostname != agentHostname {
					// Preserve network-discovered name; keep agent-reported name as an alias
					h.Aliases = appendUniqueString(h.Aliases, agentHostname)
				}
			}
			sourceFound := false
			for _, src := range h.Sources {
				if src == "agent" {
					sourceFound = true
					break
				}
			}
			if !sourceFound {
				h.Sources = append(h.Sources, "agent")
			}
		} else {
			// New host
			newHost := goapistore.HostEntry{
				ID:       hostIP,
				IP:       hostIP,
				Hostname: agentHostname,
				Sources:  []string{"agent"},
			}
			scanResult.Hosts = append(scanResult.Hosts, newHost)
		}

		// Append extracted vulnerabilities with host_id (deduped)
		if len(matchedVulns) > 0 {
			var added int
			scanResult.Vulnerabilities, added = appendAgentVulnerabilitiesDeduped(
				scanResult.Vulnerabilities, matchedVulns, hostIP, agentID)
			vulnsAdded = added
			s.logger.Info("Added agent vulnerabilities to currentScan",
				zap.Int("new_vulns", added),
				zap.Int("total_vulns", len(scanResult.Vulnerabilities)))
		}
	} else if len(matchedVulns) > 0 && result.ExitCode == 0 {
		s.logger.Warn("Agent reported vulnerabilities but host IP could not be resolved; not appending to currentScan",
			zap.String("agent_id", agentID),
			zap.String("scan_id", scanID),
			zap.Int("skipped_vulns", len(matchedVulns)))
	}

	// Write back to ValKey
	updatedJSON, err := json.Marshal(scanResult)
	if err != nil {
		s.logger.Error("Failed to marshal updated currentScan",
			zap.Error(err))
		return
	}

	encodedStr := base64Encode(updatedJSON)
	setCmd := s.valkeyClient.B().Set().Key("currentScan").Value(encodedStr).Build()
	if err := s.valkeyClient.Do(ctx, setCmd).Error(); err != nil {
		s.logger.Error("Failed to write updated currentScan to ValKey",
			zap.Error(err))
		return
	}

	// Also update the agent_scan:{scanID} status key
	s.updateAgentScanStatus(ctx, scanID, agentID, result)

	s.logger.Info("Successfully merged agent scan results into currentScan",
		zap.String("scan_id", scanID),
		zap.String("agent_id", agentID),
		zap.Int("vulnerabilities_added", vulnsAdded),
		zap.String("host_ip", hostIP))
}

// updateAgentScanStatus updates the agent_scan:{scanId} status key in ValKey
func (s *Server) updateAgentScanStatus(ctx context.Context, scanID, agentID string, result *pb.CommandResult) {
	statusKey := fmt.Sprintf("agent_scan:%s", scanID)

	getCmd := s.valkeyClient.B().Get().Key(statusKey).Build()
	resp := s.valkeyClient.Do(ctx, getCmd)

	if resp.Error() != nil {
		// Key doesn't exist yet (race condition: agent completed before frontend wrote the key)
		// Create the status key with the agent's result
		s.logger.Info("agent_scan status key not found, creating it",
			zap.String("key", statusKey),
			zap.String("agent_id", agentID))

		agentStatus := "completed"
		if result.ExitCode != 0 {
			agentStatus = "failed"
		}

		newStatus := map[string]interface{}{
			"scanId":         scanID,
			"status":         agentStatus,
			"totalAgents":    1,
			"completedAgents": 1,
			"failedAgents":   0,
			"agentStatuses": []map[string]interface{}{
				{
					"agentId":              agentID,
					"status":               agentStatus,
					"hostsFound":           0,
					"vulnerabilitiesFound": 0,
				},
			},
			"completedAt": time.Now().Format(time.RFC3339),
		}

		statusBytes, _ := json.Marshal(newStatus)
		setCmd := s.valkeyClient.B().Set().Key(statusKey).Value(string(statusBytes)).Ex(3600 * time.Second).Build()
		if err := s.valkeyClient.Do(ctx, setCmd).Error(); err != nil {
			s.logger.Error("Failed to create agent_scan status key", zap.Error(err))
		}
		return
	}

	statusStr, err := resp.ToString()
	if err != nil {
		s.logger.Warn("Could not convert agent_scan status to string",
			zap.Error(err))
		return
	}

	var status map[string]interface{}
	if err := json.Unmarshal([]byte(statusStr), &status); err != nil {
		s.logger.Warn("Could not parse agent_scan status",
			zap.Error(err))
		return
	}

	// Update agent status within the status object
	if agentStatuses, ok := status["agentStatuses"].([]interface{}); ok {
		for i, s := range agentStatuses {
			if sm, ok := s.(map[string]interface{}); ok {
				if sm["agentId"] == agentID {
					if result.ExitCode == 0 {
						sm["status"] = "completed"
					} else {
						sm["status"] = "failed"
						sm["error"] = result.Error
					}
					agentStatuses[i] = sm
					break
				}
			}
		}
	}

	// Count completed
	completedCount := 0
	if agentStatuses, ok := status["agentStatuses"].([]interface{}); ok {
		for _, s := range agentStatuses {
			if sm, ok := s.(map[string]interface{}); ok {
				st, _ := sm["status"].(string)
				if st == "completed" || st == "failed" {
					completedCount++
				}
			}
		}
	}
	status["completedAgents"] = completedCount

	totalAgents := 0
	if ta, ok := status["totalAgents"].(float64); ok {
		totalAgents = int(ta)
	}

	if completedCount >= totalAgents && totalAgents > 0 {
		status["status"] = "completed"
	}

	updatedJSON, err := json.Marshal(status)
	if err != nil {
		return
	}

	// Write back with TTL
	setCmd := s.valkeyClient.B().Set().Key(statusKey).Value(string(updatedJSON)).Ex(3600 * time.Second).Build()
	s.valkeyClient.Do(ctx, setCmd)
}

// base64Decode decodes a base64-encoded string
func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// base64Encode encodes bytes to base64
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// startTemplateSync is deprecated - now using repository-based sync via RabbitMQ
// The new system uses RepositoryManager which syncs multiple repositories based on
// messages received from the agent.template.sync.jobs queue.
// This provides better control and allows scheduled syncs managed by the UI.
/*
func (s *Server) startTemplateSync() {
	s.logger.Info("Starting periodic template sync")

	ticker := time.NewTicker(s.templateManager.config.SyncInterval)
	defer ticker.Stop()

	// Perform initial sync
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := s.templateManager.SyncFromGitHub(ctx); err != nil {
		s.logger.Error("Initial template sync failed", zap.Error(err))
	} else {
		s.logger.Info("Initial template sync completed successfully")
	}

	// Periodic sync
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := s.templateManager.SyncFromGitHub(ctx); err != nil {
				s.logger.Error("Periodic template sync failed", zap.Error(err))
			} else {
				s.logger.Info("Periodic template sync completed successfully")
			}
			cancel()
		}
	}
}
*/
