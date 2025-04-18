package status

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/sysinfo"
	"github.com/SiriusScan/go-api/sirius"
	"go.uber.org/zap"
)

// StatusCommand implements the internal status command.
type StatusCommand struct{}

// Ensures StatusCommand implements the Command interface at compile time.
var _ commands.Command = (*StatusCommand)(nil)

func init() {
	commands.Register("internal:status", &StatusCommand{})
}

// Execute gathers agent status, updates the backend API, and returns the status string.
func (c *StatusCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (output string, err error) {
	agentInfo.Logger.Info("Executing internal status command via command system")
	startTime := time.Now()

	// --- Start: Update Backend Host Record (Side Effect) ---
	// Run in background goroutine like before
	go func() {
		apiCtx := context.Background()
		hostname, _ := os.Hostname()
		hostData := sirius.Host{
			HID:       agentInfo.Config.HostID,
			OS:        runtime.GOOS,
			OSVersion: sysinfo.GetOSVersion(),
			IP:        sysinfo.GetPrimaryIP(),
			Hostname:  hostname,
		}

		if err := agentInfo.APIClient.UpdateHostRecord(apiCtx, agentInfo.Config.ApiBaseURL, hostData); err != nil {
			agentInfo.Logger.Error("Failed to update host record via API (from status command)",
				zap.String("host_id", agentInfo.Config.HostID),
				zap.Error(err))
		} else {
			agentInfo.Logger.Info("Successfully updated host record via API (from status command)", zap.String("host_id", agentInfo.Config.HostID))
		}
	}()
	// --- End: Update Backend Host Record ---

	// --- Start: Gather Local Status for Reporting Back ---
	uptime := time.Since(agentInfo.StartTime)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memAllocMB := float64(memStats.Alloc) / 1024 / 1024
	numGoroutines := runtime.NumGoroutine()

	var statusBuilder strings.Builder
	fmt.Fprintf(&statusBuilder, "Agent ID: %s\n", agentInfo.Config.AgentID)
	fmt.Fprintf(&statusBuilder, "Host ID: %s\n", agentInfo.Config.HostID)
	fmt.Fprintf(&statusBuilder, "Uptime: %s\n", uptime.Round(time.Second).String())
	fmt.Fprintf(&statusBuilder, "Server Target: %s\n", agentInfo.Config.ServerAddress)
	fmt.Fprintf(&statusBuilder, "API Target: %s\n", agentInfo.Config.ApiBaseURL)
	// Note: Cannot directly access agent.scriptingEnabled or agent.powerShellPath from AgentInfo
	// Need to add these to AgentInfo if we want them in the status command output.
	// For now, just report config value:
	fmt.Fprintf(&statusBuilder, "Scripting Config Enabled: %t\n", agentInfo.Config.EnableScripting)
	if agentInfo.Config.PowerShellPath != "" {
		fmt.Fprintf(&statusBuilder, "PowerShell Config Path: %s\n", agentInfo.Config.PowerShellPath)
	}
	fmt.Fprintf(&statusBuilder, "Go Version: %s\n", runtime.Version())
	fmt.Fprintf(&statusBuilder, "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&statusBuilder, "OS Version: %s\n", sysinfo.GetOSVersion())
	fmt.Fprintf(&statusBuilder, "Primary IP: %s\n", sysinfo.GetPrimaryIP())
	fmt.Fprintf(&statusBuilder, "Memory Allocated: %.2f MB\n", memAllocMB)
	fmt.Fprintf(&statusBuilder, "Goroutines: %d\n", numGoroutines)

	output = statusBuilder.String()
	executionTime := time.Since(startTime).Milliseconds()
	agentInfo.Logger.Info("Internal status command processed (command system)", zap.Int64("execution_time_ms", executionTime))
	// --- End: Gather Local Status ---

	// Return the gathered status output, no error expected from this command itself
	return output, nil
}
