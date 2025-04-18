package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/go-api/sirius/queue"
	"github.com/SiriusScan/go-api/sirius/store"
)

// StatusMessage represents a status update received from the queue
type StatusMessage struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Manager handles the core agent functionality
type Manager struct {
	cfg     *config.Config
	logger  *slog.Logger
	store   store.KVStore
	queueCh chan struct{} // Channel to signal queue shutdown
	wg      sync.WaitGroup
}

// defaultStoreFactory is the default implementation for creating a KV store
func defaultStoreFactory() (store.KVStore, error) {
	return store.NewValkeyStore()
}

// NewManager creates a new instance of Manager
func NewManager(cfg *config.Config, logger *slog.Logger, storeFactory func() (store.KVStore, error)) (*Manager, error) {
	if storeFactory == nil {
		storeFactory = defaultStoreFactory
	}

	kvStore, err := storeFactory()
	if err != nil {
		return nil, err
	}

	return &Manager{
		cfg:     cfg,
		logger:  logger,
		store:   kvStore,
		queueCh: make(chan struct{}),
	}, nil
}

// Start initializes and starts the agent manager
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("starting agent manager")

	// Start listening to task and status queues
	queues := []string{"tasks", "status"}
	for _, qName := range queues {
		m.wg.Add(1)
		go func(queueName string) {
			defer m.wg.Done()

			// Create a channel to handle queue reconnection
			reconnect := make(chan struct{}, 1)
			reconnect <- struct{}{} // Initial connection

			for {
				select {
				case <-m.queueCh:
					return
				case <-ctx.Done():
					return
				case <-reconnect:
					go func() {
						queue.Listen(queueName, func(msg string) {
							m.handleQueueMessage(ctx, queueName, msg)
						})
						// If Listen returns, trigger reconnection
						reconnect <- struct{}{}
					}()
				}
			}
		}(qName)
	}

	<-ctx.Done()
	return nil
}

// Stop gracefully shuts down the agent manager
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info("stopping agent manager")

	// Signal queue listeners to stop
	close(m.queueCh)

	// Wait for all queue listeners to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	// Wait for either context cancellation or all listeners to finish
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	// Close store connection
	if err := m.store.Close(); err != nil {
		m.logger.Error("error closing store connection", "error", err)
	}

	return nil
}

// handleQueueMessage processes messages received from the queue
func (m *Manager) handleQueueMessage(ctx context.Context, queueName, msg string) {
	logger := m.logger.With(
		"queue", queueName,
		"message_size", len(msg),
	)

	logger.Debug("received message from queue")

	switch queueName {
	case "tasks":
		if err := m.handleTaskMessage(ctx, msg); err != nil {
			var task TaskMessage
			if jsonErr := json.Unmarshal([]byte(msg), &task); jsonErr == nil {
				logger = logger.With(
					"task_id", task.ID,
					"task_type", task.Type,
					"agent_id", task.AgentID,
				)
			}
			logger.Error("failed to handle task message",
				"error", err,
				"error_type", fmt.Sprintf("%T", err),
			)
		}
	case "status":
		if err := m.handleStatusMessage(ctx, msg); err != nil {
			var status StatusMessage
			if jsonErr := json.Unmarshal([]byte(msg), &status); jsonErr == nil {
				logger = logger.With("status_id", status.ID)
			}
			logger.Error("failed to handle status message",
				"error", err,
				"error_type", fmt.Sprintf("%T", err),
			)
		}
	default:
		logger.Warn("received message for unknown queue")
	}
}

// handleTaskMessage processes task command messages
func (m *Manager) handleTaskMessage(ctx context.Context, msg string) error {
	if msg == "" {
		return &ValidationError{Field: "message", Reason: "empty task message"}
	}

	var task TaskMessage
	if err := json.Unmarshal([]byte(msg), &task); err != nil {
		return &ParseError{Type: "task", Err: err}
	}

	logger := m.logger.With(
		"task_id", task.ID,
		"task_type", task.Type,
		"agent_id", task.AgentID,
	)

	logger.Info("processing task")

	// Store task in Valkey for persistence
	taskKey := fmt.Sprintf("task:%s", task.ID)
	if err := m.store.SetValue(ctx, taskKey, msg); err != nil {
		return &StorageError{Operation: "set", Key: taskKey, Err: err}
	}

	// Process task based on command type
	switch task.Type {
	case CommandTypeExecute:
		return m.handleExecuteCommand(ctx, &task)
	case CommandTypeStatus:
		return m.handleStatusCommand(ctx, &task)
	case CommandTypeUpdate:
		return m.handleUpdateCommand(ctx, &task)
	default:
		return &ValidationError{
			Field:  "type",
			Reason: fmt.Sprintf("unknown command type: %s", task.Type),
		}
	}
}

