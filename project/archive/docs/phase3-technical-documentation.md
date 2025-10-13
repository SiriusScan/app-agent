# SiriusScan Agent Manager - Phase 3 Technical Documentation

## Overview

Phase 3 of the SiriusScan Agent Manager builds upon the foundations established in Phases 1 and 2, focusing on system orchestration, advanced monitoring, and production-ready enhancements. This phase completes the implementation of the Test Agent, adds comprehensive metrics collection, and introduces scaling capabilities for enterprise deployments.

## Project Structure Additions

```
.
├── cmd/
│   ├── metrics/         # Metrics collection utilities
│   │   └── main.go      # Standalone metrics processor
├── internal/
│   ├── metrics/         # Metrics implementation
│   │   ├── collector.go # Metrics data collection
│   │   └── exporter.go  # Prometheus metrics export
│   ├── orchestration/   # Agent orchestration
│   │   ├── scheduler.go # Command scheduling
│   │   └── balancer.go  # Load balancing
│   └── admin/           # Administrative API
│       ├── server.go    # Admin HTTP server
│       ├── handlers.go  # Admin API handlers
│       └── metrics.go   # Metrics endpoints
├── pkg/
│   ├── simulation/      # Test load simulation
│   │   ├── generator.go # Test command generator
│   │   └── runner.go    # Simulation runner
│   └── reporting/       # Reporting utilities
│       ├── formatter.go # Report formatting
│       └── exporter.go  # Report export functions
├── ui/                  # Admin dashboard (minimal)
│   ├── templates/       # HTML templates
│   ├── static/          # Static assets
│   └── server.go        # UI web server
└── deployment/          # Deployment configurations
    ├── docker/          # Docker configurations
    │   └── compose.yaml # Docker Compose config
    └── kubernetes/      # Kubernetes manifests
        ├── deployment.yaml  # K8s deployment
        └── service.yaml     # K8s service
```

## Core Components

### 1. Test Agent Enhancements (`cmd/agent/main.go`)

Extends the test agent with advanced capabilities for load testing and system validation.

#### Key Enhancements

- Advanced command simulation with realistic workloads
- Performance metric collection and reporting
- Configurable command execution strategies
- Failure simulation and recovery testing
- Load balancing awareness

Key Functions:

```go
func simulateRealisticLoad(ctx context.Context, config *SimulationConfig) error
func collectPerformanceMetrics(ctx context.Context) *PerformanceReport
func simulateFailureModes(ctx context.Context, failureConfig *FailureConfig) error
```

Notable Features:

- Configurable simulation parameters
- Detailed timing and performance metrics
- Realistic failure scenarios
- Programmable agent behavior
- Comprehensive logging and reporting

### 2. Metrics Collection System (`internal/metrics/collector.go`)

Collects, aggregates, and exports system metrics for monitoring and alerting.

#### Key Interfaces and Types

```go
type MetricsCollector interface {
    CollectMetrics(ctx context.Context) error
    RegisterMetric(name string, metricType MetricType, help string) error
    IncrementCounter(name string, labels map[string]string, value float64) error
    ObserveHistogram(name string, labels map[string]string, value float64) error
    SetGauge(name string, labels map[string]string, value float64) error
}

type MetricType string

const (
    MetricTypeCounter   MetricType = "counter"
    MetricTypeGauge     MetricType = "gauge"
    MetricTypeHistogram MetricType = "histogram"
    MetricTypeSummary   MetricType = "summary"
)

type PrometheusCollector struct {
    registry *prometheus.Registry
    metrics  map[string]interface{}
    mu       sync.RWMutex
}
```

Key Functions:

```go
func NewPrometheusCollector() *PrometheusCollector
func (c *PrometheusCollector) RegisterMetric(name string, metricType MetricType, help string) error
func (c *PrometheusCollector) IncrementCounter(name string, labels map[string]string, value float64) error
func (c *PrometheusCollector) ObserveHistogram(name string, labels map[string]string, value float64) error
func (c *PrometheusCollector) SetGauge(name string, labels map[string]string, value float64) error
func (c *PrometheusCollector) CollectMetrics(ctx context.Context) error
```

