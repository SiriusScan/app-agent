package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/proto/scan"
	"github.com/SiriusScan/app-agent/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// Command status constants
	CommandStatusQueued    = "queued"
	CommandStatusAssigned  = "assigned"
	CommandStatusCompleted = "completed"
	CommandStatusFailed    = "failed"
	CommandStatusTimeout   = "timeout"

	// Command key prefixes
	CommandQueueKey     = "command:queue:"        // Valkey key for command queue
	CommandStatusKey    = "command:status:"       // Valkey key for command status
	CommandResultPrefix = "cmd:result:"           // Valkey key prefix for command results
	AgentCommandsKey    = "agent:%s:command:list" // Valkey key for agent-specific command list

	// Default polling intervals
	DefaultPollingInterval = 5 * time.Second
	DefaultCleanupInterval = 1 * time.Hour

	// Default batch size for processing commands
	DefaultBatchSize = 20
)

// CommandDistributor handles the distribution of commands to agents
type CommandDistributor struct {
	// Dependencies
	logger        *zap.Logger
	config        *config.Config
	store         store.KVStore
	resultStorage store.ResultStorageInterface
	grpcServer    GRPCServerInterface

	// State
	mu             sync.RWMutex
	isRunning      bool
	activeCommands map[string]*CommandInfo
	stopCh         chan struct{}

	// Configuration
	pollingInterval time.Duration
	cleanupInterval time.Duration
	batchSize       int
}

