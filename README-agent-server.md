# SiriusScan Agent-Server Architecture

This repository implements a bidirectional gRPC-based communication system between an agent and a server. The agent can execute different types of commands received from the server and report results back.

## Key Components

### Protocol Buffers (proto/agent.proto)

The communication protocol is defined using Protocol Buffers (protobuf), providing a structured approach to message exchange:

- `AgentMessage`: Messages from agent to server
- `CommandMessage`: Commands from server to agent
- `ResultMessage`: Command execution results
- `HeartbeatMessage`: Agent health data
- `AuthRequest`: Agent authentication data

### Agent (cmd/agent/main.go)

The agent component:

1. Connects to the gRPC server
2. Authenticates using capabilities information
3. Listens for commands from the server
4. Executes commands (scan, status, exec)
5. Reports results back to the server
6. Sends periodic heartbeats

### Server (internal/grpc/server.go)

The server component:

1. Listens for gRPC connections
2. Authenticates connecting agents
3. Distributes commands to agents
4. Processes and stores command results
5. Manages agent connections
6. Handles agent disconnections gracefully

## Communication Flow

1. **Authentication**:
   - First message from agent must be authentication
   - Server registers agent with capabilities
2. **Command Distribution**:
   - Server sends commands to agents
   - Commands include type, parameters, timeout
3. **Result Reporting**:
   - Agents report command results back to server
   - Results include status, output, error info
4. **Health Monitoring**:
   - Agents send periodic heartbeats
   - Server tracks agent status and last seen time

## Command Types

- **scan**: Security scan operations
- **status**: Agent health/status reporting
- **exec**: Shell command execution

## Running the Application

### Building Locally

```bash
# Build agent
go build -o agent-app ./cmd/agent

# Build server
go build -o server-app ./cmd/app-agent
```

### Running Locally

```bash
# Start server
./server-app

# Start agent
./agent-app --server=localhost:50051
```

Or use the provided script to run both:

```bash
./scripts/run-both.sh
```

### Running in Docker Container

The recommended way to run the application is inside the sirius-engine Docker container, which has all necessary dependencies pre-configured:

```bash
# Run both server and agent
./scripts/run-in-container.sh

# Run only the server
./scripts/run-in-container.sh --server-only

# Run only the agent
./scripts/run-in-container.sh --agent-only

# Specify a custom port
./scripts/run-in-container.sh --port 50052
```

You can also run the components manually inside the container:

```bash
# Start the server
docker exec -d sirius-engine sh -c 'cd /app-agent && go run cmd/app-agent/main.go'

# Start the agent
docker exec -it sirius-engine sh -c 'cd /app-agent && go run cmd/agent/main.go --server=localhost:50051'
```

## Security Considerations

- TLS support available for secure communication
- Agent authentication required on connection
- Timeouts implemented for command execution
- Error handling for connection failures

## Future Improvements

- Add mutual TLS authentication
- Implement agent command queueing
- Add more sophisticated agent management
- Develop web interface for command monitoring