Default Metrics:

- `agent_commands_received`: Counter of received commands by type
- `agent_commands_executed`: Counter of executed commands by type and status
- `command_execution_time`: Histogram of command execution times
- `agent_connections`: Gauge of currently connected agents
- `message_queue_depth`: Gauge of RabbitMQ queue depths
- `valkey_operations`: Counter of Valkey operations by type
- `grpc_requests`: Counter of gRPC requests by method
- `grpc_errors`: Counter of gRPC errors by method and code
- `system_memory_usage`: Gauge of memory usage
- `system_cpu_usage`: Gauge of CPU usage

### 3. Command Orchestration System (`internal/orchestration/scheduler.go`)

Intelligent scheduling and distribution of commands to available agents.

#### Key Interfaces and Types

```go
type CommandScheduler interface {
    ScheduleCommand(ctx context.Context, cmd *models.Command) error
    CancelCommand(ctx context.Context, commandID string) error
    GetCommandStatus(ctx context.Context, commandID string) (*models.CommandStatus, error)
}

type AgentSelector interface {
    SelectAgentForCommand(ctx context.Context, cmd *models.Command, availableAgents []*models.AgentInfo) (string, error)
}

type DefaultScheduler struct {
    logger        *zap.Logger
    commandStore  store.CommandStore
    agentManager  *agent.Manager
    agentSelector AgentSelector
    metrics       metrics.MetricsCollector
}
```

Key Functions:

```go
func NewDefaultScheduler(logger *zap.Logger, commandStore store.CommandStore, agentManager *agent.Manager, selector AgentSelector, metrics metrics.MetricsCollector) *DefaultScheduler
func (s *DefaultScheduler) ScheduleCommand(ctx context.Context, cmd *models.Command) error
func (s *DefaultScheduler) CancelCommand(ctx context.Context, commandID string) error
func (s *DefaultScheduler) GetCommandStatus(ctx context.Context, commandID string) (*models.CommandStatus, error)
```

Scheduling Strategies:

- **Capability-based**: Match commands to agents with required capabilities
- **Load-balanced**: Distribute commands evenly across available agents
- **Locality-aware**: Prefer agents in the same network segment as targets
- **Priority-based**: Execute high-priority commands before lower priority
- **Deadline-aware**: Schedule commands based on time constraints

### 4. Load Balancing System (`internal/orchestration/balancer.go`)

Distributes commands across available agents for optimal resource utilization.

#### Key Interfaces and Types

```go
type LoadBalancer interface {
    RegisterAgent(agent *models.AgentInfo) error
    UnregisterAgent(agentID string) error
    UpdateAgentLoad(agentID string, load float64) error
    GetAgentsForCommand(cmd *models.Command) ([]*models.AgentInfo, error)
}

type StrategyType string

const (
    StrategyRoundRobin StrategyType = "round_robin"
    StrategyLeastLoaded StrategyType = "least_loaded"
    StrategyCapabilityMatch StrategyType = "capability_match"
)

type DefaultLoadBalancer struct {
    logger   *zap.Logger
    strategy StrategyType
    agents   map[string]*models.AgentInfo
    mu       sync.RWMutex
}
```

Key Functions:

```go
func NewDefaultLoadBalancer(logger *zap.Logger, strategy StrategyType) *DefaultLoadBalancer
func (b *DefaultLoadBalancer) RegisterAgent(agent *models.AgentInfo) error
func (b *DefaultLoadBalancer) UnregisterAgent(agentID string) error
func (b *DefaultLoadBalancer) UpdateAgentLoad(agentID string, load float64) error
func (b *DefaultLoadBalancer) GetAgentsForCommand(cmd *models.Command) ([]*models.AgentInfo, error)
```

Load Balancing Algorithms:

