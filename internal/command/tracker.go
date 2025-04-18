package command

import (
	"errors"
	"sync"
	"time"
)

// Common errors
var (
	ErrCommandNotFound = errors.New("command not found")
)

// TrackedCommand represents a command being tracked through its lifecycle
type TrackedCommand struct {
	ID            string         // Unique identifier for the command
	AgentID       string         // ID of the agent this command is for
	Command       string         // The actual command string
	Status        ResponseStatus // Current status of the command
	CreatedAt     time.Time      // When the command was created
	StartedAt     time.Time      // When the command started executing
	CompletedAt   time.Time      // When the command completed/failed
	ExecutionTime int64          // Time taken to execute in milliseconds
	Error         error          // Error if command failed
}

// Tracker manages command tracking and status updates
type Tracker struct {
	mu       sync.RWMutex
	commands map[string]*TrackedCommand
	// Maximum age for completed/failed commands before cleanup
	maxAge time.Duration
}

// NewTracker creates a new command tracker
func NewTracker(maxAge time.Duration) *Tracker {
	t := &Tracker{
		commands: make(map[string]*TrackedCommand),
		maxAge:   maxAge,
	}

	// Start cleanup routine
	go t.cleanupRoutine()

	return t
}

// Track starts tracking a new command
func (t *Tracker) Track(id, agentID, command string) *TrackedCommand {
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd := &TrackedCommand{
		ID:        id,
		AgentID:   agentID,
		Command:   command,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}

	t.commands[id] = cmd
	return cmd
}

// UpdateStatus updates the status of a tracked command
func (t *Tracker) UpdateStatus(id string, status ResponseStatus) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd, exists := t.commands[id]
	if !exists {
		return ErrCommandNotFound
	}

	cmd.Status = status

	switch status {
	case StatusRunning:
		cmd.StartedAt = time.Now()
	case StatusCompleted, StatusFailed:
		cmd.CompletedAt = time.Now()
		if !cmd.StartedAt.IsZero() {
			cmd.ExecutionTime = time.Since(cmd.StartedAt).Milliseconds()
		}
	}

	return nil
}

// SetError sets an error for a tracked command
func (t *Tracker) SetError(id string, err error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd, exists := t.commands[id]
	if !exists {
		return ErrCommandNotFound
	}

	cmd.Error = err
	cmd.Status = StatusFailed
	cmd.CompletedAt = time.Now()
	if !cmd.StartedAt.IsZero() {
		cmd.ExecutionTime = time.Since(cmd.StartedAt).Milliseconds()
	}

	return nil
}

// Get retrieves a tracked command by ID
func (t *Tracker) Get(id string) (*TrackedCommand, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cmd, exists := t.commands[id]
	if !exists {
		return nil, ErrCommandNotFound
	}

	return cmd, nil
}

// ListByStatus returns all commands with the specified status
func (t *Tracker) ListByStatus(status ResponseStatus) []*TrackedCommand {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*TrackedCommand
	for _, cmd := range t.commands {
		if cmd.Status == status {
			result = append(result, cmd)
		}
	}

	return result
}

// ListByAgent returns all commands for a specific agent
func (t *Tracker) ListByAgent(agentID string) []*TrackedCommand {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*TrackedCommand
	for _, cmd := range t.commands {
		if cmd.AgentID == agentID {
			result = append(result, cmd)
		}
	}

	return result
}

// cleanupRoutine periodically removes old completed/failed commands
func (t *Tracker) cleanupRoutine() {
	ticker := time.NewTicker(t.maxAge / 2)
	defer ticker.Stop()

	for range ticker.C {
		t.cleanup()
	}
}

// cleanup removes commands that have been completed/failed for longer than maxAge
func (t *Tracker) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for id, cmd := range t.commands {
		if (cmd.Status == StatusCompleted || cmd.Status == StatusFailed) &&
			!cmd.CompletedAt.IsZero() &&
			now.Sub(cmd.CompletedAt) > t.maxAge {
			delete(t.commands, id)
		}
	}
}
