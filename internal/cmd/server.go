package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/apiclient"
	"github.com/SiriusScan/app-agent/internal/config"
	siriusbootstrap "github.com/SiriusScan/app-agent/internal/family/sirius/bootstrap"
	"github.com/SiriusScan/app-agent/internal/family/sirius/connector"
)

// NewServerCommand creates the server command for agent mode.
func NewServerCommand() *cobra.Command {
	var serverAddress string
	var agentID string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start agent in server mode (connects to Sirius server)",
		Long: `Starts the agent in server mode, connecting to the Sirius gRPC server via
bidirectional stream. The agent will:
  1. Connect to the server at the specified address
  2. Send periodic heartbeats
  3. Listen for commands from the server
  4. Execute commands (including template scans)
  5. Report results back to the server

The agent will continue running until interrupted (Ctrl+C).

Examples:
  sirius-agent server
  sirius-agent server --address localhost:50051
  sirius-agent server --agent-id my-agent-123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize LOG_LEVEL-aware logger
			logger := config.NewLogger()
			defer func() {
				_ = logger.Sync()
			}()

			logger.Info("Starting Sirius Agent in server mode")

			// Load configuration (will use env vars or defaults)
			cfg := config.LoadAgentConfig()

			// Override with command-line flags if provided
			if serverAddress != "" {
				cfg.ServerAddress = serverAddress
			}
			if agentID != "" {
				cfg.AgentID = agentID
			}

			logger.Info("Configuration loaded",
				zap.String("server_address", cfg.ServerAddress),
				zap.String("agent_id", cfg.AgentID))
			if !apiclient.ServiceAPIKeyConfigured() {
				logger.Warn("Service API key is not configured; template-scan results cannot be submitted to Sirius API",
					zap.String("api_base_url", cfg.ApiBaseURL),
					zap.String("required_env_vars", strings.Join(apiclient.ServiceAPIKeyEnvNames(), ", ")))
			}

			siriusbootstrap.LoadCompatibilityRuntime()

			// Create a context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			runner := connector.NewRunner(cfg, logger)
			if err := runner.Start(ctx); err != nil {
				logger.Fatal("Failed to connect agent", zap.Error(err))
			}
			defer runner.Stop()

			// Handle termination signals
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

			// Run in a goroutine so we can handle termination signals
			errChan := make(chan error, 1)
			go func() {
				logger.Info("Starting to listen for commands from server")
				errChan <- <-runner.Errors()
			}()

			// Wait for either an error or a termination signal
			select {
			case err := <-errChan:
				if err != nil {
					logger.Error("Error waiting for commands", zap.Error(err))
					return err
				}
			case sig := <-signalChan:
				logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
				cancel() // Cancel the context to stop waiting for commands
			}

			logger.Info("Agent shutting down")
			return nil
		},
	}

	cmd.Flags().StringVarP(&serverAddress, "address", "a", "", "Server address (default: from env SERVER_ADDRESS or localhost:50051)")
	cmd.Flags().StringVar(&agentID, "agent-id", "", "Agent ID (default: from env AGENT_ID or hostname)")

	return cmd
}
