package filecontent

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/common/errors"
	"github.com/SiriusScan/app-agent/internal/common/files"
	"github.com/SiriusScan/app-agent/internal/common/patterns"
	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

const (
	// DefaultMaxFileSize is the maximum file size to read (10 MB)
	DefaultMaxFileSize = 10 * 1024 * 1024

	// DefaultMatchTimeout is the maximum time to spend on regex matching
	DefaultMatchTimeout = 5 * time.Second
)

// FileContentModule is a detection module that searches for regex patterns in file contents.
type FileContentModule struct{}

// Execute performs the file content pattern matching.
func (m *FileContentModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	path := config.GetString("path")
	if path == "" {
		return nil, errors.NewInvalidConfigError("config field 'path' is required")
	}

	regex := config.GetString("regex")
	if regex == "" {
		return nil, errors.NewInvalidConfigError("config field 'regex' is required")
	}

	// Optional: multiline flag (defaults to false)
	multiline := config.GetBool("multiline")

	// Read the file with size limit
	content, err := files.ReadFileWithLimit(path, DefaultMaxFileSize)
	if err != nil {
		// Check if it's a known error type using type assertion
		if _, ok := err.(*files.FileTooLargeError); ok {
			return &modules.Result{
				Matched: false,
				Error:   fmt.Sprintf("file too large: %v", err),
			}, nil
		}
		if _, ok := err.(*files.FileNotFoundError); ok {
			return &modules.Result{
				Matched: false,
				Error:   fmt.Sprintf("file not found: %s", path),
			}, nil
		}
		if _, ok := err.(*files.PermissionDeniedError); ok {
			return &modules.Result{
				Matched: false,
				Error:   fmt.Sprintf("permission denied: %s", path),
			}, nil
		}
		return &modules.Result{
			Matched: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	// Perform regex matching with timeout
	matchCtx, cancel := context.WithTimeout(ctx, DefaultMatchTimeout)
	defer cancel()

	var matched bool
	var matchedText string
	var lineNumber int

	if multiline {
		// Multiline mode: match across the entire content
		matched, err = patterns.MatchWithTimeout(matchCtx, regex, string(content))
		if err != nil {
			if err == context.DeadlineExceeded {
				return &modules.Result{
					Matched: false,
					Error:   "regex matching timed out (possible ReDoS pattern)",
				}, nil
			}
			return &modules.Result{
				Matched: false,
				Error:   fmt.Sprintf("regex matching failed: %v", err),
			}, nil
		}

		if matched {
			// Extract the matched text
			matchedText, _ = patterns.ExtractMatchWithTimeout(matchCtx, regex, string(content))
			lineNumber = 0 // Not applicable for multiline
		}
	} else {
		// Line-by-line mode: search each line
		matched, matchedText, lineNumber, err = m.searchLines(matchCtx, regex, content)
		if err != nil {
			if err == context.DeadlineExceeded {
				return &modules.Result{
					Matched: false,
					Error:   "regex matching timed out (possible ReDoS pattern)",
				}, nil
			}
			return &modules.Result{
				Matched: false,
				Error:   fmt.Sprintf("regex matching failed: %v", err),
			}, nil
		}
	}

	// Build evidence
	evidence := map[string]interface{}{
		"file_path": path,
		"pattern":   regex,
		"matched":   matched,
		"multiline": multiline,
	}

	if matched {
		evidence["matched_text"] = matchedText
		if !multiline {
			evidence["matched_line"] = lineNumber
		}
	}

	return &modules.Result{
		Matched:  matched,
		Evidence: evidence,
	}, nil
}

// searchLines searches for a regex pattern in file contents line by line.
// Returns: matched, matchedText, lineNumber, error
func (m *FileContentModule) searchLines(ctx context.Context, regex string, content []byte) (bool, string, int, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		// Check for context cancellation (timeout)
		select {
		case <-ctx.Done():
			return false, "", 0, ctx.Err()
		default:
		}

		line := scanner.Text()

		// Match against the line
		matched, err := patterns.MatchWithTimeout(ctx, regex, line)
		if err != nil {
			return false, "", 0, err
		}

		if matched {
			// Extract the matched text
			matchedText, _ := patterns.ExtractMatchWithTimeout(ctx, regex, line)
			return true, matchedText, lineNumber, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, "", 0, fmt.Errorf("error scanning file: %w", err)
	}

	return false, "", 0, nil
}

// init registers the FileContent module in the global registry
func init() {
	descriptor := modules.Descriptor{
		Type:        "file_content",
		Name:        "File Content Pattern Matcher",
		Description: "Searches for regex patterns within file contents to detect vulnerable configurations or code patterns",
		Version:     "1.0.0",
		Author:      "Sirius Scan",
		SupportedOS: []string{string(types.PlatformLinux), string(types.PlatformDarwin), string(types.PlatformWindows)},
		ConfigDocs: map[string]string{
			"path":      "Path to the file to search",
			"regex":     "Regular expression pattern to match",
			"multiline": "Optional: Enable multiline matching (default: false)",
		},
	}

	// Register module
	if err := registry.Register(&FileContentModule{}, descriptor); err != nil {
		panic(fmt.Sprintf("failed to register file_content module: %v", err))
	}
}

