package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/command"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/store"
)

type receiverConfig struct {
	commandID    string
	agentID      string
	pollInterval time.Duration
	timeout      time.Duration
	watchMode    bool
	outputFormat string
}

func main() {
	// Parse command line flags
	cfg := parseFlags()

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Info("Starting command receiver",
		zap.String("command_id", cfg.commandID),
		zap.String("agent_id", cfg.agentID),
		zap.Duration("poll_interval", cfg.pollInterval),
		zap.Duration("timeout", cfg.timeout))

	// Initialize store client
	storeClient, err := store.NewClient(config.LoadStoreConfig())
	if err != nil {
		logger.Fatal("Failed to initialize store client", zap.Error(err))
	}
	defer storeClient.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		cancel()
	}()

	// Start response retrieval
	if cfg.watchMode {
		watchResponses(ctx, storeClient, cfg, logger)
	} else {
		getResponse(ctx, storeClient, cfg, logger)
	}
}

func parseFlags() *receiverConfig {
	cfg := &receiverConfig{}

	flag.StringVar(&cfg.commandID, "command", "", "Command ID to retrieve response for")
	flag.StringVar(&cfg.agentID, "agent", "", "Agent ID to filter responses")
	flag.DurationVar(&cfg.pollInterval, "poll", 1*time.Second, "Polling interval for watch mode")
	flag.DurationVar(&cfg.timeout, "timeout", 5*time.Minute, "Timeout for response retrieval")
	flag.BoolVar(&cfg.watchMode, "watch", false, "Watch for responses continuously")
	flag.StringVar(&cfg.outputFormat, "format", "text", "Output format (text or json)")

	flag.Parse()

	// Validate required flags
	if cfg.commandID == "" && cfg.agentID == "" {
		fmt.Println("Error: either command ID or agent ID must be specified")
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func getResponse(ctx context.Context, client store.Client, cfg *receiverConfig, logger *zap.Logger) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	// Generate response key if we have both command and agent ID
	var key string
	if cfg.commandID != "" && cfg.agentID != "" {
		resp := &command.CommandResponse{
			CommandID: cfg.commandID,
			AgentID:   cfg.agentID,
		}
		key = resp.GenerateKey()
	}

	// Keep trying until timeout
	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				logger.Error("Timed out waiting for response")
				os.Exit(1)
			}
			return
		case <-ticker.C:
			if key != "" {
				// Get specific response
				data, err := client.Get(timeoutCtx, key)
				if err != nil {
					if err != store.ErrNotFound {
						logger.Error("Failed to get response", zap.Error(err))
					}
					continue
				}
				resp, err := command.FromJSON(data)
				if err != nil {
					logger.Error("Failed to parse response", zap.Error(err))
					continue
				}
				displayResponse(resp, cfg.outputFormat)
				return
			} else {
				// List responses for agent
				responses, err := client.List(timeoutCtx, fmt.Sprintf("cmd:response:%s:*", cfg.agentID))
				if err != nil {
					logger.Error("Failed to list responses", zap.Error(err))
					continue
				}
				for _, data := range responses {
					resp, err := command.FromJSON(data)
					if err != nil {
						logger.Error("Failed to parse response", zap.Error(err))
						continue
					}
					displayResponse(resp, cfg.outputFormat)
				}
				return
			}
		}
	}
}

func watchResponses(ctx context.Context, client store.Client, cfg *receiverConfig, logger *zap.Logger) {
	// Create patterns for watching responses
	patterns := []string{
		fmt.Sprintf("cmd:response:%s:*", cfg.agentID), // New format
		fmt.Sprintf("cmd:%s:*", cfg.agentID),          // Legacy format
		fmt.Sprintf("status:%s:*", cfg.agentID),       // Status format
	}

	logger.Info("Watching for responses", zap.Strings("patterns", patterns))

	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()

	seen := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Try each pattern
			for _, pattern := range patterns {
				responses, err := client.List(ctx, pattern)
				if err != nil {
					logger.Error("Failed to list responses",
						zap.Error(err),
						zap.String("pattern", pattern))
					continue
				}

				for key, data := range responses {
					if seen[key] {
						continue
					}

					resp, err := command.FromJSON(data)
					if err != nil {
						logger.Error("Failed to parse response",
							zap.Error(err),
							zap.String("key", key),
							zap.String("data", data))
						continue
					}

					if cfg.commandID != "" && resp.CommandID != cfg.commandID {
						continue
					}

					displayResponse(resp, cfg.outputFormat)
					seen[key] = true
				}
			}
		}
	}
}

func displayResponse(resp *command.CommandResponse, format string) {
	switch format {
	case "json":
		if data, err := resp.ToJSON(); err == nil {
			fmt.Println(data)
		}
	default:
		fmt.Printf("\n=== Command Response ===\n")
		fmt.Printf("Command ID: %s\n", resp.CommandID)
		fmt.Printf("Agent ID: %s\n", resp.AgentID)
		fmt.Printf("Command: %s\n", resp.Command)
		fmt.Printf("Status: %s\n", resp.Status)
		if resp.Output != "" {
			fmt.Printf("\nOutput:\n%s\n", resp.Output)
		}
		if resp.Error != "" {
			fmt.Printf("\nError:\n%s\n", resp.Error)
		}
		fmt.Printf("\nTiming:\n")
		fmt.Printf("Start Time: %s\n", resp.StartTime.Format(time.RFC3339))
		if resp.EndTime != nil {
			fmt.Printf("End Time: %s\n", resp.EndTime.Format(time.RFC3339))
		}
		fmt.Printf("Exit Code: %d\n", resp.ExitCode)
		fmt.Printf("=====================\n\n")
	}
}
