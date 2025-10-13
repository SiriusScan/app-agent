package detect

import (
	"context"
	"time"
)

// DetectionEngine is the main orchestrator for vulnerability detection
type DetectionEngine interface {
	// LoadTemplates loads vulnerability templates from the repository
	LoadTemplates(ctx context.Context) error

	// LoadScripts loads detection scripts from the repository
	LoadScripts(ctx context.Context) error

	// ExecuteDetection runs all configured detection methods
	ExecuteDetection(ctx context.Context, options DetectionOptions) (*DetectionReport, error)

	// ExecuteTemplate runs a specific template by ID
	ExecuteTemplate(ctx context.Context, templateID string) (*DetectionResult, error)

	// ExecuteScript runs a specific script by name
	ExecuteScript(ctx context.Context, scriptName string, args []string) (*DetectionResult, error)
}

// TemplateEngine handles YAML template parsing and execution
type TemplateEngine interface {
	// LoadTemplate parses and validates a YAML template
	LoadTemplate(templatePath string) (*VulnTemplate, error)

	// ValidateTemplate checks template syntax and structure
	ValidateTemplate(template *VulnTemplate) error

	// ExecuteTemplate runs vulnerability detection based on template
	ExecuteTemplate(ctx context.Context, template *VulnTemplate) (*DetectionResult, error)

	// ListTemplates returns all loaded templates
	ListTemplates() []*VulnTemplate
}

// ScriptEngine handles script execution with security controls
type ScriptEngine interface {
	// ExecuteScript runs a detection script with security sandbox
	ExecuteScript(ctx context.Context, script *DetectionScript, args []string) (*DetectionResult, error)

	// ValidateScript checks script permissions and safety
	ValidateScript(script *DetectionScript) error

	// ListScripts returns all available scripts
	ListScripts() []*DetectionScript
}

// HashEngine handles file hash calculations and matching
type HashEngine interface {
	// CalculateFileHash computes hash for a file
	CalculateFileHash(filePath string, algorithm HashAlgorithm) (string, error)

	// VerifyFileHash checks if file matches expected hash
	VerifyFileHash(filePath string, expectedHash string, algorithm HashAlgorithm) (bool, error)

	// BatchHashCheck verifies multiple files against hash database
	BatchHashCheck(ctx context.Context, targets []HashTarget) ([]*HashResult, error)
}

// DetectionOptions configures detection execution
type DetectionOptions struct {
	// EnableTemplates controls template-based detection
	EnableTemplates bool

	// EnableScripts controls script-based detection
	EnableScripts bool

	// EnableHashCheck controls hash-based detection
	EnableHashCheck bool

	// TemplateFilter filters which templates to execute
	TemplateFilter []string

	// ScriptFilter filters which scripts to execute
	ScriptFilter []string

	// MaxConcurrency limits parallel execution
	MaxConcurrency int

	// Timeout sets maximum execution time
	Timeout time.Duration

	// Platform specifies target platform (windows, linux, darwin)
	Platform string
}

// DetectionReport contains results from all detection methods
type DetectionReport struct {
	// ExecutionID unique identifier for this detection run
	ExecutionID string `json:"execution_id"`

	// StartTime when detection started
	StartTime time.Time `json:"start_time"`

	// EndTime when detection completed
	EndTime time.Time `json:"end_time"`

	// Duration total execution time
	Duration time.Duration `json:"duration"`

	// Platform where detection ran
	Platform string `json:"platform"`

	// TemplateResults results from template-based detection
	TemplateResults []*DetectionResult `json:"template_results"`

	// ScriptResults results from script-based detection
	ScriptResults []*DetectionResult `json:"script_results"`

	// HashResults results from hash-based detection
	HashResults []*HashResult `json:"hash_results"`

	// Summary aggregated vulnerability findings
	Summary DetectionSummary `json:"summary"`

	// Errors any execution errors encountered
	Errors []DetectionError `json:"errors,omitempty"`
}

