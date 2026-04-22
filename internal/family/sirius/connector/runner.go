package connector

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/debugtrace"
	"github.com/SiriusScan/app-agent/internal/repository"
	templateagent "github.com/SiriusScan/app-agent/internal/template/agent"
	"go.uber.org/zap"
)

type Runner struct {
	logger          *zap.Logger
	cfg             *config.AgentConfig
	agent           *agent.Agent
	repoIntegration *repository.RepositoryIntegration
	syncManager     *templateagent.AgentSyncManager
	baseCtx         context.Context
	cancel          context.CancelFunc
	errCh           chan error

	mu       sync.RWMutex
	lastErr  error
	started  bool
	starting bool
}

func NewRunner(cfg *config.AgentConfig, logger *zap.Logger) *Runner {
	return &Runner{
		logger: logger,
		cfg:    cfg,
		errCh:  make(chan error, 1),
	}
}

func (r *Runner) Start(ctx context.Context) error {
	// #region agent log
	debugtrace.Log("pre-fix", "H2,H4", "internal/family/sirius/connector/runner.go:42", "connector_runner_start", map[string]interface{}{
		"agentId":       r.cfg.AgentID,
		"serverAddress": r.cfg.ServerAddress,
		"tokenFilePath": r.cfg.TokenFilePath,
		"hasToken":      r.cfg.AuthToken != "",
	})
	// #endregion
	r.mu.Lock()
	if r.started || r.starting {
		r.mu.Unlock()
		return nil
	}
	r.starting = true
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.starting = false
		r.mu.Unlock()
	}()

	// Use a long-lived background context for the Sirius HelloService stream so
	// it is not torn down when the caller's bootstrap context is canceled.
	runCtx, cancel := context.WithCancel(context.Background())
	a := agent.NewAgent(r.cfg, r.logger)
	if err := a.Connect(runCtx); err != nil {
		cancel()
		return err
	}

	repoIntegration := repository.NewRepositoryIntegration(r.logger)
	if err := repoIntegration.Initialize(runCtx, r.cfg.AgentID, r.cfg.ServerAddress); err != nil {
		r.logger.Warn("Failed to initialize repository integration", zap.Error(err))
	} else {
		if syncManager := repoIntegration.GetSyncManager(); syncManager != nil {
			syncManager.SetGRPCStream(a.GetStream())
			syncManager.SetStreamSendFunc(a.StreamSend)
			a.SetSyncManager(syncManager)
			r.syncManager = syncManager
			r.scheduleInitialSync(runCtx, syncManager)
		}
	}

	r.mu.Lock()
	r.baseCtx = ctx
	r.cancel = cancel
	r.agent = a
	r.repoIntegration = repoIntegration
	r.started = true
	r.lastErr = nil
	r.mu.Unlock()

	go func() {
		err := a.WaitForCommands(runCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			r.mu.Lock()
			r.lastErr = err
			r.mu.Unlock()
		}
		// #region agent log
		debugtrace.Log("pre-fix", "H2,H4,H5", "internal/family/sirius/connector/runner.go:92", "connector_wait_for_commands_exit", map[string]interface{}{
			"agentId": r.cfg.AgentID,
			"error": func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}(),
		})
		// #endregion
		select {
		case r.errCh <- err:
		default:
		}
	}()

	return nil
}

func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return nil
	}
	if r.cancel != nil {
		r.cancel()
	}

	r.started = false
	if r.agent != nil {
		return r.agent.Close()
	}
	return nil
}

func (r *Runner) Reconnect(ctx context.Context) error {
	if err := r.Stop(); err != nil {
		return err
	}
	return r.Start(ctx)
}

func (r *Runner) Errors() <-chan error {
	return r.errCh
}

func (r *Runner) SyncNow(ctx context.Context) error {
	r.mu.RLock()
	syncManager := r.syncManager
	r.mu.RUnlock()
	if syncManager == nil {
		return fmt.Errorf("sync manager not initialized")
	}
	return syncManager.SyncFromServer(ctx)
}

func (r *Runner) Status() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := map[string]interface{}{
		"agent_id":         r.cfg.AgentID,
		"server_address":   r.cfg.ServerAddress,
		"started":          r.started,
		"has_sync_manager": r.syncManager != nil,
	}
	if r.lastErr != nil {
		status["last_error"] = r.lastErr.Error()
	}
	return status
}

func (r *Runner) scheduleInitialSync(ctx context.Context, syncManager *templateagent.AgentSyncManager) {
	go func() {
		time.Sleep(3 * time.Second)
		syncCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		r.logger.Info("Requesting initial template sync from server")
		if err := syncManager.SyncFromServer(syncCtx); err != nil {
			r.logger.Warn("Initial template sync failed (will retry on next sync)", zap.Error(err))
			return
		}
		r.logger.Info("Initial template sync request sent successfully")
	}()
}
