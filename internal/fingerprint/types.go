package fingerprint

import (
	"time"
)

// SystemFingerprint represents comprehensive system information
type SystemFingerprint struct {
	// CollectedAt when fingerprinting was performed
	CollectedAt time.Time `json:"collected_at"`

	// Platform operating system platform
	Platform Platform `json:"platform"`

	// Hardware hardware information
	Hardware *HardwareInfo `json:"hardware"`

	// Network network configuration
	Network *NetworkInfo `json:"network"`

	// Users user and group information
	Users *UserInfo `json:"users"`

	// Services service and process information
	Services *ServiceInfo `json:"services"`

	// Certificates certificate store information
	Certificates *CertificateInfo `json:"certificates"`

	// Metadata additional system metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CollectionDuration time taken to collect all information
	CollectionDuration time.Duration `json:"collection_duration"`

	// Errors any errors encountered during collection
	Errors []string `json:"errors,omitempty"`
}

// HardwareInfo contains system hardware information
type HardwareInfo struct {
	// CPU processor information
	CPU *CPUInfo `json:"cpu"`

	// Memory memory information
	Memory *MemoryInfo `json:"memory"`

	// Storage storage devices
	Storage []*StorageDevice `json:"storage"`

	// System general system information
	System *SystemInfo `json:"system"`

	// CollectedAt when hardware info was collected
	CollectedAt time.Time `json:"collected_at"`
}

// CPUInfo contains processor information
type CPUInfo struct {
	// Model CPU model name
	Model string `json:"model"`

	// Vendor CPU vendor
	Vendor string `json:"vendor"`

	// Cores number of CPU cores
	Cores int `json:"cores"`

	// Threads number of logical processors
	Threads int `json:"threads"`

	// Architecture CPU architecture
	Architecture string `json:"architecture"`

	// ClockSpeed CPU clock speed in MHz
	ClockSpeed float64 `json:"clock_speed_mhz,omitempty"`

	// CacheSize CPU cache size in KB
	CacheSize int64 `json:"cache_size_kb,omitempty"`

	// Features CPU features and capabilities
	Features []string `json:"features,omitempty"`
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	// TotalGB total memory in GB
	TotalGB float64 `json:"total_gb"`

	// AvailableGB available memory in GB
	AvailableGB float64 `json:"available_gb"`

	// UsedGB used memory in GB
	UsedGB float64 `json:"used_gb"`

	// UsagePercent memory usage percentage
	UsagePercent float64 `json:"usage_percent"`

	// SwapTotalGB total swap space in GB
	SwapTotalGB float64 `json:"swap_total_gb,omitempty"`

	// SwapUsedGB used swap space in GB
	SwapUsedGB float64 `json:"swap_used_gb,omitempty"`

	// MemoryModules physical memory modules
	MemoryModules []*MemoryModule `json:"memory_modules,omitempty"`
}

// MemoryModule represents a physical memory module
type MemoryModule struct {
	// Slot memory slot identifier
	Slot string `json:"slot"`

	// SizeGB module size in GB
	SizeGB float64 `json:"size_gb"`

	// Type memory type (DDR4, DDR5, etc.)
	Type string `json:"type"`

	// Speed memory speed in MHz
	Speed int `json:"speed_mhz,omitempty"`

	// Manufacturer module manufacturer
	Manufacturer string `json:"manufacturer,omitempty"`
}

// StorageDevice contains storage device information
type StorageDevice struct {
	// Device device identifier
	Device string `json:"device"`

	// Model device model
	Model string `json:"model,omitempty"`

	// SizeGB device size in GB
	SizeGB float64 `json:"size_gb"`

	// Type storage type (SSD, HDD, NVMe)
	Type string `json:"type"`

	// Interface storage interface (SATA, NVMe, USB)
	Interface string `json:"interface,omitempty"`

	// FileSystem file system type
	FileSystem string `json:"filesystem,omitempty"`

	// MountPoint mount point
	MountPoint string `json:"mount_point,omitempty"`

	// UsedGB used space in GB
	UsedGB float64 `json:"used_gb,omitempty"`

	// AvailableGB available space in GB
	AvailableGB float64 `json:"available_gb,omitempty"`

	// UsagePercent disk usage percentage
	UsagePercent float64 `json:"usage_percent,omitempty"`

	// SerialNumber device serial number
	SerialNumber string `json:"serial_number,omitempty"`
}

