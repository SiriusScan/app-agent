# gRPC Agent for Hello World Application

This is the agent (client) component of a simple gRPC-based Hello World application that demonstrates basic client-server communication.

## Features

- Connects to a gRPC server
- Implements two main functionalities:
  - `Ping`: Simple ping-pong for testing connectivity
  - `ExecuteCommand`: Executes shell commands and sends results to the server
- Executes the `ls` command and displays the result
- Periodically sends command results to the server
- Graceful shutdown on termination signals
- Structured logging with zap

## Configuration

The agent can be configured via environment variables:

- `SERVER_ADDRESS`: The address of the server to connect to (default: `localhost:50051`)
- `AGENT_ID`: A unique identifier for this agent (default: hostname)

## Prerequisites

- Go 1.21 or higher
- Protocol Buffers compiler (protoc)
- Go plugins for Protocol Buffers

## Building

To build the agent:

```bash
go build -o agent cmd/agent/main.go
```

## Running

To run the agent:

```bash
# With default settings
./agent

# With custom server address
SERVER_ADDRESS=example.com:8080 ./agent

# With custom agent ID
AGENT_ID=test-agent-1 ./agent
```

## Implementation Details

The agent performs the following actions:

1. Connects to the gRPC server specified in the configuration
2. Sends a ping request to test connectivity
3. Executes the `ls -la` command
4. Sends the command result to the server
5. Periodically executes the `ls -la` command every 10 seconds

## Architecture

The agent follows a clean, layered architecture:

- `cmd/agent/main.go`: Application entry point
- `internal/agent/agent.go`: Agent implementation
- `internal/config/config.go`: Configuration management
- `proto/hello/hello.proto`: Service definition

## Development

To regenerate gRPC code after modifying the proto file:

```bash
./scripts/generate_proto.sh
```

## Security Considerations

This is a Hello World example and does not implement security features that would be required in a production environment, such as:

- TLS encryption for gRPC communication
- Authentication and authorization
- Command validation and sandboxing
- Rate limiting

For production use, these security features should be implemented.
