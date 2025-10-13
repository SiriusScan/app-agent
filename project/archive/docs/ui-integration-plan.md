# UI Integration Plan for Agent Command System

## Overview

This document outlines the integration plan for connecting the front-end UI components with the backend services for agent command execution and response handling.

## Architecture Components

### 1. Backend Services

- REST API Server (`:8080`)
- RabbitMQ Message Queue (`sirius-rabbitmq:5672`)
- Valkey Data Store (`sirius-valkey:6379`)

### 2. Frontend Components

- Terminal Component
- Command Input Interface
- Response Display
- Session Management
- Queue Communication Layer
- Data Store Integration

## Integration Points

### 1. REST API Endpoints

#### Command Execution

```typescript
POST /app/agent.commands
Content-Type: application/json

{
    "agent_id": string,
    "command": string
}
```

#### Response Format

```typescript
{
    "success": boolean,
    "output": string
}
```

### 2. Valkey Store Integration

#### Command Response Keys

- Pattern: `cmd:response:{agent_id}:{command_id}`
- Value Format:

```typescript
{
    "command_id": string,
    "agent_id": string,
    "status": string,
    "output": string,
    "timestamp": string
}
```

## Implementation Steps

### 1. Terminal Component Setup

```typescript
// components/Terminal/Terminal.tsx
interface TerminalProps {
  agentId: string;
  onCommand: (command: string) => Promise<void>;
}

interface CommandResponse {
  success: boolean;
  output: string;
}
```

### 2. Command Execution Flow

1. User Input:

```typescript
const handleCommand = async (command: string) => {
  try {
    const response = await fetch("/app/agent.commands", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        agent_id: agentId,
        command: command,
      }),
    });

    const result = await response.json();
    return result;
  } catch (error) {
    console.error("Command execution failed:", error);
    throw error;
  }
};
```

### 3. Response Handling

1. Direct Response:

```typescript
const watchCommandResponse = async (commandId: string) => {
  const key = `cmd:response:${agentId}:${commandId}`;
  const response = await valkeyClient.getValue(key);
  return JSON.parse(response);
};
```

2. Streaming Updates:

```typescript
const streamResponses = async (commandId: string) => {
  const pollInterval = 500; // ms
  const maxAttempts = 20;
  let attempts = 0;

  const poll = async () => {
    const response = await watchCommandResponse(commandId);
    if (response.status === "completed" || attempts >= maxAttempts) {
      return response;
    }
    attempts++;
    await new Promise((resolve) => setTimeout(resolve, pollInterval));
    return poll();
  };

  return poll();
};
```

### 4. Session Management

```typescript
interface TerminalSession {
  sessionId: string;
  agentId: string;
  startTime: string;
}

const initializeSession = async (agentId: string): Promise<TerminalSession> => {
  const session = await terminalRouter.initializeSession.mutate();
  return {
    ...session,
    agentId,
  };
};
```

## Error Handling

1. Command Execution Errors:

```typescript
try {
  const result = await handleCommand(command);
  displayOutput(result.output);
} catch (error) {
  displayError(`Command execution failed: ${error.message}`);
}
```

2. Connection Errors:

```typescript
const handleConnectionError = (error: Error) => {
  displayError("Connection lost. Attempting to reconnect...");
  retryConnection(maxRetries);
};
```

## Security Considerations

1. Authentication:

   - All requests must include valid authentication tokens
   - Session validation for terminal connections
   - Rate limiting for command execution

2. Input Validation:
   - Sanitize command inputs
   - Validate agent IDs
   - Check command length and format

## Testing Plan

1. Unit Tests:

   - Command parsing
   - Response handling
   - Error scenarios

2. Integration Tests:

   - End-to-end command execution
   - Response streaming
   - Session management

3. Performance Tests:
   - Multiple concurrent commands
   - Large response handling
   - Connection stability

## Deployment Considerations

1. Environment Configuration:

```typescript
const config = {
  API_URL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080",
  VALKEY_HOST: process.env.NEXT_PUBLIC_VALKEY_HOST || "sirius-valkey",
  VALKEY_PORT: process.env.NEXT_PUBLIC_VALKEY_PORT || 6379,
  RABBITMQ_URL:
    process.env.NEXT_PUBLIC_RABBITMQ_URL ||
    "amqp://guest:guest@sirius-rabbitmq:5672",
};
```

2. Service Dependencies:
   - Ensure all services (API, RabbitMQ, Valkey) are available
   - Handle service discovery
   - Implement circuit breakers for service failures

## Next Steps

1. Implement the Terminal component
2. Set up command execution service
3. Integrate response handling
4. Add session management
5. Implement error handling
6. Add security measures
7. Write tests
8. Deploy and monitor

## Additional Resources

- API Documentation
- Backend Service Documentation
- Component Library Documentation
- Testing Guidelines
