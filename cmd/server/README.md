# gRPC Server for Hello World Application

This is the server component of a simple gRPC-based Hello World application that demonstrates basic client-server communication.

## Features

- Implements a gRPC server with two RPC methods:
  - `Ping`: Simple ping-pong for testing connectivity
  - `ExecuteCommand`: Receives command execution results from agents
- Handles multiple concurrent client connections
- Graceful shutdown on termination signals
- Structured logging with zap

## Configuration

The server can be configured via environment variables:

- `SERVER_ADDRESS`: The address and port to listen on (default: `:50051`)

## Prerequisites

- Go 1.21 or higher
- Protocol Buffers compiler (protoc)
- Go plugins for Protocol Buffers

## Building

To build the server:

```bash
go build -o server cmd/server/main.go
```

## Running

To run the server:

```bash
# With default settings
./server

# With custom address
SERVER_ADDRESS=:8080 ./server
```

## Protocol

The server implements the following gRPC service:

```protobuf
service HelloService {
  // Ping is a simple ping method to verify connectivity
  rpc Ping(PingRequest) returns (PingResponse) {}

  // ExecuteCommand executes a shell command on the agent and returns the result
  rpc ExecuteCommand(CommandRequest) returns (CommandResponse) {}
}
```

## Architecture

The server follows a clean, layered architecture:

- `cmd/server/main.go`: Application entry point
- `internal/server/server.go`: Server implementation
- `internal/config/config.go`: Configuration management
- `proto/hello/hello.proto`: Service definition

## Development

To regenerate gRPC code after modifying the proto file:

```bash
./scripts/generate_proto.sh
```

## Testing

You can test the server using grpcurl:

```bash
# Install grpcurl if not already installed
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List available services
grpcurl -plaintext localhost:50051 list

# Call the Ping method
grpcurl -plaintext -d '{"agent_id": "test-agent"}' localhost:50051 hello.HelloService/Ping
```