// DetectionResult represents result from a single detection method
type DetectionResult struct {
	// DetectionID unique identifier for this detection
	DetectionID string `json:"detection_id"`

	// Method detection method used (template, script, hash)
	Method DetectionMethod `json:"method"`

	// SourceID template ID, script name, or hash target
	SourceID string `json:"source_id"`

	// VulnerabilityID CVE or custom vulnerability identifier
	VulnerabilityID string `json:"vulnerability_id"`

	// Vulnerable whether vulnerability was detected
	Vulnerable bool `json:"vulnerable"`

	// Confidence detection confidence score (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// Severity vulnerability severity level
	Severity SeverityLevel `json:"severity"`

	// Evidence proof of vulnerability detection
	Evidence []Evidence `json:"evidence"`

	// Metadata additional detection context
	Metadata map[string]interface{} `json:"metadata"`

	// ExecutedAt when this detection ran
	ExecutedAt time.Time `json:"executed_at"`

	// ExecutionTime how long detection took
	ExecutionTime time.Duration `json:"execution_time"`

	// Error any error that occurred during detection
	Error string `json:"error,omitempty"`
}

// Evidence provides proof of vulnerability detection
type Evidence struct {
	// Type evidence type (file_hash, registry_key, config_pattern, script_output)
	Type EvidenceType `json:"type"`

	// Location file path, registry key, or other location
	Location string `json:"location"`

	// Expected expected value (hash, pattern, etc.)
	Expected string `json:"expected"`

	// Actual actual value found
	Actual string `json:"actual"`

	// Description human-readable description
	Description string `json:"description"`

	// Context additional context information
	Context map[string]interface{} `json:"context,omitempty"`
}

// DetectionSummary provides aggregated results
type DetectionSummary struct {
	// TotalDetections number of detection methods executed
	TotalDetections int `json:"total_detections"`

	// VulnerabilitiesFound total vulnerabilities detected
	VulnerabilitiesFound int `json:"vulnerabilities_found"`

	// SeverityBreakdown vulnerabilities by severity
	SeverityBreakdown map[SeverityLevel]int `json:"severity_breakdown"`

	// MethodBreakdown results by detection method
	MethodBreakdown map[DetectionMethod]int `json:"method_breakdown"`

	// HighConfidenceFindings findings with confidence >= 0.8
	HighConfidenceFindings int `json:"high_confidence_findings"`
}

// DetectionError represents an error during detection
type DetectionError struct {
	// Method detection method where error occurred
	Method DetectionMethod `json:"method"`

	// SourceID template/script/target that caused error
	SourceID string `json:"source_id"`

	// Error error message
	Error string `json:"error"`

	// Timestamp when error occurred
	Timestamp time.Time `json:"timestamp"`
}

// DetectionMethod represents the type of detection used
type DetectionMethod string

const (
	DetectionMethodTemplate DetectionMethod = "template"
	DetectionMethodScript   DetectionMethod = "script"
	DetectionMethodHash     DetectionMethod = "hash"
)

// SeverityLevel represents vulnerability severity
type SeverityLevel string

const (
	SeverityLevelCritical SeverityLevel = "critical"
	SeverityLevelHigh     SeverityLevel = "high"
	SeverityLevelMedium   SeverityLevel = "medium"
	SeverityLevelLow      SeverityLevel = "low"
	SeverityLevelInfo     SeverityLevel = "info"
)

// EvidenceType represents the type of evidence found
type EvidenceType string

const (
	EvidenceTypeFileHash      EvidenceType = "file_hash"
	EvidenceTypeRegistryKey   EvidenceType = "registry_key"
	EvidenceTypeConfigPattern EvidenceType = "config_pattern"
	EvidenceTypeScriptOutput  EvidenceType = "script_output"
	EvidenceTypeFileContent   EvidenceType = "file_content"
	EvidenceTypeProcessList   EvidenceType = "process_list"
	EvidenceTypeNetworkPort   EvidenceType = "network_port"
)