- **Round Robin**: Distribute commands sequentially
- **Least Loaded**: Assign to agent with lowest current load
- **Capability Matching**: Match command requirements with agent capabilities
- **Weighted**: Consider agent capacity during selection
- **Adaptive**: Adjust based on historical performance

### 5. Administrative API (`internal/admin/server.go`)

Provides a REST API for administrative operations and monitoring.

#### Key Interfaces and Types

```go
type AdminServer struct {
    logger        *zap.Logger
    config        *config.Config
    commandStore  store.CommandStore
    agentManager  *agent.Manager
    scheduler     orchestration.CommandScheduler
    metrics       metrics.MetricsCollector
    router        *http.ServeMux
    server        *http.Server
}
```

Key Endpoints:

- `GET /api/agents`: List all registered agents
- `GET /api/agents/:id`: Get agent details
- `GET /api/commands`: List commands with filtering
- `POST /api/commands`: Create a new command
- `GET /api/commands/:id`: Get command details
- `PUT /api/commands/:id/cancel`: Cancel a command
- `GET /api/metrics`: Get system metrics
- `GET /api/status`: Get system status information
- `POST /api/admin/shutdown`: Trigger graceful shutdown
- `POST /api/admin/maintenance`: Enter/exit maintenance mode

Key Functions:

```go
func NewAdminServer(logger *zap.Logger, config *config.Config, commandStore store.CommandStore, agentManager *agent.Manager, scheduler orchestration.CommandScheduler, metrics metrics.MetricsCollector) *AdminServer
func (s *AdminServer) Start(ctx context.Context) error
func (s *AdminServer) Stop() error
func (s *AdminServer) registerRoutes()
```

Authentication and Security:

- API key-based authentication
- Role-based access control
- TLS encryption
- Rate limiting
- Request logging

### 6. Dashboard UI (`ui/server.go`)

Simple web dashboard for system monitoring and management.

#### Key Features

- Agent status overview
- Command execution statistics
- Real-time metrics visualization
- Log viewing and filtering
- Command submission interface
- System status indicators

Key Technologies:

- Go templates for HTML rendering
- Simple embedded HTTP server
- WebSockets for real-time updates
- Bootstrap for responsive design
- Chart.js for metric visualization

### 7. Enhanced Agent Management (`internal/agent/manager.go`)

Extends the agent manager with advanced capabilities for large-scale deployments.

#### Key Interfaces and Types

```go
type Manager struct {
    logger     *zap.Logger
    config     *config.Config
    store      store.KVStore
    agents     map[string]*AgentConnection
    metrics    metrics.MetricsCollector
    balancer   orchestration.LoadBalancer
    mu         sync.RWMutex
}

type AgentFilter struct {
    Status        pb.AgentStatus
    Capabilities  map[string]string
    MinVersion    string
    MaxLoad       float64
}
```

Key Functions:

```go
func NewManager(logger *zap.Logger, config *config.Config, store store.KVStore, metrics metrics.MetricsCollector, balancer orchestration.LoadBalancer) *Manager
func (m *Manager) RegisterAgent(agent *AgentConnection) error
func (m *Manager) UnregisterAgent(agentID string) error
func (m *Manager) GetAgent(agentID string) (*AgentConnection, error)
func (m *Manager) FindAgents(filter *AgentFilter) ([]*AgentConnection, error)
func (m *Manager) SendCommandToAgent(ctx context.Context, agentID string, cmd *pb.CommandMessage) error
func (m *Manager) BroadcastCommand(ctx context.Context, cmd *pb.CommandMessage, filter *AgentFilter) error
```

Advanced Features:

- Agent capability discovery and tracking
- Version-based agent targeting
- Agent health monitoring and reporting
- Background agent pruning (remove disconnected)
- Agent load tracking and reporting
- Broadcast commands to agent groups

## System Architecture Enhancements

### High-Availability Design

