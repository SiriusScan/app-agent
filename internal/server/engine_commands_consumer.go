package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/SiriusScan/go-api/sirius/queue"
	"go.uber.org/zap"
)

const (
	engineCommandsQueueName = "engine.commands"
)

// EngineCommandMessage is the legacy envelope still used by some
// producers (notably the pre-PR3 sirius-api delete path and any
// third-party integrations that publish directly to engine.commands).
//
// Two commands matter for template-sync purposes:
//   - "internal:template upload"  - new custom template was written.
//   - "internal:template delete"  - custom template was removed.
//
// In both cases we just need to nudge connected agents to re-pull the
// template:meta:* namespace; the repository-level sync is unaffected.
// Anything else is acked and ignored so we don't block other producers
// that might land on this queue.
type EngineCommandMessage struct {
	Command    string `json:"command"`
	TemplateID string `json:"template_id,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

// isTemplateNotifyCommand reports whether the legacy command string
// should trigger an agent re-pull. Kept as a small pure helper so the
// classification can be unit-tested without spinning up RabbitMQ.
func isTemplateNotifyCommand(cmd string) bool {
	switch cmd {
	case "internal:template upload", "internal:template delete":
		return true
	default:
		return false
	}
}

// EngineCommandQueueProcessor is the defense-in-depth consumer that
// catches any producer still publishing to engine.commands. PR 3 made
// the canonical path agent.template.sync.jobs / notify_agents, so this
// consumer is strictly additive: if PR 3's path keeps working this
// consumer logs but never has anything to do.
type EngineCommandQueueProcessor struct {
	repositoryMgr *RepositoryManager
	logger        *zap.Logger
	ctx           context.Context
	cancelFunc    context.CancelFunc
}

// NewEngineCommandQueueProcessor wires the consumer to the repository
// manager (notify-agents path).
func NewEngineCommandQueueProcessor(
	repositoryMgr *RepositoryManager,
	logger *zap.Logger,
) *EngineCommandQueueProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &EngineCommandQueueProcessor{
		repositoryMgr: repositoryMgr,
		logger:        logger,
		ctx:           ctx,
		cancelFunc:    cancel,
	}
}

// StartListening spawns the goroutine that pumps messages off
// engine.commands and routes recognized template commands into the
// notify-agents helper. Mirrors the shape of TemplateSyncQueueProcessor.
func (ecp *EngineCommandQueueProcessor) StartListening() error {
	ecp.logger.Info("Starting engine.commands queue processor (defense-in-depth)")

	go func() {
		processor := func(msg string) {
			ecp.logger.Debug("Received engine.commands message", zap.String("message", msg))

			var cmd EngineCommandMessage
			if err := json.Unmarshal([]byte(msg), &cmd); err != nil {
				ecp.logger.Warn("Failed to parse engine.commands message",
					zap.Error(err),
					zap.String("raw", msg))
				return
			}

			processCtx, cancel := context.WithTimeout(ecp.ctx, 2*time.Minute)
			defer cancel()

			if isTemplateNotifyCommand(cmd.Command) {
				ecp.logger.Info("Routing engine.commands template event to notify-agents",
					zap.String("command", cmd.Command),
					zap.String("template_id", cmd.TemplateID))
				ecp.repositoryMgr.NotifyAgents(processCtx)
			} else {
				// Other producers (engine health pings, scan dispatch,
				// etc.) legitimately share this queue. Ack-and-ignore.
				ecp.logger.Debug("Unhandled engine.commands command",
					zap.String("command", cmd.Command))
			}
		}

		queue.Listen(engineCommandsQueueName, processor)
		ecp.logger.Info("engine.commands queue processor stopped")
	}()

	return nil
}

// Stop cancels the consumer's context. The underlying queue.Listen does
// not currently honour cancellation (matches TemplateSyncQueueProcessor),
// but we keep the same lifecycle shape so future plumbing is symmetric.
func (ecp *EngineCommandQueueProcessor) Stop() {
	if ecp.cancelFunc != nil {
		ecp.cancelFunc()
	}
}
