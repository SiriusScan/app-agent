package patterns

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// DefaultMatchTimeout is the default timeout for regex matching (5s)
	DefaultMatchTimeout = 5 * time.Second
)

// MatchOptions configures pattern matching behavior
type MatchOptions struct {
	// Timeout is the maximum time to spend matching (default: 5s)
	Timeout time.Duration

	// CaseInsensitive enables case-insensitive matching
	CaseInsensitive bool
}

// DefaultMatchOptions returns the default options for pattern matching
func DefaultMatchOptions() MatchOptions {
	return MatchOptions{
		Timeout:         DefaultMatchTimeout,
		CaseInsensitive: false,
	}
}

// MatchResult contains the results of a pattern match
type MatchResult struct {
	// Matched indicates whether the pattern matched
	Matched bool

	// MatchedText is the text that was matched (if any)
	MatchedText string

	// Line is the line number where the match occurred (1-based, 0 if not applicable)
	Line int

	// AllMatches contains all matched strings (for FindAll operations)
	AllMatches []string
}

// Match checks if a pattern matches the given text with timeout protection.
func Match(pattern, text string) (bool, error) {
	return MatchWithOptions(pattern, text, DefaultMatchOptions())
}

// MatchWithOptions checks if a pattern matches with custom options.
func MatchWithOptions(pattern, text string, opts MatchOptions) (bool, error) {
	// Compile the regex
	if opts.CaseInsensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// Run regex matching with timeout
	type matchResult struct {
		matched bool
	}
	resultChan := make(chan matchResult, 1)

	go func() {
		matched := re.MatchString(text)
		resultChan <- matchResult{matched: matched}
	}()

	select {
	case <-ctx.Done():
		return false, &TimeoutError{
			Pattern: pattern,
			Timeout: opts.Timeout,
		}
	case result := <-resultChan:
		return result.matched, nil
	}
}

// Find finds the first match of a pattern in text.
func Find(pattern, text string) (*MatchResult, error) {
	return FindWithOptions(pattern, text, DefaultMatchOptions())
}

// FindWithOptions finds the first match with custom options.
func FindWithOptions(pattern, text string, opts MatchOptions) (*MatchResult, error) {
	// Compile the regex
	if opts.CaseInsensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// Run regex matching with timeout
	type findResult struct {
		match string
		err   error
	}
	resultChan := make(chan findResult, 1)

	go func() {
		match := re.FindString(text)
		resultChan <- findResult{match: match}
	}()

	select {
	case <-ctx.Done():
		return nil, &TimeoutError{
			Pattern: pattern,
			Timeout: opts.Timeout,
		}
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}

		matched := result.match != ""
		return &MatchResult{
			Matched:     matched,
			MatchedText: result.match,
		}, nil
	}
}

// FindInLines finds matches in text and returns line numbers.
func FindInLines(pattern, text string) ([]*MatchResult, error) {
	return FindInLinesWithOptions(pattern, text, DefaultMatchOptions())
}

// FindInLinesWithOptions finds matches in text with line numbers using custom options.
func FindInLinesWithOptions(pattern, text string, opts MatchOptions) ([]*MatchResult, error) {
	// Compile the regex
	if opts.CaseInsensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// Run regex matching with timeout
	type findResult struct {
		results []*MatchResult
		err     error
	}
	resultChan := make(chan findResult, 1)

	go func() {
		lines := strings.Split(text, "\n")
		var results []*MatchResult

		for lineNum, line := range lines {
			if re.MatchString(line) {
				matchedText := re.FindString(line)
				results = append(results, &MatchResult{
					Matched:     true,
					MatchedText: matchedText,
					Line:        lineNum + 1, // 1-based line numbers
				})
			}
		}

		resultChan <- findResult{results: results}
	}()

	select {
	case <-ctx.Done():
		return nil, &TimeoutError{
			Pattern: pattern,
			Timeout: opts.Timeout,
		}
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		return result.results, nil
	}
}

// FindAll finds all matches in text.
func FindAll(pattern, text string) ([]string, error) {
	return FindAllWithOptions(pattern, text, DefaultMatchOptions())
}

// FindAllWithOptions finds all matches with custom options.
func FindAllWithOptions(pattern, text string, opts MatchOptions) ([]string, error) {
	// Compile the regex
	if opts.CaseInsensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// Run regex matching with timeout
	type findResult struct {
		matches []string
		err     error
	}
	resultChan := make(chan findResult, 1)

	go func() {
		matches := re.FindAllString(text, -1)
		resultChan <- findResult{matches: matches}
	}()

	select {
	case <-ctx.Done():
		return nil, &TimeoutError{
			Pattern: pattern,
			Timeout: opts.Timeout,
		}
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		return result.matches, nil
	}
}

// TimeoutError indicates a pattern matching operation timed out
type TimeoutError struct {
	Pattern string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("pattern matching timed out: pattern %q exceeded timeout %v", e.Pattern, e.Timeout)
}