```
                         +-------------------+
                         |  Load Balancer    |
                         +-------------------+
                                  |
                 +----------------+----------------+
                 |                                 |
        +--------v--------+              +---------v-------+
        |  App Instance 1 |              |  App Instance 2 |
        +--------+--------+              +---------+-------+
                 |                                 |
        +--------v--------+              +---------v-------+
        |    RabbitMQ     |<------------>|     RabbitMQ    |
        +--------+--------+              +---------+-------+
                 |                                 |
        +--------v--------+              +---------v-------+
        |     Valkey      |<------------>|     Valkey      |
        +-----------------+              +-----------------+
                                  ^
                                  |
                         +--------+--------+
                         |  Monitoring     |
                         |     System      |
                         +-----------------+
```

1. **Load Balancer**: Routes traffic to available instances
2. **Clustered RabbitMQ**: Replicated message queues
3. **Clustered Valkey**: Distributed key-value store
4. **Monitoring System**: Prometheus and Grafana integration

### Scaling Strategy

- **Horizontal Scaling**: Multiple application instances
- **Vertical Scaling**: Increase resources for high-load components
- **Cluster Scaling**: Add nodes to RabbitMQ and Valkey clusters
- **Auto-Scaling**: Dynamic scaling based on load metrics

### Fault Tolerance

- **Service Discovery**: Dynamic service registration and discovery
- **Circuit Breakers**: Prevent cascading failures
- **Rate Limiting**: Protect services from overload
- **Retry Mechanisms**: Automatic retry with backoff
- **Fallback Mechanisms**: Graceful degradation under load

## Deployment

### Docker Deployment

```yaml
# docker-compose.yml
version: "3.8"

services:
  app-agent:
    image: siriusscan/app-agent:latest
    restart: always
    ports:
      - "50051:50051"
      - "8080:8080"
    environment:
      - APP_VERSION=1.0.0
      - APP_ENV=production
      - GRPC_SERVER_ADDR=:50051
      - SIRIUS_LOG_LEVEL=info
      - SIRIUS_LOG_FORMAT=json
      - METRICS_ADDR=:2112
      - ADMIN_API_ADDR=:8080
      - ADMIN_API_KEY=${ADMIN_API_KEY}
      - TLS_ENABLED=true
      - TLS_CERT_FILE=/certs/server.crt
      - TLS_KEY_FILE=/certs/server.key
    volumes:
      - ./certs:/certs
    depends_on:
      - rabbitmq
      - valkey

  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"
      - "15672:15672"
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

  valkey:
    image: valkey/valkey:latest
    ports:
      - "6379:6379"
    volumes:
      - valkey_data:/data
    command: ["valkey-server", "/etc/valkey/valkey.conf"]

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
    depends_on:
      - prometheus

volumes:
  rabbitmq_data:
  valkey_data:
  prometheus_data:
  grafana_data:
```

### Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-agent
  namespace: sirius-scan
spec:
  replicas: 3
  selector:
    matchLabels:
      app: app-agent
  template:
    metadata:
      labels:
        app: app-agent
    spec:
      containers:
        - name: app-agent
          image: siriusscan/app-agent:latest
          ports:
            - containerPort: 50051
              name: grpc
            - containerPort: 8080
              name: http
            - containerPort: 2112
              name: metrics
          env:
            - name: APP_VERSION
              value: "1.0.0"
            - name: APP_ENV
              value: "production"
            - name: GRPC_SERVER_ADDR
              value: ":50051"
            - name: SIRIUS_LOG_LEVEL
              value: "info"
            - name: SIRIUS_LOG_FORMAT
              value: "json"
            - name: METRICS_ADDR
              value: ":2112"
            - name: ADMIN_API_ADDR
              value: ":8080"
            - name: ADMIN_API_KEY
              valueFrom:
                secretKeyRef:
                  name: app-agent-secrets
                  key: admin-api-key
            - name: TLS_ENABLED
              value: "true"
          volumeMounts:
            - name: tls-certs
              mountPath: /certs
          resources:
            limits:
              cpu: "1"
              memory: "1Gi"
            requests:
              cpu: "500m"
              memory: "512Mi"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
      volumes:
        - name: tls-certs
          secret:
            secretName: app-agent-tls
