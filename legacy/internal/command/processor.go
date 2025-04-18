package command

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SiriusScan/go-api/sirius/queue"
	"github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"
)

// Command represents a command message received from RabbitMQ
type Command struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	AgentID   string            `json:"agent_id"`
	Params    map[string]string `json:"params"`
	Timestamp int64             `json:"timestamp"`
}

// Processor handles commands received from the queue
type Processor struct {
	logger *zap.Logger
	store  store.KVStore
	ctx    context.Context
	cancel context.CancelFunc
}

// NewProcessor creates a new command processor
func NewProcessor(logger *zap.Logger, store store.KVStore) *Processor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Processor{
		logger: logger.Named("command-processor"),
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins listening for commands on the specified queue
func (p *Processor) Start(queueName string) {
	p.logger.Info("Starting command processor", zap.String("queue", queueName))

	// Use the Sirius queue API to listen for messages
	go queue.Listen(queueName, p.processMessage)
}

// Stop gracefully shuts down the processor
func (p *Processor) Stop() {
	p.logger.Info("Stopping command processor")
	p.cancel()
}

// processMessage handles a single message from the queue
func (p *Processor) processMessage(msg string) {
	p.logger.Debug("Received message from queue", zap.String("message", msg))

	// Parse the command
	var cmd Command
	if err := json.Unmarshal([]byte(msg), &cmd); err != nil {
		p.logger.Error("Failed to unmarshal command", zap.Error(err), zap.String("message", msg))
		return
	}

	p.logger.Info("Processing command",
		zap.String("command_id", cmd.ID),
		zap.String("type", cmd.Type),
		zap.String("agent_id", cmd.AgentID))

	// Store the command in Valkey for later distribution to agents
	if err := p.storeCommand(cmd); err != nil {
		p.logger.Error("Failed to store command", zap.Error(err), zap.String("command_id", cmd.ID))
		return
	}

	p.logger.Info("Command stored successfully", zap.String("command_id", cmd.ID))
}

// storeCommand saves the command to the key-value store
func (p *Processor) storeCommand(cmd Command) error {
	// Create keys for the command - both an ID-specific key and a "latest" key
	idKey := fmt.Sprintf("cmd:%s:%s", cmd.AgentID, cmd.ID)
	latestKey := fmt.Sprintf("cmd:%s:latest", cmd.AgentID)

	// Serialize the command to JSON
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Store the command in Valkey with both keys
	// First with the ID-specific key
	if err := p.store.SetValue(p.ctx, idKey, string(cmdJSON)); err != nil {
		return fmt.Errorf("failed to store command with ID key: %w", err)
	}

	// Then with the "latest" key
	if err := p.store.SetValue(p.ctx, latestKey, string(cmdJSON)); err != nil {
		return fmt.Errorf("failed to store command with latest key: %w", err)
	}

	p.logger.Debug("Command stored in Valkey",
		zap.String("id_key", idKey),
		zap.String("latest_key", latestKey),
		zap.String("command_id", cmd.ID))

	return nil
}
