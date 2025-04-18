# Agent Message Tester

A test utility for simulating UI messages to the SiriusScan Agent via RabbitMQ.

## Overview

The Agent Message Tester allows developers to send test messages to the agent's RabbitMQ queue, simulating commands that would normally come from the UI. This tool is essential for testing and debugging the agent's message handling capabilities.

## Features

- Send predefined test messages (scan, status, stop)
- Support for custom message payloads
- Pretty-printed JSON output
- Graceful shutdown handling
- Context-aware message sending
- Uses the Sirius Go API for RabbitMQ integration

## Installation

```bash
# Inside the sirius-engine container
cd /app-agent
go install ./cmd/agent-message-tester
```

## Usage

### Basic Commands

```bash
# List available message types
agent-message-tester -list

# Send a predefined message
agent-message-tester -type scan
agent-message-tester -type status
agent-message-tester -type stop

# Send a message with custom payload
agent-message-tester -type scan -payload '{"target": "10.0.0.1", "profile": "quick"}'
```

## Message Types

### 1. Scan Command

Initiates a vulnerability scan on a target.

```json
{
  "type": "task",
  "id": "scan-001",
  "timestamp": "2024-03-19T10:00:00Z",
  "payload": {
    "action": "scan",
    "target": "192.168.1.100",
    "profile": "full"
  }
}
```

### 2. Status Request

Queries the status of a running task.

```json
{
  "type": "status",
  "id": "status-001",
  "timestamp": "2024-03-19T10:00:00Z",
  "payload": {
    "task_id": "scan-001",
    "status": "running"
  }
}
```

### 3. Stop Command

Stops a running task.

```json
{
  "type": "stop",
  "id": "stop-001",
  "timestamp": "2024-03-19T10:00:00Z",
  "payload": {
    "task_id": "scan-001",
    "reason": "user_requested"
  }
}
```

## Example Workflows

### 1. Full Scan Workflow

```bash
# Start a scan
agent-message-tester -type scan -payload '{"target": "192.168.1.100", "profile": "full"}'

# Check status
agent-message-tester -type status -payload '{"task_id": "scan-001"}'

# Stop scan if needed
agent-message-tester -type stop -payload '{"task_id": "scan-001", "reason": "complete"}'
```

### 2. Quick Scan

```bash
agent-message-tester -type scan -payload '{"target": "10.0.0.1", "profile": "quick", "timeout": "300"}'
```

## Development

The tester uses the Sirius Go API's queue package for RabbitMQ operations. Messages are sent to the `agent-tasks` queue, which the agent monitors for commands.

### Error Handling

- Invalid message types: Displays available types and exits
- JSON parsing errors: Shows detailed error message
- Queue connection issues: Reports connection problems
- Graceful shutdown: Handles SIGINT/SIGTERM signals

### Adding New Message Types

1. Add the message type constant
2. Create a predefined test message
3. Update documentation

## Troubleshooting

### Common Issues

1. Connection Errors

   ```
   Error: sending message to queue: connection refused
   ```

   - Check if RabbitMQ is running
   - Verify container network connectivity

2. Invalid Payload
   ```
   Error: Invalid custom payload JSON
   ```
   - Ensure JSON payload is properly formatted
   - Check for missing quotes or commas

## Notes

- All timestamps are automatically set to the current time
- Message IDs are predefined but can be overridden in custom payloads
- The tool runs inside the sirius-engine container to ensure proper connectivity
