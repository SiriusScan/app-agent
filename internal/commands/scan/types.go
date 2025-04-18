package scan

// ScanResult holds the aggregated results from a scan command.
type ScanResult struct {
	OSInfo         OSInfo                      `json:"osInfo"`
	Packages       []InstalledPackage          `json:"packages,omitempty"`
	Patches        interface{}                 `json:"patches,omitempty"`        // Placeholder for future implementation
	Configurations interface{}                 `json:"configurations,omitempty"` // Placeholder for future implementation
	NetworkInfo    interface{}                 `json:"networkInfo,omitempty"`    // Placeholder for future implementation
	UserInfo       interface{}                 `json:"userInfo,omitempty"`       // Placeholder for future implementation
	CustomResults  map[string]CustomScanOutput `json:"customResults,omitempty"`  // Results from custom scripts
	ScanErrors     []string                    `json:"scanErrors,omitempty"`     // Errors encountered during scan execution
}

// OSInfo contains basic operating system details.
type OSInfo struct {
	OS        string `json:"os"`
	Version   string `json:"version"`
	Hostname  string `json:"hostname"`
	PrimaryIP string `json:"primaryIp"`
}

// InstalledPackage represents a software package found on the system.
type InstalledPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"` // e.g., "dpkg", "rpm", "brew", "windows_registry", "winget"
}

// CustomScanOutput holds the raw results from executing a custom script.
type CustomScanOutput struct {
	ScriptName string `json:"scriptName"`
	StdOut     string `json:"stdout,omitempty"`
	StdErr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exitCode"`
	Error      string `json:"error,omitempty"` // Error message from agent execution (e.g., script not found, execution failed)
}
