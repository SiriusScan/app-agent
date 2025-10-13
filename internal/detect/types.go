package detect

import (
	"time"
)

// VulnTemplate represents a YAML vulnerability template
type VulnTemplate struct {
	// ID unique template identifier
	ID string `yaml:"id" json:"id"`

	// Info template metadata
	Info TemplateInfo `yaml:"info" json:"info"`

	// Detection detection configuration
	Detection DetectionConfig `yaml:"detection" json:"detection"`

	// Remediation remediation information
	Remediation RemediationInfo `yaml:"remediation,omitempty" json:"remediation,omitempty"`

	// FilePath path to the template file
	FilePath string `yaml:"-" json:"file_path,omitempty"`

	// LoadedAt when template was loaded
	LoadedAt time.Time `yaml:"-" json:"loaded_at,omitempty"`
}

// TemplateInfo contains template metadata
type TemplateInfo struct {
	// Name human-readable template name
	Name string `yaml:"name" json:"name"`

	// Author template author
	Author string `yaml:"author" json:"author"`

	// Severity vulnerability severity
	Severity SeverityLevel `yaml:"severity" json:"severity"`

	// Description template description
	Description string `yaml:"description" json:"description"`

	// References related links and documentation
	References []string `yaml:"references,omitempty" json:"references,omitempty"`

	// CVE associated CVE identifier
	CVE string `yaml:"cve,omitempty" json:"cve,omitempty"`

	// Tags classification tags
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Version template version
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// DetectionConfig specifies how to detect the vulnerability
type DetectionConfig struct {
	// Type detection type (file-hash, registry, config-file)
	Type DetectionType `yaml:"type" json:"type"`

	// Method specific detection method
	Method string `yaml:"method,omitempty" json:"method,omitempty"`

	// Targets files, registry keys, or other targets to check
	Targets []DetectionTarget `yaml:"targets,omitempty" json:"targets,omitempty"`

	// Files configuration files to analyze
	Files []FileTarget `yaml:"files,omitempty" json:"files,omitempty"`

	// Keys registry keys to check (Windows only)
	Keys []RegistryTarget `yaml:"keys,omitempty" json:"keys,omitempty"`

	// Conditions conditions that must be met
	Conditions []DetectionCondition `yaml:"conditions" json:"conditions"`

	// Metadata additional detection metadata
	Metadata map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// DetectionType represents the type of detection method
type DetectionType string

const (
	DetectionTypeFileHash   DetectionType = "file-hash"
	DetectionTypeRegistry   DetectionType = "registry"
	DetectionTypeConfigFile DetectionType = "config-file"
	DetectionTypeProcess    DetectionType = "process"
	DetectionTypeService    DetectionType = "service"
	DetectionTypeNetwork    DetectionType = "network"
)

// DetectionTarget represents a file or system component to check
type DetectionTarget struct {
	// Path file path or system identifier
	Path string `yaml:"path" json:"path"`

	// Hash expected hash value
	Hash string `yaml:"hash,omitempty" json:"hash,omitempty"`

	// Description target description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Platform target platform (windows, linux, darwin)
	Platform []string `yaml:"platform,omitempty" json:"platform,omitempty"`
}

// FileTarget represents a configuration file to analyze
type FileTarget struct {
	// Path file path
	Path string `yaml:"path" json:"path"`

	// Patterns regex patterns to match
	Patterns []PatternMatch `yaml:"patterns" json:"patterns"`

	// Encoding file encoding (utf-8, ascii, etc.)
	Encoding string `yaml:"encoding,omitempty" json:"encoding,omitempty"`

	// MaxSize maximum file size to process
	MaxSize int64 `yaml:"max_size,omitempty" json:"max_size,omitempty"`
}

// RegistryTarget represents a Windows registry key to check
type RegistryTarget struct {
	// Path registry key path
	Path string `yaml:"path" json:"path"`

	// Value registry value name
	Value string `yaml:"value,omitempty" json:"value,omitempty"`

	// Pattern regex pattern to match against value
	Pattern string `yaml:"pattern,omitempty" json:"pattern,omitempty"`

	// Description target description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// PatternMatch represents a regex pattern to match
type PatternMatch struct {
	// Regex regular expression pattern
	Regex string `yaml:"regex" json:"regex"`

	// Description pattern description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Negate whether to negate the match
	Negate bool `yaml:"negate,omitempty" json:"negate,omitempty"`
}

// DetectionCondition represents a condition that must be met
type DetectionCondition struct {
	// Type condition type
	Type ConditionType `yaml:"type" json:"type"`

	// Value expected value
	Value interface{} `yaml:"value,omitempty" json:"value,omitempty"`

	// Operator comparison operator
	Operator string `yaml:"operator,omitempty" json:"operator,omitempty"`
}

// ConditionType represents the type of condition
type ConditionType string

const (
	ConditionTypeFileExists          ConditionType = "file_exists"
	ConditionTypeHashMatch           ConditionType = "hash_match"
	ConditionTypeFileExecutable      ConditionType = "file_executable"
	ConditionTypeKeyExists           ConditionType = "key_exists"
	ConditionTypeValueMatchesPattern ConditionType = "value_matches_pattern"
	ConditionTypePatternFound        ConditionType = "pattern_found"
)

// RemediationInfo provides remediation guidance
type RemediationInfo struct {
	// Description remediation description
	Description string `yaml:"description" json:"description"`

	// Commands platform-specific remediation commands
	Commands map[string]string `yaml:"commands,omitempty" json:"commands,omitempty"`

	// Verification verification command and expected output
	Verification VerificationInfo `yaml:"verification,omitempty" json:"verification,omitempty"`

	// References additional remediation resources
	References []string `yaml:"references,omitempty" json:"references,omitempty"`
}

// VerificationInfo describes how to verify remediation
type VerificationInfo struct {
	// Command verification command
	Command string `yaml:"command" json:"command"`

	// ExpectedPattern expected output pattern
	ExpectedPattern string `yaml:"expected_pattern,omitempty" json:"expected_pattern,omitempty"`

	// SuccessCodes expected exit codes for success
	SuccessCodes []int `yaml:"success_codes,omitempty" json:"success_codes,omitempty"`
}

// DetectionScript represents a vulnerability detection script
type DetectionScript struct {
	// Name script name
	Name string `json:"name"`

	// Path script file path
	Path string `json:"path"`

	// Platform target platform (windows, linux, darwin)
	Platform string `json:"platform"`

	// Language script language (powershell, bash, python)
	Language ScriptLanguage `json:"language"`

	// VulnerabilityID associated vulnerability
	VulnerabilityID string `json:"vulnerability_id"`

	// Severity vulnerability severity
	Severity SeverityLevel `json:"severity"`

	// Description script description
	Description string `json:"description"`

	// Author script author
	Author string `json:"author"`

	// Version script version
	Version string `json:"version"`

	// Timeout maximum execution time
	Timeout time.Duration `json:"timeout"`

	// RequiredPrivileges privileges needed to run
	RequiredPrivileges []string `json:"required_privileges,omitempty"`

	// Parameters script parameters
	Parameters []ScriptParameter `json:"parameters,omitempty"`

	// Checksum script content checksum
	Checksum string `json:"checksum"`

	// LoadedAt when script was loaded
	LoadedAt time.Time `json:"loaded_at,omitempty"`
}

// ScriptLanguage represents script execution language
type ScriptLanguage string

const (
	ScriptLanguagePowerShell ScriptLanguage = "powershell"
	ScriptLanguageBash       ScriptLanguage = "bash"
	ScriptLanguagePython     ScriptLanguage = "python"
	ScriptLanguageShell      ScriptLanguage = "shell"
)

// ScriptParameter represents a script parameter
type ScriptParameter struct {
	// Name parameter name
	Name string `json:"name"`

	// Type parameter type
	Type string `json:"type"`

	// Required whether parameter is required
	Required bool `json:"required"`

	// Default default value
	Default interface{} `json:"default,omitempty"`

	// Description parameter description
	Description string `json:"description,omitempty"`
}

// HashAlgorithm represents supported hash algorithms
type HashAlgorithm string

const (
	HashAlgorithmSHA256 HashAlgorithm = "sha256"
	HashAlgorithmSHA1   HashAlgorithm = "sha1"
	HashAlgorithmMD5    HashAlgorithm = "md5"
	HashAlgorithmSHA512 HashAlgorithm = "sha512"
)

// HashTarget represents a file to hash check
type HashTarget struct {
	// Path file path
	Path string `json:"path"`

	// ExpectedHash expected hash value
	ExpectedHash string `json:"expected_hash"`

	// Algorithm hash algorithm to use
	Algorithm HashAlgorithm `json:"algorithm"`

	// Description target description
	Description string `json:"description,omitempty"`

	// VulnerabilityID associated vulnerability
	VulnerabilityID string `json:"vulnerability_id,omitempty"`
}

// HashResult represents result of hash verification
type HashResult struct {
	// Target original hash target
	Target HashTarget `json:"target"`

	// ActualHash calculated hash value
	ActualHash string `json:"actual_hash"`

	// Matches whether hashes match
	Matches bool `json:"matches"`

	// FileExists whether file exists
	FileExists bool `json:"file_exists"`

	// Error any error that occurred
	Error string `json:"error,omitempty"`

	// CheckedAt when hash was calculated
	CheckedAt time.Time `json:"checked_at"`
}
