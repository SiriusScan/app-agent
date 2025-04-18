package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/SiriusScan/app-agent/internal/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Constants for result retrieval
const (
	OutputFormatJSON  = "json"
	OutputFormatTable = "table"
	OutputFormatCSV   = "csv"
	OutputFormatPlain = "plain"

	// Default Valkey key prefixes from the server
	TaskKeyPrefix         = "task:"
	TaskResultKeyPrefix   = "task-result:"
	AgentKeyPrefix        = "agent:"
	AgentCommandKeyPrefix = "agent-command:"
)

// Formats the time specified in Unix timestamp to a human-readable format
func formatTime(timestamp int64) string {
	if timestamp == 0 {
		return "N/A"
	}
	t := time.Unix(timestamp, 0)
	return t.Format("2006-01-02 15:04:05")
}

// TaskResult represents a task execution result
type TaskResult struct {
	CommandID   string            `json:"command_id"`
	Status      string            `json:"status"`
	Data        map[string]string `json:"data"`
	Error       string            `json:"error,omitempty"`
	CompletedAt int64             `json:"completed_at"`
	AgentID     string            `json:"agent_id"`
}

// ResultRetriever manages result retrieval operations
type ResultRetriever struct {
	store      store.KVStore
	logger     *zap.Logger
	format     string
	commandID  string
	agentID    string
	status     string
	monitor    bool
	refreshInt time.Duration
	maxResults int
	outputFile string
}

// NewResultRetriever creates a new result retriever
func NewResultRetriever(
	store store.KVStore,
	logger *zap.Logger,
	format string,
	commandID string,
	agentID string,
	status string,
	monitor bool,
	refreshInt time.Duration,
	maxResults int,
	outputFile string,
) *ResultRetriever {
	return &ResultRetriever{
		store:      store,
		logger:     logger,
		format:     format,
		commandID:  commandID,
		agentID:    agentID,
		status:     status,
		monitor:    monitor,
		refreshInt: refreshInt,
		maxResults: maxResults,
		outputFile: outputFile,
	}
}

// Retrieve performs the result retrieval operation
func (r *ResultRetriever) Retrieve(ctx context.Context) error {
	if r.monitor {
		return r.monitorResults(ctx)
	}
	return r.fetchResults(ctx)
}

// fetchResults retrieves results according to filters
func (r *ResultRetriever) fetchResults(ctx context.Context) error {
	results, err := r.getFilteredResults(ctx)
	if err != nil {
		return err
	}

	return r.outputResults(results)
}

// monitorResults continuously retrieves and displays results
func (r *ResultRetriever) monitorResults(ctx context.Context) error {
	r.logger.Info("Starting result monitoring mode",
		zap.Duration("refresh_interval", r.refreshInt))

	ticker := time.NewTicker(r.refreshInt)
	defer ticker.Stop()

	// Create a signal channel to handle Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Show initial results
	if err := r.fetchResults(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.fetchResults(ctx); err != nil {
				r.logger.Error("Failed to fetch results during monitoring",
					zap.Error(err))
				continue
			}
		case <-sigCh:
			r.logger.Info("Monitoring stopped by user")
			return nil
		}
	}
}

// getFilteredResults retrieves and filters results from storage
func (r *ResultRetriever) getFilteredResults(ctx context.Context) ([]TaskResult, error) {
	var results []TaskResult

	// If a specific command ID is provided, retrieve just that result
	if r.commandID != "" {
		result, err := r.getResultByCommandID(ctx, r.commandID)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
		return results, nil
	}

	// Otherwise, scan through all results with pattern matching
	// In a real implementation, we would query a list of keys or use a specialized
	// query method, but for simplicity, we'll mock this for now
	mockResults := []TaskResult{
		{
			CommandID: "task-20240601-123456-1",
			Status:    "completed",
			Data: map[string]string{
				"output":    "total 24\ndrwxr-xr-x  8 user group 4096 Jun 1 12:34 .",
				"exit_code": "0",
			},
			Error:       "",
			CompletedAt: time.Now().Add(-time.Minute).Unix(),
			AgentID:     "test-agent-1",
		},
		{
			CommandID:   "task-20240601-123457-1",
			Status:      "failed",
			Data:        map[string]string{},
			Error:       "command timeout exceeded",
			CompletedAt: time.Now().Add(-2 * time.Minute).Unix(),
			AgentID:     "test-agent-2",
		},
		{
			CommandID: "task-20240601-123458-1",
			Status:    "completed",
			Data: map[string]string{
				"cpu":    "45.2",
				"memory": "1256MB",
				"disk":   "10.5GB free",
				"uptime": "3d 12h 30m",
			},
			Error:       "",
			CompletedAt: time.Now().Add(-3 * time.Minute).Unix(),
			AgentID:     "test-agent-1",
		},
	}

	// Apply filters
	for _, result := range mockResults {
		if r.agentID != "" && result.AgentID != r.agentID {
			continue
		}
		if r.status != "" && result.Status != r.status {
			continue
		}
		results = append(results, result)
	}

	// Sort by completed time (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CompletedAt > results[j].CompletedAt
	})

	// Limit results if needed
	if r.maxResults > 0 && len(results) > r.maxResults {
		results = results[:r.maxResults]
	}

	return results, nil
}

