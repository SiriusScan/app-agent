package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/command"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/grpc"
	internalStore "github.com/SiriusScan/app-agent/internal/store"
	"github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Create a custom encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create a console core that writes to stdout
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	// Create the logger
	logger := zap.New(core)
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("❌ Failed to load configuration",
			zap.Error(err),
			zap.String("config_type", "app"),
			zap.String("startup_phase", "configuration"))
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("🛑 Received shutdown signal",
			zap.String("signal", sig.String()),
			zap.String("phase", "shutdown"),
			zap.Time("timestamp", time.Now()))
		cancel()
	}()

	logger.Info("🚀 Starting SiriusScan Agent Manager",
		zap.String("version", cfg.Version),
		zap.String("environment", cfg.Environment),
		zap.String("grpc_server", cfg.GRPCServerAddr),
		zap.String("metrics_addr", cfg.MetricsAddr))

	// Initialize store using the SDK
	goAPIStore, err := store.NewValkeyStore()
	if err != nil {
		logger.Fatal("❌ Failed to initialize store",
			zap.Error(err),
			zap.String("component", "valkey"),
			zap.String("startup_phase", "store_initialization"))
	}
	defer goAPIStore.Close()

	// Create an adapter to convert between the Go API's KVStore and our internal KVStore
	kvStore := internalStore.NewStoreAdapter(goAPIStore)

	// Create the result storage
	resultStorage := internalStore.NewResultStorage(kvStore, logger)

	// Create message processor
	processor := agent.NewMessageProcessor(logger, goAPIStore, cfg)

	// Create gRPC server
	grpcServer, err := grpc.NewServer(logger, cfg, kvStore)
	if err != nil {
		logger.Fatal("❌ Failed to create gRPC server", zap.Error(err))
	}

	// Create command distribution service
	commandService := command.NewService(
		logger,
		cfg,
		kvStore,
		resultStorage,
		grpcServer,
	)

	// Start gRPC server
	if err := grpcServer.Start(ctx); err != nil {
		logger.Fatal("❌ Failed to start gRPC server", zap.Error(err))
	}

	// Start command distribution service
	if err := commandService.Start(ctx); err != nil {
		logger.Fatal("❌ Failed to start command distribution service",
			zap.Error(err),
			zap.String("component", "command_distributor"),
			zap.String("startup_phase", "distributor_initialization"))
	}

	logger.Info("✅ Command distribution service started successfully",
		zap.String("status", "running"),
		zap.Time("startup_time", time.Now()))

	// Start message processor
	if err := processor.Start(ctx); err != nil {
		logger.Fatal("❌ Failed to start message processor",
			zap.Error(err),
			zap.String("component", "processor"),
			zap.String("startup_phase", "processor_initialization"))
	}

	logger.Info("✅ Message processor started successfully",
		zap.String("status", "running"),
		zap.Time("startup_time", time.Now()))

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("🛑 Shutting down Sirius Scan Agent Manager",
		zap.String("phase", "shutdown_initiated"),
		zap.Time("shutdown_time", time.Now()))

	// Shutdown services
	commandService.Stop()

	// Give services time to shut down gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Wait for processor to stop
	<-shutdownCtx.Done()
	logger.Info("👋 Shutdown complete",
		zap.String("phase", "shutdown_complete"),
		zap.Time("completion_time", time.Now()))
}
