# Queue-Based Command Distribution System

## Overview

The Sirius Agent Manager uses RabbitMQ for asynchronous command distribution to agents. This system works alongside the existing gRPC-based communication to provide a flexible and scalable command execution framework.

## Architecture

### Components

1. **Command Sender**

   - Standalone binary for sending commands via queue
   - Located in `cmd/command-sender/`
   - Uses Sirius queue package for message delivery

2. **Server Queue Processor**

   - Integrated into the main server
   - Listens for commands on `agent.commands` queue
   - Routes commands to appropriate agents
   - Manages command lifecycle and status tracking

3. **Command Message Format**
   ```json
   {
     "message_id": "msg-1234567890",
     "agent_id": "agent1",
     "command": "ls -la",
     "timestamp": "2024-04-15T07:22:00Z"
   }
   ```

## Configuration Requirements

### RabbitMQ Configuration

The system expects RabbitMQ to be configured with the following settings:

```env
SIRIUS_RABBITMQ=amqp://guest:guest@sirius-rabbitmq:5672/
```

### Queue Configuration

- Queue Name: `agent.commands`
- Durability: Durable
- Auto-delete: False
- Arguments: None

## Usage Guide

### Command Sender Usage

```bash
# Send a command to a specific agent
command-sender -agent=agent1 -command="ls -la"

# Example with complex command
command-sender -agent=agent2 -command="find /var/log -name '*.log'"
```

### Server CLI Commands

The server provides an interactive CLI with the following commands:

```
Available commands:
  <agent_id> <command>  - Send command to specific agent
  list                  - Show connected agents
  status [cmd_id]       - Show status of all or specific command
  pending               - Show pending commands
```

### Command Status Lifecycle

Commands go through the following states:

1. `pending` - Command created but not yet sent
2. `sent` - Command sent to agent
3. `completed` - Command executed successfully
4. `failed` - Command failed to execute

## Troubleshooting

### Common Issues

1. **Queue Connection Issues**

   - Verify RabbitMQ is running: `docker ps | grep rabbitmq`
   - Check connection string in configuration
   - Ensure network connectivity to RabbitMQ

2. **Command Not Reaching Agent**

   - Verify agent is connected (use `list` command)
   - Check server logs for routing errors
   - Verify command format is correct

3. **Command Execution Failures**
   - Check agent logs for execution errors
   - Verify command syntax is supported on agent
   - Check agent permissions for command execution

### Logging

- Server logs are written to `server_commands.log`
- Command execution results include:
  - Command ID
  - Agent ID
  - Execution time
  - Exit code
  - Output/error streams

### Monitoring

Monitor command execution using the server CLI:

```bash
# Check all command statuses
> status

# Check specific command
> status cmd-1234567890

# List pending commands
> pending
```

## Message Format Reference

### Command Message

```json
{
  "message_id": "msg-1234567890",
  "agent_id": "agent1",
  "command": "ls -la",
  "timestamp": "2024-04-15T07:22:00Z"
}
```

### Command Result

```json
{
  "command": "ls -la",
  "output": "total 123...",
  "error": "",
  "exit_code": 0,
  "execution_time": 45
}
```

## Best Practices

1. **Command Sending**

   - Use meaningful agent IDs
   - Validate commands before sending
   - Handle command timeouts appropriately
   - Monitor command status

2. **Error Handling**

   - Check command status after sending
   - Monitor agent connectivity
   - Handle command failures gracefully
   - Log all errors for debugging

3. **Security**
   - Validate command input
   - Restrict command execution permissions
   - Monitor failed command patterns
   - Audit command execution logs
