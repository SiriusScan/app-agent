package fingerprint

import (
	"context"
	"time"
)

// SystemFingerprinter is the main interface for system fingerprinting
type SystemFingerprinter interface {
	// CollectSystemInfo gathers comprehensive system information
	CollectSystemInfo(ctx context.Context) (*SystemFingerprint, error)

	// CollectHardwareInfo gathers hardware information
	CollectHardwareInfo(ctx context.Context) (*HardwareInfo, error)

	// CollectNetworkInfo gathers network configuration
	CollectNetworkInfo(ctx context.Context) (*NetworkInfo, error)

	// CollectUserInfo gathers user and group information
	CollectUserInfo(ctx context.Context) (*UserInfo, error)

	// CollectServiceInfo gathers running services information
	CollectServiceInfo(ctx context.Context) (*ServiceInfo, error)

	// CollectCertificateInfo gathers certificate store information
	CollectCertificateInfo(ctx context.Context) (*CertificateInfo, error)
}

// HardwareCollector collects hardware-specific information
type HardwareCollector interface {
	// GetCPUInfo retrieves CPU information
	GetCPUInfo(ctx context.Context) (*CPUInfo, error)

	// GetMemoryInfo retrieves memory information
	GetMemoryInfo(ctx context.Context) (*MemoryInfo, error)

	// GetStorageInfo retrieves storage device information
	GetStorageInfo(ctx context.Context) ([]*StorageDevice, error)

	// GetSystemInfo retrieves general system information
	GetSystemInfo(ctx context.Context) (*SystemInfo, error)
}

// NetworkCollector collects network configuration information
type NetworkCollector interface {
	// GetNetworkInterfaces retrieves all network interfaces
	GetNetworkInterfaces(ctx context.Context) ([]*NetworkInterface, error)

	// GetRoutingTable retrieves routing table information
	GetRoutingTable(ctx context.Context) ([]*RouteEntry, error)

	// GetDNSConfiguration retrieves DNS configuration
	GetDNSConfiguration(ctx context.Context) (*DNSConfig, error)

	// GetActiveConnections retrieves active network connections
	GetActiveConnections(ctx context.Context) ([]*NetworkConnection, error)
}

// UserCollector collects user and group information
type UserCollector interface {
	// GetLocalUsers retrieves local user accounts
	GetLocalUsers(ctx context.Context) ([]*UserAccount, error)

	// GetLocalGroups retrieves local groups
	GetLocalGroups(ctx context.Context) ([]*UserGroup, error)

	// GetCurrentUserPrivileges retrieves current user privileges
	GetCurrentUserPrivileges(ctx context.Context) (*UserPrivileges, error)

	// GetLoginSessions retrieves active login sessions
	GetLoginSessions(ctx context.Context) ([]*LoginSession, error)
}

// ServiceCollector collects service and process information
type ServiceCollector interface {
	// GetRunningServices retrieves running services
	GetRunningServices(ctx context.Context) ([]*ServiceDetails, error)

	// GetRunningProcesses retrieves running processes
	GetRunningProcesses(ctx context.Context) ([]*ProcessDetails, error)

	// GetStartupPrograms retrieves startup programs
	GetStartupPrograms(ctx context.Context) ([]*StartupProgram, error)

	// GetInstalledSoftware retrieves installed software list
	GetInstalledSoftware(ctx context.Context) ([]*InstalledSoftware, error)
}

// CertificateCollector collects certificate store information
type CertificateCollector interface {
	// CollectCertificateInfo gathers certificate store information
	CollectCertificateInfo(ctx context.Context) (*CertificateInfo, error)

	// GetSystemCertificates retrieves system certificate store
	GetSystemCertificates(ctx context.Context) ([]*CertificateDetails, error)

	// GetUserCertificates retrieves user certificate store
	GetUserCertificates(ctx context.Context) ([]*CertificateDetails, error)

	// GetSSLCertificates retrieves SSL certificates from common locations
	GetSSLCertificates(ctx context.Context) ([]*CertificateDetails, error)

	// ValidateCertificates checks certificate validity and expiration
	ValidateCertificates(ctx context.Context, certs []*CertificateDetails) ([]*CertificateValidation, error)
}

// PlatformSpecificCollector handles platform-specific data collection
type PlatformSpecificCollector interface {
	// GetPlatform returns the current platform
	GetPlatform() Platform

	// IsSupported checks if a collection method is supported on this platform
	IsSupported(method CollectionMethod) bool

	// ExecutePlatformCommand executes a platform-specific command
	ExecutePlatformCommand(ctx context.Context, command PlatformCommand) (*CommandResult, error)
}

// Platform represents the operating system platform
type Platform string

const (
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
	PlatformMacOS   Platform = "darwin"
	PlatformUnknown Platform = "unknown"
)

// CollectionMethod represents different data collection methods
type CollectionMethod string

const (
	CollectionMethodWMI        CollectionMethod = "wmi"
	CollectionMethodRegistry   CollectionMethod = "registry"
	CollectionMethodFileSystem CollectionMethod = "filesystem"
	CollectionMethodCommand    CollectionMethod = "command"
	CollectionMethodAPI        CollectionMethod = "api"
	CollectionMethodProcFS     CollectionMethod = "procfs"
	CollectionMethodSysFS      CollectionMethod = "sysfs"
)

// PlatformCommand represents a platform-specific command to execute
type PlatformCommand struct {
	// Command command to execute
	Command string `json:"command"`

	// Args command arguments
	Args []string `json:"args"`

	// Timeout command timeout
	Timeout time.Duration `json:"timeout"`

	// WorkingDir working directory
	WorkingDir string `json:"working_dir,omitempty"`

	// Environment environment variables
	Environment map[string]string `json:"environment,omitempty"`

	// RequiresSudo whether command requires elevated privileges
	RequiresSudo bool `json:"requires_sudo"`
}

// CommandResult represents the result of executing a platform command
type CommandResult struct {
	// ExitCode command exit code
	ExitCode int `json:"exit_code"`

	// Stdout standard output
	Stdout string `json:"stdout"`

	// Stderr standard error
	Stderr string `json:"stderr"`

	// Duration execution duration
	Duration time.Duration `json:"duration"`

	// Error any error that occurred
	Error string `json:"error,omitempty"`

	// ExecutedAt when command was executed
	ExecutedAt time.Time `json:"executed_at"`
}
