# gRPC Hello World Agent-Server Application

A simple gRPC-based client-server application that demonstrates basic command execution and result reporting.

## Overview

This project implements a basic agent-server architecture using gRPC:

- **Server**: Listens for connections from agents and receives command execution results
- **Agent**: Connects to the server, executes commands (specifically `ls`), and reports results back to the server

The communication is bidirectional, with the server capable of acknowledging the agent's messages and the agent sending periodic updates to the server.

## Project Structure

```
.
├── cmd/                    # Application entry points
│   ├── agent/              # Agent application
│   │   ├── main.go         # Agent entry point
│   │   └── README.md       # Agent documentation
│   └── server/             # Server application
│       ├── main.go         # Server entry point
│       └── README.md       # Server documentation
├── internal/               # Private application code
│   ├── agent/              # Agent implementation
│   ├── config/             # Configuration management
│   └── server/             # Server implementation
├── proto/                  # Protocol Buffers definitions
│   └── hello/              # Hello service definition
│       └── hello.proto     # Service and message definitions
├── scripts/                # Utility scripts
│   └── generate_proto.sh   # Script to generate Go code from proto files
├── go.mod                  # Go module definition
└── README.md               # This file
```

## Prerequisites

- Go 1.21 or higher
- Protocol Buffers compiler (protoc)
- Go plugins for Protocol Buffers:
  ```
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```

## Quick Start

1. **Generate gRPC code from proto files**:

   ```bash
   ./scripts/generate_proto.sh
   ```

2. **Build both applications**:

   ```bash
   go build -o server cmd/server/main.go
   go build -o agent cmd/agent/main.go
   ```

3. **Run the server**:

   ```bash
   ./server
   ```

4. **Run the agent** (in a separate terminal):
   ```bash
   ./agent
   ```

## Configuration

### Server

- `SERVER_ADDRESS`: The address and port to listen on (default: `:50051`)

### Agent

- `SERVER_ADDRESS`: The address of the server to connect to (default: `localhost:50051`)
- `AGENT_ID`: A unique identifier for this agent (default: hostname)

## Features

- **gRPC Communication**: Fast, efficient communication using Protocol Buffers and gRPC
- **Command Execution**: Agent executes the `ls` command and reports results
- **Periodic Updates**: Agent sends results to the server every 10 seconds
- **Graceful Shutdown**: Both components handle termination signals properly
- **Structured Logging**: Detailed logging with zap for better observability

## Architecture

This application follows a clean architecture approach:

- **Proto Definitions**: Define the contract between client and server
- **Config Layer**: Handle configuration through environment variables
- **Service Layer**: Implement the core logic of the server and agent
- **Main Applications**: Tie everything together in the cmd directory

## Security Considerations

This is a Hello World example and does not implement security features that would be required in a production environment. For production use, consider implementing:

- TLS encryption for gRPC communication
- Authentication and authorization
- Command validation and sandboxing
- Rate limiting
- Proper error handling and recovery

## Development

See the individual READMEs in the cmd/agent and cmd/server directories for more detailed information on each component.
