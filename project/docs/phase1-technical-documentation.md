# SiriusScan Agent Manager - Phase 1 Technical Documentation

## Overview

The SiriusScan Agent Manager is a critical component in the SiriusScan vulnerability scanning ecosystem, focusing on distributed task management and coordination. Phase 1 implements the core infrastructure for RabbitMQ-based command handling and state management using the Sirius Go API SDK.

## Project Structure

```
.
├── cmd/
│   └── app-agent/       # Application entry point
│       └── main.go      # Main application initialization
├── internal/
│   ├── agent/          # Core agent management logic
│   ├── config/         # Configuration management
│   │   └── config.go   # Configuration structure and loading
│   └── store/          # Storage interfaces
│       └── store.go    # KVStore interface and mock implementation
├── pkg/                # Public packages (empty in Phase 1)
└── project/
    └── docs/          # Project documentation
```

## Core Components

### 1. Main Application (`cmd/app-agent/main.go`)

The entry point of the application, responsible for:

- Initializing the logger (using `zap`)
- Loading configuration
- Setting up graceful shutdown handling
- Initializing the key-value store
- Starting the message processor

Key Functions:

```go
func main() {
    // Initializes logger, config, and starts the application
}
```

Notable Features:

- Uses context for cancellation and shutdown
- Implements graceful shutdown with 5-second timeout
- Proper resource cleanup with deferred calls
- Structured logging with zap

### 2. Configuration Management (`internal/config/config.go`)

Handles all application configuration through environment variables and `.env` files.

#### Config Structure

```go
type Config struct {
    Version        string // Application version
    Environment    string // development, staging, production
    GRPCServerAddr string // gRPC server address
    LogLevel       string // Logging level
    LogFormat      string // Logging format
    MetricsAddr    string // Metrics endpoint address
}
```

Key Functions:

```go
func Load() (*Config, error)                    // Loads configuration from environment
func loadEnvFile() error                        // Attempts to load .env file
func getEnv(key, defaultValue string) string    // Gets environment variable with default
```

Environment Variables:

```bash
APP_VERSION=1.0.0                 # Application version
APP_ENV=development               # Environment
GRPC_SERVER_ADDR=:50051          # gRPC server address
SIRIUS_LOG_LEVEL=info            # Log level
SIRIUS_LOG_FORMAT=json           # Log format
METRICS_ADDR=:2112               # Metrics address
```

### 3. Storage Interface (`internal/store/store.go`)

Defines the interface for key-value storage operations and provides a mock implementation for testing.

#### Key Interfaces and Types

```go
type KVStore interface {
    SetValue(ctx context.Context, key, value string) error
    GetValue(ctx context.Context, key string) (ValkeyResponse, error)
    Close() error
}

type Message struct {
    Value string
}

type ValkeyResponse struct {
    Message Message
}
```

Mock Implementation:

- `MockStore`: In-memory implementation for testing
- Thread-safe operations
- Simple key-value storage

## Integration with Sirius Go API

### RabbitMQ Integration

The application uses the Sirius Go API's queue package for RabbitMQ operations:

```go
import "github.com/SiriusScan/go-api/sirius/queue"
```

Key Features:

- Automatic queue declaration
- Message processing in goroutines
- Built-in error handling
- No manual configuration needed (handled by SDK)

### Valkey Store Integration

Uses the Sirius Go API's store package for key-value operations:

```go
import "github.com/SiriusScan/go-api/sirius/store"
```

Key Features:

- Automatic connection management
- Context-aware operations
- Built-in error handling
- No manual configuration needed (handled by SDK)

## Development Guide

### Prerequisites

1. Go 1.21 or later
2. Access to RabbitMQ server (configured via SDK)
3. Access to Valkey store (configured via SDK)

### Setting Up Development Environment

1. Clone the repository:

   ```bash
   git clone <repository-url>
   cd app-agent
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Create a `.env` file in the project root:

   ```bash
   APP_VERSION=1.0.0
   APP_ENV=development
   GRPC_SERVER_ADDR=:50051
   SIRIUS_LOG_LEVEL=debug
   SIRIUS_LOG_FORMAT=text
   METRICS_ADDR=:2112
   ```

4. Run tests:
   ```bash
   go test ./...
   ```

### Development Workflow

1. **Code Organization**

   - Place new agent-related code in `internal/agent/`
   - Add new configuration options in `internal/config/config.go`
   - Implement interfaces defined in `internal/store/store.go`

2. **Testing**

   - Write unit tests for all new functionality
   - Use `MockStore` for testing storage operations
   - Follow table-driven test patterns

3. **Logging**
   - Use the provided zap logger
   - Include relevant fields in structured logging
   - Follow log level guidelines:
     - Debug: Detailed development info
     - Info: General operational events
     - Warn: Concerning but non-critical issues
     - Error: Critical issues requiring attention

### Common Development Tasks

1. **Adding New Configuration Options**

   ```go
   // 1. Add to Config struct in config.go
   type Config struct {
       NewOption string
   }

   // 2. Add to Load function
   cfg := &Config{
       NewOption: getEnv("NEW_OPTION", "default"),
   }
   ```

2. **Implementing Storage Operations**

   ```go
   // 1. Use the KVStore interface
   store.SetValue(ctx, "key", "value")

   // 2. Handle errors appropriately
   if err != nil {
       logger.Error("Failed to set value", zap.Error(err))
   }
   ```

3. **Graceful Shutdown Handling**

   ```go
   // 1. Use context cancellation
   ctx, cancel := context.WithCancel(context.Background())
   defer cancel()

   // 2. Listen for shutdown signals
   sigCh := make(chan os.Signal, 1)
   signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
   ```

## Testing Guide

### Unit Tests

1. **Configuration Testing**

   - Test environment variable loading
   - Test .env file loading
   - Test default values

2. **Store Testing**
   - Use MockStore for storage operations
   - Test all KVStore interface methods
   - Test error conditions

### Integration Tests

1. **Store Integration**

   - Test with actual Valkey store
   - Verify connection handling
   - Test concurrent operations

2. **RabbitMQ Integration**
   - Test message processing
   - Verify queue operations
   - Test error handling

## Deployment

### Container Environment

The application is designed to run in a container:

- Container name: sirius-engine
- Mount point: /app-agent
- Environment variables injected via container runtime

### Health Checks

Monitor the following endpoints:

- Metrics: `http://<host>:2112/metrics`
- gRPC health: `<host>:50051`

## Troubleshooting

### Common Issues

1. **Configuration Loading**

   - Check .env file location
   - Verify environment variables
   - Check file permissions

2. **Store Connection**

   - Verify Valkey service availability
   - Check connection logs
   - Verify credentials

3. **RabbitMQ Issues**
   - Check queue connectivity
   - Verify message processing
   - Monitor queue length

### Debugging

1. Set log level to debug:

   ```bash
   export SIRIUS_LOG_LEVEL=debug
   ```

2. Use structured logging fields:
   ```go
   logger.Debug("Detailed operation info",
       zap.String("component", "store"),
       zap.Error(err))
   ```

## Future Considerations

Phase 1 establishes the foundation for:

1. Message queue-based command handling
2. Persistent state management
3. Graceful shutdown handling
4. Structured logging

Future phases will build upon this to add:

1. gRPC-based scan execution
2. Enhanced security features
3. Advanced orchestration capabilities
4. Metrics and monitoring improvements
