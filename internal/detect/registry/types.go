package registry

import (
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
)

// RegistryResult represents the result of checking a Windows registry key
type RegistryResult struct {
	// Target the original registry target that was checked
	Target detect.RegistryTarget `json:"target"`

	// KeyExists whether the registry key exists
	KeyExists bool `json:"key_exists"`

	// ValueExists whether the specific value exists (if checked)
	ValueExists bool `json:"value_exists"`

	// ValueData the actual value data (as string)
	ValueData string `json:"value_data,omitempty"`

	// ValueType the registry value type (REG_SZ, REG_DWORD, etc.)
	ValueType string `json:"value_type,omitempty"`

	// PatternMatches whether the value matches the specified pattern
	PatternMatches bool `json:"pattern_matches"`

	// CheckedAt when the registry check was performed
	CheckedAt time.Time `json:"checked_at"`

	// ProcessingTime how long the registry check took
	ProcessingTime time.Duration `json:"processing_time"`

	// Error any error that occurred during the check
	Error string `json:"error,omitempty"`
}

// RegistryValue represents a Windows registry value with type information
type RegistryValue struct {
	// Exists whether the value exists
	Exists bool `json:"exists"`

	// Data the value data as a string
	Data string `json:"data"`

	// Type the registry value type
	Type string `json:"type"`
}

// RegistryCheckRequest represents a request to check registry keys
type RegistryCheckRequest struct {
	// Keys list of registry targets to check
	Keys []detect.RegistryTarget `json:"keys"`

	// Timeout maximum processing time
	Timeout string `json:"timeout,omitempty"`

	// Concurrent whether to process keys concurrently
	Concurrent bool `json:"concurrent,omitempty"`
}

// RegistryCheckResponse contains results of registry key checks
type RegistryCheckResponse struct {
	// Results check results for each registry key
	Results []*RegistryResult `json:"results"`

	// Summary overall check summary
	Summary RegistryCheckSummary `json:"summary"`

	// ExecutionTime total time taken for all checks
	ExecutionTime time.Duration `json:"execution_time"`

	// Platform the platform where checks were performed
	Platform string `json:"platform"`

	// PowerShellAvailable whether PowerShell was available for checks
	PowerShellAvailable bool `json:"powershell_available"`
}

// RegistryCheckSummary provides aggregated registry check statistics
type RegistryCheckSummary struct {
	// TotalKeys total number of keys checked
	TotalKeys int `json:"total_keys"`

	// ExistingKeys number of keys that exist
	ExistingKeys int `json:"existing_keys"`

	// ValuesChecked number of values that were checked
	ValuesChecked int `json:"values_checked"`

	// ExistingValues number of values that exist
	ExistingValues int `json:"existing_values"`

	// PatternMatches number of values that matched patterns
	PatternMatches int `json:"pattern_matches"`

	// ErrorKeys number of keys that had check errors
	ErrorKeys int `json:"error_keys"`

	// AverageProcessingTime average time per key check
	AverageProcessingTime time.Duration `json:"average_processing_time"`
}

// RegistryKeyInfo contains metadata about a registry key
type RegistryKeyInfo struct {
	// Path registry key path
	Path string `json:"path"`

	// Exists whether the key exists
	Exists bool `json:"exists"`

	// ValueCount number of values in the key
	ValueCount int `json:"value_count"`

	// SubkeyCount number of subkeys
	SubkeyCount int `json:"subkey_count"`

	// LastWriteTime when the key was last modified
	LastWriteTime time.Time `json:"last_write_time,omitempty"`

	// AccessError any error accessing the key
	AccessError string `json:"access_error,omitempty"`
}

// RegistryValueInfo contains detailed information about a registry value
type RegistryValueInfo struct {
	// Name value name
	Name string `json:"name"`

	// Data value data
	Data interface{} `json:"data"`

	// Type registry value type
	Type RegistryValueType `json:"type"`

	// Size data size in bytes
	Size int `json:"size"`

	// Readable whether the value is readable
	Readable bool `json:"readable"`
}

// RegistryValueType represents Windows registry value types
type RegistryValueType string

const (
	// Registry value types
	RegistryValueTypeString       RegistryValueType = "REG_SZ"
	RegistryValueTypeExpandString RegistryValueType = "REG_EXPAND_SZ"
	RegistryValueTypeBinary       RegistryValueType = "REG_BINARY"
	RegistryValueTypeDWORD        RegistryValueType = "REG_DWORD"
	RegistryValueTypeQWORD        RegistryValueType = "REG_QWORD"
	RegistryValueTypeMultiString  RegistryValueType = "REG_MULTI_SZ"
	RegistryValueTypeNone         RegistryValueType = "REG_NONE"
)

// RegistryHive represents Windows registry hives
type RegistryHive string

const (
	// Registry hives
	HiveClassesRoot   RegistryHive = "HKEY_CLASSES_ROOT"
	HiveCurrentUser   RegistryHive = "HKEY_CURRENT_USER"
	HiveLocalMachine  RegistryHive = "HKEY_LOCAL_MACHINE"
	HiveUsers         RegistryHive = "HKEY_USERS"
	HiveCurrentConfig RegistryHive = "HKEY_CURRENT_CONFIG"
)

// IsValidHive checks if a registry hive is valid
func (rh RegistryHive) IsValid() bool {
	switch rh {
	case HiveClassesRoot, HiveCurrentUser, HiveLocalMachine, HiveUsers, HiveCurrentConfig:
		return true
	default:
		return false
	}
}

// RegistryError represents a registry operation error
type RegistryError struct {
	// Operation the operation that failed
	Operation string `json:"operation"`

	// KeyPath the registry key path
	KeyPath string `json:"key_path"`

	// ValueName the value name (if applicable)
	ValueName string `json:"value_name,omitempty"`

	// ErrorCode Windows error code (if available)
	ErrorCode int `json:"error_code,omitempty"`

	// Message error message
	Message string `json:"message"`

	// Timestamp when the error occurred
	Timestamp time.Time `json:"timestamp"`
}
