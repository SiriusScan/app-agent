// Package output provides a unified output formatting system for the Sirius agent.
// It supports multiple output formats (JSON, table, CSV, etc.) with consistent
// interfaces across all commands.
package output

import (
	"os"
	"time"

	"github.com/SiriusScan/app-agent/internal/sysinfo"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// Format represents an output format type.
type Format string

// Supported output formats.
const (
	FormatJSON     Format = "json"
	FormatJSONL    Format = "jsonl"
	FormatTable    Format = "table"
	FormatText     Format = "text"
	FormatCSV      Format = "csv"
	FormatQuiet    Format = "quiet"
	FormatMarkdown Format = "markdown"
)

// ValidFormats returns all supported output formats.
func ValidFormats() []Format {
	return []Format{
		FormatJSON,
		FormatJSONL,
		FormatTable,
		FormatText,
		FormatCSV,
		FormatQuiet,
		FormatMarkdown,
	}
}

// IsValidFormat checks if a format string is a valid format.
func IsValidFormat(f string) bool {
	switch Format(f) {
	case FormatJSON, FormatJSONL, FormatTable, FormatText, FormatCSV, FormatQuiet, FormatMarkdown:
		return true
	}
	return false
}

// ScanSummary contains summary statistics for a scan operation.
type ScanSummary struct {
	// TotalTemplates is the number of templates that were executed
	TotalTemplates int `json:"total_templates"`

	// Matched is the number of templates that matched (vulnerabilities detected)
	Matched int `json:"matched"`

	// NotMatched is the number of templates that did not match
	NotMatched int `json:"not_matched"`

	// Errors is the number of templates that had errors
	Errors int `json:"errors"`

	// ExecutionTimeMs is the total execution time in milliseconds
	ExecutionTimeMs int64 `json:"execution_time_ms"`

	// Workers is the number of parallel workers used
	Workers int `json:"workers"`

	// StartTime is when the scan started
	StartTime time.Time `json:"start_time,omitempty"`

	// EndTime is when the scan completed
	EndTime time.Time `json:"end_time,omitempty"`

	// Host is the hostname where the scan was executed
	Host string `json:"host,omitempty"`

	// PrimaryIP is the agent's primary non-loopback IPv4 address
	PrimaryIP string `json:"primary_ip,omitempty"`
}

// NewScanSummary creates a ScanSummary from scan results.
func NewScanSummary(results []*types.Result, executionTime time.Duration, workers int) *ScanSummary {
	hostname, _ := os.Hostname()

	summary := &ScanSummary{
		TotalTemplates:  len(results),
		ExecutionTimeMs: executionTime.Milliseconds(),
		Workers:         workers,
		EndTime:         time.Now(),
		Host:            hostname,
		PrimaryIP:       sysinfo.GetPrimaryIP(),
	}

	for _, result := range results {
		if result == nil {
			summary.Errors++
			continue
		}
		if len(result.Errors) > 0 {
			summary.Errors++
		}
		if result.Matched {
			summary.Matched++
		} else {
			summary.NotMatched++
		}
	}

	return summary
}

// ValidationResult contains the result of a template validation.
type ValidationResult struct {
	// Valid indicates whether the template is valid
	Valid bool `json:"valid"`

	// TemplateID is the ID of the validated template
	TemplateID string `json:"id"`

	// TemplateName is the name of the validated template
	TemplateName string `json:"name"`

	// Severity is the template's severity level
	Severity string `json:"severity"`

	// StepCount is the number of detection steps
	StepCount int `json:"steps"`

	// Errors contains validation error messages
	Errors []string `json:"errors,omitempty"`

	// Warnings contains validation warning messages
	Warnings []string `json:"warnings,omitempty"`
}

// TemplateInfo is a simplified template representation for listing.
type TemplateInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Severity    string   `json:"severity"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
	StepCount   int      `json:"steps"`
	FilePath    string   `json:"file_path,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Version     string   `json:"version,omitempty"`
}

// NewTemplateInfo creates a TemplateInfo from a Template.
func NewTemplateInfo(t *types.Template) *TemplateInfo {
	return &TemplateInfo{
		ID:          t.ID,
		Name:        t.Info.Name,
		Severity:    string(t.Info.Severity),
		Author:      t.Info.Author,
		Description: t.Info.Description,
		StepCount:   len(t.Detection.Steps),
		FilePath:    t.FilePath,
		Tags:        t.Info.Tags,
		Version:     t.Info.Version,
	}
}

// ScanOutput wraps scan results with summary for JSON output.
type ScanOutput struct {
	Summary         *ScanSummary    `json:"summary"`
	Results         []*types.Result `json:"results"`
	DiscoveryErrors []string        `json:"discovery_errors,omitempty"`
	ExecutionErrors []string        `json:"execution_errors,omitempty"`
}

// SystemScanResult represents the output of a system inventory scan.
type SystemScanResult struct {
	OSInfo        OSInfo                    `json:"os_info"`
	Packages      []PackageInfo             `json:"packages"`
	CustomResults map[string]CustomResult   `json:"custom_results,omitempty"`
	ScanErrors    []string                  `json:"scan_errors,omitempty"`
}

// OSInfo contains operating system information.
type OSInfo struct {
	OS        string `json:"os"`
	Version   string `json:"version"`
	Hostname  string `json:"hostname"`
	PrimaryIP string `json:"primary_ip"`
}

// PackageInfo represents an installed package.
type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source,omitempty"`
}

// CustomResult represents the result of a custom script execution.
type CustomResult struct {
	ScriptName string `json:"script_name"`
	StdOut     string `json:"stdout,omitempty"`
	StdErr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exit_code"`
	Error      string `json:"error,omitempty"`
}



