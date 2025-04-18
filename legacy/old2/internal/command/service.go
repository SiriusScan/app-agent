package command

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/grpc"
	"github.com/SiriusScan/app-agent/internal/store"
	"github.com/SiriusScan/app-agent/proto"
	"github.com/SiriusScan/go-api/sirius/queue"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles the distribution of commands to agents
type Service struct {
	logger         *zap.Logger
	config         *config.Config
	store          store.KVStore
	resultStorage  store.ResultStorageInterface
	grpcServer     *grpc.Server
	queueClient    queue.Client
	isRunning      bool
	shutdownSignal chan struct{}
	mu             sync.Mutex
}

// NewService creates a new command distribution service
func NewService(
	logger *zap.Logger,
	config *config.Config,
	store store.KVStore,
	resultStorage store.ResultStorageInterface,
	grpcServer *grpc.Server,
) *Service {
	return &Service{
		logger:         logger.Named("command-service"),
		config:         config,
		store:          store,
		resultStorage:  resultStorage,
		grpcServer:     grpcServer,
		shutdownSignal: make(chan struct{}),
	}
}

// Start initializes and starts the command distribution service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("command service already running")
	}

	// Initialize queue client
	queueClient, err := queue.NewClient(queue.Config{
		KafkaBootstrapServers: s.config.KafkaBootstrapServers,
		KafkaTopic:            s.config.KafkaTaskTopic,
		KafkaGroupID:          s.config.KafkaGroupID,
	})
	if err != nil {
		return fmt.Errorf("failed to create queue client: %w", err)
	}
	s.queueClient = queueClient

	// Start listening for tasks from the queue
	go s.listenToTasksQueue(ctx)

	s.isRunning = true
	s.logger.Info("Command service started successfully")
	return nil
}

// Stop terminates the command distribution service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	close(s.shutdownSignal)
	if s.queueClient != nil {
		s.queueClient.Close()
	}
	s.isRunning = false
	s.logger.Info("Command service stopped")
}

// listenToTasksQueue subscribes to the tasks queue and processes incoming messages
func (s *Service) listenToTasksQueue(ctx context.Context) {
	s.logger.Info("Started listening to tasks queue",
		zap.String("topic", s.config.KafkaTaskTopic),
		zap.String("group_id", s.config.KafkaGroupID))

	// Start consuming messages in a separate goroutine
	go func() {
		err := s.queueClient.Listen(ctx, func(msg []byte) error {
			s.logger.Debug("Received message from queue", zap.ByteString("message", msg))
			return s.processMessage(ctx, msg)
		})
		if err != nil {
			s.logger.Error("Error listening to queue", zap.Error(err))
		}
	}()

	// Wait for shutdown signal or context cancellation
	select {
	case <-s.shutdownSignal:
		s.logger.Info("Shutting down queue listener")
	case <-ctx.Done():
		s.logger.Info("Context canceled, shutting down queue listener")
	}
}

// TaskMessage represents a task message received from the queue
type TaskMessage struct {
	AgentID string         `json:"agent_id"`
	Command *proto.Command `json:"command"`
}

// processMessage parses a task message and sends commands to the appropriate agent
func (s *Service) processMessage(ctx context.Context, message []byte) error {
	var task TaskMessage
	if err := json.Unmarshal(message, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task message: %w", err)
	}

	if task.AgentID == "" {
		return fmt.Errorf("agent ID is missing in task message")
	}

	if task.Command == nil {
		return fmt.Errorf("command is missing in task message")
	}

	s.logger.Info("Processing task",
		zap.String("agent_id", task.AgentID),
		zap.String("command_id", task.Command.Id),
		zap.String("command_type", task.Command.Type))

	// Send command to the agent through the gRPC server
	result, err := s.grpcServer.SendCommandToAgent(ctx, task.AgentID, task.Command)
	if err != nil {
		s.logger.Error("Failed to send command to agent",
			zap.String("agent_id", task.AgentID),
			zap.String("command_id", task.Command.Id),
			zap.Error(err))
		return err
	}

	if result != nil {
		// Store command result
		commandResult := &store.CommandResult{
			CommandID: task.Command.Id,
			AgentID:   task.AgentID,
			// Map other fields as needed from result to CommandResult
		}
		err = s.resultStorage.Store(ctx, commandResult)
		if err != nil {
			s.logger.Error("Failed to store command result",
				zap.String("command_id", task.Command.Id),
				zap.Error(err))
		} else {
			s.logger.Info("Command result stored successfully",
				zap.String("command_id", task.Command.Id),
				zap.String("agent_id", task.AgentID))
		}
	}

	return nil
}