// CommandInfo tracks information about a command
type CommandInfo struct {
	CommandID   string            `json:"command_id"`
	AgentID     string            `json:"agent_id"`
	CommandType scan.CommandType  `json:"command_type"`
	Parameters  map[string]string `json:"parameters"`
	Status      string            `json:"status"`
	QueuedAt    time.Time         `json:"queued_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Timeout     int64             `json:"timeout"`
	Attempts    int               `json:"attempts"`
}

// GRPCServerInterface defines the interface for interacting with the gRPC server
type GRPCServerInterface interface {
	SendCommand(ctx context.Context, req *scan.SendCommandRequest) (*scan.SendCommandResponse, error)
	ListAgents(ctx context.Context, req *scan.ListAgentsRequest) (*scan.ListAgentsResponse, error)
}

// NewCommandDistributor creates a new command distributor
func NewCommandDistributor(
	logger *zap.Logger,
	cfg *config.Config,
	kvStore store.KVStore,
	resultStorage store.ResultStorageInterface,
	grpcServer GRPCServerInterface,
) *CommandDistributor {
	return &CommandDistributor{
		logger:          logger.Named("command_distributor"),
		config:          cfg,
		store:           kvStore,
		resultStorage:   resultStorage,
		grpcServer:      grpcServer,
		activeCommands:  make(map[string]*CommandInfo),
		stopCh:          make(chan struct{}),
		pollingInterval: DefaultPollingInterval,
		cleanupInterval: DefaultCleanupInterval,
		batchSize:       DefaultBatchSize,
	}
}

// Start begins the command distribution process
func (d *CommandDistributor) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.isRunning {
		d.mu.Unlock()
		return fmt.Errorf("command distributor is already running")
	}
	d.isRunning = true
	d.mu.Unlock()

	d.logger.Info("🚀 Starting command distributor",
		zap.Duration("polling_interval", d.pollingInterval),
		zap.Duration("cleanup_interval", d.cleanupInterval),
		zap.Int("batch_size", d.batchSize),
	)

	// Start the polling loop
	pollingTicker := time.NewTicker(d.pollingInterval)
	cleanupTicker := time.NewTicker(d.cleanupInterval)
	defer pollingTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("🛑 Context cancelled, stopping command distributor")
			d.Stop()
			return nil
		case <-d.stopCh:
			d.logger.Info("🛑 Stop signal received, stopping command distributor")
			return nil
		case <-pollingTicker.C:
			if err := d.pollAndProcessCommands(ctx); err != nil {
				d.logger.Error("❌ Error processing commands",
					zap.Error(err),
				)
			}
		case <-cleanupTicker.C:
			if err := d.cleanupTimedOutCommands(ctx); err != nil {
				d.logger.Error("❌ Error cleaning up timed out commands",
					zap.Error(err),
				)
			}
		}
	}
}

// Stop halts the command distribution process
func (d *CommandDistributor) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isRunning {
		return
	}

	d.logger.Info("🛑 Stopping command distributor")
	close(d.stopCh)
	d.isRunning = false
}

// QueueCommand adds a command to the distribution queue
func (d *CommandDistributor) QueueCommand(ctx context.Context, commandObj interface{}) error {
	var command *CommandInfo

	// Check the type of the command object and convert accordingly
	switch cmd := commandObj.(type) {
	case *CommandInfo:
		// Already in the right format
		command = cmd
	case *scan.SendCommandRequest:
		// Convert from SendCommandRequest to CommandInfo
		command = &CommandInfo{
			CommandID:   cmd.Command.CommandId,
			AgentID:     cmd.AgentId,
			CommandType: cmd.Command.Type,
			Parameters:  cmd.Command.Parameters,
			Status:      CommandStatusQueued,
			QueuedAt:    time.Now(),
			Timeout:     cmd.Command.TimeoutMs / 1000, // Convert from milliseconds to seconds
		}
	default:
		return fmt.Errorf("unsupported command type: %T", commandObj)
	}

	// Validate command
	if command.CommandID == "" {
		command.CommandID = uuid.New().String()
	}
	if command.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if command.QueuedAt.IsZero() {
		command.QueuedAt = time.Now()
	}
	if command.Status == "" {
		command.Status = CommandStatusQueued
	}

	// Store command in Valkey
	commandJSON, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Store in command queue
	key := fmt.Sprintf("%s%s", CommandQueueKey, command.CommandID)
	if err := d.store.SetValue(ctx, key, string(commandJSON)); err != nil {
		return fmt.Errorf("failed to store command in queue: %w", err)
	}

	// Add to agent's command list
	agentCommandsKey := fmt.Sprintf(AgentCommandsKey, command.AgentID)
	// In a real implementation, we would use a list or set operation
	// For simplicity, we'll just store the command ID
	if err := d.store.SetValue(ctx, fmt.Sprintf("%s:%s", agentCommandsKey, command.CommandID), command.CommandID); err != nil {
		d.logger.Warn("Failed to add command to agent's list",
			zap.Error(err),
			zap.String("agent_id", command.AgentID),
			zap.String("command_id", command.CommandID),
		)
	}

	d.logger.Info("✅ Command queued",
		zap.String("command_id", command.CommandID),
		zap.String("agent_id", command.AgentID),
		zap.String("type", command.CommandType.String()),
	)

	return nil
}

// pollAndProcessCommands polls for queued commands and processes them
func (d *CommandDistributor) pollAndProcessCommands(ctx context.Context) error {
	d.logger.Debug("🔄 Polling for queued commands")

	// Get a list of connected agents
	connectedAgents, err := d.getConnectedAgents(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connected agents: %w", err)
	}

	if len(connectedAgents) == 0 {
		d.logger.Debug("No connected agents, skipping command processing")
		return nil
	}

	// Process commands for each connected agent
	for agentID := range connectedAgents {
		if err := d.processAgentCommands(ctx, agentID); err != nil {
			d.logger.Error("❌ Error processing commands for agent",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
			// Continue with other agents
			continue
		}
	}

	return nil
}

// getConnectedAgents retrieves a list of connected agents
func (d *CommandDistributor) getConnectedAgents(ctx context.Context) (map[string]struct{}, error) {
	connectedAgents := make(map[string]struct{})

	// Call the gRPC server to get connected agents
	resp, err := d.grpcServer.ListAgents(ctx, &scan.ListAgentsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	for _, agent := range resp.Agents {
		// Only consider agents that are connected
		if agent.Status == "connected" {
			connectedAgents[agent.AgentId] = struct{}{}
		}
	}

	return connectedAgents, nil
}

// processAgentCommands processes commands for a specific agent
func (d *CommandDistributor) processAgentCommands(ctx context.Context, agentID string) error {
	// In a real implementation, we would use the agent's command list to find commands
	// agentCommandsKey := fmt.Sprintf(AgentCommandsKey, agentID)

	// In a real implementation, we would use a SCAN or similar operation to get all keys
	// For simplicity, we'll just list all commands in the queue and filter by agent ID
	pattern := fmt.Sprintf("%s*", CommandQueueKey)
	keys, err := d.store.Keys(ctx, pattern)
	if err != nil {
		if strings.Contains(err.Error(), "not supported") {
			d.logger.Warn("Keys operation not supported by underlying store",
				zap.String("pattern", pattern),
			)
			return nil
		}
		return fmt.Errorf("failed to list command keys: %w", err)
	}

	// Process up to batch size commands
	processedCount := 0
	for _, key := range keys {
		// Check if we've processed enough commands
		if processedCount >= d.batchSize {
			break
		}

		// Get the command
		resp, err := d.store.GetValue(ctx, key)
		if err != nil {
			d.logger.Warn("Failed to get command",
				zap.Error(err),
				zap.String("key", key),
			)
			continue
		}

		// Parse the command
		var cmd CommandInfo
		if err := json.Unmarshal([]byte(resp.Message.Value), &cmd); err != nil {
			d.logger.Warn("Failed to unmarshal command",
				zap.Error(err),
				zap.String("key", key),
			)
			continue
		}

		// Check if this command is for the current agent and is queued
		if cmd.AgentID == agentID && cmd.Status == CommandStatusQueued {
			// Send the command to the agent
			if err := d.sendCommandToAgent(ctx, &cmd); err != nil {
				d.logger.Error("❌ Failed to send command to agent",
					zap.String("command_id", cmd.CommandID),
					zap.String("agent_id", cmd.AgentID),
					zap.Error(err),
				)
				continue
			}

			// Update the command status
			now := time.Now()
			cmd.Status = CommandStatusAssigned
			cmd.StartedAt = &now
			cmd.Attempts++

			// Update the command in the store
			if err := d.updateCommandStatus(ctx, &cmd); err != nil {
				d.logger.Warn("Failed to update command status",
					zap.Error(err),
					zap.String("command_id", cmd.CommandID),
				)
			}

			// Track the command locally
			d.mu.Lock()
			d.activeCommands[cmd.CommandID] = &cmd
			d.mu.Unlock()

			processedCount++
		}
	}

	if processedCount > 0 {
		d.logger.Info("✅ Processed commands for agent",
			zap.String("agent_id", agentID),
			zap.Int("count", processedCount),
		)
	}

	return nil
}

// sendCommandToAgent sends a command to an agent via gRPC
func (d *CommandDistributor) sendCommandToAgent(ctx context.Context, cmd *CommandInfo) error {
	// Convert CommandInfo to CommandMessage
	commandMsg := &scan.CommandMessage{
		CommandId:  cmd.CommandID,
		AgentId:    cmd.AgentID,
		Type:       cmd.CommandType,
		Parameters: cmd.Parameters,
		TimeoutMs:  cmd.Timeout * 1000, // Convert seconds to milliseconds
	}

	// Send command via gRPC
	req := &scan.SendCommandRequest{
		AgentId: cmd.AgentID,
		Command: commandMsg,
	}

	resp, err := d.grpcServer.SendCommand(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	if resp.CommandId == "" {
		return fmt.Errorf("server did not return a command ID")
	}

	d.logger.Info("✅ Command sent to agent",
		zap.String("command_id", cmd.CommandID),
		zap.String("agent_id", cmd.AgentID),
	)

	return nil
}

// updateCommandStatus updates the status of a command in the store
func (d *CommandDistributor) updateCommandStatus(ctx context.Context, cmd *CommandInfo) error {
	// Marshal command to JSON
	commandJSON, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Update in the store
	key := fmt.Sprintf("%s%s", CommandQueueKey, cmd.CommandID)
	if err := d.store.SetValue(ctx, key, string(commandJSON)); err != nil {
		return fmt.Errorf("failed to update command: %w", err)
	}

	// Also store the status separately for easier querying
	statusKey := fmt.Sprintf("%s%s", CommandStatusKey, cmd.CommandID)
	statusData := map[string]interface{}{
		"command_id":   cmd.CommandID,
		"agent_id":     cmd.AgentID,
		"status":       cmd.Status,
		"updated_at":   time.Now().Unix(),
		"command_type": cmd.CommandType.String(),
	}

	statusJSON, err := json.Marshal(statusData)
	if err != nil {
		d.logger.Warn("Failed to marshal status data",
			zap.Error(err),
			zap.String("command_id", cmd.CommandID),
		)
	} else {
		if err := d.store.SetValue(ctx, statusKey, string(statusJSON)); err != nil {
			d.logger.Warn("Failed to update command status key",
				zap.Error(err),
				zap.String("command_id", cmd.CommandID),
			)
		}
	}

	return nil
}

// cleanupTimedOutCommands finds and marks timed out commands
func (d *CommandDistributor) cleanupTimedOutCommands(ctx context.Context) error {
	d.logger.Debug("🧹 Cleaning up timed out commands")

	// Get the current list of active commands
	d.mu.RLock()
	activeCommands := make(map[string]*CommandInfo, len(d.activeCommands))
	for id, cmd := range d.activeCommands {
		activeCommands[id] = cmd
	}
	d.mu.RUnlock()

	// Check each active command for timeout
	for _, cmd := range activeCommands {
		if cmd.Status == CommandStatusAssigned && cmd.StartedAt != nil {
			// Calculate how long the command has been running
			runningTime := time.Since(*cmd.StartedAt)
			timeoutDuration := time.Duration(cmd.Timeout) * time.Second

			// Check if the command has timed out
			if runningTime > timeoutDuration {
				d.logger.Warn("⏱️ Command timed out",
					zap.String("command_id", cmd.CommandID),
					zap.String("agent_id", cmd.AgentID),
					zap.Duration("running_time", runningTime),
					zap.Duration("timeout", timeoutDuration),
				)

				// Update command status to timeout
				cmd.Status = CommandStatusTimeout
				now := time.Now()
				cmd.CompletedAt = &now

				// Update in the store
				if err := d.updateCommandStatus(ctx, cmd); err != nil {
					d.logger.Error("❌ Failed to update timed out command status",
						zap.String("command_id", cmd.CommandID),
						zap.Error(err),
					)
				}

				// Create a timeout result
				result := &store.CommandResult{
					CommandID:   cmd.CommandID,
					AgentID:     cmd.AgentID,
					Status:      "TIMEOUT",
					Output:      "",
					ExitCode:    -1,
					Type:        "command",
					Metadata:    cmd.Parameters,
					ReceivedAt:  time.Now(),
					CompletedAt: time.Now(),
					Error:       fmt.Sprintf("Command timed out after %v", timeoutDuration),
				}

				// Store the result
				if err := d.resultStorage.Store(ctx, result); err != nil {
					d.logger.Error("❌ Failed to store timeout result",
						zap.String("command_id", cmd.CommandID),
						zap.Error(err),
					)
				}

				// Remove from active commands
				d.mu.Lock()
				delete(d.activeCommands, cmd.CommandID)
				d.mu.Unlock()
			}
		}
	}

	return nil
}

// HandleCommandResult processes a command result
func (d *CommandDistributor) HandleCommandResult(ctx context.Context, result *scan.ResultMessage) error {
	// Get the command info from active commands
	d.mu.RLock()
	cmd, exists := d.activeCommands[result.CommandId]
	d.mu.RUnlock()

	// Update command status
	status := CommandStatusCompleted
	if result.Status == scan.ResultStatus_RESULT_STATUS_ERROR {
		status = CommandStatusFailed
	}

	if exists {
		// Update the command status
		now := time.Now()
		cmd.Status = status
		cmd.CompletedAt = &now

		// Update in the store
		if err := d.updateCommandStatus(ctx, cmd); err != nil {
			d.logger.Warn("Failed to update command status on result",
				zap.Error(err),
				zap.String("command_id", result.CommandId),
			)
		}

		// Remove from active commands
		d.mu.Lock()
		delete(d.activeCommands, result.CommandId)
		d.mu.Unlock()
	} else {
		// This might be a result for a command we're not tracking
		// Try to get it from the store
		key := fmt.Sprintf("%s%s", CommandQueueKey, result.CommandId)
		resp, err := d.store.GetValue(ctx, key)
		if err == nil && resp.Message.Value != "" {
			var cmdInfo CommandInfo
			if err := json.Unmarshal([]byte(resp.Message.Value), &cmdInfo); err == nil {
				// Update the command status
				now := time.Now()
				cmdInfo.Status = status
				cmdInfo.CompletedAt = &now

				// Update in the store
				if err := d.updateCommandStatus(ctx, &cmdInfo); err != nil {
					d.logger.Warn("Failed to update command status on result",
						zap.Error(err),
						zap.String("command_id", result.CommandId),
					)
				}
			}
		}
	}

	d.logger.Info("✅ Command result processed",
		zap.String("command_id", result.CommandId),
		zap.String("agent_id", result.AgentId),
		zap.String("status", result.Status.String()),
	)

	return nil
}

// SetPollingInterval sets the polling interval
func (d *CommandDistributor) SetPollingInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pollingInterval = interval
}

// SetCleanupInterval sets the cleanup interval
func (d *CommandDistributor) SetCleanupInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cleanupInterval = interval
}

// SetBatchSize sets the batch size
func (d *CommandDistributor) SetBatchSize(batchSize int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.batchSize = batchSize
}