// SystemInfo contains general system information
type SystemInfo struct {
	// Hostname system hostname
	Hostname string `json:"hostname"`

	// OS operating system name
	OS string `json:"os"`

	// OSVersion operating system version
	OSVersion string `json:"os_version"`

	// Kernel kernel version
	Kernel string `json:"kernel,omitempty"`

	// Architecture system architecture
	Architecture string `json:"architecture"`

	// Uptime system uptime in seconds
	Uptime int64 `json:"uptime_seconds,omitempty"`

	// BootTime system boot time
	BootTime time.Time `json:"boot_time,omitempty"`

	// TimeZone system timezone
	TimeZone string `json:"timezone,omitempty"`

	// Language system language
	Language string `json:"language,omitempty"`

	// Domain domain membership
	Domain string `json:"domain,omitempty"`

	// Manufacturer hardware manufacturer
	Manufacturer string `json:"manufacturer,omitempty"`

	// ProductName product name
	ProductName string `json:"product_name,omitempty"`

	// SerialNumber system serial number
	SerialNumber string `json:"serial_number,omitempty"`
}

// NetworkInfo contains network configuration information
type NetworkInfo struct {
	// Interfaces network interfaces
	Interfaces []*NetworkInterface `json:"interfaces"`

	// Routes routing table entries
	Routes []*RouteEntry `json:"routes,omitempty"`

	// DNS DNS configuration
	DNS *DNSConfig `json:"dns"`

	// Connections active network connections
	Connections []*NetworkConnection `json:"connections,omitempty"`

	// CollectedAt when network info was collected
	CollectedAt time.Time `json:"collected_at"`
}

// NetworkConfiguration represents comprehensive network configuration
type NetworkConfiguration struct {
	// CollectedAt when network configuration was collected
	CollectedAt time.Time `json:"collected_at"`

	// Platform operating system platform
	Platform Platform `json:"platform"`

	// Interfaces network interfaces
	Interfaces []NetworkInterface `json:"interfaces"`

	// RoutingTable routing table entries
	RoutingTable []RouteEntry `json:"routing_table"`

	// DNSServers DNS server addresses
	DNSServers []string `json:"dns_servers"`

	// NetworkStats network statistics
	NetworkStats *NetworkStats `json:"network_stats,omitempty"`
}

// InterfaceStatistics represents network interface statistics
type InterfaceStatistics struct {
	// BytesReceived total bytes received
	BytesReceived uint64 `json:"bytes_received"`

	// BytesTransmitted total bytes transmitted
	BytesTransmitted uint64 `json:"bytes_transmitted"`

	// PacketsReceived total packets received
	PacketsReceived uint64 `json:"packets_received"`

	// PacketsTransmitted total packets transmitted
	PacketsTransmitted uint64 `json:"packets_transmitted"`

	// ReceiveErrors receive errors
	ReceiveErrors uint64 `json:"receive_errors"`

	// TransmitErrors transmit errors
	TransmitErrors uint64 `json:"transmit_errors"`

	// ReceiveDropped receive dropped packets
	ReceiveDropped uint64 `json:"receive_dropped"`

	// TransmitDropped transmit dropped packets
	TransmitDropped uint64 `json:"transmit_dropped"`
}

