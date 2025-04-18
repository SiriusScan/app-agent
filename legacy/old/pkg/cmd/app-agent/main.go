package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/go-api/sirius/store"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.LoadFromEnv("")
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create context that will be canceled on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Create store factory function
	storeFactory := func() (store.KVStore, error) {
		return store.NewValkeyStore()
	}

	// Initialize and start agent manager
	mgr, err := agent.NewManager(cfg, logger, storeFactory)
	if err != nil {
		slog.Error("failed to create agent manager", "error", err)
		os.Exit(1)
	}

	slog.Info("starting app-agent")
	if err := mgr.Start(ctx); err != nil {
		slog.Error("failed to start agent manager", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := mgr.Stop(shutdownCtx); err != nil {
		slog.Error("error during shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("app-agent shutdown complete")
}