// handleStatusMessage processes status update messages
func (m *Manager) handleStatusMessage(ctx context.Context, msg string) error {
	if msg == "" {
		return fmt.Errorf("empty status message")
	}

	var status StatusMessage
	if err := json.Unmarshal([]byte(msg), &status); err != nil {
		return fmt.Errorf("failed to unmarshal status message: %w", err)
	}

	m.logger.Info("processing status update",
		"id", status.ID,
		"status", status.Status,
	)

	// Store status in Valkey with consistent key format
	statusKey := fmt.Sprintf("agent:%s:metrics", status.ID)
	if err := m.store.SetValue(ctx, statusKey, msg); err != nil {
		return fmt.Errorf("failed to store status: %w", err)
	}

	return nil
}

// handleExecuteCommand processes execute commands
func (m *Manager) handleExecuteCommand(ctx context.Context, task *TaskMessage) error {
	// Validate required scan arguments
	target, ok := task.Args["target"]
	if !ok {
		return fmt.Errorf("execute command missing required 'target' argument")
	}

	m.logger.Info("starting scan",
		"id", task.ID,
		"target", target,
	)

	// TODO: Implement scan logic once scan package is available
	return nil
}

// handleStatusCommand processes status check commands
func (m *Manager) handleStatusCommand(ctx context.Context, task *TaskMessage) error {
	scanID, ok := task.Args["scan_id"]
	if !ok {
		return &ValidationError{
			Field:  "scan_id",
			Reason: "status command missing required 'scan_id' argument",
			Value:  fmt.Sprintf("%v", task.Args),
		}
	}

	logger := m.logger.With(
		"task_id", task.ID,
		"scan_id", scanID,
	)
	logger.Info("checking scan status")

	// Retrieve status from Valkey using consistent key format
	statusKey := fmt.Sprintf("agent:%s:metrics", scanID)
	resp, err := m.store.GetValue(ctx, statusKey)
	if err != nil {
		return &StorageError{Operation: "get", Key: statusKey, Err: err}
	}

	if resp.Message.Value == "" {
		return &StorageError{
			Operation: "get",
			Key:       statusKey,
			Err:       fmt.Errorf("status not found"),
		}
	}

	// Parse the stored status message
	var storedStatus StatusMessage
	if err := json.Unmarshal([]byte(resp.Message.Value), &storedStatus); err != nil {
		return &ParseError{Type: "status", Err: err}
	}

	// Send status back through queue
	statusMsg := StatusMessage{
		ID:     scanID,
		Status: storedStatus.Status,
	}

	statusJSON, err := json.Marshal(statusMsg)
	if err != nil {
		return &ParseError{Type: "status", Err: err}
	}

	if err := queue.Send("status", string(statusJSON)); err != nil {
		return &QueueError{Operation: "send", Queue: "status", Err: err}
	}

	logger.Info("scan status sent", "status", statusMsg.Status)
	return nil
}

// handleUpdateCommand processes update commands
func (m *Manager) handleUpdateCommand(ctx context.Context, task *TaskMessage) error {
	m.logger.Info("processing update command",
		"id", task.ID,
		"args", task.Args,
	)

	// Store update status
	statusKey := fmt.Sprintf("update:%s:status", task.ID)
	if err := m.store.SetValue(ctx, statusKey, "in_progress"); err != nil {
		return fmt.Errorf("failed to store update status: %w", err)
	}

	// TODO: Implement actual update logic
	// This will be implemented as part of Task 7: Agent Update System

	return nil
}

// Error types for better error handling
type ValidationError struct {
	Field  string
	Reason string
	Value  string
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("validation error: %s (%s = %s)", e.Reason, e.Field, e.Value)
	}
	return fmt.Sprintf("validation error: %s (%s)", e.Reason, e.Field)
}

type ParseError struct {
	Type string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error: failed to handle %s message: %v", e.Type, e.Err)
}

type StorageError struct {
	Operation string
	Key       string
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error: failed to %s key '%s': %v", e.Operation, e.Key, e.Err)
}

type QueueError struct {
	Operation string
	Queue     string
	Err       error
}

func (e *QueueError) Error() string {
	return fmt.Sprintf("queue error: failed to %s to queue '%s': %v", e.Operation, e.Queue, e.Err)
}
