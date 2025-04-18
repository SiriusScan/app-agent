package commands

import (
	"context"
	"time"

	"github.com/SiriusScan/app-agent/internal/apiclient"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/go-api/sirius"
	"go.uber.org/zap"
)

// AgentInfo provides necessary dependencies from the main agent to commands
// without passing the entire agent struct, promoting loose coupling.
type AgentInfo struct {
	Logger           *zap.Logger
	Config           *config.AgentConfig
	APIClient        APIClient // Interface for needed API calls
	StartTime        time.Time
	ScriptingEnabled bool   // Is PowerShell/scripting available and enabled?
	PowerShellPath   string // Path to the PowerShell executable
	// Add other shared resources if needed
}

// APIClient defines the interface for API functions needed by internal commands.
// This allows mocking the API client for command testing.
type APIClient interface {
	UpdateHostRecord(ctx context.Context, apiBaseURL string, hostData sirius.Host) error
}

// --- Default API Client Implementation ---

// Ensure apiClientAdapter implements APIClient at compile time.
var _ APIClient = (*apiClientAdapter)(nil)

// apiClientAdapter provides a concrete implementation of the APIClient interface
// using the actual functions from the internal/apiclient package.
type apiClientAdapter struct{}

// NewAPIClientAdapter creates a new adapter.
func NewAPIClientAdapter() APIClient {
	return &apiClientAdapter{}
}

// UpdateHostRecord calls the underlying apiclient function.
func (a *apiClientAdapter) UpdateHostRecord(ctx context.Context, apiBaseURL string, hostData sirius.Host) error {
	// Call the actual function from the apiclient package
	return apiclient.UpdateHostRecord(ctx, apiBaseURL, hostData)
}

// --- End Default API Client Implementation ---

// Command defines the interface for all internal agent commands.
type Command interface {
	// Execute runs the command logic.
	// commandString is the full original command string received by the agent.
	// args is the portion of the commandString after the prefix (trimmed).
	Execute(ctx context.Context, agentInfo AgentInfo, commandString string, args string) (output string, err error)

	// Help returns a brief help string for the command (optional but good practice).
	// Help() string
}
