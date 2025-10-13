# SiriusScan Agent Manager - Technical Documentation

## Executive Summary

The SiriusScan Agent Manager provides a scalable, distributed architecture for vulnerability scanning across diverse environments. It consists of two primary components:

1. **Agent Manager (Server)**: A centralized coordination component that:

   - Receives commands from the UI via RabbitMQ
   - Maintains state in Valkey storage
   - Exposes a gRPC interface for agent communication
   - Orchestrates scan execution across distributed agents
   - Processes and stores scan results

2. **Test Agent (Client)**: A lightweight, standalone component that:
   - Connects to the server exclusively via gRPC
   - Operates independently of backend infrastructure (no direct RabbitMQ/Valkey access)
   - Can run from any external system with network access to the gRPC server
   - Maintains minimal state locally (just enough for operation)
   - Executes scans and reports results back to the server

This architecture enables a clear separation of concerns where the Agent Manager handles command orchestration, state management, and result processing, while the Test Agent focuses solely on scan execution. This design allows agents to be deployed in various network environments without requiring direct access to backend services.

## System Architecture Diagram

```
                                 +------------------+
                                 |    UI / Client   |
                                 +------------------+
                                          |
                                +---------v---------+
                                |  RabbitMQ Queues  |
                                +---------+---------+
                                          |
+------------+                  +---------v---------+     +---------------+
|            |                  |                   |     |               |
| Prometheus +<-----------------+  Agent Manager    +---->+ Valkey Store  |
|            |                  |     (Server)      |     |               |
+------------+                  +---------+---------+     +---------------+
                                          |
                                +---------v---------+
                                |  gRPC Interface   |
                                +---------+---------+
                                          |
                                          |
          +-------------------+--------------------+
          |                   |                    |
+---------v---------+ +-------v-------+   +--------v--------+
|                   | |               |   |                 |
|    Agent 1        | |    Agent 2    |   |     Agent N     |
|  (Remote Host)    | | (Remote Host) |   |  (Remote Host)  |
+-------------------+ +---------------+   +-----------------+
```

## Project Structure

```
.
├── cmd/
│   ├── app-agent/       # Server application entry point
│   │   └── main.go      # Main server initialization
│   └── agent/          # Test agent entry point
│       └── main.go      # Lightweight gRPC client implementation
├── internal/
│   ├── agent/          # Core agent management logic
│   │   ├── manager.go  # Agent connection management
│   │   └── types.go    # Agent data structures
│   ├── config/         # Configuration management
│   │   └── config.go   # Configuration structure and loading
│   ├── queue/          # RabbitMQ integration (server-side only)
│   │   ├── listener.go # Message queue listener
│   │   └── processor.go # Message processing logic
│   ├── command/        # Command handling
│   │   ├── parser.go   # Command validation and parsing
│   │   ├── executor.go # Command execution logic
│   │   └── types.go    # Command type definitions
│   ├── grpc/           # gRPC implementation
│   │   ├── server.go   # gRPC server implementation
│   │   └── client.go   # gRPC client utilities
│   ├── metrics/        # Metrics implementation (server-side)
│   │   ├── collector.go # Metrics data collection
│   │   └── exporter.go # Prometheus metrics export
│   ├── orchestration/  # Agent orchestration (server-side)
│   │   ├── scheduler.go # Command scheduling
│   │   └── balancer.go # Load balancing
│   ├── admin/          # Administrative API (server-side)
│   │   ├── server.go   # Admin HTTP server
│   │   ├── handlers.go # Admin API handlers
│   │   └── metrics.go  # Metrics endpoints
│   └── store/          # Storage interfaces (server-side only)
│       ├── valkey.go   # Valkey store implementation
│       └── command.go  # Command storage operations
├── pkg/
│   ├── models/         # Shared data models
│   │   ├── command.go  # Command data structures
│   │   └── agent.go    # Agent data structures
│   ├── simulation/     # Test load simulation
│   │   ├── generator.go # Test command generator
│   │   └── runner.go   # Simulation runner
│   └── reporting/      # Reporting utilities
│       ├── formatter.go # Report formatting
│       └── exporter.go  # Report export functions
├── proto/
│   └── scan/           # Protocol buffer definitions
│       └── agent.proto # Agent service definition
├── ui/                 # Admin dashboard (minimal)
│   ├── templates/      # HTML templates
│   ├── static/         # Static assets
│   └── server.go       # UI web server
└── deployment/         # Deployment configurations
    ├── docker/         # Docker configurations
    │   └── compose.yaml # Docker Compose config
    └── kubernetes/     # Kubernetes manifests
        ├── deployment.yaml # K8s deployment
        └── service.yaml    # K8s service
```

