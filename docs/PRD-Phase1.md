# SiriusScan Agent Manager - Phase 1 PRD

## Overview

The SiriusScan Agent Manager serves as the central coordination component for distributed vulnerability scanning. This PRD outlines the first three phases of implementation, focusing on establishing core communication infrastructure using the Sirius Go API for RabbitMQ integration, implementing a gRPC server for agent connections, and creating a test agent for validation.

## Phase 1: RabbitMQ Integration

### Objective

Implement a server that listens for commands from the UI via RabbitMQ using the Sirius Go API.

### Technical Requirements

- **SDK Integration**
  - Use `github.com/SiriusScan/go-api/sirius/queue` package for RabbitMQ operations
  - Implement message processing using the `queue.Listen()` function
  - Store received commands in Valkey using `github.com/SiriusScan/go-api/sirius/store`

### Components

1. **Message Listener**

   ```go
   type MessageProcessor func(msg string)
   queue.Listen("scan-queue", messageProcessor)
   ```

2. **Command Storage**
   - Use Valkey KVStore interface for command persistence
   - Key format: `task:<command_id>`
   - Store command details including type, arguments, and status

### Data Structures

```go
type TaskMessage struct {
    ID      string            `json:"id"`
    Type    string            `json:"type"`
    Args    map[string]string `json:"args"`
    Status  string            `json:"status"`
}
```

### Success Criteria

- Successfully receive and parse commands from RabbitMQ
- Store commands in Valkey with proper formatting
- Handle basic error cases (malformed messages, storage failures)

## Phase 2: gRPC Server Implementation

### Objective

Create a gRPC server that allows agents to connect and receive commands.

### Technical Requirements

- Define proto service for agent-server communication
- Implement secure agent authentication
- Enable bidirectional streaming for command distribution

### Components

1. **Proto Definitions**

   ```protobuf
   service AgentService {
       rpc Connect(ConnectRequest) returns (stream Command);
       rpc ReportResult(CommandResult) returns (CommandAck);
   }
   ```

2. **Server Implementation**
   - Secure agent authentication system
   - Command distribution mechanism
   - Result processing and storage

### Security

- TLS encryption for all gRPC connections
- Token-based agent authentication
- Secure command validation

### Success Criteria

- gRPC server accepts secure connections
- Successfully authenticates connecting agents
- Maintains persistent command streams

## Phase 3: Test Agent Development

### Objective

Create a test agent to validate the complete communication flow.

### Technical Requirements

- Implement gRPC client functionality
- Handle command stream processing
- Report execution results

### Components

1. **Agent Implementation**

   - gRPC client with reconnection logic
   - Command execution framework
   - Result reporting mechanism

2. **Test Scenarios**

   ```
   1. Agent Connection
      - Connect to server
      - Authenticate successfully
      - Maintain command stream

   2. Command Processing
      - Receive command from stream
      - Execute mock operation
      - Report results

   3. Error Handling
      - Connection loss recovery
      - Invalid command handling
      - Authentication failures
   ```

### Success Criteria

- Agent successfully connects to server
- Processes commands received via gRPC
- Reports results back to server
- Handles connection interruptions gracefully

## Integration Flow

```mermaid
sequenceDiagram
    participant UI
    participant RabbitMQ
    participant Server
    participant Valkey
    participant Agent

    UI->>RabbitMQ: Send Command
    RabbitMQ->>Server: Deliver Command
    Server->>Valkey: Store Command
    Agent->>Server: Connect (gRPC)
    Server->>Agent: Stream Command
    Agent->>Server: Report Result
    Server->>Valkey: Store Result
```

## Environment Configuration

```bash
# RabbitMQ Configuration
SIRIUS_RABBITMQ=amqp://user:pass@localhost:5672/

# Valkey Configuration
SIRIUS_VALKEY=valkey://localhost:6379/

# gRPC Server Configuration
GRPC_SERVER_ADDR=:50051
GRPC_SERVER_TLS_CERT=/path/to/cert
GRPC_SERVER_TLS_KEY=/path/to/key
```

## Development Workflow

1. **Phase 1 Implementation**

   - Set up project structure
   - Implement RabbitMQ listener
   - Add Valkey storage integration
   - Add comprehensive logging
   - Write unit tests

2. **Phase 2 Implementation**

   - Define proto service
   - Implement gRPC server
   - Add authentication system
   - Implement command streaming
   - Write integration tests

3. **Phase 3 Implementation**
   - Create test agent
   - Implement command processing
   - Add result reporting
   - Write end-to-end tests

## Success Metrics

1. **Functionality**

   - Command flow from UI to agent works end-to-end
   - Results are properly stored and accessible
   - System handles errors gracefully

2. **Performance**

   - Command delivery latency < 100ms
   - Agent connection handling scales to 100+ agents
   - No message loss during normal operation

3. **Reliability**
   - System recovers from network interruptions
   - No data loss during component failures
   - Proper error reporting and logging

## Future Considerations

- Load balancing for multiple agents
- Enhanced security features
- Metrics and monitoring
- Advanced command types
- Agent capability discovery
