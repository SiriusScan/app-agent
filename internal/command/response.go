package command

import (
	"encoding/json"
	"fmt"
	"time"
)

// ResponseStatus represents the status of a command execution
type ResponseStatus string

const (
	// StatusPending indicates the command is waiting to be executed
	StatusPending ResponseStatus = "pending"
	// StatusRunning indicates the command is currently executing
	StatusRunning ResponseStatus = "running"
	// StatusCompleted indicates the command completed successfully
	StatusCompleted ResponseStatus = "completed"
	// StatusFailed indicates the command failed to execute
	StatusFailed ResponseStatus = "failed"
)

// CommandResponse represents the response from an agent executing a command
type CommandResponse struct {
	// CommandID is the unique identifier for the command
	CommandID string `json:"command_id"`
	// AgentID is the identifier of the agent that executed the command
	AgentID string `json:"agent_id"`
	// Command is the command that was executed
	Command string `json:"command"`
	// Status indicates the current status of the command execution
	Status ResponseStatus `json:"status"`
	// Output contains the command's output (stdout/stderr)
	Output string `json:"output,omitempty"`
	// Error contains any error message if the command failed
	Error string `json:"error,omitempty"`
	// StartTime is when the command started executing
	StartTime time.Time `json:"start_time"`
	// EndTime is when the command finished executing
	EndTime *time.Time `json:"end_time,omitempty"`
	// ExitCode is the command's exit code
	ExitCode int `json:"exit_code"`
}

// Validate checks if the response has all required fields
func (r *CommandResponse) Validate() error {
	if r.CommandID == "" {
		return fmt.Errorf("command_id is required")
	}
	if r.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if r.Command == "" {
		return fmt.Errorf("command is required")
	}
	if r.Status == "" {
		return fmt.Errorf("status is required")
	}
	return nil
}

// NewCommandResponse creates a new CommandResponse with the given command and agent IDs
func NewCommandResponse(commandID, agentID, command string) *CommandResponse {
	return &CommandResponse{
		CommandID: commandID,
		AgentID:   agentID,
		Command:   command,
		Status:    StatusPending,
		StartTime: time.Now(),
	}
}

// SetRunning marks the command as running
func (r *CommandResponse) SetRunning() {
	r.Status = StatusRunning
}

// SetCompleted marks the command as completed with the given output and exit code
func (r *CommandResponse) SetCompleted(output string, exitCode int) {
	r.Status = StatusCompleted
	r.Output = output
	r.ExitCode = exitCode
	now := time.Now()
	r.EndTime = &now
}

// SetFailed marks the command as failed with the given error
func (r *CommandResponse) SetFailed(err error) {
	r.Status = StatusFailed
	r.Error = err.Error()
	now := time.Now()
	r.EndTime = &now
	r.ExitCode = 1
}

// GenerateKey generates a Valkey store key for this response
func (r *CommandResponse) GenerateKey() string {
	return fmt.Sprintf("cmd:response:%s:%s", r.AgentID, r.CommandID)
}

// ToJSON converts the response to a JSON string
func (r *CommandResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal command response: %w", err)
	}
	return string(data), nil
}

// FromJSON creates a CommandResponse from a JSON string
func FromJSON(data string) (*CommandResponse, error) {
	var response CommandResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command response: %w", err)
	}
	return &response, nil
}