```

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: app-agent
  namespace: sirius-scan
spec:
  selector:
    app: app-agent
  ports:
    - name: grpc
      port: 50051
      targetPort: 50051
    - name: http
      port: 8080
      targetPort: 8080
    - name: metrics
      port: 2112
      targetPort: 2112
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-agent-admin
  namespace: sirius-scan
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  rules:
    - host: admin.siriusscan.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: app-agent
                port:
                  name: http
  tls:
    - hosts:
        - admin.siriusscan.com
      secretName: siriusscan-tls
```

## Configuration

Additional environment variables for Phase 3:

```bash
# Admin API Configuration
ADMIN_API_ADDR=:8080            # Admin API address
ADMIN_API_KEY=                  # Admin API authentication key
ADMIN_TLS_ENABLED=true          # Enable TLS for Admin API
ADMIN_TLS_CERT_FILE=            # Admin API TLS certificate
ADMIN_TLS_KEY_FILE=             # Admin API TLS key

# Metrics Configuration
METRICS_RETENTION_DAYS=30       # Metrics retention period
METRICS_SCRAPE_INTERVAL=15s     # Prometheus scrape interval
ENABLE_DETAILED_METRICS=true    # Enable detailed metrics collection

# Orchestration Configuration
MAX_AGENT_LOAD=0.8              # Maximum agent load (0.0-1.0)
LOAD_BALANCING_STRATEGY=least_loaded  # Default strategy
COMMAND_QUEUE_SIZE=1000         # Command queue size
AGENT_TIMEOUT=300               # Agent timeout in seconds
MAINTENANCE_MODE=false          # System maintenance mode
```

## Monitoring Setup

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "app-agent"
    static_configs:
      - targets: ["app-agent:2112"]

  - job_name: "rabbitmq"
    static_configs:
      - targets: ["rabbitmq-exporter:9419"]

  - job_name: "valkey"
    static_configs:
      - targets: ["valkey-exporter:9121"]
```

### Grafana Dashboards

1. **System Overview**
   - Agent connection status
   - Command execution rates
   - Error rates
   - Resource utilization
2. **Queue Monitoring**
   - Queue depths
   - Message processing rates
   - Processing latency
   - Dead-letter monitoring
3. **Agent Performance**
   - Command execution times
   - Agent load distribution
   - Connection stability
   - Error rates by agent

## Performance Benchmarks

### Command Processing Capacity

| Deployment Size | Agents | Commands/second | Max Concurrent Commands | Memory Usage |
| --------------- | ------ | --------------- | ----------------------- | ------------ |
| Small           | 1-10   | 50              | 100                     | 512 MB       |
| Medium          | 10-50  | 200             | 500                     | 1 GB         |
| Large           | 50-200 | 1,000           | 2,000                   | 2 GB         |
| Enterprise      | 200+   | 5,000+          | 10,000+                 | 4+ GB        |

### Scaling Characteristics

| Component         | Scaling Method | Bottleneck Factor | Recommended Scaling Trigger |
| ----------------- | -------------- | ----------------- | --------------------------- |
| GRPC Server       | Horizontal     | CPU               | 70% CPU utilization         |
| Command Processor | Horizontal     | Memory            | 80% Memory utilization      |
| RabbitMQ          | Cluster        | Message Rate      | 5,000 msgs/sec per node     |
| Valkey            | Cluster        | Operations/second | 10,000 ops/sec per node     |
| Admin API         | Horizontal     | Requests/second   | 1,000 req/sec per instance  |

## Development Guide

### Implementing a Custom Metrics Collector

```go
// 1. Create a custom collector
type CustomMetricsCollector struct {
    logger *zap.Logger
    // Additional fields
}

