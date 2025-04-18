package command

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"

	pb "github.com/SiriusScan/app-agent/proto/scan"
)

// CommandStatus represents the status of a command
type CommandStatus string

const (
	// CommandStatusPending indicates the command is waiting to be sent to an agent
	CommandStatusPending CommandStatus = "pending"
	// CommandStatusSent indicates the command has been sent to an agent
	CommandStatusSent CommandStatus = "sent"
	// CommandStatusCompleted indicates the command has been completed by an agent
	CommandStatusCompleted CommandStatus = "completed"
	// CommandStatusFailed indicates the command failed to execute
	CommandStatusFailed CommandStatus = "failed"

	// Polling interval for checking new commands
	defaultPollingInterval = 5 * time.Second
	// Command key prefix in Valkey
	commandKeyPrefix = "cmd:"
)

// AgentCommandSender is an interface for sending commands to agents
type AgentCommandSender interface {
	SendCommand(agentID string, cmd *pb.CommandMessage) error
	IsAgentConnected(agentID string) bool
	GetAgentIDs() []string
	SendCommandToAgent(agentID string, cmd Command) error
	GetConnectedAgentIDs() []string
}

// Distributor distributes commands from Valkey to connected agents
type Distributor struct {
	logger       *zap.Logger
	store        store.KVStore
	sender       AgentCommandSender
	interval     time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	isRunning    bool
	runningMutex sync.Mutex
}

// NewDistributor creates a new command distributor
func NewDistributor(logger *zap.Logger, store store.KVStore, sender AgentCommandSender) *Distributor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Distributor{
		logger:   logger.Named("command-distributor"),
		store:    store,
		sender:   sender,
		interval: defaultPollingInterval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the command distribution process
func (d *Distributor) Start() {
	d.runningMutex.Lock()
	defer d.runningMutex.Unlock()

	if d.isRunning {
		d.logger.Info("Distributor already running")
		return
	}

	d.isRunning = true
	d.wg.Add(1)

	d.logger.Info("Starting command distributor", zap.Duration("interval", d.interval))

	go func() {
		defer d.wg.Done()
		d.distributionLoop()
	}()
}

// Stop halts the command distribution process
func (d *Distributor) Stop() {
	d.runningMutex.Lock()
	if !d.isRunning {
		d.runningMutex.Unlock()
		return
	}
	d.isRunning = false
	d.runningMutex.Unlock()

	d.logger.Info("Stopping command distributor")
	d.cancel()
	d.wg.Wait()
}

// distributionLoop runs periodically to check for pending commands and distribute them
func (d *Distributor) distributionLoop() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Do an initial distribution on startup
	d.distributeCommands()

	for {
		select {
		case <-ticker.C:
			d.distributeCommands()
		case <-d.ctx.Done():
			d.logger.Info("Distribution loop stopped")
			return
		}
	}
}

// distributeCommands gets and processes commands for all connected agents
func (d *Distributor) distributeCommands() {
	// Get list of connected agents
	agentIDs := d.sender.GetConnectedAgentIDs()
	if len(agentIDs) == 0 {
		d.logger.Debug("No connected agents")
		return
	}

	// Process the latest command for each connected agent
	for _, agentID := range agentIDs {
		d.processLatestCommandForAgent(agentID)
	}
}

// processLatestCommandForAgent gets and processes the latest command for a specific agent
func (d *Distributor) processLatestCommandForAgent(agentID string) {
	// Get latest command for this agent
	cmdKey := fmt.Sprintf("cmd:%s:latest", agentID)
	cmdResp, err := d.store.GetValue(d.ctx, cmdKey)
	if err != nil {
		d.logger.Error("Failed to get command key",
			zap.Error(err),
			zap.String("agent_id", agentID),
			zap.String("key", cmdKey))
		return
	}

	// Check if we have a valid command
	if cmdResp.Type == "nil" || cmdResp.Message.Value == "" {
		d.logger.Debug("No pending command for agent", zap.String("agent_id", agentID))
		return
	}

	// Deserialize command
	var cmd Command
	if err := json.Unmarshal([]byte(cmdResp.Message.Value), &cmd); err != nil {
		d.logger.Error("Failed to unmarshal command",
			zap.Error(err),
			zap.String("agent_id", agentID),
			zap.String("command", cmdResp.Message.Value))
		return
	}

	// Validate command
	if cmd.ID == "" || cmd.Type == "" || cmd.AgentID != agentID {
		d.logger.Error("Invalid command",
			zap.String("agent_id", agentID),
			zap.String("command_id", cmd.ID),
			zap.String("command_type", string(cmd.Type)),
			zap.String("command_agent_id", cmd.AgentID))
		return
	}

	// Check if command has already been sent
	statusKey := fmt.Sprintf("status:%s:%s", cmd.AgentID, cmd.ID)
	statusResp, err := d.store.GetValue(d.ctx, statusKey)
	if err == nil && statusResp.Type != "nil" && statusResp.Message.Value != "" {
		status := CommandStatus(statusResp.Message.Value)
		if status == CommandStatusSent || status == CommandStatusCompleted {
			d.logger.Debug("Command already processed, skipping",
				zap.String("agent_id", agentID),
				zap.String("command_id", cmd.ID),
				zap.String("status", string(status)))
			return
		}
	}

	// Skip if agent not connected
	if !d.sender.IsAgentConnected(agentID) {
		d.logger.Debug("Agent not connected, skipping command",
			zap.String("agent_id", agentID),
			zap.String("command_id", cmd.ID))
		return
	}

	// Send command to agent
	d.logger.Info("Sending command to agent",
		zap.String("agent_id", cmd.AgentID),
		zap.String("command_id", cmd.ID),
		zap.String("command_type", string(cmd.Type)))

	// Convert to gRPC command message
	cmdMsg := &pb.CommandMessage{
		Id:             cmd.ID,
		Type:           cmd.Type,
		Params:         cmd.Params,
		TimeoutSeconds: 60, // Default timeout of 60 seconds
	}

	if err := d.sender.SendCommand(cmd.AgentID, cmdMsg); err != nil {
		d.logger.Error("Failed to send command to agent",
			zap.Error(err),
			zap.String("agent_id", cmd.AgentID),
			zap.String("command_id", cmd.ID))

		// Update command status to failed
		if err := d.store.SetValue(d.ctx, statusKey, string(CommandStatusFailed)); err != nil {
			d.logger.Error("Failed to update command status",
				zap.Error(err),
				zap.String("key", statusKey),
				zap.String("value", string(CommandStatusFailed)))
		}
		return
	}

	// Update command status to sent
	if err := d.store.SetValue(d.ctx, statusKey, string(CommandStatusSent)); err != nil {
		d.logger.Error("Failed to update command status",
			zap.Error(err),
			zap.String("key", statusKey),
			zap.String("value", string(CommandStatusSent)))
	}
}
