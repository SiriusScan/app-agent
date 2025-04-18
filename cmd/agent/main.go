package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/config"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Info("Starting gRPC Hello World Agent")

	// Load configuration
	cfg := config.LoadAgentConfig()
	logger.Info("Configuration loaded",
		zap.String("server_address", cfg.ServerAddress),
		zap.String("agent_id", cfg.AgentID))

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and connect the agent
	a := agent.NewAgent(cfg, logger)

	if err := a.Connect(ctx); err != nil {
		logger.Fatal("Failed to connect agent", zap.Error(err))
	}
	defer a.Close()

	// Handle termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Run in a goroutine so we can handle termination signals
	errChan := make(chan error, 1)
	go func() {
		logger.Info("Starting to listen for commands from server")
		errChan <- a.WaitForCommands(ctx)
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