// ProcessCommandResult handles a command result
func (s *Service) ProcessCommandResult(ctx context.Context, result *proto.ResultMessage) error {
	s.logger.Info("Processing command result",
		zap.String("command_id", result.CommandId),
		zap.String("agent_id", result.AgentId),
		zap.String("status", result.Status.String()))

	// Process the result as needed
	// This could involve updating command state, notifying clients, etc.

	return nil
}

// QueueCommand adds a command to the queue for processing
func (s *Service) QueueCommand(ctx context.Context, agentID, commandType string, args map[string]string) (string, error) {
	// Generate a command ID
	commandID := uuid.New().String()

	// Create a task message
	task := agent.TaskMessage{
		ID:        commandID,
		Type:      commandType,
		AgentID:   agentID,
		Args:      args,
		Status:    agent.TaskStatusQueued,
		UpdatedAt: time.Now(),
	}

	// Convert to JSON
	taskData, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task message: %w", err)
	}

	// Send to queue
	if err := queue.Send("tasks", string(taskData)); err != nil {
		return "", fmt.Errorf("failed to send task to queue: %w", err)
	}

	s.logger.Info("Command queued successfully",
		zap.String("command_id", commandID),
		zap.String("agent_id", agentID))

	return commandID, nil
}

// GetCommandStatus retrieves the status of a command
func (s *Service) GetCommandStatus(ctx context.Context, commandID string) (string, error) {
	if commandID == "" {
		return "", fmt.Errorf("command ID is required")
	}

	// Get the command status from the store
	statusKey := fmt.Sprintf("%s%s", CommandStatusKey, commandID)
	resp, err := s.store.GetValue(ctx, statusKey)
	if err != nil {
		return "", fmt.Errorf("failed to get command status: %w", err)
	}

	if resp.Message.Value == "" {
		return "", fmt.Errorf("command not found")
	}

	// Extract the status from the response
	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Message.Value), &statusData); err != nil {
		return "", fmt.Errorf("failed to parse command status: %w", err)
	}

	status, ok := statusData["status"].(string)
	if !ok {
		return "", fmt.Errorf("invalid command status format")
	}

	return status, nil
}

// GetActiveCommands retrieves the list of active commands for an agent
func (s *Service) GetActiveCommands(ctx context.Context, agentID string) ([]string, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	// Get the agent's command list
	agentCommandsKey := fmt.Sprintf(AgentCommandsKey, agentID)
	pattern := fmt.Sprintf("%s:*", agentCommandsKey)

	keys, err := s.store.Keys(ctx, pattern)
	if err != nil {
		if isOperationNotSupportedError(err) {
			s.logger.Warn("Keys operation not supported by underlying store",
				zap.String("pattern", pattern),
			)
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get active commands: %w", err)
	}

	commandIDs := make([]string, 0, len(keys))
	for _, key := range keys {
		resp, err := s.store.GetValue(ctx, key)
		if err != nil {
			s.logger.Warn("Failed to get command",
				zap.Error(err),
				zap.String("key", key),
			)
			continue
		}

		if resp.Message.Value != "" {
			commandIDs = append(commandIDs, resp.Message.Value)
		}
	}

	return commandIDs, nil
}

// isOperationNotSupportedError checks if an error indicates an unsupported operation
func isOperationNotSupportedError(err error) bool {
	return err != nil && fmt.Sprint(err) == "operation not supported"
}
