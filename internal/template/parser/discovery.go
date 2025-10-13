package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

const (
	// DefaultDiscoveryTimeout is the maximum time to spend discovering templates
	DefaultDiscoveryTimeout = 5 * time.Minute
)

// DiscoveryError represents an error that occurred during template discovery
type DiscoveryError struct {
	Path string
	Err  error
}

// Error implements the error interface
func (de *DiscoveryError) Error() string {
	return fmt.Sprintf("%s: %v", de.Path, de.Err)
}

// DiscoverTemplates discovers and loads all templates from a directory.
// It walks the directory recursively, finds all .yaml/.yml files, and attempts
// to parse and validate each one. It collects errors but continues processing,
// returning all successfully loaded templates and a list of errors for failed ones.
func DiscoverTemplates(dir string) ([]*types.Template, []error) {
	return DiscoverTemplatesWithTimeout(dir, DefaultDiscoveryTimeout)
}

// DiscoverTemplatesWithTimeout discovers templates with a custom timeout.
// It walks the directory recursively and processes .yaml/.yml files.
func DiscoverTemplatesWithTimeout(dir string, timeout time.Duration) ([]*types.Template, []error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return DiscoverTemplatesWithContext(ctx, dir)
}

// DiscoverTemplatesWithContext discovers templates with a context for cancellation.
func DiscoverTemplatesWithContext(ctx context.Context, dir string) ([]*types.Template, []error) {
	// Validate directory exists
	if dir == "" {
		return nil, []error{fmt.Errorf("directory path cannot be empty")}
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []error{fmt.Errorf("directory does not exist: %s", dir)}
		}
		return nil, []error{fmt.Errorf("failed to access directory %s: %w", dir, err)}
	}

	if !info.IsDir() {
		return nil, []error{fmt.Errorf("path is not a directory: %s", dir)}
	}

	var templates []*types.Template
	var errors []error

	// Walk the directory tree
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Handle walk errors
		if err != nil {
			errors = append(errors, &DiscoveryError{
				Path: path,
				Err:  fmt.Errorf("walk error: %w", err),
			})
			return nil // Continue walking
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file is a YAML template
		if !isTemplateFile(path) {
			return nil
		}

		// Parse and validate the template
		template, parseErr := ParseTemplate(path)
		if parseErr != nil {
			errors = append(errors, &DiscoveryError{
				Path: path,
				Err:  parseErr,
			})
			return nil // Continue with other files
		}

		// Validate the template
		if validateErr := ValidateTemplate(template); validateErr != nil {
			errors = append(errors, &DiscoveryError{
				Path: path,
				Err:  validateErr,
			})
			return nil // Continue with other files
		}

		// Successfully loaded template
		templates = append(templates, template)
		return nil
	})

	// Handle context timeout or cancellation
	if err != nil {
		if err == context.DeadlineExceeded {
			errors = append(errors, fmt.Errorf("template discovery timed out after %v", DefaultDiscoveryTimeout))
		} else if err == context.Canceled {
			errors = append(errors, fmt.Errorf("template discovery was canceled"))
		} else {
			errors = append(errors, fmt.Errorf("failed to walk directory: %w", err))
		}
	}

	return templates, errors
}

// isTemplateFile checks if a file path represents a template file based on extension
func isTemplateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