// getResultByCommandID retrieves a specific result by command ID
func (r *ResultRetriever) getResultByCommandID(ctx context.Context, commandID string) (TaskResult, error) {
	key := TaskResultKeyPrefix + commandID
	resp, err := r.store.GetValue(ctx, key)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to retrieve result for command %s: %w", commandID, err)
	}

	var result TaskResult
	if err := json.Unmarshal([]byte(resp.Message.Value), &result); err != nil {
		return TaskResult{}, fmt.Errorf("failed to parse result data: %w", err)
	}

	return result, nil
}

// outputResults formats and outputs results according to the specified format
func (r *ResultRetriever) outputResults(results []TaskResult) error {
	if len(results) == 0 {
		fmt.Println("No results found matching the specified criteria")
		return nil
	}

	var output string
	var err error

	switch r.format {
	case OutputFormatJSON:
		output, err = r.formatJSON(results)
	case OutputFormatTable:
		output, err = r.formatTable(results)
	case OutputFormatCSV:
		output, err = r.formatCSV(results)
	case OutputFormatPlain:
		output, err = r.formatPlain(results)
	default:
		return fmt.Errorf("unsupported output format: %s", r.format)
	}

	if err != nil {
		return err
	}

	// If monitoring mode is enabled, clear screen before output
	if r.monitor {
		fmt.Print("\033[H\033[2J") // ANSI escape sequence to clear screen
		fmt.Printf("Result Monitor - Last Updated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	// Output to file if specified
	if r.outputFile != "" {
		if err := os.WriteFile(r.outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}
		fmt.Printf("Results written to %s\n", r.outputFile)
	} else {
		// Otherwise print to stdout
		fmt.Println(output)
	}

	return nil
}

// formatJSON formats results as JSON
func (r *ResultRetriever) formatJSON(results []TaskResult) (string, error) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format results as JSON: %w", err)
	}
	return string(data), nil
}

// formatTable formats results as a table
func (r *ResultRetriever) formatTable(results []TaskResult) (string, error) {
	var sb strings.Builder

	// Create a table
	table := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', tabwriter.AlignRight)

	// Write header
	fmt.Fprintln(table, "COMMAND ID\tAGENT ID\tSTATUS\tCOMPLETED AT\tERROR")
	fmt.Fprintln(table, "-----------\t--------\t------\t------------\t-----")

	// Write rows
	for _, result := range results {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			result.CommandID,
			result.AgentID,
			result.Status,
			formatTime(result.CompletedAt),
			result.Error,
		)
	}

	table.Flush()

	// Add data section for detailed view of the first result or if only one result
	if len(results) == 1 {
		result := results[0]
		sb.WriteString("\nCommand Details:\n")

		if len(result.Data) > 0 {
			sb.WriteString("\nData:\n")
			for k, v := range result.Data {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
			}
		}
	}

	return sb.String(), nil
}

// formatCSV formats results as CSV
func (r *ResultRetriever) formatCSV(results []TaskResult) (string, error) {
	var sb strings.Builder

	// Write header
	sb.WriteString("command_id,agent_id,status,completed_at,error\n")

	// Write rows
	for _, result := range results {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			result.CommandID,
			result.AgentID,
			result.Status,
			formatTime(result.CompletedAt),
			result.Error,
		))
	}

	return sb.String(), nil
}

// formatPlain formats results as plain text
func (r *ResultRetriever) formatPlain(results []TaskResult) (string, error) {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	// Write header
	fmt.Fprintln(w, "COMMAND ID\tAGENT ID\tSTATUS\tCOMPLETED AT\tERROR")
	fmt.Fprintln(w, "----------\t--------\t------\t------------\t-----")

	// Write rows
	for _, result := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			result.CommandID,
			result.AgentID,
			result.Status,
			formatTime(result.CompletedAt),
			result.Error,
		)
	}

	w.Flush()

	return sb.String(), nil
}

func main() {
	// Parse command line flags
	commandID := flag.String("command", "", "Filter by command ID")
	agentID := flag.String("agent", "", "Filter by agent ID")
	status := flag.String("status", "", "Filter by status (completed, failed, etc.)")
	format := flag.String("format", "table", "Output format (json, table, csv, plain)")
	monitor := flag.Bool("monitor", false, "Enable monitoring mode (continuous refresh)")
	refreshInt := flag.Int("refresh", 5, "Refresh interval in seconds for monitoring mode")
	maxResults := flag.Int("limit", 10, "Maximum number of results to show (0 for unlimited)")
	outputFile := flag.String("output", "", "Output file (omit for stdout)")
	_ = flag.String("valkey", "localhost:6379", "Valkey server address")

	flag.Parse()

	// Initialize logger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	logger, _ := config.Build()
	defer logger.Sync()

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Shutting down gracefully...")
		cancel()
	}()

	// Create store instance
	// In a real implementation, this would connect to actual Valkey
	// For this example, we'll use a mock implementation
	mockStore := store.NewMockStore()

	// Create result retriever
	retriever := NewResultRetriever(
		mockStore,
		logger,
		*format,
		*commandID,
		*agentID,
		*status,
		*monitor,
		time.Duration(*refreshInt)*time.Second,
		*maxResults,
		*outputFile,
	)

	// Execute retrieval
	logger.Info("Starting result retrieval",
		zap.String("format", *format),
		zap.String("command_id", *commandID),
		zap.String("agent_id", *agentID),
		zap.String("status", *status))

	if err := retriever.Retrieve(ctx); err != nil {
		logger.Fatal("Failed to retrieve results", zap.Error(err))
	}

	logger.Info("Result retrieval completed successfully")
}
