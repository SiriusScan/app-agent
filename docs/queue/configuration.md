# Queue System Configuration Guide

## Environment Variables

### Required Variables

```env
# RabbitMQ Connection
SIRIUS_RABBITMQ=amqp://guest:guest@sirius-rabbitmq:5672/

# Server Configuration
SERVER_ADDRESS=:50053      # gRPC server address
LOG_LEVEL=info            # Log level (debug, info, warn, error)

# Agent Configuration
AGENT_ID=agent1          # Unique identifier for the agent
```

### Optional Variables

```env
# Queue Configuration
QUEUE_NAME=agent.commands    # Default queue name
QUEUE_PREFETCH=1            # Number of messages to prefetch
QUEUE_RECONNECT_DELAY=5s    # Delay between reconnection attempts

# Command Processing
COMMAND_TIMEOUT=300s        # Maximum command execution time
MAX_PENDING_COMMANDS=100    # Maximum number of pending commands
```

## RabbitMQ Setup

### Queue Creation

The queue is created automatically with these settings:

```go
QueueConfig {
    Name:       "agent.commands",
    Durable:    true,
    AutoDelete: false,
    Exclusive:  false,
    NoWait:     false,
    Args:       nil,
}
```

### Exchange Configuration

No custom exchange is required. The system uses the default exchange.

## Server Configuration

### Command Processing Settings

```yaml
# server.yaml
queue:
  name: agent.commands
  prefetch: 1
  reconnect_delay: 5s

command:
  timeout: 300s
  max_pending: 100
  status_cleanup_interval: 1h

logging:
  level: info
  file: server_commands.log
  format: json
```

### Logging Configuration

The server logs command processing events with the following structure:

```json
{
  "timestamp": "2024-04-15T07:22:00Z",
  "level": "info",
  "event": "command_received",
  "command_id": "cmd-1234567890",
  "agent_id": "agent1",
  "command": "ls -la",
  "status": "pending"
}
```

## Agent Configuration

### Command Execution Settings

```yaml
# agent.yaml
server:
  address: localhost:50053
  reconnect_delay: 5s

command:
  timeout: 300s
  shell: /bin/bash
  working_dir: /tmp

logging:
  level: info
  file: agent.log
  format: json
```

## Security Configuration

### RabbitMQ Security

1. **Authentication**

   ```env
   RABBITMQ_DEFAULT_USER=sirius
   RABBITMQ_DEFAULT_PASS=secure_password
   ```

2. **TLS Configuration**
   ```env
   RABBITMQ_SSL_CERT_FILE=/path/to/cert.pem
   RABBITMQ_SSL_KEY_FILE=/path/to/key.pem
   RABBITMQ_SSL_CA_FILE=/path/to/ca.pem
   ```

### Command Execution Security

1. **Allowed Commands**

   ```yaml
   # security.yaml
   allowed_commands:
     - pattern: "^ls"
     - pattern: "^cat /var/log"
     - pattern: "^find /home"
   ```

2. **Resource Limits**
   ```yaml
   resource_limits:
     max_output_size: 1048576 # 1MB
     max_execution_time: 300 # seconds
     max_memory: 104857600 # 100MB
   ```

## Monitoring Configuration

### Metrics Collection

```yaml
# metrics.yaml
prometheus:
  enabled: true
  port: 9090
  path: /metrics

metrics:
  command_execution_time:
    type: histogram
    buckets: [0.1, 0.5, 1, 2, 5, 10, 30]
  command_status:
    type: gauge
    labels: [status, agent_id]
  queue_depth:
    type: gauge
```

### Health Checks

```yaml
# health.yaml
checks:
  queue:
    interval: 30s
    timeout: 5s
  agent_connection:
    interval: 60s
    timeout: 10s
```

## Development Configuration

### Local Development Setup

```env
# dev.env
SIRIUS_RABBITMQ=amqp://guest:guest@localhost:5672/
SERVER_ADDRESS=:50053
AGENT_ID=dev-agent
LOG_LEVEL=debug
```

### Testing Configuration

```yaml
# test.yaml
queue:
  mock: true
  mock_delay: 100ms

command:
  timeout: 10s
  max_pending: 10

logging:
  level: debug
  output: stdout
```
