package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/server"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Info("Starting gRPC Hello World Server")

	// Load configuration
	cfg := config.LoadServerConfig()
	logger.Info("Configuration loaded", zap.String("address", cfg.Address))

	// Create and start the server
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Handle termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("Server started successfully")

	// Wait for termination signal
	sig := <-signalChan
	logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))

	// Stop the server
	srv.Stop()
	logger.Info("Server shutdown complete")
}
