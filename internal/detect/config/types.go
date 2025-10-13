package config

import (
	"regexp"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
)

// ConfigFileResult represents the result of analyzing a configuration file
type ConfigFileResult struct {
	// FilePath path to the analyzed file
	FilePath string `json:"file_path"`

	// FileExists whether the file exists
	FileExists bool `json:"file_exists"`

	// FileSize size of the file in bytes
	FileSize int64 `json:"file_size"`

	// Encoding file encoding (utf-8, ascii, etc.)
	Encoding string `json:"encoding,omitempty"`

	// TotalLines total number of lines in the file
	TotalLines int `json:"total_lines"`

	// MatchingLines number of lines that matched patterns
	MatchingLines int `json:"matching_lines"`

	// PatternMatches details of pattern matches found
	PatternMatches []PatternMatch `json:"pattern_matches"`

	// ProcessedAt when the analysis was performed
	ProcessedAt time.Time `json:"processed_at"`

	// ProcessingTime how long the analysis took
	ProcessingTime time.Duration `json:"processing_time"`

	// Error any error that occurred during analysis
	Error string `json:"error,omitempty"`
}

// PatternMatch represents a successful pattern match in a configuration file
type PatternMatch struct {
	// LineNumber line number where match was found (1-based)
	LineNumber int `json:"line_number"`

	// Line the full line content that matched
	Line string `json:"line"`

	// Pattern the regex pattern that matched
	Pattern string `json:"pattern"`

	// Description human-readable description of what was matched
	Description string `json:"description"`

	// MatchGroups regex capture groups from the match
	MatchGroups []string `json:"match_groups,omitempty"`
}

// CompiledPattern represents a compiled regex pattern with metadata
type CompiledPattern struct {
	// Original the original pattern configuration
	Original detect.PatternMatch `json:"original"`

	// Compiled the compiled regex pattern
	Compiled *regexp.Regexp `json:"-"` // Don't serialize the compiled regex

	// Description human-readable description
	Description string `json:"description"`

	// Negate whether to negate the match result
	Negate bool `json:"negate"`
}

// ConfigAnalysisRequest represents a request to analyze configuration files
type ConfigAnalysisRequest struct {
	// Files list of files to analyze
	Files []detect.FileTarget `json:"files"`

	// MaxFileSize maximum file size to process (bytes)
	MaxFileSize int64 `json:"max_file_size,omitempty"`

	// Timeout maximum processing time
	Timeout string `json:"timeout,omitempty"`

	// Concurrent whether to process files concurrently
	Concurrent bool `json:"concurrent,omitempty"`
}

// ConfigAnalysisResponse contains results of configuration file analysis
type ConfigAnalysisResponse struct {
	// Results analysis results for each file
	Results []*ConfigFileResult `json:"results"`

	// Summary overall analysis summary
	Summary ConfigAnalysisSummary `json:"summary"`

	// ExecutionTime total time taken for analysis
	ExecutionTime time.Duration `json:"execution_time"`
}

// ConfigAnalysisSummary provides aggregated analysis statistics
type ConfigAnalysisSummary struct {
	// TotalFiles total number of files analyzed
	TotalFiles int `json:"total_files"`

	// SuccessfulFiles number of files successfully analyzed
	SuccessfulFiles int `json:"successful_files"`

	// FilesWithMatches number of files that had pattern matches
	FilesWithMatches int `json:"files_with_matches"`

	// TotalMatches total number of pattern matches found
	TotalMatches int `json:"total_matches"`

	// ErrorFiles number of files that had analysis errors
	ErrorFiles int `json:"error_files"`

	// AverageProcessingTime average time per file
	AverageProcessingTime time.Duration `json:"average_processing_time"`

	// TotalLinesProcessed total lines processed across all files
	TotalLinesProcessed int `json:"total_lines_processed"`
}

// ConfigFileInfo contains metadata about a configuration file
type ConfigFileInfo struct {
	// Path file path
	Path string `json:"path"`

	// Size file size in bytes
	Size int64 `json:"size"`

	// ModTime last modification time
	ModTime time.Time `json:"mod_time"`

	// Readable whether file is readable
	Readable bool `json:"readable"`

	// IsBinary whether file appears to be binary
	IsBinary bool `json:"is_binary"`

	// DetectedEncoding detected file encoding
	DetectedEncoding string `json:"detected_encoding,omitempty"`
}
