package errors

import "fmt"

// PermissionDenied indicates a permission error
type PermissionDenied struct {
	Resource  string
	Operation string
}

func (e *PermissionDenied) Error() string {
	return fmt.Sprintf("permission denied: cannot %s %s", e.Operation, e.Resource)
}

// FileNotFound indicates a file does not exist
type FileNotFound struct {
	Path string
}

func (e *FileNotFound) Error() string {
	return fmt.Sprintf("file not found: %s", e.Path)
}

// Timeout indicates an operation timed out
type Timeout struct {
	Operation string
	Duration  interface{}
}

func (e *Timeout) Error() string {
	return fmt.Sprintf("operation timed out: %s exceeded %v", e.Operation, e.Duration)
}

// InvalidConfig indicates invalid configuration
type InvalidConfig struct {
	Field   string
	Message string
}

func (e *InvalidConfig) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("invalid configuration for field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("invalid configuration: %s", e.Message)
}

// ModuleNotFound indicates a module type is not registered
type ModuleNotFound struct {
	Type string
}

func (e *ModuleNotFound) Error() string {
	return fmt.Sprintf("module not found: %s", e.Type)
}

// ExecutionError indicates a module execution error
type ExecutionError struct {
	Module  string
	Message string
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("execution error in module %s: %s", e.Module, e.Message)
}

// Helper functions for creating common errors

// NewInvalidConfigError creates a new InvalidConfig error
func NewInvalidConfigError(message string) error {
	return &InvalidConfig{Message: message}
}

// NewFileNotFoundError creates a new FileNotFound error
func NewFileNotFoundError(path string) error {
	return &FileNotFound{Path: path}
}

// NewPermissionDeniedError creates a new PermissionDenied error
func NewPermissionDeniedError(resource string) error {
	return &PermissionDenied{Resource: resource, Operation: "access"}
}

// NewTimeoutError creates a new Timeout error
func NewTimeoutError(operation string, duration interface{}) error {
	return &Timeout{Operation: operation, Duration: duration}
}

// IsFileNotFound checks if an error is a FileNotFound error
func IsFileNotFound(err error) bool {
	_, ok := err.(*FileNotFound)
	return ok
}

// IsPermissionDenied checks if an error is a PermissionDenied error
func IsPermissionDenied(err error) bool {
	_, ok := err.(*PermissionDenied)
	return ok
}

// IsTimeout checks if an error is a Timeout error
func IsTimeout(err error) bool {
	_, ok := err.(*Timeout)
	return ok
}

