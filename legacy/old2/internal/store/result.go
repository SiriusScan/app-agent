package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ResultStorageFormat constants
const (
	CommandResultPrefix  = "cmd:result:"        // Prefix for command results
	AgentResultPrefix    = "agent:result:"      // Prefix for agent-specific results
	ResultIndexPrefix    = "result:index:"      // Prefix for result indices
	CommandResultSetKey  = "result:command:set" // Set of all command result keys
	ResultRetentionDays  = 90                   // Default retention in days
	ResultBatchSize      = 100                  // Default batch size for operations
	CompressionThreshold = 1024 * 5             // Compress data over 5KB
)

// ResultStorage handles storing and retrieving command results
type ResultStorage struct {
	store  KVStore
	logger *zap.Logger
}

// CommandResult represents a command execution result
type CommandResult struct {
	CommandID   string            `json:"command_id"`
	AgentID     string            `json:"agent_id"`
	Status      string            `json:"status"`
	Output      string            `json:"output"`
	ExitCode    int32             `json:"exit_code"`
	Type        string            `json:"type"`
	Metadata    map[string]string `json:"metadata"`
	ReceivedAt  time.Time         `json:"received_at"`
	CompletedAt time.Time         `json:"completed_at"`
	Error       string            `json:"error"`
}

// ResultQuery defines parameters for querying results
type ResultQuery struct {
	CommandID string     `json:"command_id,omitempty"`
	AgentID   string     `json:"agent_id,omitempty"`
	Type      string     `json:"type,omitempty"`
	Status    string     `json:"status,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Page      int        `json:"page,omitempty"`
	PageSize  int        `json:"page_size,omitempty"`
	SortBy    string     `json:"sort_by,omitempty"`
	SortOrder string     `json:"sort_order,omitempty"` // asc or desc
}

// ResultStorageInterface provides an interface for storing and retrieving command results
type ResultStorageInterface interface {
	// Store stores a command result
	Store(ctx context.Context, result *CommandResult) error

	// StoreBatch stores multiple command results in a batch operation
	StoreBatch(ctx context.Context, results []*CommandResult) error

	// Get retrieves a command result by ID
	Get(ctx context.Context, commandID string) (*CommandResult, error)

	// GetByAgent retrieves all results for an agent
	GetByAgent(ctx context.Context, agentID string, query *ResultQuery) ([]*CommandResult, error)

	// Query retrieves results matching the given criteria
	Query(ctx context.Context, query *ResultQuery) ([]*CommandResult, error)

	// DeleteOlderThan deletes results older than the given duration
	DeleteOlderThan(ctx context.Context, age time.Duration) (int, error)

	// PurgeByCommandID deletes results for a specific command
	PurgeByCommandID(ctx context.Context, commandID string) error
}

// NewResultStorage creates a new ResultStorage instance
func NewResultStorage(store KVStore, logger *zap.Logger) *ResultStorage {
	return &ResultStorage{
		store:  store,
		logger: logger,
	}
}

// Store saves a command result to the KV store
func (rs *ResultStorage) Store(ctx context.Context, result *CommandResult) error {
	if result.CommandID == "" {
		return fmt.Errorf("command ID cannot be empty")
	}

	// Generate key for the result
	key := fmt.Sprintf("cmd:result:%s", result.CommandID)

	// Marshal result to JSON
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Store in KV store
	err = rs.store.SetValue(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	// Store additional index for agent-based lookups
	agentKey := fmt.Sprintf("agent:%s:cmd:%s", result.AgentID, result.CommandID)
	err = rs.store.SetValue(ctx, agentKey, result.CommandID)
	if err != nil {
		rs.logger.Warn("Failed to store agent command index",
			zap.Error(err),
			zap.String("agent_id", result.AgentID),
			zap.String("command_id", result.CommandID))
	}

	return nil
}

// Get retrieves a command result by its ID
func (rs *ResultStorage) Get(ctx context.Context, commandID string) (*CommandResult, error) {
	if commandID == "" {
		return nil, fmt.Errorf("command ID cannot be empty")
	}

	key := fmt.Sprintf("cmd:result:%s", commandID)
	resp, err := rs.store.GetValue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve result: %w", err)
	}

	if resp.Message.Value == "" {
		return nil, fmt.Errorf("result not found: %s", commandID)
	}

	var result CommandResult
	if err := json.Unmarshal([]byte(resp.Message.Value), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// ListByAgent retrieves command results for a specific agent
func (rs *ResultStorage) ListByAgent(ctx context.Context, agentID string, limit int) ([]*CommandResult, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	pattern := fmt.Sprintf("agent:%s:cmd:*", agentID)
	keys, err := rs.store.Keys(ctx, pattern)
	if err != nil {
		if strings.Contains(err.Error(), "not supported") {
			rs.logger.Warn("Keys operation not supported by underlying store",
				zap.String("agent_id", agentID),
				zap.String("pattern", pattern))
			return []*CommandResult{}, nil
		}
		return nil, fmt.Errorf("failed to list agent command keys: %w", err)
	}

	var results []*CommandResult
	for i, key := range keys {
		if limit > 0 && i >= limit {
			break
		}

		// Extract command ID from the key
		resp, err := rs.store.GetValue(ctx, key)
		if err != nil {
			rs.logger.Warn("Failed to get command ID from index",
				zap.Error(err),
				zap.String("key", key))
			continue
		}

		result, err := rs.Get(ctx, resp.Message.Value)
		if err != nil {
			rs.logger.Warn("Failed to retrieve command result",
				zap.Error(err),
				zap.String("command_id", resp.Message.Value))
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// Cleanup removes command results older than the retention period
func (rs *ResultStorage) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive")
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	pattern := "cmd:result:*"
	keys, err := rs.store.Keys(ctx, pattern)
	if err != nil {
		if strings.Contains(err.Error(), "not supported") {
			rs.logger.Warn("Keys operation not supported by underlying store",
				zap.Int("retention_days", retentionDays),
				zap.String("pattern", pattern))
			return 0, nil
		}
		return 0, fmt.Errorf("failed to list command result keys: %w", err)
	}

	deletedCount := 0
	for _, key := range keys {
		resp, err := rs.store.GetValue(ctx, key)
		if err != nil {
			rs.logger.Warn("Failed to get command result",
				zap.Error(err),
				zap.String("key", key))
			continue
		}

		var result CommandResult
		if err := json.Unmarshal([]byte(resp.Message.Value), &result); err != nil {
			rs.logger.Warn("Failed to unmarshal result",
				zap.Error(err),
				zap.String("key", key))
			continue
		}

		if result.ReceivedAt.Before(cutoff) {
			// Delete the result
			if err := rs.store.Del(ctx, key); err != nil {
				if strings.Contains(err.Error(), "not supported") {
					rs.logger.Warn("Del operation not supported by underlying store",
						zap.String("key", key))
					continue
				}
				rs.logger.Warn("Failed to delete expired result",
					zap.Error(err),
					zap.String("key", key))
				continue
			}

			// Delete the agent index
			agentKey := fmt.Sprintf("agent:%s:cmd:%s", result.AgentID, result.CommandID)
			if err := rs.store.Del(ctx, agentKey); err != nil && !strings.Contains(err.Error(), "not supported") {
				rs.logger.Warn("Failed to delete agent command index",
					zap.Error(err),
					zap.String("key", agentKey))
			}

			deletedCount++
		}
	}

	return deletedCount, nil
}

// StoreBatch implements ResultStorage.StoreBatch
func (s *ResultStorage) StoreBatch(ctx context.Context, results []*CommandResult) error {
	if len(results) == 0 {
		return nil
	}

	// Process each result individually
	// In a real implementation, we would use a batch/pipeline operation for better performance
	for _, result := range results {
		if err := s.Store(ctx, result); err != nil {
			return fmt.Errorf("failed to store result batch: %w", err)
		}
	}

	return nil
}

// GetByAgent implements ResultStorage.GetByAgent
func (s *ResultStorage) GetByAgent(ctx context.Context, agentID string, query *ResultQuery) ([]*CommandResult, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	// Set default query parameters if not provided
	if query == nil {
		query = &ResultQuery{}
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	// Get all results for this agent
	allResults, err := s.ListByAgent(ctx, agentID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent results: %w", err)
	}

	// Apply filters
	var filteredResults []*CommandResult
	for _, result := range allResults {
		if s.matchesQuery(result, query) {
			filteredResults = append(filteredResults, result)
		}
	}

	// Apply sorting
	sortResults(filteredResults, query.SortBy, query.SortOrder)

	// Apply pagination
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start >= len(filteredResults) {
		return []*CommandResult{}, nil
	}
	if end > len(filteredResults) {
		end = len(filteredResults)
	}

	return filteredResults[start:end], nil
}

// Query implements ResultStorage.Query
func (s *ResultStorage) Query(ctx context.Context, query *ResultQuery) ([]*CommandResult, error) {
	// Set default query parameters if not provided
	if query == nil {
		query = &ResultQuery{}
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	// If agent ID is specified, use GetByAgent
	if query.AgentID != "" {
		return s.GetByAgent(ctx, query.AgentID, query)
	}

	// Otherwise, we need to scan all command results
	pattern := "cmd:result:*"
	keys, err := s.store.Keys(ctx, pattern)
	if err != nil {
		if strings.Contains(err.Error(), "not supported") {
			s.logger.Warn("Keys operation not supported by underlying store",
				zap.String("pattern", pattern))
			return []*CommandResult{}, nil
		}
		return nil, fmt.Errorf("failed to list command keys: %w", err)
	}

	// Fetch all matching results
	var allResults []*CommandResult
	for _, key := range keys {
		// Extract command ID from key
		cmdID := strings.TrimPrefix(key, "cmd:result:")
		result, err := s.Get(ctx, cmdID)
		if err != nil {
			s.logger.Warn("Failed to retrieve result during query",
				zap.Error(err),
				zap.String("command_id", cmdID))
			continue
		}

		if s.matchesQuery(result, query) {
			allResults = append(allResults, result)
		}
	}

	// Apply sorting
	sortResults(allResults, query.SortBy, query.SortOrder)

	// Apply pagination
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start >= len(allResults) {
		return []*CommandResult{}, nil
	}
	if end > len(allResults) {
		end = len(allResults)
	}

	return allResults[start:end], nil
}

// DeleteOlderThan implements ResultStorage.DeleteOlderThan
func (s *ResultStorage) DeleteOlderThan(ctx context.Context, age time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-age)
	cutoffTimestamp := cutoffTime.Unix()

	s.logger.Info("Deleting results older than cutoff time",
		zap.Time("cutoff_time", cutoffTime),
		zap.Int64("cutoff_timestamp", cutoffTimestamp))

	// Similar to Query, a real implementation would require SCAN support
	// For now, we'll log that this operation would happen but can't be fully implemented

	s.logger.Warn("DeleteOlderThan not fully implemented with current KVStore interface",
		zap.Time("cutoff_time", cutoffTime))

	// In a complete implementation, we would:
	// 1. Use SCAN to iterate keys matching the time index prefix
	// 2. Parse the timestamp from each key
	// 3. If timestamp < cutoffTimestamp, delete the corresponding result
	// 4. Delete the index key as well

	return 0, nil
}

// PurgeByCommandID implements ResultStorage.PurgeByCommandID
func (s *ResultStorage) PurgeByCommandID(ctx context.Context, commandID string) error {
	if commandID == "" {
		return fmt.Errorf("command ID is required")
	}

	// Get the result first to extract the agent ID
	result, err := s.Get(ctx, commandID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// If already not found, consider it purged
			return nil
		}
		return fmt.Errorf("failed to retrieve result: %w", err)
	}

	// Delete the result key
	key := fmt.Sprintf("cmd:result:%s", commandID)
	if err := s.store.Del(ctx, key); err != nil {
		if strings.Contains(err.Error(), "not supported") {
			s.logger.Warn("Del operation not supported by underlying store",
				zap.String("key", key))
			return nil
		}
		return fmt.Errorf("failed to delete result: %w", err)
	}

	// Delete the agent index
	agentKey := fmt.Sprintf("agent:%s:cmd:%s", result.AgentID, commandID)
	if err := s.store.Del(ctx, agentKey); err != nil && !strings.Contains(err.Error(), "not supported") {
		s.logger.Warn("Failed to delete agent command index",
			zap.Error(err),
			zap.String("key", agentKey))
	}

	return nil
}

// Helper function to check if a result matches query criteria
func (s *ResultStorage) matchesQuery(result *CommandResult, query *ResultQuery) bool {
	// Match by command ID
	if query.CommandID != "" && result.CommandID != query.CommandID {
		return false
	}

	// Match by type
	if query.Type != "" && result.Type != query.Type {
		return false
	}

	// Match by status
	if query.Status != "" && result.Status != query.Status {
		return false
	}

	// Match by time range
	if query.StartTime != nil && result.CompletedAt.Before(*query.StartTime) {
		return false
	}
	if query.EndTime != nil && result.CompletedAt.After(*query.EndTime) {
		return false
	}

	return true
}

// Helper function to sort results based on query parameters
func sortResults(results []*CommandResult, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "completed_at" // Default sort by completion time
	}
	if sortOrder == "" {
		sortOrder = "desc" // Default sort order is descending (newest first)
	}

	// Define sort function based on criteria
	sort.Slice(results, func(i, j int) bool {
		// Determine if sorting ascending or descending
		asc := sortOrder != "desc"

		// Sort based on the specified field
		switch sortBy {
		case "command_id":
			if asc {
				return results[i].CommandID < results[j].CommandID
			}
			return results[i].CommandID > results[j].CommandID
		case "type":
			if asc {
				return results[i].Type < results[j].Type
			}
			return results[i].Type > results[j].Type
		case "status":
			if asc {
				return results[i].Status < results[j].Status
			}
			return results[i].Status > results[j].Status
		case "exit_code":
			if asc {
				return results[i].ExitCode < results[j].ExitCode
			}
			return results[i].ExitCode > results[j].ExitCode
		case "completed_at":
			if asc {
				return results[i].CompletedAt.Before(results[j].CompletedAt)
			}
			return results[i].CompletedAt.After(results[j].CompletedAt)
		default:
			// Default to sorting by completion time
			if asc {
				return results[i].CompletedAt.Before(results[j].CompletedAt)
			}
			return results[i].CompletedAt.After(results[j].CompletedAt)
		}
	})
}
