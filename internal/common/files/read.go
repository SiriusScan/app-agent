package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	// DefaultMaxFileSize is the default maximum file size to read (10MB)
	DefaultMaxFileSize = 10 * 1024 * 1024

	// DefaultReadTimeout is the default timeout for reading files
	DefaultReadTimeout = 5 * time.Second
)

// ReadOptions configures file reading behavior
type ReadOptions struct {
	// MaxSize is the maximum file size to read (default: 10MB)
	MaxSize int64

	// Timeout is the maximum time to spend reading (default: 5s)
	Timeout time.Duration
}

// DefaultReadOptions returns the default options for reading files
func DefaultReadOptions() ReadOptions {
	return ReadOptions{
		MaxSize: DefaultMaxFileSize,
		Timeout: DefaultReadTimeout,
	}
}

// ReadFile reads a file with size and timeout restrictions.
// It returns the file contents or an error if the file is too large,
// reading times out, or the file cannot be accessed.
func ReadFile(path string) ([]byte, error) {
	return ReadFileWithOptions(path, DefaultReadOptions())
}

// ReadFileWithOptions reads a file with custom options.
func ReadFileWithOptions(path string, opts ReadOptions) ([]byte, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileNotFoundError{Path: path}
		}
		if os.IsPermission(err) {
			return nil, &PermissionDeniedError{Path: path, Operation: "read"}
		}
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer file.Close()

	// Check file size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", path, err)
	}

	if stat.Size() > opts.MaxSize {
		return nil, &FileTooLargeError{
			Path:     path,
			Size:     stat.Size(),
			MaxSize:  opts.MaxSize,
		}
	}

	// Read file contents with timeout
	type readResult struct {
		data []byte
		err  error
	}
	resultChan := make(chan readResult, 1)

	go func() {
		data, err := io.ReadAll(file)
		resultChan <- readResult{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, &TimeoutError{
			Path:    path,
			Timeout: opts.Timeout,
		}
	case result := <-resultChan:
		if result.err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", path, result.err)
		}
		return result.data, nil
	}
}

// ReadFileString reads a file and returns it as a string.
func ReadFileString(path string) (string, error) {
	data, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

