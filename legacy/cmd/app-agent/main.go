package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/grpc"
)

func main() {
	// Initialize logger
	logger := initLogger("info", "console")
	defer logger.Sync()

	logger.Info("Starting Agent Manager (Server)")

	// Load configuration
	cfg := config.LoadServerConfig()
	logger.Info("Configuration loaded",
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
		zap.String("grpc_addr", cfg.GRPCServerAddr),
	)

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	setupSignalHandler(cancel, logger)

	// Initialize gRPC server
	server := grpc.NewServer(cfg, logger)

	// Start the server
	if err := server.Start(ctx); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}

// initLogger initializes the logger based on the configuration
func initLogger(level, format string) *zap.Logger {
	var cfg zap.Config

	// Configure based on format
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level
	var logLevel zapcore.Level
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		logLevel = zapcore.InfoLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(logLevel)

	// Build logger
	logger, err := cfg.Build()
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	return logger
}

// setupSignalHandler configures signal handling for graceful shutdown
func setupSignalHandler(cancel context.CancelFunc, logger *zap.Logger) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()
}
