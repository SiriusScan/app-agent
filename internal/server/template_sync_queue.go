package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SiriusScan/go-api/sirius/queue"
	"go.uber.org/zap"
)

const (
	syncJobQueueName = "agent.template.sync.jobs"
)

// SyncJobMessage represents a sync job message from RabbitMQ.
//
// Action values:
//   - "sync_repository"   - re-clone a single GitHub repository and broadcast.
//   - "sync_all"          - re-sync every enabled repository and broadcast.
//   - "delete_repository" - remove repository files + KV entries.
//   - "notify_agents"     - tell connected agents to re-pull templates from
//     Valkey. Used by sirius-api after writing a custom template envelope so
//     the new template appears in agent inventories without requiring a full
//     repository sync. TemplateID is informational (currently logged only).
type SyncJobMessage struct {
	Action           string `json:"action"`
	RepositoryID     string `json:"repository_id,omitempty"`
	RepositoryURL    string `json:"repository_url,omitempty"`
	RepositoryBranch string `json:"repository_branch,omitempty"`
	TemplateID       string `json:"template_id,omitempty"`
	TriggeredBy      string `json:"triggered_by"`
	Timestamp        string `json:"timestamp"`
	JobID            string `json:"job_id"`
}

// TemplateSyncQueueProcessor processes template sync jobs from RabbitMQ
type TemplateSyncQueueProcessor struct {
	repositoryMgr *RepositoryManager
	logger        *zap.Logger
	ctx           context.Context
	cancelFunc    context.CancelFunc
}

// NewTemplateSyncQueueProcessor creates a new sync queue processor
func NewTemplateSyncQueueProcessor(
	repositoryMgr *RepositoryManager,
	logger *zap.Logger,
) *TemplateSyncQueueProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &TemplateSyncQueueProcessor{
		repositoryMgr: repositoryMgr,
		logger:        logger,
		ctx:           ctx,
		cancelFunc:    cancel,
	}
}

// StartListening begins consuming from the sync job queue
func (tsp *TemplateSyncQueueProcessor) StartListening() error {
	tsp.logger.Info("Starting template sync queue processor")

	// Use the queue package's Listen function
	go func() {
		processor := func(msg string) {
			tsp.logger.Info("Received sync job message", zap.String("message", msg))

			// Parse message
			var syncMsg SyncJobMessage
			if err := json.Unmarshal([]byte(msg), &syncMsg); err != nil {
				tsp.logger.Error("Failed to parse sync job message", zap.Error(err))
				return
			}

			// Create context with timeout
			processCtx, cancel := context.WithTimeout(tsp.ctx, 10*time.Minute)
			defer cancel()

			// Process based on action
			var err error
			switch syncMsg.Action {
			case "sync_repository":
				err = tsp.processSyncRepository(processCtx, &syncMsg)

			case "sync_all":
				err = tsp.processSyncAll(processCtx, &syncMsg)

			case "delete_repository":
				err = tsp.processDeleteRepository(processCtx, &syncMsg)

			case "notify_agents":
				err = tsp.processNotifyAgents(processCtx, &syncMsg)

			default:
				tsp.logger.Warn("Unknown sync action", zap.String("action", syncMsg.Action))
				return
			}

			if err != nil {
				tsp.logger.Error("Sync job failed",
					zap.String("job_id", syncMsg.JobID),
					zap.String("action", syncMsg.Action),
					zap.Error(err))
				return
			}

			tsp.logger.Info("Sync job completed successfully",
				zap.String("job_id", syncMsg.JobID),
				zap.String("action", syncMsg.Action))
		}

		queue.Listen(syncJobQueueName, processor)
		tsp.logger.Info("Template sync queue processor stopped")
	}()

	return nil
}

// Stop stops the queue processor
func (tsp *TemplateSyncQueueProcessor) Stop() {
	if tsp.cancelFunc != nil {
		tsp.cancelFunc()
	}
}

// processSyncRepository syncs a specific repository
func (tsp *TemplateSyncQueueProcessor) processSyncRepository(ctx context.Context, msg *SyncJobMessage) error {
	if msg.RepositoryID == "" {
		return fmt.Errorf("missing repository_id in sync message")
	}

	tsp.logger.Info("Processing repository sync",
		zap.String("repo_id", msg.RepositoryID),
		zap.String("job_id", msg.JobID))

	return tsp.repositoryMgr.SyncRepository(ctx, msg.RepositoryID)
}

// processSyncAll syncs all enabled repositories
func (tsp *TemplateSyncQueueProcessor) processSyncAll(ctx context.Context, msg *SyncJobMessage) error {
	tsp.logger.Info("Processing sync all repositories",
		zap.String("job_id", msg.JobID))

	return tsp.repositoryMgr.SyncAllRepositories(ctx)
}

// processNotifyAgents broadcasts a template-sync command to every connected
// agent. Used after a producer (typically sirius-api) has written a new
// custom template envelope to Valkey and needs agents to re-enumerate the
// template:meta:* namespace without doing a full repository sync.
func (tsp *TemplateSyncQueueProcessor) processNotifyAgents(ctx context.Context, msg *SyncJobMessage) error {
	tsp.logger.Info("Processing notify_agents request",
		zap.String("template_id", msg.TemplateID),
		zap.String("triggered_by", msg.TriggeredBy),
		zap.String("job_id", msg.JobID))
	tsp.repositoryMgr.NotifyAgents(ctx)
	return nil
}

// processDeleteRepository deletes a repository
func (tsp *TemplateSyncQueueProcessor) processDeleteRepository(ctx context.Context, msg *SyncJobMessage) error {
	if msg.RepositoryID == "" {
		return fmt.Errorf("missing repository_id in delete message")
	}

	tsp.logger.Info("Processing repository deletion",
		zap.String("repo_id", msg.RepositoryID),
		zap.String("job_id", msg.JobID))

	return tsp.repositoryMgr.DeleteRepository(ctx, msg.RepositoryID)
}
