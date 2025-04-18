package monitoring

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Server metrics
	connectedAgents = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sirius_connected_agents",
		Help: "Number of currently connected agents",
	})

	commandsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sirius_commands_processed_total",
		Help: "Total number of commands processed by type",
	}, []string{"type", "status"})

	commandDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sirius_command_duration_seconds",
		Help:    "Duration of command execution",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})

	// Agent metrics
	agentMemoryUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sirius_agent_memory_bytes",
		Help: "Memory usage of agents in bytes",
	}, []string{"agent_id", "type"})

	agentCPUUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sirius_agent_cpu_usage",
		Help: "CPU usage percentage of agents",
	}, []string{"agent_id"})

	// Queue metrics
	queueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sirius_command_queue_size",
		Help: "Number of commands in agent queues",
	}, []string{"agent_id"})

	// Error metrics
	errorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sirius_errors_total",
		Help: "Total number of errors by type",
	}, []string{"type"})
)

// MetricsCollector handles metric collection and updates
type MetricsCollector struct{}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// UpdateConnectedAgents updates the number of connected agents
func (m *MetricsCollector) UpdateConnectedAgents(count float64) {
	connectedAgents.Set(count)
}

// RecordCommandProcessed increments the command counter
func (m *MetricsCollector) RecordCommandProcessed(cmdType, status string) {
	commandsProcessed.WithLabelValues(cmdType, status).Inc()
}

// RecordCommandDuration records the duration of a command
func (m *MetricsCollector) RecordCommandDuration(cmdType string, duration float64) {
	commandDuration.WithLabelValues(cmdType).Observe(duration)
}

// UpdateAgentMemory updates agent memory metrics
func (m *MetricsCollector) UpdateAgentMemory(agentID string, total, free float64) {
	agentMemoryUsage.WithLabelValues(agentID, "total").Set(total)
	agentMemoryUsage.WithLabelValues(agentID, "free").Set(free)
}

// UpdateAgentCPU updates agent CPU usage metrics
func (m *MetricsCollector) UpdateAgentCPU(agentID string, usage float64) {
	agentCPUUsage.WithLabelValues(agentID).Set(usage)
}

// UpdateQueueSize updates the command queue size metric
func (m *MetricsCollector) UpdateQueueSize(agentID string, size float64) {
	queueSize.WithLabelValues(agentID).Set(size)
}

// RecordError increments the error counter
func (m *MetricsCollector) RecordError(errorType string) {
	errorCounter.WithLabelValues(errorType).Inc()
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func StartMetricsServer(addr string) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting metrics server: %v", err)
		}
	}()

	return server, nil
}
