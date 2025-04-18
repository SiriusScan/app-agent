package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// MessageHandler processes messages from RabbitMQ
type MessageHandler struct {
	processor *CommandProcessor
	logger    *zap.Logger
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(processor *CommandProcessor, logger *zap.Logger) *MessageHandler {
	return &MessageHandler{
		processor: processor,
		logger:    logger,
	}
}

// HandleTaskMessage processes task messages from RabbitMQ
func (h *MessageHandler) HandleTaskMessage(msg string) {
	ctx := context.Background()

	h.logger.Debug("Received task message", zap.String("message", msg))

	if err := h.processor.ProcessCommand(ctx, []byte(msg)); err != nil {
		h.logger.Error("Failed to process task message",
			zap.String("message", msg),
			zap.Error(err))
		return
	}
}

// HandleStatusMessage processes status messages from RabbitMQ
func (h *MessageHandler) HandleStatusMessage(msg string) {
	h.logger.Debug("Received status message", zap.String("message", msg))

	var statusMsg struct {
		AgentID string            `json:"agent_id"`
		Status  string            `json:"status"`
		Metrics map[string]string `json:"metrics"`
	}

	if err := json.Unmarshal([]byte(msg), &statusMsg); err != nil {
		h.logger.Error("Failed to unmarshal status message",
			zap.String("message", msg),
			zap.Error(err))
		return
	}

	// TODO: Implement status message handling
	// This will be implemented as part of Task 6: Agent Management
}

// SetupQueues initializes the RabbitMQ queues and starts listening
func (h *MessageHandler) SetupQueues(queueClient QueueClient) error {
	// Set up task queue
	if err := queueClient.Listen("tasks", h.HandleTaskMessage); err != nil {
		return fmt.Errorf("failed to set up task queue: %w", err)
	}

	// Set up status queue
	if err := queueClient.Listen("status", h.HandleStatusMessage); err != nil {
		return fmt.Errorf("failed to set up status queue: %w", err)
	}

	h.logger.Info("Message queues initialized")
	return nil
}
