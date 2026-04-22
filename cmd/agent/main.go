package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/config"
	siriusbootstrap "github.com/SiriusScan/app-agent/internal/family/sirius/bootstrap"
	"github.com/SiriusScan/app-agent/internal/family/sirius/connector"
)

func main() {
	// Initialize LOG_LEVEL-aware logger
	logger := config.NewLogger()
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("Starting gRPC Hello World Agent")

	// Load configuration
	cfg := config.LoadAgentConfig()
	logger.Info("Configuration loaded",
		zap.String("server_address", cfg.ServerAddress),
		zap.String("agent_id", cfg.AgentID))

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
		}
	case sig := <-signalChan:
		logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		cancel() // Cancel the context to stop waiting for commands
	}

	logger.Info("Agent shutting down")
}