## Implementation Phases

The system has been implemented in three distinct phases:

### Phase 1: Foundation

- Core infrastructure setup
- Configuration management
- Logging system
- Graceful shutdown
- Environment bootstrapping

### Phase 2: Command Processing

- RabbitMQ command reception
- Command validation and parsing
- Valkey state storage
- gRPC server implementation
- Basic agent communication

### Phase 3: Orchestration and Test Agent

- Test agent implementation (gRPC client)
- Metrics collection and reporting
- Command orchestration and scheduling
- Agent load balancing
- Administrative API and dashboard

## Core Components

### Server Component: Agent Manager

#### 1. Main Application (`cmd/app-agent/main.go`)

The entry point of the server application, responsible for:

- Initializing the logger (using `zap`)
- Loading configuration
- Setting up graceful shutdown handling
- Initializing the key-value store
- Starting the RabbitMQ message processor
- Starting the gRPC server

#### 2. Configuration Management (`internal/config/config.go`)

Handles all application configuration through environment variables and `.env` files.

##### Server Config Structure

```go
type Config struct {
    Version        string // Application version
    Environment    string // development, staging, production
    GRPCServerAddr string // gRPC server address
    LogLevel       string // Logging level
    LogFormat      string // Logging format
    MetricsAddr    string // Metrics endpoint address
    AdminAPIAddr   string // Admin API address
    TLSEnabled     bool   // Enable TLS for gRPC
    TLSCertFile    string // TLS certificate file
    TLSKeyFile     string // TLS key file
}
```

#### 3. RabbitMQ Listener (`internal/queue/listener.go`)

Connects to RabbitMQ and processes incoming messages from the UI.

```go
type MessageListener struct {
    logger     *zap.Logger
    processor  MessageProcessor
    stopCh     chan struct{}
    queueName  string
}

func (l *MessageListener) Start(ctx context.Context) error
func (l *MessageListener) Stop() error
```

#### 4. Command Processor (`internal/queue/processor.go`)

Processes incoming RabbitMQ messages and converts them to command objects.

```go
type CommandProcessor struct {
    logger     *zap.Logger
    parser     command.Parser
    executor   command.Executor
    store      store.CommandStore
}

func (p *CommandProcessor) ProcessMessage(ctx context.Context, message string) error
```

#### 5. Valkey Store (`internal/store/valkey.go`)

Server-side storage implementation using Valkey.

```go
type ValkeyStore struct {
    logger    *zap.Logger
    client    *valkey.Client
    connected bool
    mu        sync.RWMutex
}

func (s *ValkeyStore) SetValue(ctx context.Context, key, value string) error
func (s *ValkeyStore) GetValue(ctx context.Context, key string) (store.ValkeyResponse, error)
```

#### 6. gRPC Server (`internal/grpc/server.go`)

Implements the gRPC interface for agent communication.

```go
type Server struct {
    pb.UnimplementedAgentServiceServer
    logger     *zap.Logger
    config     *config.Config
    grpcServer *grpc.Server
    agents     map[string]*AgentConnection
    mu         sync.RWMutex
}

func (s *Server) Connect(stream pb.AgentService_ConnectServer) error
func (s *Server) Start(ctx context.Context) error
func (s *Server) Stop() error
```

#### 7. Command Orchestration (`internal/orchestration/scheduler.go`)

Server-side scheduling and distribution of commands to agents.

```go
type CommandScheduler interface {
    ScheduleCommand(ctx context.Context, cmd *models.Command) error
    CancelCommand(ctx context.Context, commandID string) error
    GetCommandStatus(ctx context.Context, commandID string) (*models.CommandStatus, error)
}
```

### Client Component: Test Agent

#### 1. Agent Application (`cmd/agent/main.go`)

The lightweight agent implementation that:

- Connects to the server via gRPC
- Receives and executes commands
- Reports results back to the server
- Maintains heartbeat communication

