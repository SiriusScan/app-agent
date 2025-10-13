package types

import (
	"time"
)

// Severity levels for vulnerabilities
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Platform types
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformDarwin  Platform = "darwin"
	PlatformWindows Platform = "windows"
)

// DetectionLogic defines how detection steps are evaluated
type DetectionLogic string

const (
	LogicAll DetectionLogic = "all" // AND - all steps must match
	LogicAny DetectionLogic = "any" // OR - at least one step must match
)

// Template represents a complete vulnerability detection template
type Template struct {
	// ID is the unique identifier for this template
	ID string `json:"id" yaml:"id"`

	// Info contains template metadata
	Info TemplateInfo `json:"info" yaml:"info"`

	// Detection defines the detection logic and steps
	Detection DetectionConfig `json:"detection" yaml:"detection"`

	// Metadata (set during parsing, not in YAML)
	FilePath string    `json:"file_path,omitempty" yaml:"-"`
	LoadedAt time.Time `json:"loaded_at,omitempty" yaml:"-"`
}

// TemplateInfo contains metadata about the template
type TemplateInfo struct {
	// Name is the human-readable template name
	Name string `json:"name" yaml:"name"`

	// Author is the template author
	Author string `json:"author,omitempty" yaml:"author,omitempty"`

	// Severity indicates the severity level
	Severity Severity `json:"severity" yaml:"severity"`

	// Description explains what this template detects
	Description string `json:"description" yaml:"description"`

	// References are URLs to related information
	References []string `json:"references,omitempty" yaml:"references,omitempty"`

	// CVE identifiers if applicable
	CVE []string `json:"cve,omitempty" yaml:"cve,omitempty"`

	// Tags for categorization
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Version is the template version
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// DetectionConfig defines how detection is performed
type DetectionConfig struct {
	// Logic determines how steps are evaluated (all/any)
	Logic DetectionLogic `json:"logic,omitempty" yaml:"logic,omitempty"` // defaults to "all"

	// Steps are the detection steps to execute
	Steps []DetectionStep `json:"steps" yaml:"steps"`
}

// DetectionStep represents a single detection step
type DetectionStep struct {
	// Type is the module type to use (e.g., "file_hash", "file_content")
	Type string `json:"type" yaml:"type"`

	// Platforms specifies which platforms this step applies to
	// Empty means all platforms
	Platforms []Platform `json:"platforms,omitempty" yaml:"platforms,omitempty"`

	// Weight affects confidence calculation (0.0-1.0, default 1.0)
	Weight float64 `json:"weight,omitempty" yaml:"weight,omitempty"`

	// Config contains module-specific configuration
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// Result represents the outcome of template execution
type Result struct {
	// TemplateID is the ID of the template that was executed
	TemplateID string `json:"template_id"`

	// TemplateName is the name of the template
	TemplateName string `json:"template_name,omitempty"`

	// Severity is the template's severity level
	Severity Severity `json:"severity,omitempty"`

	// Matched indicates whether the template matched (vulnerability detected)
	Matched bool `json:"matched"`

	// Confidence is a score from 0.0 to 1.0 indicating detection confidence
	Confidence float64 `json:"confidence"`

	// Steps contains the results of each detection step
	Steps []StepResult `json:"steps,omitempty"`

	// Errors contains any errors that occurred during execution
	Errors []string `json:"errors,omitempty"`

	// Timestamp is when the template was executed
	Timestamp time.Time `json:"timestamp"`

	// Host is the hostname where the template was executed
	Host string `json:"host,omitempty"`
}

// StepResult represents the outcome of a single detection step
type StepResult struct {
	// Step is the index of the step (0-based)
	Step int `json:"step"`

	// Type is the module type that was used
	Type string `json:"type"`

	// Matched indicates whether this step matched
	Matched bool `json:"matched"`

	// Evidence contains module-specific evidence data
	Evidence map[string]interface{} `json:"evidence,omitempty"`

	// Error contains any error message if the step failed
	Error string `json:"error,omitempty"`

	// Duration is how long the step took to execute
	Duration time.Duration `json:"duration,omitempty"`
}

// ValidSeverities returns all valid severity levels
func ValidSeverities() []Severity {
	return []Severity{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
		SeverityInfo,
	}
}

// ValidPlatforms returns all valid platform names
func ValidPlatforms() []Platform {
	return []Platform{
		PlatformLinux,
		PlatformDarwin,
		PlatformWindows,
	}
}

// IsSeverityValid checks if a severity string is valid
func IsSeverityValid(s string) bool {
	switch Severity(s) {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	}
	return false
}

// IsPlatformValid checks if a platform string is valid
func IsPlatformValid(p string) bool {
	switch Platform(p) {
	case PlatformLinux, PlatformDarwin, PlatformWindows:
		return true
	}
	return false
}

