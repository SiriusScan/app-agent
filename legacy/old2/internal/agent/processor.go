package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/go-api/sirius/queue"
	"github.com/SiriusScan/go-api/sirius/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	// Task status constants
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskMessage represents a command message received from RabbitMQ
type TaskMessage struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Args      map[string]string `json:"args"`
	Status    TaskStatus        `json:"status"`
	AgentID   string            `json:"agent_id,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	Error     string            `json:"error,omitempty"`
}

// MessageProcessor handles incoming messages from RabbitMQ
type MessageProcessor struct {
	logger *zap.Logger
	store  store.KVStore
	cfg    *config.Config
}

// NewMessageProcessor creates a new message processor instance
func NewMessageProcessor(logger *zap.Logger, store store.KVStore, cfg *config.Config) *MessageProcessor {
	// Create a console encoder config
	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	consoleConfig.EncodeDuration = zapcore.StringDurationEncoder
	consoleConfig.ConsoleSeparator = "  "

	// Create a console encoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)

	// Create stdout core
	stdout := zapcore.Lock(os.Stdout)
	consoleCore := zapcore.NewCore(consoleEncoder, stdout, zap.NewAtomicLevelAt(zap.DebugLevel))

	// Create logger with only console output
	logger = zap.New(consoleCore, zap.AddCaller())

	p := &MessageProcessor{
		logger: logger,
		store:  store,
		cfg:    cfg,
	}

	return p
}

// ProcessMessage handles an incoming message from RabbitMQ
func (p *MessageProcessor) ProcessMessage(ctx context.Context, msg string) error {
	logger := p.logger.With(
		zap.String("trace_id", uuid.New().String()),
		zap.Int("message_size", len(msg)),
		zap.String("processor_id", p.cfg.Version),
		zap.Time("processing_start", time.Now()),
	)

	logger.Debug("📝 Processing message",
		zap.String("message", msg),
		zap.String("phase", "start"))

	// Parse the message
	var task TaskMessage
	if err := json.Unmarshal([]byte(msg), &task); err != nil {
		logger.Error("❌ Failed to parse message",
			zap.Error(err),
			zap.String("phase", "parsing"),
			zap.String("error_type", fmt.Sprintf("%T", err)))
		return fmt.Errorf("failed to parse message: %w", err)
	}

	logger = logger.With(
		zap.String("task_type", task.Type),
		zap.String("task_id", task.ID),
	)

	// Validate the task
	if err := validateTask(&task); err != nil {
		logger.Error("❌ Invalid task",
			zap.Error(err),
			zap.String("phase", "validation"),
			zap.String("validation_error", err.Error()))
		return fmt.Errorf("invalid task: %w", err)
	}

	// Generate task ID if not provided
	if task.ID == "" {
		task.ID = generateTaskID()
		logger = logger.With(zap.String("generated_task_id", task.ID))
		logger.Debug("🆔 Generated new task ID")
	}

	// Convert "exec" type to "stop" for agent processing
	// (since we're reusing COMMAND_TYPE_STOP for exec commands)
	originalType := task.Type
	if task.Type == "exec" {
		// Store the command type for later use
		if task.Args == nil {
			task.Args = make(map[string]string)
		}
		task.Args["original_type"] = "exec"
		logger.Info("🔄 Converting exec command to stop command type",
			zap.String("original_type", originalType))
	}

	// Store the task in Valkey
	storeStart := time.Now()
	if err := p.storeTask(ctx, &task); err != nil {
		logger.Error("❌ Failed to store task",
			zap.Error(err),
			zap.String("phase", "storage"),
			zap.Duration("store_attempt_duration", time.Since(storeStart)))
		return fmt.Errorf("failed to store task: %w", err)
	}

	logger.Info("✅ Successfully processed message",
		zap.String("task_id", task.ID),
		zap.String("type", originalType),
		zap.Duration("total_duration", time.Since(storeStart)),
		zap.String("phase", "complete"))

	return nil
}

// Start initializes the message processor and starts listening for messages
func (p *MessageProcessor) Start(ctx context.Context) error {
	p.logger.Info("🚀 Starting message processor",
		zap.String("version", p.cfg.Version),
		zap.String("phase", "initialization"))

	// Create message handler that includes context and enhanced logging
	handler := func(msg string) {
		// Check if context is done
		select {
		case <-ctx.Done():
			p.logger.Info("🛑 Processor shutting down, stopping message processing",
				zap.String("phase", "shutdown"))
			return
		default:
		}

		startTime := time.Now()
		logger := p.logger.With(
			zap.String("message_handler", "task_queue"),
			zap.Time("handler_start", startTime))

		logger.Info("📥 Received message from queue",
			zap.Int("message_size", len(msg)),
			zap.String("phase", "message_received"))

		if err := p.ProcessMessage(ctx, msg); err != nil {
			logger.Error("❌ Error processing message",
				zap.Error(err),
				zap.Duration("processing_duration", time.Since(startTime)),
				zap.String("phase", "processing_failed"))
		} else {
			logger.Info("✅ Message processing complete",
				zap.Duration("processing_duration", time.Since(startTime)),
				zap.String("phase", "processing_success"))
		}
	}

	// Start listening for messages using Sirius Go API queue package
	p.logger.Info("🔄 Setting up message queue listener",
		zap.String("queue", "tasks"),
		zap.String("phase", "queue_setup"))

	// Create a channel to keep the goroutine alive
	done := make(chan struct{})
	go func() {
		// Wrap the queue.Listen call with our logger
		p.logger.Info("🎧 Starting queue listener",
			zap.String("queue", "tasks"),
			zap.String("phase", "queue_start"))

		// Create a wrapper handler that logs
		wrappedHandler := func(msg string) {
			p.logger.Info("📥 Received message",
				zap.Int("message_size", len(msg)),
				zap.String("phase", "message_received"))
			handler(msg)
		}

		queue.Listen("tasks", wrappedHandler)
		p.logger.Info("⏹️ Queue listener stopped",
			zap.String("phase", "queue_stopped"))
		close(done)
	}()

	// Wait for context cancellation or done signal
	select {
	case <-ctx.Done():
		p.logger.Info("🛑 Context cancelled, stopping message processor",
			zap.String("phase", "shutdown"))
	case <-done:
		p.logger.Info("⏹️ Message queue listener stopped",
			zap.String("phase", "listener_stopped"))
	}

	p.logger.Info("✅ Message queue listener started",
		zap.String("phase", "queue_ready"))

	return nil
}

// storeTask stores a task in Valkey with enhanced logging
func (p *MessageProcessor) storeTask(ctx context.Context, task *TaskMessage) error {
	logger := p.logger.With(
		zap.String("task_id", task.ID),
		zap.String("operation", "store_task"))

	// Convert task to JSON
	taskData, err := json.Marshal(task)
	if err != nil {
		logger.Error("❌ Failed to marshal task",
			zap.Error(err),
			zap.String("phase", "json_marshal"))
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Store in Valkey
	storeStart := time.Now()
	key := fmt.Sprintf("task:%s", task.ID)
	if err := p.store.SetValue(ctx, key, string(taskData)); err != nil {
		logger.Error("❌ Failed to store task in Valkey",
			zap.Error(err),
			zap.String("key", key),
			zap.Duration("attempt_duration", time.Since(storeStart)),
			zap.String("phase", "valkey_store"))
		return fmt.Errorf("failed to store task in Valkey: %w", err)
	}

	logger.Debug("💾 Task stored successfully",
		zap.String("key", key),
		zap.Duration("store_duration", time.Since(storeStart)),
		zap.String("phase", "store_complete"))

	return nil
}

// UpdateTaskStatus updates the status of a task
func (p *MessageProcessor) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, errorMsg string) error {
	logger := p.logger.With(zap.String("task_id", taskID), zap.String("status", string(status)))
	logger.Debug("🔄 Updating task status")

	// Retrieve existing task
	task, err := p.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update task status
	task.Status = status
	task.UpdatedAt = time.Now()
	if errorMsg != "" {
		task.Error = errorMsg
	}

	// Store updated task
	if err := p.storeTask(ctx, task); err != nil {
		logger.Error("Failed to store updated task status", zap.Error(err))
		return fmt.Errorf("failed to store task status: %w", err)
	}

	// Send status update message
	statusMsg := fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, taskID, status)
	if err := queue.Send("task-status", statusMsg); err != nil {
		logger.Error("Failed to send status update", zap.Error(err))
		return fmt.Errorf("failed to send status update: %w", err)
	}

	logger.Info("Successfully updated task status")
	return nil
}

// GetTask retrieves a task from storage
func (p *MessageProcessor) GetTask(ctx context.Context, taskID string) (*TaskMessage, error) {
	key := fmt.Sprintf("task:%s", taskID)
	resp, err := p.store.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task from store: %w", err)
	}

	var task TaskMessage
	if err := json.Unmarshal([]byte(resp.Message.Value), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// validateTask performs validation on a task message
func validateTask(task *TaskMessage) error {
	if task.Type == "" {
		return fmt.Errorf("task type is required")
	}

	// Validate task type
	validTypes := map[string]bool{
		"scan":    true,
		"execute": true,
		"exec":    true,
		"status":  true,
		"update":  true,
	}
	if !validTypes[task.Type] {
		return fmt.Errorf("invalid task type: %s", task.Type)
	}

	// Validate status if provided
	if task.Status != "" {
		validStatus := map[TaskStatus]bool{
			TaskStatusQueued:     true,
			TaskStatusInProgress: true,
			TaskStatusCompleted:  true,
			TaskStatusFailed:     true,
			TaskStatusCancelled:  true,
		}
		if !validStatus[task.Status] {
			return fmt.Errorf("invalid task status: %s", task.Status)
		}
	}

	// Validate required arguments based on type
	switch task.Type {
	case "scan":
		if _, ok := task.Args["target"]; !ok {
			return fmt.Errorf("scan task requires 'target' argument")
		}
	case "execute", "exec":
		if _, ok := task.Args["command"]; !ok {
			return fmt.Errorf("execute task requires 'command' argument")
		}
	case "status":
		if _, ok := task.Args["scan_id"]; !ok && task.AgentID == "" {
			return fmt.Errorf("status task requires either 'scan_id' argument or agent_id")
		}
	}

	return nil
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task-%s", generateTraceID())
}

// generateTraceID generates a unique trace ID for request tracking
func generateTraceID() string {
	return uuid.New().String()
}