// 2. Implement the MetricsCollector interface
func (c *CustomMetricsCollector) RegisterMetric(name string, metricType metrics.MetricType, help string) error {
    // Implementation
}

func (c *CustomMetricsCollector) IncrementCounter(name string, labels map[string]string, value float64) error {
    // Implementation
}

func (c *CustomMetricsCollector) ObserveHistogram(name string, labels map[string]string, value float64) error {
    // Implementation
}

func (c *CustomMetricsCollector) SetGauge(name string, labels map[string]string, value float64) error {
    // Implementation
}

func (c *CustomMetricsCollector) CollectMetrics(ctx context.Context) error {
    // Implementation
}

// 3. Register the collector
metricsCollector := &CustomMetricsCollector{
    logger: logger.Named("custom_metrics"),
}

// 4. Register default metrics
metricsCollector.RegisterMetric("command_processing_time", metrics.MetricTypeHistogram, "Time taken to process commands")
```

### Custom Load Balancing Strategy

```go
// 1. Define custom strategy
type CustomLoadBalancer struct {
    logger *zap.Logger
    agents map[string]*models.AgentInfo
    mu     sync.RWMutex
}

// 2. Implement the LoadBalancer interface
func (b *CustomLoadBalancer) RegisterAgent(agent *models.AgentInfo) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.agents[agent.ID] = agent
    return nil
}

func (b *CustomLoadBalancer) UnregisterAgent(agentID string) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    delete(b.agents, agentID)
    return nil
}

func (b *CustomLoadBalancer) UpdateAgentLoad(agentID string, load float64) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    if agent, exists := b.agents[agentID]; exists {
        agent.CurrentLoad = load
    }
    return nil
}

func (b *CustomLoadBalancer) GetAgentsForCommand(cmd *models.Command) ([]*models.AgentInfo, error) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    // Custom agent selection logic
    var selectedAgents []*models.AgentInfo

    // Example: Select agents with specific capability and low load
    for _, agent := range b.agents {
        if capability, exists := agent.Capabilities["scan_type"]; exists &&
           capability == cmd.Params["scan_type"] &&
           agent.CurrentLoad < 0.5 {
            selectedAgents = append(selectedAgents, agent)
        }
    }

    return selectedAgents, nil
}