```go
func main() {
    // Initialize logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // Load configuration
    config, err := loadConfig()
    if err != nil {
        logger.Fatal("Failed to load config", zap.Error(err))
    }

    // Set up context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    go handleShutdown(cancel)

    // Initialize gRPC client
    client, err := newGRPCClient(config.ServerAddr, config.TLSEnabled, config.TLSCertFile)
    if err != nil {
        logger.Fatal("Failed to create gRPC client", zap.Error(err))
    }

    // Start agent
    if err := runAgent(ctx, client, logger, config); err != nil {
        logger.Fatal("Agent failed", zap.Error(err))
    }
}
```

#### 2. Agent Configuration

The agent has minimal configuration requirements:

```go
type AgentConfig struct {
    AgentID      string // Unique agent identifier
    ServerAddr   string // gRPC server address
    TLSEnabled   bool   // Enable TLS for gRPC
    TLSCertFile  string // TLS certificate file
    LogLevel     string // Logging level
    LogFormat    string // Logging format
    Capabilities map[string]string // Agent capabilities
}
```

#### 3. gRPC Client Implementation (`internal/grpc/client.go`)

The agent's connection to the server:

```go
type Client struct {
    logger *zap.Logger
    conn   *grpc.ClientConn
    client pb.AgentServiceClient
}

func NewClient(serverAddr string, enableTLS bool, certFile string, logger *zap.Logger) (*Client, error)
func (c *Client) Connect(ctx context.Context, agentID string, capabilities map[string]string) (pb.AgentService_ConnectClient, error)
func (c *Client) Close() error
```

#### 4. Command Execution

The agent executes commands received through the gRPC stream. It uses an internal dispatcher that first checks if the received command string matches a known internal command prefix (e.g., `internal:status`).

- **Internal Commands**: If a prefix matches, the corresponding internal Go function is executed within the agent. These commands do _not_ interact with the external shell.
- **Script Commands**: If no internal prefix matches, the entire command string is treated as a script. If scripting is enabled (via configuration and successful PowerShell/pwsh detection), the script is executed using the configured PowerShell executable (`-NonInteractive -Command -`). Results (stdout, stderr, exit code) are captured and sent back.
- **Scripting Disabled**: If scripting is disabled or PowerShell is not found, script commands will fail, and an error result is sent back.

##### Internal Command: `internal:status`

This command performs two actions:

1.  **Backend Update (Side-Effect)**: It gathers basic host information (OS, OS Version, IP, Hostname, Host ID) and sends it to the configured backend API endpoint (`API_BASE_URL/host` via HTTP POST) to update the central host record. This API call happens in the background and does not block the command response.
2.  **Local Status Report**: It gathers detailed local agent status (Agent ID, Host ID, Uptime, Go Version, OS/Arch, Memory, Goroutines, Configuration details like scripting status and paths) and sends this information back to the connected gRPC server as the command's output.

```go
// Example of current command handling logic (simplified)
func (a *Agent) processCommandString(ctx context.Context, commandString string) {
    switch {
    case strings.HasPrefix(commandString, "internal:status"):
        a.handleInternalStatusCommand(ctx, commandString)
    // other internal commands...
    default:
        a.executeScriptCommand(ctx, commandString)
    }
}
```

## Communication Protocol

### gRPC Protocol (`proto/scan/agent.proto`)

```protobuf
syntax = "proto3";

package scan;

option go_package = "github.com/SiriusScan/app-agent/proto/scan";

service AgentService {
    // Bidirectional streaming RPC for agent communication
    rpc Connect(stream AgentMessage) returns (stream CommandMessage);
}

// Agent status values
enum AgentStatus {
    UNKNOWN = 0;
    IDLE = 1;
    BUSY = 2;
    ERROR = 3;
    MAINTENANCE = 4;
}

// Message sent from agent to server
message AgentMessage {
    string agent_id = 1;
    AgentStatus status = 2;
    oneof payload {
        HeartbeatMessage heartbeat = 3;
        CommandResult command_result = 4;
    }
    map<string, string> metadata = 5;
}

// Message sent from server to agent
message CommandMessage {
    string id = 1;
    string type = 2;
    map<string, string> params = 3;
    int64 timeout_seconds = 4;
}

// Heartbeat message
message HeartbeatMessage {
    int64 timestamp = 1;
    double cpu_usage = 2;
    double memory_usage = 3;
    int32 active_commands = 4;
}

// Command result message
message CommandResult {
    string command_id = 1;
    string status = 2;
    map<string, string> data = 3;
    string error = 4;
    int64 completed_at = 5;
}
```

