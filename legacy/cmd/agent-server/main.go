package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/command"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/grpc"
)

const (
	defaultQueueName = "agent-commands"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting Agent Manager Server")

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg := config.LoadServerConfig()
	logger.Info("Configuration loaded",
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
		zap.String("grpc_addr", cfg.GRPCServerAddr),
	)

	// Initialize the Valkey store using the Sirius API
	kvStore, err := store.NewValkeyStore()
	if err != nil {
		logger.Fatal("Failed to connect to Valkey", zap.Error(err))
	}
	defer func() {
		if err := kvStore.Close(); err != nil {
			logger.Error("Error closing Valkey connection", zap.Error(err))
		}
	}()
	logger.Info("Connected to Valkey")

	// Create the gRPC server
	grpcServer := grpc.NewServer(cfg, logger)

	// Get the service instance to use as the command sender
	service := grpcServer.GetAgentService()

	// Create and start the command processor
	cmdProcessor := command.NewProcessor(logger, kvStore)
	cmdProcessor.Start(defaultQueueName)
	logger.Info("Command processor started", zap.String("queue", defaultQueueName))

	// Create and start the command distributor
	cmdDistributor := command.NewDistributor(logger, kvStore, service)
	cmdDistributor.Start()
	logger.Info("Command distributor started")

	// Start the gRPC server in a goroutine
	go func() {
		if err := grpcServer.Start(ctx); err != nil {
			logger.Error("Failed to start gRPC server", zap.Error(err))

			// Check if it's the "address in use" error
			if strings.Contains(err.Error(), "address already in use") {
				logger.Warn("Port is already in use. Either another instance is running or the port wasn't properly released.")
				logger.Info("You may need to manually kill the process using this port or select a different port.")
				logger.Info("On Linux/Mac, you can find the process using: lsof -i :50051")
				logger.Info("On Windows, you can find the process using: netstat -ano | findstr :50051")
			}

			// Cancel the context to trigger shutdown
			cancel()
		}
	}()
	logger.Info("gRPC server started")

	// Wait for interrupt signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	logger.Info("Received shutdown signal, gracefully stopping services")

	// Stop the gRPC server
	grpcServer.Stop()
	logger.Info("gRPC server stopped")

	// Stop the command distributor
	cmdDistributor.Stop()
	logger.Info("Command distributor stopped")

	// Stop the command processor
	cmdProcessor.Stop()
	logger.Info("Command processor stopped")

	logger.Info("Agent Manager Server shutdown complete")
}