// 3. Register the custom balancer
balancer := &CustomLoadBalancer{
    logger: logger.Named("custom_balancer"),
    agents: make(map[string]*models.AgentInfo),
}
```

## Testing Strategy

### Load Testing

Configure the test agent for load simulation:

```go
func runLoadTest() {
    // Configure simulation
    config := &simulation.Config{
        AgentCount:       10,
        CommandsPerAgent: 100,
        CommandTypes: []string{
            models.CommandTypeScan,
            models.CommandTypeStatus,
            models.CommandTypeUpdate,
        },
        RampUpPeriod:     30 * time.Second,
        TestDuration:     5 * time.Minute,
        ReportingInterval: 5 * time.Second,
    }

    // Run simulation
    report, err := simulation.RunLoadTest(context.Background(), config)
    if err != nil {
        log.Fatalf("Failed to run load test: %v", err)
    }

    // Print report
    fmt.Printf("Load Test Report:\n")
    fmt.Printf("Total Commands: %d\n", report.TotalCommands)
    fmt.Printf("Successful: %d (%.2f%%)\n", report.SuccessfulCommands,
               float64(report.SuccessfulCommands)/float64(report.TotalCommands)*100)
    fmt.Printf("Failed: %d (%.2f%%)\n", report.FailedCommands,
               float64(report.FailedCommands)/float64(report.TotalCommands)*100)
    fmt.Printf("Average Response Time: %v\n", report.AverageResponseTime)
    fmt.Printf("95th Percentile: %v\n", report.Percentile95)
    fmt.Printf("99th Percentile: %v\n", report.Percentile99)
    fmt.Printf("Max Response Time: %v\n", report.MaxResponseTime)
    fmt.Printf("Throughput: %.2f commands/sec\n", report.CommandsPerSecond)
}
```

### Chaos Testing

Test system resilience with controlled failures:

```go
func runChaosTest() {
    // Configure chaos test
    config := &chaos.Config{
        TargetServices: []string{"app-agent", "rabbitmq", "valkey"},
        TestDuration:   30 * time.Minute,
        Scenarios: []chaos.Scenario{
            {
                Name:        "Network Partition",
                Description: "Simulate network partition between services",
                Duration:    5 * time.Minute,
            },
            {
                Name:        "Service Restart",
                Description: "Restart services randomly",
                Duration:    2 * time.Minute,
            },
            {
                Name:        "Resource Exhaustion",
                Description: "Simulate CPU/memory pressure",
                Duration:    10 * time.Minute,
            },
        },
    }

    // Run chaos test
    report, err := chaos.RunTest(context.Background(), config)
    if err != nil {
        log.Fatalf("Failed to run chaos test: %v", err)
    }

    // Print report
    fmt.Printf("Chaos Test Report:\n")
    for _, result := range report.ScenarioResults {
        fmt.Printf("Scenario: %s\n", result.Name)
        fmt.Printf("  Duration: %v\n", result.Duration)
        fmt.Printf("  Recovery Time: %v\n", result.RecoveryTime)
        fmt.Printf("  Commands During Chaos: %d\n", result.CommandsDuringChaos)
        fmt.Printf("  Success Rate: %.2f%%\n", result.SuccessRate*100)
        fmt.Printf("  System Stability: %.2f%%\n", result.SystemStability*100)
    }
}
```

## Future Considerations

Phase 3 completes the core functionality of the SiriusScan Agent Manager, but future enhancements could include:

1. **Enhanced Security**

   - Certificate rotation
   - Advanced authentication mechanisms
   - Security vulnerability scanning

2. **Extended Analytics**

   - Machine learning for anomaly detection
   - Predictive scaling
   - Historical trend analysis

3. **External Integrations**

   - SIEM integration
   - Ticketing system integration
   - CI/CD pipeline integration

4. **Multi-Region Support**

   - Geographic distribution of agents
   - Cross-region coordination
   - Region-aware command routing

5. **Advanced UI**
   - Interactive command builder
   - Custom dashboard creation
   - Advanced visualization options

## Appendix

### System Architecture Diagram

```
                      +------------------+
                      |    UI / Client   |
                      +------------------+
                               |
                     +---------v---------+
                     |  RabbitMQ Queues  |
                     +---------+---------+
                               |
 +------------+      +---------v---------+     +---------------+
 |            |      |                   |     |               |
 | Prometheus +<-----+  Agent Manager    +---->+ Valkey Store  |
 |            |      |                   |     |               |
 +------------+      +---------+---------+     +---------------+
                               |
                     +---------v---------+
                     |  gRPC Interface   |
                     +---------+---------+
                               |
          +-------------------+--------------------+
          |                   |                    |
+---------v---------+ +-------v-------+   +--------v--------+
|                   | |               |   |                 |
|    Agent 1        | |    Agent 2    |   |     Agent N     |
|                   | |               |   |                 |
+-------------------+ +---------------+   +-----------------+
```

### Performance Optimization Tips

1. **RabbitMQ Optimizations**

   - Use durable queues for critical messages
   - Configure appropriate QoS settings
   - Use prefetch counts to manage consumer load
   - Enable publisher confirms

2. **Valkey Optimizations**

   - Use appropriate data structures
   - Set realistic TTLs for ephemeral data
   - Configure proper eviction policies
   - Monitor memory usage closely

3. **Agent Communication**

   - Use bi-directional streams for efficiency
   - Implement backpressure mechanisms
   - Batch status updates when possible
   - Use compression for large payloads

4. **Resource Management**
   - Implement graceful degradation under load
   - Configure appropriate resource limits
   - Use resource quotas in Kubernetes
   - Implement circuit breakers for critical services

```

```
