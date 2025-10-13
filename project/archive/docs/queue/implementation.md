# Queue System Implementation Guide

## Overview

This guide details how to implement the queue-based command distribution system in the Sirius Agent Manager. The system uses RabbitMQ for reliable message delivery between the server and agents.

## Core Components

### 1. Queue Manager

```go
// QueueManager handles RabbitMQ connection and channel management
type QueueManager struct {
    conn    *amqp.Connection
    channel *amqp.Channel
    config  QueueConfig
}

type QueueConfig struct {
    URI            string
    QueueName      string
    PrefetchCount  int
    ReconnectDelay time.Duration
}

// NewQueueManager creates a new queue manager instance
func NewQueueManager(config QueueConfig) (*QueueManager, error) {
    qm := &QueueManager{config: config}
    if err := qm.connect(); err != nil {
        return nil, err
    }
    return qm, nil
}
```

### 2. Command Publisher

```go
// CommandPublisher handles sending commands to the queue
type CommandPublisher struct {
    qm *QueueManager
}

// PublishCommand sends a command to the specified agent
func (p *CommandPublisher) PublishCommand(cmd *Command) error {
    body, err := json.Marshal(cmd)
    if err != nil {
        return fmt.Errorf("marshal command: %w", err)
    }

    return p.qm.channel.Publish(
        "",               // exchange
        p.qm.config.QueueName, // routing key
        false,           // mandatory
        false,           // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
}
```

### 3. Command Consumer

```go
// CommandConsumer handles receiving and processing commands
type CommandConsumer struct {
    qm       *QueueManager
    executor CommandExecutor
}

// StartConsuming begins consuming commands from the queue
func (c *CommandConsumer) StartConsuming() error {
    msgs, err := c.qm.channel.Consume(
        c.qm.config.QueueName, // queue
        "",                    // consumer
        false,                // auto-ack
        false,                // exclusive
        false,                // no-local
        false,                // no-wait
        nil,                  // args
    )
    if err != nil {
        return fmt.Errorf("start consuming: %w", err)
    }

    go func() {
        for d := range msgs {
            var cmd Command
            if err := json.Unmarshal(d.Body, &cmd); err != nil {
                log.Errorf("unmarshal command: %v", err)
                d.Nack(false, false)
                continue
            }

            result := c.executor.Execute(&cmd)
            // Handle result (e.g., send to results queue)

            d.Ack(false)
        }
    }()

    return nil
}
```

### 4. Command Executor

```go
// CommandExecutor handles the actual execution of commands
type CommandExecutor interface {
    Execute(cmd *Command) *CommandResult
}

type ShellCommandExecutor struct {
    timeout time.Duration
    shell   string
}

func (e *ShellCommandExecutor) Execute(cmd *Command) *CommandResult {
    ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
    defer cancel()

    execCmd := exec.CommandContext(ctx, e.shell, "-c", cmd.Command)
    output, err := execCmd.CombinedOutput()

    return &CommandResult{
        CommandID: cmd.ID,
        AgentID:   cmd.AgentID,
        Output:    string(output),
        Error:     err,
        ExitCode:  execCmd.ProcessState.ExitCode(),
    }
}
```

## Implementation Steps

### 1. Server-Side Implementation

1. Initialize Queue Manager:

```go
config := QueueConfig{
    URI:            os.Getenv("SIRIUS_RABBITMQ"),
    QueueName:      "agent.commands",
    PrefetchCount:  1,
    ReconnectDelay: 5 * time.Second,
}

qm, err := NewQueueManager(config)
if err != nil {
    log.Fatalf("create queue manager: %v", err)
}
```

2. Set up Command Publisher:

```go
publisher := &CommandPublisher{qm: qm}

// Example usage
cmd := &Command{
    ID:      uuid.New().String(),
    AgentID: "agent1",
    Command: "ls -la",
}

if err := publisher.PublishCommand(cmd); err != nil {
    log.Errorf("publish command: %v", err)
}
```

### 2. Agent-Side Implementation

1. Initialize Command Consumer:

```go
executor := &ShellCommandExecutor{
    timeout: 5 * time.Minute,
    shell:   "/bin/bash",
}

consumer := &CommandConsumer{
    qm:       qm,
    executor: executor,
}

if err := consumer.StartConsuming(); err != nil {
    log.Fatalf("start consuming: %v", err)
}
```

