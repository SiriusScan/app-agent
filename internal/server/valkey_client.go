package server

import (
	"context"
	"fmt"
	"os"
	"time"

	valkey "github.com/valkey-io/valkey-go"
	"go.uber.org/zap"
)

// NewValkeyClient creates a new ValKey client from environment variables
func NewValkeyClient(logger *zap.Logger) (valkey.Client, error) {
	// Get configuration from environment
	host := os.Getenv("VALKEY_HOST")
	port := os.Getenv("VALKEY_PORT")

	if host == "" {
		host = "localhost" // Default fallback
	}
	if port == "" {
		port = "6379" // Default fallback
	}

	address := fmt.Sprintf("%s:%s", host, port)

	logger.Info("Initializing ValKey client",
		zap.String("address", address))

	// Create ValKey client with proper configuration
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{address},
		SelectDB:    0, // Use database 0 for templates

		// Connection pooling
		ConnWriteTimeout: 10 * time.Second,

		// Retry configuration
		DisableRetry: false,

		// Enable pipelining for better performance
		DisableCache: false,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create ValKey client: %w", err)
	}

	// Test connection with a ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pingCmd := client.B().Ping().Build()
	if err := client.Do(ctx, pingCmd).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping ValKey server: %w", err)
	}

	logger.Info("ValKey client connected successfully",
		zap.String("address", address))

	return client, nil
}
