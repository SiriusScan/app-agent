package agent

import (
	"context"

	"github.com/SiriusScan/app-agent/internal/commands"
	templateagent "github.com/SiriusScan/app-agent/internal/template/agent"
)

type templateSyncAdapter struct {
	manager *templateagent.AgentSyncManager
}

func (a *templateSyncAdapter) SyncFromServer(ctx context.Context) error {
	return a.manager.SyncFromServer(ctx)
}

func (a *templateSyncAdapter) GetStatus(ctx context.Context) (*commands.TemplateSyncStatus, error) {
	_ = ctx

	status, err := a.manager.GetCacheStatus()
	if err != nil {
		return nil, err
	}

	return &commands.TemplateSyncStatus{
		LastSync:          status.LastSync,
		TotalTemplates:    status.Statistics.TotalTemplates,
		StandardTemplates: status.Statistics.StandardTemplates,
		CustomTemplates:   status.Statistics.CustomTemplates,
		CacheSize:         status.Statistics.CacheSize,
	}, nil
}
