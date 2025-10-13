package config

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
)

// ConfigFileReader handles configuration file analysis and pattern matching
type ConfigFileReader struct {
	logger          *zap.Logger
	maxFileSize     int64 // Maximum file size to process (bytes)
	maxLineLength   int   // Maximum line length to process
	timeout         time.Duration
	compiledRegexes map[string]*regexp.Regexp // Cache for compiled regexes
}

// NewConfigFileReader creates a new configuration file reader
func NewConfigFileReader(logger *zap.Logger) *ConfigFileReader {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ConfigFileReader{
		logger:          logger,
		maxFileSize:     10 * 1024 * 1024, // 10MB default
		maxLineLength:   1024 * 16,        // 16KB line limit
		timeout:         30 * time.Second, // 30 second timeout
		compiledRegexes: make(map[string]*regexp.Regexp),
	}
}

// SetMaxFileSize sets the maximum file size to process
func (cfr *ConfigFileReader) SetMaxFileSize(size int64) {
	cfr.maxFileSize = size
}

// SetTimeout sets the maximum processing time
func (cfr *ConfigFileReader) SetTimeout(timeout time.Duration) {
	cfr.timeout = timeout
}

// AnalyzeConfigFile performs pattern matching analysis on a configuration file
func (cfr *ConfigFileReader) AnalyzeConfigFile(ctx context.Context, fileTarget detect.FileTarget) (*ConfigFileResult, error) {
	cfr.logger.Debug("Analyzing configuration file",
		zap.String("path", fileTarget.Path),
		zap.Int("patterns", len(fileTarget.Patterns)))

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, cfr.timeout)
	defer cancel()

	startTime := time.Now()

	result := &ConfigFileResult{
		FilePath:       fileTarget.Path,
		FileExists:     false,
		Encoding:       fileTarget.Encoding,
		PatternMatches: []PatternMatch{},
		ProcessedAt:    startTime,
	}

	// Check if file exists and get basic info
	fileInfo, err := os.Stat(fileTarget.Path)
	if os.IsNotExist(err) {
		result.Error = fmt.Sprintf("file does not exist: %s", fileTarget.Path)
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}
	if err != nil {
		result.Error = fmt.Sprintf("failed to access file: %v", err)
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	result.FileExists = true
	result.FileSize = fileInfo.Size()

	// Check file size limits
	if fileTarget.MaxSize > 0 && fileInfo.Size() > fileTarget.MaxSize {
		result.Error = fmt.Sprintf("file too large: %d bytes (limit: %d)", fileInfo.Size(), fileTarget.MaxSize)
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}
	if fileInfo.Size() > cfr.maxFileSize {
		result.Error = fmt.Sprintf("file exceeds maximum size: %d bytes (limit: %d)", fileInfo.Size(), cfr.maxFileSize)
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	// Check if file is binary (basic heuristic)
	if cfr.isBinaryFile(fileTarget.Path) {
		result.Error = "file appears to be binary, skipping pattern analysis"
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	// Compile regex patterns
	patterns, err := cfr.compilePatterns(fileTarget.Patterns)
	if err != nil {
		result.Error = fmt.Sprintf("pattern compilation failed: %v", err)
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	// Process file line by line
	matches, err := cfr.processFileWithPatterns(execCtx, fileTarget.Path, patterns)
	if err != nil {
		result.Error = err.Error()
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	result.PatternMatches = matches
	result.TotalLines = cfr.countFileLines(fileTarget.Path)
	result.MatchingLines = len(matches)
	result.ProcessingTime = time.Since(startTime)

	cfr.logger.Debug("Configuration file analysis completed",
		zap.String("path", fileTarget.Path),
		zap.Int("total_lines", result.TotalLines),
		zap.Int("matching_lines", result.MatchingLines),
		zap.Duration("processing_time", result.ProcessingTime))

	return result, nil
}

// BatchAnalyzeConfigFiles processes multiple configuration files
func (cfr *ConfigFileReader) BatchAnalyzeConfigFiles(ctx context.Context, fileTargets []detect.FileTarget) ([]*ConfigFileResult, error) {
	cfr.logger.Info("Starting batch configuration file analysis",
		zap.Int("files", len(fileTargets)))

	results := make([]*ConfigFileResult, len(fileTargets))

	for i, target := range fileTargets {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result, err := cfr.AnalyzeConfigFile(ctx, target)
		if err != nil {
			// Create error result
			result = &ConfigFileResult{
				FilePath:    target.Path,
				FileExists:  false,
				Error:       err.Error(),
				ProcessedAt: time.Now(),
			}
		}
		results[i] = result
	}

	cfr.logger.Info("Batch configuration file analysis completed",
		zap.Int("total_files", len(results)),
		zap.Int("successful", cfr.countSuccessfulResults(results)),
		zap.Int("errors", cfr.countErrorResults(results)))

	return results, nil
}

// compilePatterns compiles regex patterns and caches them
func (cfr *ConfigFileReader) compilePatterns(patterns []detect.PatternMatch) ([]*CompiledPattern, error) {
	compiled := make([]*CompiledPattern, len(patterns))

	for i, pattern := range patterns {
		// Check cache first
		if regex, exists := cfr.compiledRegexes[pattern.Regex]; exists {
			compiled[i] = &CompiledPattern{
				Original:    pattern,
				Compiled:    regex,
				Description: pattern.Description,
				Negate:      pattern.Negate,
			}
			continue
		}

		// Compile new regex
		regex, err := regexp.Compile(pattern.Regex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile regex '%s': %w", pattern.Regex, err)
		}

		// Cache compiled regex
		cfr.compiledRegexes[pattern.Regex] = regex

		compiled[i] = &CompiledPattern{
			Original:    pattern,
			Compiled:    regex,
			Description: pattern.Description,
			Negate:      pattern.Negate,
		}
	}

	return compiled, nil
}

// processFileWithPatterns reads file line by line and applies patterns
func (cfr *ConfigFileReader) processFileWithPatterns(ctx context.Context, filePath string, patterns []*CompiledPattern) ([]PatternMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var matches []PatternMatch
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	// Set scanner buffer size limit
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, cfr.maxLineLength)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}

		lineNumber++
		line := scanner.Text()

		// Apply each pattern to the line
		for _, pattern := range patterns {
			if cfr.lineMatchesPattern(line, pattern) {
				match := PatternMatch{
					LineNumber:  lineNumber,
					Line:        line,
					Pattern:     pattern.Original.Regex,
					Description: pattern.Description,
					MatchGroups: pattern.Compiled.FindStringSubmatch(line),
				}
				matches = append(matches, match)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return matches, fmt.Errorf("error scanning file: %w", err)
	}

	return matches, nil
}

// lineMatchesPattern checks if a line matches a compiled pattern
func (cfr *ConfigFileReader) lineMatchesPattern(line string, pattern *CompiledPattern) bool {
	matches := pattern.Compiled.MatchString(line)

	// Apply negation if specified
	if pattern.Negate {
		return !matches
	}

	return matches
}

// isBinaryFile performs basic binary file detection
func (cfr *ConfigFileReader) isBinaryFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false // Assume text if we can't read
	}
	defer file.Close()

	// Read first 512 bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	return false
}

// countFileLines counts total lines in a file efficiently
func (cfr *ConfigFileReader) countFileLines(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}

	return lines
}

// Helper functions for batch processing

func (cfr *ConfigFileReader) countSuccessfulResults(results []*ConfigFileResult) int {
	count := 0
	for _, result := range results {
		if result.Error == "" {
			count++
		}
	}
	return count
}

func (cfr *ConfigFileReader) countErrorResults(results []*ConfigFileResult) int {
	count := 0
	for _, result := range results {
		if result.Error != "" {
			count++
		}
	}
	return count
}

// GetStats returns processing statistics
func (cfr *ConfigFileReader) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"max_file_size":   cfr.maxFileSize,
		"max_line_length": cfr.maxLineLength,
		"timeout_seconds": cfr.timeout.Seconds(),
		"cached_regexes":  len(cfr.compiledRegexes),
	}
}