2. Implement Result Handling:

```go
type ResultHandler struct {
    qm *QueueManager
}

func (h *ResultHandler) HandleResult(result *CommandResult) error {
    body, err := json.Marshal(result)
    if err != nil {
        return fmt.Errorf("marshal result: %w", err)
    }

    return h.qm.channel.Publish(
        "",                     // exchange
        "agent.command.results", // routing key
        false,                 // mandatory
        false,                 // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
}
```

## Error Handling

### 1. Connection Recovery

```go
func (qm *QueueManager) reconnect() error {
    for {
        if err := qm.connect(); err == nil {
            return nil
        }
        time.Sleep(qm.config.ReconnectDelay)
    }
}

func (qm *QueueManager) watchConnection() {
    notifyClose := qm.conn.NotifyClose(make(chan *amqp.Error))
    go func() {
        for range notifyClose {
            log.Warn("RabbitMQ connection lost, attempting to reconnect...")
            if err := qm.reconnect(); err != nil {
                log.Errorf("reconnect failed: %v", err)
            }
        }
    }()
}
```

### 2. Message Acknowledgment

```go
func (c *CommandConsumer) processMessage(d amqp.Delivery) {
    var cmd Command
    if err := json.Unmarshal(d.Body, &cmd); err != nil {
        log.Errorf("unmarshal command: %v", err)
        d.Nack(false, true) // requeue message
        return
    }

    result := c.executor.Execute(&cmd)
    if result.Error != nil {
        log.Errorf("execute command: %v", result.Error)
        // Decide whether to requeue based on error type
        d.Nack(false, isRetryableError(result.Error))
        return
    }

    if err := c.handleResult(result); err != nil {
        log.Errorf("handle result: %v", err)
        d.Nack(false, true)
        return
    }

    d.Ack(false)
}
```

## Monitoring and Metrics

### 1. Queue Metrics

```go
type QueueMetrics struct {
    messagesSent     prometheus.Counter
    messagesReceived prometheus.Counter
    processingTime   prometheus.Histogram
    queueDepth       prometheus.Gauge
}

func NewQueueMetrics() *QueueMetrics {
    return &QueueMetrics{
        messagesSent: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "queue_messages_sent_total",
            Help: "Total number of messages sent to the queue",
        }),
        messagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "queue_messages_received_total",
            Help: "Total number of messages received from the queue",
        }),
        processingTime: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "command_processing_duration_seconds",
            Help:    "Time taken to process commands",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
        }),
        queueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "queue_depth",
            Help: "Current number of messages in the queue",
        }),
    }
}
```

### 2. Health Checks

```go
type HealthCheck struct {
    qm *QueueManager
}

func (h *HealthCheck) Check() error {
    if h.qm.conn == nil || h.qm.conn.IsClosed() {
        return errors.New("queue connection is closed")
    }

    if h.qm.channel == nil {
        return errors.New("queue channel is nil")
    }

    return nil
}
```

## Testing

### 1. Mock Queue Manager

```go
type MockQueueManager struct {
    messages chan []byte
}

func NewMockQueueManager() *MockQueueManager {
    return &MockQueueManager{
        messages: make(chan []byte, 100),
    }
}

func (m *MockQueueManager) Publish(body []byte) error {
    m.messages <- body
    return nil
}

func (m *MockQueueManager) Consume() (<-chan []byte, error) {
    return m.messages, nil
}
```

### 2. Integration Tests

```go
func TestCommandExecution(t *testing.T) {
    // Set up test environment
    qm := NewMockQueueManager()
    publisher := &CommandPublisher{qm: qm}
    consumer := &CommandConsumer{
        qm: qm,
        executor: &MockExecutor{},
    }

    // Start consuming in background
    go consumer.StartConsuming()

    // Publish test command
    cmd := &Command{
        ID:      "test-1",
        AgentID: "test-agent",
        Command: "echo 'hello'",
    }

    err := publisher.PublishCommand(cmd)
    require.NoError(t, err)

    // Wait for result
    select {
    case result := <-resultChan:
        assert.Equal(t, cmd.ID, result.CommandID)
        assert.Equal(t, "hello\n", result.Output)
        assert.Equal(t, 0, result.ExitCode)
    case <-time.After(5 * time.Second):
        t.Fatal("timeout waiting for command result")
    }
}
```