## Deployment

### Agent Manager Deployment (Docker)

```yaml
# docker-compose.yml for the server
version: "3.8"

services:
  app-agent:
    image: siriusscan/app-agent:latest
    restart: always
    ports:
      - "50051:50051"
      - "8080:8080"
    environment:
      - APP_VERSION=1.0.0
      - APP_ENV=production
      - GRPC_SERVER_ADDR=:50051
      - SIRIUS_LOG_LEVEL=info
      - SIRIUS_LOG_FORMAT=json
      - METRICS_ADDR=:2112
      - ADMIN_API_ADDR=:8080
      - TLS_ENABLED=false
    depends_on:
      - rabbitmq
      - valkey

  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"
      - "15672:15672"
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

  valkey:
    image: valkey/valkey:latest
    ports:
      - "6379:6379"
    volumes:
      - valkey_data:/data

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    depends_on:
      - prometheus

volumes:
  rabbitmq_data:
  valkey_data:
```

### Test Agent Deployment

The test agent can be deployed independently of the server with minimal dependencies:

```yaml
# docker-compose.yml for the agent
version: "3.8"

services:
  app-agent-client:
    image: siriusscan/app-agent-client:latest
    restart: always
    environment:
      - AGENT_ID=agent-001
      - SERVER_ADDR=server-hostname:50051
      - TLS_ENABLED=false
      - SIRIUS_LOG_LEVEL=info
      - SIRIUS_LOG_FORMAT=json
      - CAPABILITIES={"scan_type":"full","max_targets":"100"}
```

Alternatively, the agent can be deployed directly on a host:

```bash
# Run the agent directly
./app-agent-client --server-addr=server-hostname:50051 --agent-id=agent-001
```

## Configuration

### Server Environment Variables

```bash
# Server Configuration
APP_VERSION=1.0.0                 # Application version
APP_ENV=development               # Environment
GRPC_SERVER_ADDR=:50051          # gRPC server address
SIRIUS_LOG_LEVEL=info            # Log level
SIRIUS_LOG_FORMAT=json           # Log format
METRICS_ADDR=:2112               # Metrics server address
ADMIN_API_ADDR=:8080             # Admin API address
TLS_ENABLED=false                # Enable TLS for gRPC
TLS_CERT_FILE=                   # TLS certificate file path
TLS_KEY_FILE=                    # TLS key file path

# RabbitMQ Configuration (handled by Sirius Go API)
# SIRIUS_RABBITMQ=amqp://guest:guest@rabbitmq:5672/

# Valkey Configuration (handled by Sirius Go API)
# SIRIUS_VALKEY=valkey:6379
```

### Agent Environment Variables

```bash
# Agent Configuration
AGENT_ID=agent-001               # Agent ID (generated if not provided, defaults to hostname)
SERVER_ADDR=server:50051         # gRPC server address (e.g., server-hostname:50051 or localhost:50051)
TLS_ENABLED=false                # Enable TLS for gRPC
TLS_CERT_FILE=                   # TLS certificate file path
SIRIUS_LOG_LEVEL=info            # Log level (e.g., debug, info, warn, error)
SIRIUS_LOG_FORMAT=json           # Log format (e.g., json, console)
CAPABILITIES={"scan_type":"full"} # JSON-encoded agent capabilities (DEPRECATED - capabilities now sent via gRPC metadata)
HOST_ID=                         # Optional: Specific ID for the host record in the backend API (defaults to AGENT_ID if not set)
API_BASE_URL=                    # Optional: Base URL for the backend REST API (e.g., http://192.168.1.100:9001). Defaults based on SERVER_ADDR host, using port 9001.
POWERSHELL_PATH=                 # Optional: Explicit path to the PowerShell executable (powershell.exe or pwsh). If not set, agent searches PATH.
ENABLE_SCRIPTING=true            # Optional: Set to "false" to disable PowerShell script execution even if found. Defaults to true.
```

## Development Guide

### Server Development

1. **Setup Local Environment**

   ```bash
   # Clone the repository
   git clone https://github.com/SiriusScan/app-agent.git
   cd app-agent

   # Install dependencies
   go mod download

   # Generate protocol buffers
   ./scripts/generate-protos.sh

   # Run the server
   go run cmd/app-agent/main.go
   ```