// NetworkStats represents overall network statistics
type NetworkStats struct {
	// ActiveConnections number of active connections
	ActiveConnections int `json:"active_connections"`

	// ListeningPorts number of listening ports
	ListeningPorts int `json:"listening_ports"`

	// TotalBandwidthMbps total bandwidth in Mbps
	TotalBandwidthMbps float64 `json:"total_bandwidth_mbps,omitempty"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	// Name interface name
	Name string `json:"name"`

	// DisplayName display name
	DisplayName string `json:"display_name,omitempty"`

	// MACAddress MAC address
	MACAddress string `json:"mac_address"`

	// IPv4Addresses IPv4 addresses
	IPv4Addresses []string `json:"ipv4_addresses"`

	// IPv6Addresses IPv6 addresses
	IPv6Addresses []string `json:"ipv6_addresses"`

	// State interface state (up, down)
	State string `json:"state"`

	// Type interface type (ethernet, wireless, loopback)
	Type string `json:"type"`

	// Speed link speed in Mbps
	Speed int64 `json:"speed_mbps,omitempty"`

	// MTU maximum transmission unit
	MTU int `json:"mtu,omitempty"`

	// Statistics interface statistics
	Statistics *InterfaceStatistics `json:"statistics,omitempty"`

	// BytesSent bytes sent (deprecated, use Statistics)
	BytesSent int64 `json:"bytes_sent,omitempty"`

	// BytesReceived bytes received (deprecated, use Statistics)
	BytesReceived int64 `json:"bytes_received,omitempty"`
}

// RouteEntry represents a routing table entry
type RouteEntry struct {
	// Destination destination network
	Destination string `json:"destination"`

	// Gateway gateway address
	Gateway string `json:"gateway"`

	// Interface outgoing interface
	Interface string `json:"interface"`

	// Metric route metric
	Metric int `json:"metric,omitempty"`

	// Type route type
	Type string `json:"type,omitempty"`
}

// DNSConfig contains DNS configuration
type DNSConfig struct {
	// Servers DNS servers
	Servers []string `json:"servers"`

	// SearchDomains search domains
	SearchDomains []string `json:"search_domains,omitempty"`

	// Domain primary domain
	Domain string `json:"domain,omitempty"`

	// Hostname resolved hostname
	Hostname string `json:"hostname,omitempty"`
}

// NetworkConnection represents an active network connection
type NetworkConnection struct {
	// LocalAddress local address
	LocalAddress string `json:"local_address"`

	// LocalPort local port
	LocalPort int `json:"local_port"`

	// RemoteAddress remote address
	RemoteAddress string `json:"remote_address,omitempty"`

	// RemotePort remote port
	RemotePort int `json:"remote_port,omitempty"`

	// Protocol protocol (TCP, UDP)
	Protocol string `json:"protocol"`

	// State connection state
	State string `json:"state"`

	// ProcessID process ID
	ProcessID int `json:"process_id,omitempty"`

	// ProcessName process name
	ProcessName string `json:"process_name,omitempty"`
}

// UserInfo contains user and group information
type UserInfo struct {
	// Users local user accounts
	Users []*UserAccount `json:"users"`

	// Groups local groups
	Groups []*UserGroup `json:"groups"`

	// CurrentUser current user privileges
	CurrentUser *UserPrivileges `json:"current_user"`

	// LoginSessions active login sessions
	LoginSessions []*LoginSession `json:"login_sessions,omitempty"`

	// CollectedAt when user info was collected
	CollectedAt time.Time `json:"collected_at"`
}

// UserAccount represents a user account
type UserAccount struct {
	// Username user name
	Username string `json:"username"`

	// FullName full display name
	FullName string `json:"full_name,omitempty"`

	// UID user ID
	UID string `json:"uid"`

	// GID primary group ID
	GID string `json:"gid,omitempty"`

	// HomeDirectory home directory
	HomeDirectory string `json:"home_directory,omitempty"`

	// Shell login shell
	Shell string `json:"shell,omitempty"`

	// Enabled whether account is enabled
	Enabled bool `json:"enabled"`

	// LastLogin last login time
	LastLogin *time.Time `json:"last_login,omitempty"`

	// Groups groups user belongs to
	Groups []string `json:"groups,omitempty"`

	// PasswordSet whether password is set
	PasswordSet bool `json:"password_set"`

	// AccountType account type (local, domain)
	AccountType string `json:"account_type,omitempty"`
}

// UserGroup represents a user group
type UserGroup struct {
	// Name group name
	Name string `json:"name"`

	// GID group ID
	GID string `json:"gid"`

	// Description group description
	Description string `json:"description,omitempty"`

	// Members group members
	Members []string `json:"members,omitempty"`

	// Type group type (local, domain)
	Type string `json:"type,omitempty"`
}

// UserPrivileges represents user privileges and permissions
type UserPrivileges struct {
	// Username current username
	Username string `json:"username"`

	// IsAdmin whether user has admin privileges
	IsAdmin bool `json:"is_admin"`

	// Groups user's groups
	Groups []string `json:"groups"`

	// Privileges specific privileges
	Privileges []string `json:"privileges,omitempty"`

	// SudoAccess sudo access (Linux/macOS)
	SudoAccess bool `json:"sudo_access,omitempty"`

	// UAC user account control status (Windows)
	UAC *UACStatus `json:"uac,omitempty"`
}

// UACStatus represents Windows UAC status
type UACStatus struct {
	// Enabled whether UAC is enabled
	Enabled bool `json:"enabled"`

	// Level UAC level
	Level string `json:"level,omitempty"`

	// ElevationRequired whether elevation is required
	ElevationRequired bool `json:"elevation_required"`
}

// LoginSession represents an active login session
type LoginSession struct {
	// SessionID session identifier
	SessionID string `json:"session_id"`

	// Username session username
	Username string `json:"username"`

	// SessionType session type (console, remote)
	SessionType string `json:"session_type"`

	// LoginTime login time
	LoginTime time.Time `json:"login_time"`

	// RemoteAddress remote address (for remote sessions)
	RemoteAddress string `json:"remote_address,omitempty"`

	// Status session status
	Status string `json:"status"`
}

// ServiceInfo contains service and process information
type ServiceInfo struct {
	// Services running services
	Services []*ServiceDetails `json:"services"`

	// Processes running processes
	Processes []*ProcessDetails `json:"processes,omitempty"`

	// StartupPrograms startup programs
	StartupPrograms []*StartupProgram `json:"startup_programs,omitempty"`

	// InstalledSoftware installed software
	InstalledSoftware []*InstalledSoftware `json:"installed_software,omitempty"`

	// CollectedAt when service info was collected
	CollectedAt time.Time `json:"collected_at"`
}

// ServiceDetails represents a system service
type ServiceDetails struct {
	// Name service name
	Name string `json:"name"`

	// DisplayName display name
	DisplayName string `json:"display_name,omitempty"`

	// Status service status (running, stopped, etc.)
	Status string `json:"status"`

	// StartType start type (auto, manual, disabled)
	StartType string `json:"start_type,omitempty"`

	// ProcessID process ID
	ProcessID int `json:"process_id,omitempty"`

	// BinaryPath service binary path
	BinaryPath string `json:"binary_path,omitempty"`

	// Description service description
	Description string `json:"description,omitempty"`

	// Version service version
	Version string `json:"version,omitempty"`

	// Dependencies service dependencies
	Dependencies []string `json:"dependencies,omitempty"`
}

// ProcessDetails represents a running process
type ProcessDetails struct {
	// ProcessID process ID
	ProcessID int `json:"process_id"`

	// ParentProcessID parent process ID
	ParentProcessID int `json:"parent_process_id,omitempty"`

	// Name process name
	Name string `json:"name"`

	// ExecutablePath executable path
	ExecutablePath string `json:"executable_path,omitempty"`

	// CommandLine command line
	CommandLine string `json:"command_line,omitempty"`

	// Username process owner
	Username string `json:"username,omitempty"`

	// StartTime process start time
	StartTime time.Time `json:"start_time,omitempty"`

	// CPUPercent CPU usage percentage
	CPUPercent float64 `json:"cpu_percent,omitempty"`

	// MemoryMB memory usage in MB
	MemoryMB float64 `json:"memory_mb,omitempty"`

	// Status process status
	Status string `json:"status,omitempty"`
}

// StartupProgram represents a startup program
type StartupProgram struct {
	// Name program name
	Name string `json:"name"`

	// Path program path
	Path string `json:"path"`

	// Arguments program arguments
	Arguments string `json:"arguments,omitempty"`

	// Location startup location (registry, folder, etc.)
	Location string `json:"location"`

	// Enabled whether program is enabled
	Enabled bool `json:"enabled"`

	// RunAs run as user
	RunAs string `json:"run_as,omitempty"`
}

// InstalledSoftware represents installed software
type InstalledSoftware struct {
	// Name software name
	Name string `json:"name"`

	// Version software version
	Version string `json:"version,omitempty"`

	// Publisher software publisher
	Publisher string `json:"publisher,omitempty"`

	// InstallDate installation date
	InstallDate *time.Time `json:"install_date,omitempty"`

	// InstallLocation installation location
	InstallLocation string `json:"install_location,omitempty"`

	// SizeMB installed size in MB
	SizeMB float64 `json:"size_mb,omitempty"`

	// UninstallString uninstall command
	UninstallString string `json:"uninstall_string,omitempty"`

	// Source installation source
	Source string `json:"source,omitempty"`
}

// CertificateInfo contains certificate store information
type CertificateInfo struct {
	// SystemCertificates system certificate store
	SystemCertificates []*CertificateDetails `json:"system_certificates"`

	// UserCertificates user certificate store
	UserCertificates []*CertificateDetails `json:"user_certificates"`

	// SSLCertificates SSL certificates
	SSLCertificates []*CertificateDetails `json:"ssl_certificates,omitempty"`

	// Validations certificate validations
	Validations []*CertificateValidation `json:"validations,omitempty"`

	// CollectedAt when certificate info was collected
	CollectedAt time.Time `json:"collected_at"`
}

// CertificateDetails represents certificate information
type CertificateDetails struct {
	// Subject certificate subject
	Subject string `json:"subject"`

	// Issuer certificate issuer
	Issuer string `json:"issuer"`

	// SerialNumber certificate serial number
	SerialNumber string `json:"serial_number"`

	// NotBefore valid from date
	NotBefore time.Time `json:"not_before"`

	// NotAfter valid until date
	NotAfter time.Time `json:"not_after"`

	// Fingerprint certificate fingerprint
	Fingerprint string `json:"fingerprint"`

	// FingerprintAlgorithm fingerprint algorithm
	FingerprintAlgorithm string `json:"fingerprint_algorithm"`

	// KeyUsage key usage
	KeyUsage []string `json:"key_usage,omitempty"`

	// ExtendedKeyUsage extended key usage
	ExtendedKeyUsage []string `json:"extended_key_usage,omitempty"`

	// SubjectAlternativeNames SAN entries
	SubjectAlternativeNames []string `json:"san,omitempty"`

	// Store certificate store location
	Store string `json:"store"`

	// StoreLocation store location (system, user)
	StoreLocation string `json:"store_location"`

	// FilePath file path (for file-based certificates)
	FilePath string `json:"file_path,omitempty"`
}

// CertificateValidation represents certificate validation result
type CertificateValidation struct {
	// Certificate reference to certificate
	Certificate *CertificateDetails `json:"certificate"`

	// IsValid whether certificate is valid
	IsValid bool `json:"is_valid"`

	// IsExpired whether certificate is expired
	IsExpired bool `json:"is_expired"`

	// ExpiresInDays days until expiration
	ExpiresInDays int `json:"expires_in_days"`

	// ValidationErrors validation errors
	ValidationErrors []string `json:"validation_errors,omitempty"`

	// ValidatedAt when validation was performed
	ValidatedAt time.Time `json:"validated_at"`
}
