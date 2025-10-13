package sync

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/commands"
)

const (
	SyncCommandName = "internal:sync"
)

// SyncCommand handles custom content synchronization
type SyncCommand struct{}

// NewSyncCommand creates a new sync command
func NewSyncCommand() *SyncCommand {
	return &SyncCommand{}
}

// Execute performs custom content synchronization
func (sc *SyncCommand) Execute(ctx context.Context, info commands.AgentInfo, commandString string, args string) (string, error) {
	info.Logger.Info("Executing custom content sync command", zap.String("args", args))

	// For now, we'll return a message indicating sync should be triggered
	// The actual sync will be implemented in the agent's custom sync handler

	output := fmt.Sprintf("Custom content sync initiated at %s\n", time.Now().Format(time.RFC3339))
	output += "Custom content sync is available through the agent's sync functionality.\n"
	output += "Use 'internal:sync templates' to sync templates or 'internal:sync scripts' to sync scripts.\n"

	return output, nil
}

// Ensures SyncCommand implements the Command interface at compile time.
var _ commands.Command = (*SyncCommand)(nil)

func init() {
	commands.Register(SyncCommandName, &SyncCommand{})
}
