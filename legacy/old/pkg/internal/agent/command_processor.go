package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	scanpb "github.com/SiriusScan/app-agent/proto/scan/v1"
	"github.com/SiriusScan/go-api/sirius/queue"
	"github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"
)

// CommandType represents the type of command to execute
type CommandType string

const (
	CommandTypeScan    CommandType = "SCAN"
	CommandTypeExecute CommandType = "EXECUTE"
	CommandTypeStatus  CommandType = "STATUS"
	CommandTypeUpdate  CommandType = "UPDATE"
)

// ToProtoType converts a CommandType to scanpb.CommandType
func (t CommandType) ToProtoType() scanpb.CommandType {
	switch t {
	case CommandTypeScan:
		return scanpb.CommandType_COMMAND_TYPE_SCAN
	case CommandTypeExecute:
		return scanpb.CommandType_COMMAND_TYPE_EXECUTE
	case CommandTypeStatus:
		return scanpb.CommandType_COMMAND_TYPE_STATUS
	case CommandTypeUpdate:
		return scanpb.CommandType_COMMAND_TYPE_UPDATE
	default:
		return scanpb.CommandType_COMMAND_TYPE_UNSPECIFIED
	}
}

// FromProtoType converts a scanpb.CommandType to CommandType
func FromProtoType(t scanpb.CommandType) CommandType {
	switch t {
	case scanpb.CommandType_COMMAND_TYPE_SCAN:
		return CommandTypeScan
	case scanpb.CommandType_COMMAND_TYPE_EXECUTE:
		return CommandTypeExecute
	case scanpb.CommandType_COMMAND_TYPE_STATUS:
		return CommandTypeStatus
	case scanpb.CommandType_COMMAND_TYPE_UPDATE:
		return CommandTypeUpdate
	default:
		return ""
	}
}

// TaskMessage represents a command being processed
type TaskMessage struct {
	ID        string            `json:"command_id"`
	Type      CommandType       `json:"type"`
	AgentID   string            `json:"agent_id"`
	Args      map[string]string `json:"args"`
	Timeout   int64             `json:"timeout"`
	Status    string            `json:"status"`
	Error     string            `json:"error,omitempty"`
	Output    string            `json:"output,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`

	// Scan result fields
	ScanID   string    `json:"scan_id,omitempty"`
	Findings []Finding `json:"findings,omitempty"`

	// Execute result fields
	ExitCode   int32  `json:"exit_code,omitempty"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`

	// Status result fields
	Metrics    map[string]string `json:"metrics,omitempty"`
	SystemInfo *SystemInfo       `json:"system_info,omitempty"`

	// Update result fields
	UpdateSuccess  bool     `json:"update_success,omitempty"`
	UpdatedConfigs []string `json:"updated_configs,omitempty"`
}

