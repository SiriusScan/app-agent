package files

import "fmt"

// FileNotFoundError indicates a file does not exist
type FileNotFoundError struct {
	Path string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("file not found: %s", e.Path)
}

// PermissionDeniedError indicates insufficient permissions
type PermissionDeniedError struct {
	Path      string
	Operation string
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("permission denied: cannot %s file %s", e.Operation, e.Path)
}

// FileTooLargeError indicates a file exceeds the maximum allowed size
type FileTooLargeError struct {
	Path    string
	Size    int64
	MaxSize int64
}

func (e *FileTooLargeError) Error() string {
	return fmt.Sprintf("file too large: %s is %d bytes (max: %d bytes)", e.Path, e.Size, e.MaxSize)
}

// TimeoutError indicates a file operation timed out
type TimeoutError struct {
	Path    string
	Timeout interface{}
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("operation timed out: reading file %s exceeded timeout %v", e.Path, e.Timeout)
}

