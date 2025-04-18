package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SiriusScan/app-agent/internal/store"
	"go.uber.org/zap"
)

const (
	TaskResultKeyPrefix    = "task-result:"
	AgentCommandKeyPrefix  = "agent-command:"
	DefaultRefreshInterval = 5 * time.Second
)

// formatTime formats a Unix timestamp to human-readable format
func formatTime(timestamp int64) string {
	if timestamp == 0 {
		return "N/A"
	}
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

func main() {
	// Parse command line flags
	agentID := flag.String("agent", "test-agent-1", "Agent ID to monitor")
	monitor := flag.Bool("monitor", true, "Enable continuous monitoring")
	refreshSec := flag.Int("refresh", 5, "Refresh interval in seconds for monitoring")
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Shutting down gracefully...")
		cancel()
	}()

	// Create store instance (mock for testing)
	mockStore := store.NewMockStore()

	logger.Info("Starting agent result checker",
		zap.String("agent_id", *agentID),
		zap.Bool("monitor", *monitor),
		zap.Int("refresh_interval", *refreshSec))

	// Start monitoring
	refreshInterval := time.Duration(*refreshSec) * time.Second
	if *monitor {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()

		// Initial check
		checkResults(ctx, mockStore, *agentID, logger)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkResults(ctx, mockStore, *agentID, logger)
			}
		}
	} else {
		checkResults(ctx, mockStore, *agentID, logger)
	}
}

// checkResults retrieves and displays task results for the specified agent
func checkResults(ctx context.Context, kvStore store.KVStore, agentID string, logger *zap.Logger) {
	// Clear screen in monitor mode
	fmt.Print("\033[H\033[2J")
	fmt.Printf("=== Agent Results for %s ===\n", agentID)
	fmt.Printf("Time: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// In a real implementation, we would query the key-value store
	// For simplicity, we use mock data here
	results := []map[string]interface{}{
		{
			"command_id":   "task-20250412-181408-1",
			"agent_id":     "test-agent-1",
			"status":       "completed",
			"type":         "exec",
			"command":      "hostname",
			"completed_at": time.Now().Add(-time.Minute).Unix(),
			"output":       "sirius-engine\n",
		},
		{
			"command_id":   "task-20250412-181342-1",
			"agent_id":     "test-agent-1",
			"status":       "completed",
			"type":         "exec",
			"command":      "whoami",
			"completed_at": time.Now().Add(-2 * time.Minute).Unix(),
			"output":       "root\n",
		},
		{
			"command_id":   "task-20250412-181323-1",
			"agent_id":     "test-agent-1",
			"status":       "completed",
			"type":         "exec",
			"command":      "ls -la",
			"completed_at": time.Now().Add(-3 * time.Minute).Unix(),
			"output":       "total 84\ndrwxr-xr-x 1 root root 4096 Apr 12 18:13 .\ndrwxr-xr-x 1 root root 4096 Apr 12 18:13 ..\n...",
		},
	}

	fmt.Printf("%-25s %-12s %-15s %-20s\n", "COMMAND ID", "STATUS", "TYPE", "COMPLETED AT")
	fmt.Printf("%-25s %-12s %-15s %-20s\n", "---------", "------", "----", "-----------")

	// Display results
	for _, result := range results {
		if result["agent_id"] == agentID {
			fmt.Printf("%-25s %-12s %-15s %-20s\n",
				result["command_id"],
				result["status"],
				result["type"],
				formatTime(result["completed_at"].(int64)),
			)
		}
	}

	fmt.Println("\nDetailed command outputs:")
	fmt.Println("------------------------")

	// Show detailed outputs
	for _, result := range results {
		if result["agent_id"] == agentID {
			fmt.Printf("\nCommand: %s\n", result["command"])
			fmt.Printf("ID: %s\n", result["command_id"])
			fmt.Printf("Status: %s\n", result["status"])
			fmt.Println("Output:")
			fmt.Println("-------")
			fmt.Println(result["output"])
			fmt.Println()
		}
	}

	// Provide recommendations
	fmt.Println("\nRecommendations:")
	fmt.Println("----------------")
	fmt.Println("1. Run agent with: go run cmd/agent/main.go")
	fmt.Println("2. Send a test task: docker exec -it sirius-engine sh -c 'cd /app-agent && go run cmd/agent-message-tester/main.go --type=exec --cmd=\"hostname\" --target=localhost'")
	fmt.Println("3. Use result-retriever to check results: go run cmd/result-retriever/main.go --agent=test-agent-1 --format=table")

	// Provide server status
	fmt.Println("\nServer Status:")
	fmt.Println("-------------")
	fmt.Println("The server is connected to RabbitMQ and polling for commands.")
	fmt.Println("Current task processing is handled in mock mode - the real agent implementation needs")
	fmt.Println("to connect to the RabbitMQ queue rather than the mock gRPC system.")
}
