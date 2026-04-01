package filesearch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/common/files"
	"github.com/SiriusScan/app-agent/internal/common/patterns"
	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

const (
	DefaultMaxDepth    = 20
	DefaultMaxResults  = 100
	DefaultMaxFileSize = 10 * 1024 * 1024 // 10 MB
	DefaultMatchTimeout = 5 * time.Second
)

// FileSearchModule discovers files across directory trees using filename globs,
// path regex, and content regex. It returns structured evidence for every match.
type FileSearchModule struct{}

// fileMatch records a single matched file and its content match details.
type fileMatch struct {
	Path        string `json:"path"`
	MatchedText string `json:"matched_text,omitempty"`
	LineNumber  int    `json:"line_number,omitempty"`
}

func (m *FileSearchModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	rootPath := config.GetString("root_path")
	if rootPath == "" {
		return nil, fmt.Errorf("config field 'root_path' is required")
	}

	filenamePattern := config.GetString("filename_pattern")
	pathRegex := config.GetString("path_regex")
	contentRegex := config.GetString("content_regex")

	if filenamePattern == "" && pathRegex == "" && contentRegex == "" {
		return nil, fmt.Errorf("at least one of 'filename_pattern', 'path_regex', or 'content_regex' is required")
	}

	maxDepth := config.GetInt("max_depth")
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}

	maxResults := config.GetInt("max_results")
	if maxResults <= 0 {
		maxResults = DefaultMaxResults
	}

	excludeDirs := config.GetStringSlice("exclude_dirs")
	excludeSet := make(map[string]bool, len(excludeDirs))
	for _, d := range excludeDirs {
		excludeSet[d] = true
	}
	// Always exclude .git unless the user explicitly provided an exclude list
	// that doesn't include it (preserve default safety without overriding intent).
	if len(excludeDirs) == 0 {
		excludeSet[".git"] = true
	}

	var compiledPathRegex *regexp.Regexp
	if pathRegex != "" {
		var err error
		compiledPathRegex, err = regexp.Compile(pathRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid 'path_regex' pattern %q: %w", pathRegex, err)
		}
	}

	// Validate content_regex compiles before starting the walk.
	if contentRegex != "" {
		if _, err := regexp.Compile(contentRegex); err != nil {
			return nil, fmt.Errorf("invalid 'content_regex' pattern %q: %w", contentRegex, err)
		}
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &modules.Result{
				Matched: false,
				Error:   fmt.Sprintf("root_path does not exist: %s", rootPath),
			}, nil
		}
		return &modules.Result{
			Matched: false,
			Error:   fmt.Sprintf("cannot access root_path: %v", err),
		}, nil
	}
	if !info.IsDir() {
		return &modules.Result{
			Matched: false,
			Error:   fmt.Sprintf("root_path is not a directory: %s", rootPath),
		}, nil
	}

	// Resolve to absolute so depth calculations and evidence paths are unambiguous.
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		absRoot = rootPath
	}

	var (
		matchList    []fileMatch
		filesScanned int
		truncated    bool
	)

	walkErr := filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // skip inaccessible entries, keep walking
		}

		if d.IsDir() {
			if excludeSet[d.Name()] {
				return filepath.SkipDir
			}
			rel, relErr := filepath.Rel(absRoot, path)
			if relErr == nil && rel != "." {
				depth := strings.Count(rel, string(filepath.Separator)) + 1
				if depth > maxDepth {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// --- File-level filters (AND logic) ---

		if filenamePattern != "" {
			matched, matchErr := filepath.Match(filenamePattern, d.Name())
			if matchErr != nil || !matched {
				return nil
			}
		}

		if compiledPathRegex != nil {
			if !compiledPathRegex.MatchString(path) {
				return nil
			}
		}

		filesScanned++

		if contentRegex != "" {
			fm, ok := m.matchContent(ctx, path, contentRegex)
			if !ok {
				return nil
			}
			if len(matchList) < maxResults {
				matchList = append(matchList, fm)
			} else {
				truncated = true
			}
			return nil
		}

		// No content_regex -- file passes by existence alone.
		if len(matchList) < maxResults {
			matchList = append(matchList, fileMatch{Path: path})
		} else {
			truncated = true
		}
		return nil
	})

	if walkErr != nil && walkErr != context.DeadlineExceeded && walkErr != context.Canceled {
		return &modules.Result{
			Matched: false,
			Error:   fmt.Sprintf("walk error: %v", walkErr),
		}, nil
	}

	matched := len(matchList) > 0

	// Build serialisable matches slice for evidence.
	evidenceMatches := make([]interface{}, len(matchList))
	for i, fm := range matchList {
		m := map[string]interface{}{"path": fm.Path}
		if fm.MatchedText != "" {
			m["matched_text"] = fm.MatchedText
		}
		if fm.LineNumber > 0 {
			m["line_number"] = fm.LineNumber
		}
		evidenceMatches[i] = m
	}

	evidence := map[string]interface{}{
		"root_path":     absRoot,
		"files_scanned": filesScanned,
		"files_matched": len(matchList),
		"truncated":     truncated,
		"matches":       evidenceMatches,
	}
	if filenamePattern != "" {
		evidence["filename_pattern"] = filenamePattern
	}
	if pathRegex != "" {
		evidence["path_regex"] = pathRegex
	}
	if contentRegex != "" {
		evidence["content_regex"] = contentRegex
	}

	return &modules.Result{
		Matched:  matched,
		Evidence: evidence,
	}, nil
}

// matchContent reads a file and applies the content regex line-by-line.
// Returns the first match and true, or an empty struct and false.
func (m *FileSearchModule) matchContent(ctx context.Context, path, regex string) (fileMatch, bool) {
	content, err := files.ReadFileWithLimit(path, DefaultMaxFileSize)
	if err != nil {
		return fileMatch{}, false
	}

	matchCtx, cancel := context.WithTimeout(ctx, DefaultMatchTimeout)
	defer cancel()

	results, err := patterns.FindInLinesWithOptions(regex, string(content), patterns.MatchOptions{
		Timeout: DefaultMatchTimeout,
	})
	if err != nil {
		return fileMatch{}, false
	}

	// Check for context cancellation after pattern work.
	select {
	case <-matchCtx.Done():
		return fileMatch{}, false
	default:
	}

	if len(results) > 0 && results[0].Matched {
		return fileMatch{
			Path:        path,
			MatchedText: results[0].MatchedText,
			LineNumber:  results[0].Line,
		}, true
	}

	return fileMatch{}, false
}

func init() {
	descriptor := modules.Descriptor{
		Type:        "file_search",
		Name:        "File Search",
		Description: "Recursively searches directory trees for files matching filename globs, path patterns, and content regex",
		Version:     "1.0.0",
		Author:      "Sirius Scan",
		SupportedOS: []string{string(types.PlatformLinux), string(types.PlatformDarwin), string(types.PlatformWindows)},
		ConfigDocs: map[string]string{
			"root_path":        "Starting directory for the search (required)",
			"filename_pattern": "Glob pattern matched against filenames (e.g. 'package.json', '*.lock')",
			"path_regex":       "Regex matched against full file paths (e.g. 'node_modules/axios/package\\.json$')",
			"content_regex":    "Regex applied to file content of files that pass filename/path filters",
			"max_depth":        "Maximum directory recursion depth (default: 20)",
			"max_results":      "Maximum matches to collect (default: 100)",
			"exclude_dirs":     "Directory names to skip (default: [\".git\"])",
		},
	}

	if err := registry.Register(&FileSearchModule{}, descriptor); err != nil {
		panic(fmt.Sprintf("failed to register file_search module: %v", err))
	}
}
