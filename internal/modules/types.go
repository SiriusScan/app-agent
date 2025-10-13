package modules

import (
	"context"
)

// Module defines the interface that all detection modules must implement.
// This minimal interface allows maximum flexibility in module implementation.
type Module interface {
	// Execute runs the detection logic for this module.
	// ctx provides timeout and cancellation support.
	// config contains module-specific configuration (from template step config).
	// Returns a Result containing detection outcome and evidence.
	Execute(ctx context.Context, config StepConfig) (*Result, error)
}

// StepConfig represents the configuration for a single detection step.
// It's a flexible map that allows each module to define its own config fields.
type StepConfig map[string]interface{}

// GetString safely retrieves a string value from the config.
func (c StepConfig) GetString(key string) string {
	if val, ok := c[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetBool safely retrieves a boolean value from the config.
func (c StepConfig) GetBool(key string) bool {
	if val, ok := c[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// GetInt safely retrieves an integer value from the config.
func (c StepConfig) GetInt(key string) int {
	if val, ok := c[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetFloat safely retrieves a float value from the config.
func (c StepConfig) GetFloat(key string) float64 {
	if val, ok := c[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return 0.0
}

// GetStringSlice safely retrieves a string slice from the config.
func (c StepConfig) GetStringSlice(key string) []string {
	if val, ok := c[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		if slice, ok := val.([]string); ok {
			return slice
		}
	}
	return nil
}

// Descriptor provides metadata about a module.
type Descriptor struct {
	// Type is the unique identifier for this module (e.g., "file_hash", "file_content")
	Type string

	// Name is a human-readable name for the module
	Name string

	// Description explains what this module does
	Description string

	// Version is the module version (semver recommended)
	Version string

	// Author is the module author or organization
	Author string

	// SupportedOS lists the operating systems this module supports
	// Valid values: "linux", "darwin", "windows"
	SupportedOS []string

	// ConfigDocs documents the configuration fields this module expects
	// Key is the config field name, value is a description
	ConfigDocs map[string]string
}

// Result represents the outcome of a module execution.
type Result struct {
	// Matched indicates whether the detection logic matched
	Matched bool `json:"matched"`

	// Evidence contains module-specific evidence data
	Evidence map[string]interface{} `json:"evidence,omitempty"`

	// Error contains any error message (if execution failed)
	Error string `json:"error,omitempty"`
}