2. **Building the Server**

   ```bash
   # Build for local platform
   go build -o app-agent ./cmd/app-agent

   # Build for specific platform
   GOOS=linux GOARCH=amd64 go build -o app-agent-linux-amd64 ./cmd/app-agent
   ```

### Agent Development

1. **Setup Agent Environment**

   ```bash
   # Clone the repository (if not already done)
   git clone https://github.com/SiriusScan/app-agent.git
   cd app-agent

   # Install dependencies
   go mod download

   # Generate protocol buffers
   ./scripts/generate-protos.sh

   # Run the agent
   go run cmd/agent/main.go --server-addr=localhost:50051
   ```

2. **Building the Agent**

   ```bash
   # Build for local platform
   go build -o app-agent-client ./cmd/agent

   # Build for specific platform (e.g., for deployment on various hosts)
   GOOS=linux GOARCH=amd64 go build -o app-agent-client-linux-amd64 ./cmd/agent
   ```

### Testing

#### Unit Testing

```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/grpc
```

#### Integration Testing

```bash
# Run integration tests
go test -tags=integration ./...
```

#### Load Testing

Configure the test agent for load simulation:

```go
func runLoadTest() {
    // Configure simulation
    config := &simulation.Config{
        ServerAddresses:  []string{"localhost:50051"},
        AgentCount:       10,
        CommandsPerAgent: 100,
        CommandTypes: []string{"scan", "status"},
        RampUpPeriod:     30 * time.Second,
        TestDuration:     5 * time.Minute,
    }

    // Run simulation
    report, err := simulation.RunLoadTest(context.Background(), config)
    if err != nil {
        log.Fatalf("Failed to run load test: %v", err)
    }

    // Print report
    fmt.Printf("Load Test Report:\n")
    fmt.Printf("Total Commands: %d\n", report.TotalCommands)
    fmt.Printf("Successful: %d (%.2f%%)\n", report.SuccessfulCommands,
               float64(report.SuccessfulCommands)/float64(report.TotalCommands)*100)
    fmt.Printf("Failed: %d (%.2f%%)\n", report.FailedCommands,
               float64(report.FailedCommands)/float64(report.TotalCommands)*100)
    fmt.Printf("Average Response Time: %v\n", report.AverageResponseTime)
}
```

## Security Considerations

### Server Security

1. **TLS Encryption**

   - Enable TLS for gRPC communication
   - Use proper certificate management
   - Implement certificate rotation

2. **Authentication**

   - Validate agent authentication credentials
   - Implement token-based authentication
   - Set up role-based access control for admin API

3. **Authorization**
   - Restrict command execution based on agent capabilities
   - Implement rate limiting for command execution
   - Monitor for suspicious activity

### Agent Security

1. **Authentication**

   - Use secure authentication mechanisms to connect to server
   - Protect agent credentials

2. **Communication Security**

   - Use TLS for all gRPC communication
   - Validate server certificates

3. **Local Security**
   - Minimal permissions required for operation
   - No persistent storage of sensitive data
   - Local logging segregation

## Performance Optimization

### Server Optimizations

1. **RabbitMQ Optimizations**

   - Use durable queues for critical messages
   - Configure appropriate QoS settings
   - Use prefetch counts to manage consumer load

2. **Valkey Optimizations**

   - Use appropriate data structures
   - Set realistic TTLs for ephemeral data
   - Configure proper eviction policies

3. **gRPC Optimizations**
   - Use bi-directional streams for efficiency
   - Implement backpressure mechanisms
   - Batch updates when possible

### Agent Optimizations

1. **Connection Management**

   - Implement reconnection with exponential backoff
   - Maintain long-lived gRPC connections
   - Handle network interruptions gracefully

2. **Resource Management**
   - Limit concurrent command execution
   - Implement graceful degradation under load
   - Monitor and report resource usage

## Future Enhancements

1. **Enhanced Security**

   - Certificate rotation
   - Advanced authentication mechanisms
   - Security vulnerability scanning

2. **Extended Analytics**

   - Machine learning for anomaly detection
   - Predictive scaling
   - Historical trend analysis

3. **Multi-Region Support**

   - Geographic distribution of agents
   - Cross-region coordination
   - Region-aware command routing

4. **Advanced UI**
   - Interactive command builder
   - Custom dashboard creation
   - Advanced visualization options

## Appendix