// Finding represents a scan finding
type Finding struct {
	ID          string            `json:"id"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Details     map[string]string `json:"details"`
}

// SystemInfo contains system status information
type SystemInfo struct {
	OSVersion   string            `json:"os_version"`
	Uptime      int64             `json:"uptime"`
	MemoryTotal int64             `json:"memory_total"`
	MemoryFree  int64             `json:"memory_free"`
	CPUUsage    float32           `json:"cpu_usage"`
	Extra       map[string]string `json:"extra"`
}

// CommandProcessor handles processing of commands from RabbitMQ
type CommandProcessor struct {
	server CommandQueueServer
	store  store.KVStore
	logger *zap.Logger
	mu     sync.RWMutex
	tasks  map[string]*TaskMessage
}

// CommandState tracks the state of a command
type CommandState struct {
	CommandID   string
	AgentID     string
	Status      string
	StartTime   time.Time
	LastUpdated time.Time
	Retries     int
	Error       string
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor(server CommandQueueServer, store store.KVStore, logger *zap.Logger) *CommandProcessor {
	return &CommandProcessor{
		server: server,
		store:  store,
		logger: logger,
		tasks:  make(map[string]*TaskMessage),
	}
}

// Start begins listening for task commands from RabbitMQ
func (p *CommandProcessor) Start(ctx context.Context) error {
	p.logger.Info("starting command processor")

	// Listen for task commands
	queue.Listen("tasks", func(msg string) {
		if err := p.handleTaskMessage(ctx, msg); err != nil {
			p.logger.Error("failed to handle task message",
				zap.String("message", msg),
				zap.Error(err))
		}
	})

	<-ctx.Done()
	return nil
}

// handleTaskMessage processes a task message from RabbitMQ
func (p *CommandProcessor) handleTaskMessage(ctx context.Context, msg string) error {
	// Parse task message directly into TaskMessage
	var task TaskMessage
	if err := json.Unmarshal([]byte(msg), &task); err != nil {
		return fmt.Errorf("failed to unmarshal task message: %w", err)
	}

	// Convert type to uppercase
	task.Type = CommandType(strings.ToUpper(string(task.Type)))

	// Validate task
	if err := p.validateCommand(&task); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	// Store task in memory
	p.mu.Lock()
	p.tasks[task.ID] = &task
	p.mu.Unlock()

	// Store task in Valkey
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	key := fmt.Sprintf("task:%s", task.ID)
	if err := p.store.SetValue(ctx, key, string(taskJSON)); err != nil {
		return fmt.Errorf("failed to store task in Valkey: %w", err)
	}

	// Create and queue command for agent
	grpcCmd, err := p.createGRPCCommand(&task)
	if err != nil {
		return fmt.Errorf("failed to create gRPC command: %w", err)
	}

	if err := p.server.QueueCommand(task.AgentID, grpcCmd); err != nil {
		return fmt.Errorf("failed to queue command: %w", err)
	}

	p.logger.Info("task queued for execution",
		zap.String("task_id", task.ID),
		zap.String("agent_id", task.AgentID),
		zap.String("type", string(task.Type)))

	return nil
}

// validateCommand validates a command message
func (p *CommandProcessor) validateCommand(cmd *TaskMessage) error {
	if cmd.ID == "" {
		return fmt.Errorf("command ID is required")
	}

	if cmd.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}

	switch cmd.Type {
	case CommandTypeScan:
		if _, ok := cmd.Args["target"]; !ok {
			return fmt.Errorf("scan command requires 'target' argument")
		}
	case CommandTypeExecute:
		if _, ok := cmd.Args["command"]; !ok {
			return fmt.Errorf("execute command requires 'command' argument")
		}
	case CommandTypeStatus:
		// No additional validation needed
	case CommandTypeUpdate:
		if _, ok := cmd.Args["version"]; !ok {
			return fmt.Errorf("update command requires 'version' argument")
		}
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}

	return nil
}

// GetTask retrieves a task by ID
func (p *CommandProcessor) GetTask(ctx context.Context, taskID string) (*TaskMessage, error) {
	// Try memory first
	p.mu.RLock()
	if task, ok := p.tasks[taskID]; ok {
		p.mu.RUnlock()
		return task, nil
	}
	p.mu.RUnlock()

	// Try Valkey
	key := fmt.Sprintf("task:%s", taskID)
	resp, err := p.store.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task from Valkey: %w", err)
	}

	var task TaskMessage
	if err := json.Unmarshal([]byte(resp.Message.Value), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Cache in memory
	p.mu.Lock()
	p.tasks[taskID] = &task
	p.mu.Unlock()

	return &task, nil
}

// UpdateTaskStatus updates the status of a task
func (p *CommandProcessor) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	task, err := p.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update task status
	task.Status = status
	task.Timestamp = time.Now()

	// Store updated task
	if err := p.storeTask(ctx, task); err != nil {
		return fmt.Errorf("failed to store task: %w", err)
	}

	return nil
}

// CleanupTask removes a completed task from memory
func (p *CommandProcessor) CleanupTask(taskID string) {
	p.mu.Lock()
	delete(p.tasks, taskID)
	p.mu.Unlock()
}

// ProcessCommand handles a command received from RabbitMQ
func (p *CommandProcessor) ProcessCommand(ctx context.Context, msg []byte) error {
	var task TaskMessage
	if err := json.Unmarshal(msg, &task); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	// Convert type to uppercase
	task.Type = CommandType(strings.ToUpper(string(task.Type)))

	// Validate command
	if err := p.validateCommand(&task); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}

	// Create command state
	state := &CommandState{
		CommandID:   task.ID,
		AgentID:     task.AgentID,
		Status:      "pending",
		StartTime:   time.Now(),
		LastUpdated: time.Now(),
	}

	// Store initial state
	if err := p.storeCommandState(ctx, state); err != nil {
		return fmt.Errorf("failed to store command state: %w", err)
	}

	// Track command
	p.mu.Lock()
	p.tasks[task.ID] = &task
	p.mu.Unlock()

	// Convert to gRPC command
	grpcCmd, err := p.createGRPCCommand(&task)
	if err != nil {
		return fmt.Errorf("failed to create gRPC command: %w", err)
	}

	// Queue command for agent
	if err := p.server.QueueCommand(task.AgentID, grpcCmd); err != nil {
		state.Status = "failed"
		state.Error = err.Error()
		p.storeCommandState(ctx, state)
		return fmt.Errorf("failed to queue command: %w", err)
	}

	// Start timeout monitoring if specified
	if task.Timeout > 0 {
		go p.monitorTimeout(ctx, task.ID, time.Duration(task.Timeout)*time.Second)
	}

	p.logger.Info("Command queued",
		zap.String("command_id", task.ID),
		zap.String("agent_id", task.AgentID),
		zap.String("type", string(task.Type)))

	return nil
}

// HandleCommandResult processes the result of a command from an agent
func (p *CommandProcessor) HandleCommandResult(ctx context.Context, result *scanpb.CommandResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, ok := p.tasks[result.CommandId]
	if !ok {
		return fmt.Errorf("command %s not found", result.CommandId)
	}

	// Update common fields
	task.Status = result.ExecutionStatus
	task.Output = result.Output
	task.Metadata = result.Metadata
	task.Timestamp = time.Unix(result.Timestamp, 0)

	// Handle specific result details
	switch details := result.ResultDetails.(type) {
	case *scanpb.CommandResult_Scan:
		scanResult := details.Scan
		task.ScanID = scanResult.ScanId
		task.Findings = make([]Finding, len(scanResult.Findings))
		for i, f := range scanResult.Findings {
			task.Findings[i] = Finding{
				ID:          f.Id,
				Severity:    f.Severity,
				Description: f.Description,
				Details:     f.Details,
			}
		}
	case *scanpb.CommandResult_Execute:
		execResult := details.Execute
		task.ExitCode = execResult.ExitCode
		task.Stdout = execResult.Stdout
		task.Stderr = execResult.Stderr
		task.DurationMs = execResult.DurationMs
	case *scanpb.CommandResult_Status:
		statusResult := details.Status
		task.Metrics = statusResult.Metrics
		if sysInfo := statusResult.System; sysInfo != nil {
			task.SystemInfo = &SystemInfo{
				OSVersion:   sysInfo.OsVersion,
				Uptime:      sysInfo.Uptime,
				MemoryTotal: sysInfo.MemoryTotal,
				MemoryFree:  sysInfo.MemoryFree,
				CPUUsage:    sysInfo.CpuUsage,
				Extra:       sysInfo.Extra,
			}
		}
	case *scanpb.CommandResult_Update:
		updateResult := details.Update
		task.UpdateSuccess = updateResult.Success
		task.UpdatedConfigs = updateResult.Updated
		if updateResult.Error != "" {
			task.Error = updateResult.Error
		}
	}

	// Store updated task
	if err := p.storeTask(ctx, task); err != nil {
		return fmt.Errorf("failed to store task: %w", err)
	}

	return nil
}

// createGRPCCommand creates a gRPC command from a task message
func (p *CommandProcessor) createGRPCCommand(cmd *TaskMessage) (*scanpb.Command, error) {
	grpcCmd := &scanpb.Command{
		CommandId: cmd.ID,
		Type:      cmd.Type.ToProtoType(),
		Timeout:   cmd.Timeout,
	}

	switch cmd.Type {
	case CommandTypeScan:
		grpcCmd.CommandDetails = &scanpb.Command_Scan{
			Scan: &scanpb.ScanCommand{
				ScanType: cmd.Args["type"],
				Targets:  []string{cmd.Args["target"]},
				Options:  cmd.Args,
			},
		}
	case CommandTypeExecute:
		grpcCmd.CommandDetails = &scanpb.Command_Execute{
			Execute: &scanpb.ExecuteCommand{
				Command:    cmd.Args["command"],
				Args:       strings.Split(cmd.Args["args"], " "),
				WorkingDir: cmd.Args["working_dir"],
				Env:        cmd.Args,
			},
		}
	case CommandTypeStatus:
		grpcCmd.CommandDetails = &scanpb.Command_Status{
			Status: &scanpb.StatusCommand{
				Metrics: strings.Split(cmd.Args["metrics"], ","),
			},
		}
	case CommandTypeUpdate:
		grpcCmd.CommandDetails = &scanpb.Command_Update{
			Update: &scanpb.UpdateCommand{
				Config:          cmd.Args,
				RestartRequired: cmd.Args["restart"] == "true",
			},
		}
	}

	return grpcCmd, nil
}

// storeCommandState stores the command state in Valkey
func (p *CommandProcessor) storeCommandState(ctx context.Context, state *CommandState) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal command state: %w", err)
	}

	key := fmt.Sprintf("command:%s", state.CommandID)
	return p.store.SetValue(ctx, key, string(stateJSON))
}

// monitorTimeout monitors command execution timeout
func (p *CommandProcessor) monitorTimeout(ctx context.Context, commandID string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		p.mu.Lock()
		if task, exists := p.tasks[commandID]; exists {
			task.Status = "timeout"
			task.Error = "command execution timed out"
			p.storeCommandState(context.Background(), &CommandState{
				CommandID: commandID,
				Status:    "timeout",
				Error:     "command execution timed out",
			})
			delete(p.tasks, commandID)
		}
		p.mu.Unlock()

		p.logger.Warn("Command timed out",
			zap.String("command_id", commandID),
			zap.Duration("timeout", timeout))
	}
}

// isTerminalStatus checks if a status is terminal (completed or failed)
func isTerminalStatus(status string) bool {
	switch status {
	case "success", "failed", "timeout", "error":
		return true
	default:
		return false
	}
}

// storeTask stores a task in memory and Valkey
func (p *CommandProcessor) storeTask(ctx context.Context, task *TaskMessage) error {
	// Store in memory
	p.mu.Lock()
	p.tasks[task.ID] = task
	p.mu.Unlock()

	// Store in Valkey
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	key := fmt.Sprintf("task:%s", task.ID)
	if err := p.store.SetValue(ctx, key, string(taskJSON)); err != nil {
		return fmt.Errorf("failed to store task in Valkey: %w", err)
	}

	return nil
}
