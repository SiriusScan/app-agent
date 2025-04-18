package agent

import (
	scanpb "github.com/SiriusScan/app-agent/proto/scan/v1"
)

// CommandQueueServer defines the interface for queueing commands to agents
type CommandQueueServer interface {
	QueueCommand(agentID string, cmd *scanpb.Command) error
}