### Example: Implementing a Custom Command Handler (Server-side)

```go
// 1. Define the command handler
type ScanCommandHandler struct {
    logger *zap.Logger
    scheduler orchestration.CommandScheduler
}

// 2. Implement the CommandHandler interface
func (h *ScanCommandHandler) Execute(ctx context.Context, cmd *models.Command) (*models.CommandResult, error) {
    // Validate command parameters
    target, ok := cmd.Params["target"]
    if !ok || target == "" {
        return nil, errors.New("missing required parameter: target")
    }

    // Schedule the command for execution by an agent
    if err := h.scheduler.ScheduleCommand(ctx, cmd); err != nil {
        return nil, fmt.Errorf("failed to schedule command: %w", err)
    }

    // Return pending result
    return &models.CommandResult{
        CommandID: cmd.ID,
        Status: "pending",
        Data: map[string]string{
            "message": "Command scheduled for execution",
        },
    }, nil
}

// 3. Register the handler with the executor
executor.RegisterHandler("scan", &ScanCommandHandler{
    logger: logger.Named("scan_handler"),
    scheduler: scheduler,
})
```

### Example: Implementing Command Execution (Agent-side)

```go
// Execute a scan command on the agent
func executeScanCommand(ctx context.Context, cmd *pb.CommandMessage, logger *zap.Logger) *pb.CommandResult {
    logger.Info("Executing scan command",
        zap.String("command_id", cmd.Id),
        zap.String("target", cmd.Params["target"]))

    // Get command parameters
    target := cmd.Params["target"]
    if target == "" {
        return &pb.CommandResult{
            CommandId: cmd.Id,
            Status:    "failed",
            Error:     "missing target parameter",
        }
    }

    scanType := cmd.Params["scan_type"]
    if scanType == "" {
        scanType = "default"
    }

    // Execute the scan (implementation details omitted)
    scanResult, err := performScan(ctx, target, scanType)
    if err != nil {
        return &pb.CommandResult{
            CommandId: cmd.Id,
            Status:    "failed",
            Error:     err.Error(),
        }
    }

    // Return success result with scan data
    return &pb.CommandResult{
        CommandId:   cmd.Id,
        Status:      "completed",
        Data:        scanResult,
        CompletedAt: time.Now().Unix(),
    }
}
```

### Example: Adding a New Internal Agent Command

The agent uses a modular system (`internal/commands`) to handle commands prefixed with `internal:`. To add a new internal command (e.g., `internal:newcmd`):

1.  **Create Command File/Package:**
    Create a new file (e.g., `internal/commands/newcmd/newcmd_command.go`) or place it within an existing relevant package.

2.  **Define Command Struct:**
    Define an empty struct for your command:

    ```go
    package newcmd

    import (
        "context"
        "github.com/SiriusScan/app-agent/internal/commands"
        // other necessary imports
    )

    type NewCommand struct{}
    ```

3.  **Implement `commands.Command` Interface:**
    Implement the `Execute` method. This method receives the execution `context`, shared `agentInfo` (containing logger, config, API client, etc.), the original `commandString`, and any `args` that came after the prefix.

    ```go
    // Ensures NewCommand implements the Command interface at compile time.
    var _ commands.Command = (*NewCommand)(nil)

    func (c *NewCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (output string, err error) {
        agentInfo.Logger.Info("Executing new internal command", zap.String("args", args))

        // --- Command Logic ---
        // Use agentInfo.Logger, agentInfo.Config, agentInfo.APIClient etc.
        output = fmt.Sprintf("New command executed with args: %s", args)
        // --- End Command Logic ---

        return output, nil // Return output string and nil error on success
    }
    ```

4.  **Register Command:**
    Add an `init()` function in the same file to register the command struct with its desired prefix.

    ```go
    func init() {
        commands.Register("internal:newcmd", &NewCommand{})
    }
    ```

5.  **Import for Registration:**
    In `internal/agent/agent.go`, add an underscore import for your new command package. This ensures the `init()` function runs and the command is registered when the agent starts.
    ```go
    import (
        // ... other imports
        _ "github.com/SiriusScan/app-agent/internal/commands/newcmd" // Import for side-effect (registration)
    )
    ```

The agent's command dispatcher (`internal/commands/registry.go`) will now recognize and route commands starting with `internal:newcmd` to your `NewCommand.Execute` method.
