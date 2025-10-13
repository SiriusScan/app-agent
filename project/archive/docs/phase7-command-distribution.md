# Phase 7: Command Distribution System

## Overview

The Command Distribution System bridges the gap between command storage in Valkey and command execution by agents. This system is responsible for:

1. Tracking command status throughout the command lifecycle
2. Routing commands to the appropriate agents
3. Monitoring command execution and timeouts
4. Processing command results
5. Providing command status information

## Architecture

The Command Distribution System consists of the following components:

### CommandDistributor

The `CommandDistributor` is the core component responsible for:

- Periodically polling Valkey for queued commands
- Sending commands to connected agents
- Tracking command execution status
- Handling command timeouts
- Processing command results

### CommandService

The `CommandService` provides a high-level interface to the Command Distribution System:

- Manages the lifecycle of the CommandDistributor
- Provides an API for queuing commands
- Offers methods for retrieving command status

### Integration with gRPC Server

The Command Distribution System integrates with the gRPC server by:

- Registering as a command handler with the server
- Processing command results received from agents
- Providing command routing functionality

## Command Lifecycle

Commands in the system go through the following lifecycle states:

1. **Queued**: Command is stored in Valkey and waiting to be processed
2. **Assigned**: Command has been sent to an agent for execution
3. **Completed**: Command was successfully executed
4. **Failed**: Command execution failed
5. **Timeout**: Command did not complete within the allocated timeout period

## Implementation Details

### Command Storage

Commands are stored in Valkey using the following key formats:

- `command:queue:{command_id}`: Contains the serialized command information
- `command:status:{command_id}`: Contains the current status of the command
- `agent:{agent_id}:command:list:{command_id}`: Maps agent IDs to their commands
- `cmd:result:{command_id}`: Stores the result of the command execution

### Command Polling

The CommandDistributor polls for commands using the following approach:

1. Get a list of connected agents from the gRPC server
2. For each connected agent, look for queued commands
3. Send commands to the appropriate agents
4. Track the status of commands in the distributor's active commands map

### Timeout Handling

The CommandDistributor includes a timeout mechanism that:

1. Periodically checks for commands that have exceeded their timeout
2. Marks timed-out commands as failed in Valkey
3. Creates a result entry for the timed-out command
4. Notifies stakeholders about the timeout

## Command Distribution Process

The command distribution process follows these steps:

1. A command is queued by calling the `QueueCommand` method on the `CommandService`
2. The `CommandDistributor` polls Valkey and discovers the queued command
3. The distributor identifies the target agent and checks if it's connected
4. If the agent is connected, the command is sent via the gRPC server
5. The command status is updated to "assigned"
6. When the agent completes the command, it sends a result to the server
7. The server forwards the result to the distributor
8. The distributor updates the command status and stores the result

## Error Handling

The Command Distribution System handles errors through:

1. Command retries for transient failures
2. Timeout detection and handling
3. Result validation and error reporting
4. Graceful fallback mechanisms for disconnected agents

## Configuration Options

The Command Distribution System can be configured with the following options:

- `pollingInterval`: How often to poll for new commands (default: 5 seconds)
- `cleanupInterval`: How often to check for timed-out commands (default: 1 hour)
- `batchSize`: Maximum number of commands to process in a single poll (default: 20)

## Testing

Use the provided command-sender utility to test the Command Distribution System:

```bash
go run cmd/command-sender/main.go --agent=agent123 --type=STOP --params=reason=test
```

## Troubleshooting

Common issues and their solutions:

1. **Commands are queued but not executed**: Check agent connectivity and the command distributor logs
2. **Command timeouts**: Verify the timeout settings and ensure the agent is not overloaded
3. **Missing command results**: Check the Valkey connection and result storage implementation

## Future Improvements

Potential future enhancements for the Command Distribution System:

1. Command prioritization and queue management
2. Agent load balancing for high-volume command scenarios
3. Advanced monitoring and metrics for command execution
4. Enhanced security features for command validation
5. Result filtering and aggregation capabilities
