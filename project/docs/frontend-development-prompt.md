# Frontend Development Task: Agent Command Terminal Implementation

## Context

We need to implement a web-based terminal interface that allows users to execute commands on remote agents through our backend services. The system uses a REST API for command execution, RabbitMQ for message queuing, and Valkey for storing command responses.

## Core Requirements

1. Create a terminal component that:
   - Accepts user commands
   - Displays command output in real-time
   - Shows command execution status
   - Handles errors gracefully
   - Maintains session state

## Technical Stack

- Next.js with TypeScript
- tRPC for API communication
- Valkey client for data store access
- RabbitMQ for message queuing

## Starting Point

Begin with the Terminal component. Here's the basic interface:

```typescript
interface TerminalProps {
  agentId: string;
  onCommand: (command: string) => Promise<void>;
}

interface CommandResponse {
  success: boolean;
  output: string;
}
```

## Implementation Steps

1. **Terminal Component**

   - Create a new component at `components/Terminal/Terminal.tsx`
   - Implement command input with history support
   - Add output display with proper formatting
   - Include loading states for command execution
   - Handle command response streaming

2. **Command Execution**

   - Implement the command execution flow using the REST API
   - Use the provided endpoint: `POST /app/agent.commands`
   - Handle command responses and errors
   - Add proper error display in the terminal

3. **Response Handling**
   - Implement response polling from Valkey store
   - Add real-time updates for command output
   - Handle different response states (pending, completed, error)
   - Implement proper cleanup on component unmount

## Key Code Examples

1. **Command Execution**

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

2. **Response Polling**

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

## Required Environment Variables

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

## Development Guidelines

1. **Code Style**

   - Use TypeScript for all new components
   - Follow existing project formatting (double quotes, trailing commas)
   - Add proper type definitions for all interfaces
   - Include JSDoc comments for public functions

2. **Error Handling**

   - Implement proper error boundaries
   - Show user-friendly error messages
   - Add retry logic for failed commands
   - Log errors for debugging

3. **Testing Requirements**
   - Write unit tests for the Terminal component
   - Add integration tests for command execution
   - Test error scenarios and edge cases
   - Mock external services appropriately

## Deliverables

1. Terminal component with:

   - Command input interface
   - Output display
   - Error handling
   - Loading states
   - Command history

2. Command execution service with:

   - REST API integration
   - Response polling
   - Error handling
   - Retry logic

3. Tests:

   - Unit tests
   - Integration tests
   - Error scenario tests

4. Documentation:
   - Component usage examples
   - Props documentation
   - Error handling guide
   - Testing guide

## Success Criteria

1. Users can:

   - Enter commands in the terminal
   - See real-time command output
   - View command history
   - Handle command errors
   - See command execution status

2. The component:

   - Properly handles all error cases
   - Shows loading states
   - Updates in real-time
   - Cleans up resources on unmount

3. Tests:
   - All tests pass
   - Coverage meets project requirements
   - Edge cases are covered

## Next Steps

1. Review the UI integration plan in detail
2. Set up your development environment
3. Create the basic Terminal component
4. Implement command execution
5. Add response handling
6. Write tests
7. Document your implementation

## Questions?

If you have any questions about:

- The backend API
- Command execution flow
- Response handling
- Testing requirements

Please refer to the UI integration plan or reach out to the team for clarification.
