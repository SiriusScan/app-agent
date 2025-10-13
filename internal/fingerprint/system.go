package fingerprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SystemHardwareCollector implements HardwareCollector interface
type SystemHardwareCollector struct {
	timeout time.Duration
}

// NewSystemHardwareCollector creates a new hardware collector
func NewSystemHardwareCollector() HardwareCollector {
	return &SystemHardwareCollector{
		timeout: 30 * time.Second,
	}
}

// GetCPUInfo retrieves CPU information
func (c *SystemHardwareCollector) GetCPUInfo(ctx context.Context) (*CPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.collectLinuxCPUInfo(ctx)
	case "windows":
		return c.collectWindowsCPUInfo(ctx)
	case "darwin":
		return c.collectMacOSCPUInfo(ctx)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// GetMemoryInfo retrieves memory information
func (c *SystemHardwareCollector) GetMemoryInfo(ctx context.Context) (*MemoryInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.collectLinuxMemoryInfo(ctx)
	case "windows":
		return c.collectWindowsMemoryInfo(ctx)
	case "darwin":
		return c.collectMacOSMemoryInfo(ctx)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// GetStorageInfo retrieves storage device information
func (c *SystemHardwareCollector) GetStorageInfo(ctx context.Context) ([]*StorageDevice, error) {
	switch runtime.GOOS {
	case "linux":
		return c.collectLinuxStorageInfo(ctx)
	case "windows":
		return c.collectWindowsStorageInfo(ctx)
	case "darwin":
		return c.collectMacOSStorageInfo(ctx)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// GetSystemInfo retrieves general system information
func (c *SystemHardwareCollector) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.collectLinuxSystemInfo(ctx)
	case "windows":
		return c.collectWindowsSystemInfo(ctx)
	case "darwin":
		return c.collectMacOSSystemInfo(ctx)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// CollectHardwareInfo collects comprehensive hardware information (helper method)
func (c *SystemHardwareCollector) CollectHardwareInfo(ctx context.Context) (*HardwareInfo, error) {
	startTime := time.Now()

	hardware := &HardwareInfo{
		CollectedAt: startTime,
	}

	var errors []string

	// Collect CPU information
	if cpu, err := c.GetCPUInfo(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("CPU collection failed: %v", err))
	} else {
		hardware.CPU = cpu
	}

	// Collect Memory information
	if memory, err := c.GetMemoryInfo(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Memory collection failed: %v", err))
	} else {
		hardware.Memory = memory
	}

	// Collect Storage information
	if storage, err := c.GetStorageInfo(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("Storage collection failed: %v", err))
	} else {
		hardware.Storage = storage
	}

	// Collect System information
	if system, err := c.GetSystemInfo(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("System collection failed: %v", err))
	} else {
		hardware.System = system
	}

	// If we have any errors but some data was collected, return partial results
	if len(errors) > 0 && (hardware.CPU != nil || hardware.Memory != nil || len(hardware.Storage) > 0) {
		return hardware, fmt.Errorf("partial hardware collection completed with errors: %v", errors)
	} else if len(errors) > 0 {
		return nil, fmt.Errorf("hardware collection failed: %v", errors)
	}

	return hardware, nil
}

// Linux-specific implementations
func (c *SystemHardwareCollector) collectLinuxCPUInfo(ctx context.Context) (*CPUInfo, error) {
	cpuInfo := &CPUInfo{
		Architecture: runtime.GOARCH,
	}

	// Read /proc/cpuinfo
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/cpuinfo: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	coreCount := 0
	processorCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "model name":
			if cpuInfo.Model == "" {
				cpuInfo.Model = value
			}
		case "vendor_id":
			if cpuInfo.Vendor == "" {
				cpuInfo.Vendor = value
			}
		case "processor":
			processorCount++
		case "cpu cores":
			if cores, err := strconv.Atoi(value); err == nil && cpuInfo.Cores == 0 {
				coreCount = cores
			}
		case "siblings":
			if threads, err := strconv.Atoi(value); err == nil && cpuInfo.Threads == 0 {
				cpuInfo.Threads = threads
			}
		case "cpu MHz":
			if speed, err := strconv.ParseFloat(value, 64); err == nil && cpuInfo.ClockSpeed == 0 {
				cpuInfo.ClockSpeed = speed
			}
		case "cache size":
			if strings.Contains(value, "KB") {
				sizeStr := strings.TrimSuffix(value, " KB")
				if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
					cpuInfo.CacheSize = size
				}
			}
		case "flags":
			cpuInfo.Features = strings.Fields(value)
		}
	}

	// Set core count (fallback to processor count if cores not available)
	if coreCount > 0 {
		cpuInfo.Cores = coreCount
	} else if processorCount > 0 {
		cpuInfo.Cores = processorCount
	}

	// If threads not set, assume equal to cores
	if cpuInfo.Threads == 0 {
		cpuInfo.Threads = cpuInfo.Cores
	}

	return cpuInfo, nil
}

func (c *SystemHardwareCollector) collectLinuxMemoryInfo(ctx context.Context) (*MemoryInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %v", err)
	}

	memInfo := &MemoryInfo{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSuffix(parts[0], ":")
		valueStr := parts[1]
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		// Convert from KB to GB
		valueGB := value / 1024 / 1024

		switch key {
		case "MemTotal":
			memInfo.TotalGB = valueGB
		case "MemAvailable":
			memInfo.AvailableGB = valueGB
		case "SwapTotal":
			memInfo.SwapTotalGB = valueGB
		case "SwapFree":
			memInfo.SwapUsedGB = memInfo.SwapTotalGB - valueGB
		}
	}

	// Calculate used memory and usage percentage
	memInfo.UsedGB = memInfo.TotalGB - memInfo.AvailableGB
	if memInfo.TotalGB > 0 {
		memInfo.UsagePercent = (memInfo.UsedGB / memInfo.TotalGB) * 100
	}

	return memInfo, nil
}

func (c *SystemHardwareCollector) collectLinuxStorageInfo(ctx context.Context) ([]*StorageDevice, error) {
	var devices []*StorageDevice

	// Use lsblk to get storage information
	cmd := exec.CommandContext(ctx, "lsblk", "-J", "-b", "-o", "NAME,SIZE,TYPE,MODEL,SERIAL,FSTYPE")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk command failed: %v", err)
	}

	var lsblkOutput struct {
		BlockDevices []struct {
			Name   string `json:"name"`
			Size   string `json:"size"`
			Type   string `json:"type"`
			Model  string `json:"model"`
			Serial string `json:"serial"`
			FSType string `json:"fstype"`
		} `json:"blockdevices"`
	}

	if err := json.Unmarshal(output, &lsblkOutput); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %v", err)
	}

	for _, device := range lsblkOutput.BlockDevices {
		if device.Type == "disk" {
			sizeBytes, _ := strconv.ParseInt(device.Size, 10, 64)
			sizeGB := float64(sizeBytes) / 1024 / 1024 / 1024

			storageDevice := &StorageDevice{
				Device:       "/dev/" + device.Name,
				Model:        device.Model,
				SizeGB:       sizeGB,
				Type:         c.getLinuxStorageType(device.Name),
				Interface:    c.getLinuxStorageInterface(device.Name),
				SerialNumber: device.Serial,
			}

			devices = append(devices, storageDevice)
		}
	}

	return devices, nil
}

func (c *SystemHardwareCollector) getLinuxStorageType(deviceName string) string {
	// Check if it's an NVMe device
	if strings.HasPrefix(deviceName, "nvme") {
		return "NVMe"
	}

	// Check if it's an SSD by looking at rotational flag
	rotationalPath := fmt.Sprintf("/sys/block/%s/queue/rotational", deviceName)
	if data, err := os.ReadFile(rotationalPath); err == nil {
		if strings.TrimSpace(string(data)) == "0" {
			return "SSD"
		}
		return "HDD"
	}

	return "Unknown"
}

func (c *SystemHardwareCollector) getLinuxStorageInterface(deviceName string) string {
	if strings.HasPrefix(deviceName, "nvme") {
		return "NVMe"
	} else if strings.HasPrefix(deviceName, "sd") {
		return "SATA"
	} else if strings.HasPrefix(deviceName, "hd") {
		return "IDE"
	}
	return "Unknown"
}

func (c *SystemHardwareCollector) collectLinuxSystemInfo(ctx context.Context) (*SystemInfo, error) {
	system := &SystemInfo{
		OS:           "Linux",
		Architecture: runtime.GOARCH,
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		system.Hostname = hostname
	}

	// Get OS version from /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				version := strings.TrimPrefix(line, "PRETTY_NAME=")
				version = strings.Trim(version, "\"")
				system.OSVersion = version
				break
			}
		}
	}

	// Get kernel version
	if data, err := os.ReadFile("/proc/version"); err == nil {
		versionStr := string(data)
		re := regexp.MustCompile(`Linux version ([^\s]+)`)
		if matches := re.FindStringSubmatch(versionStr); len(matches) > 1 {
			system.Kernel = matches[1]
		}
	}

	// Get uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if uptimeFloat, err := strconv.ParseFloat(fields[0], 64); err == nil {
				system.Uptime = int64(uptimeFloat)
				system.BootTime = time.Now().Add(-time.Duration(system.Uptime) * time.Second)
			}
		}
	}

	// Get timezone
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		system.TimeZone = strings.TrimSpace(string(data))
	} else if tz := os.Getenv("TZ"); tz != "" {
		system.TimeZone = tz
	} else {
		_, offset := time.Now().Zone()
		system.TimeZone = fmt.Sprintf("UTC%+d", offset/3600)
	}

	return system, nil
}

// Windows-specific implementations
func (c *SystemHardwareCollector) collectWindowsCPUInfo(ctx context.Context) (*CPUInfo, error) {
	cpuInfo := &CPUInfo{
		Architecture: runtime.GOARCH,
	}

	// Use PowerShell to query WMI for CPU information
	script := `Get-WmiObject -Class Win32_Processor | Select-Object Name, Manufacturer, NumberOfCores, NumberOfLogicalProcessors, MaxClockSpeed, L3CacheSize | ConvertTo-Json`

	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query CPU info via WMI: %v", err)
	}

	// Handle both single CPU and multiple CPU scenarios
	var cpuDataArray []struct {
		Name                      string      `json:"Name"`
		Manufacturer              string      `json:"Manufacturer"`
		NumberOfCores             interface{} `json:"NumberOfCores"`             // Can be int or string
		NumberOfLogicalProcessors interface{} `json:"NumberOfLogicalProcessors"` // Can be int or string
		MaxClockSpeed             interface{} `json:"MaxClockSpeed"`             // Can be int or string
		L3CacheSize               interface{} `json:"L3CacheSize"`               // Can be int or string
	}

	// First try to unmarshal as array
	if err := json.Unmarshal(output, &cpuDataArray); err != nil {
		// If array fails, try as single object
		var singleCpuData struct {
			Name                      string      `json:"Name"`
			Manufacturer              string      `json:"Manufacturer"`
			NumberOfCores             interface{} `json:"NumberOfCores"`
			NumberOfLogicalProcessors interface{} `json:"NumberOfLogicalProcessors"`
			MaxClockSpeed             interface{} `json:"MaxClockSpeed"`
			L3CacheSize               interface{} `json:"L3CacheSize"`
		}
		if singleErr := json.Unmarshal(output, &singleCpuData); singleErr != nil {
			return nil, fmt.Errorf("failed to parse CPU WMI output: %v (original array error: %v)", singleErr, err)
		}
		cpuDataArray = append(cpuDataArray, singleCpuData)
	}

	if len(cpuDataArray) == 0 {
		return nil, fmt.Errorf("no CPU data found")
	}

	// Use the first CPU for now (could aggregate in the future)
	cpuData := cpuDataArray[0]

	cpuInfo.Model = cpuData.Name
	cpuInfo.Vendor = cpuData.Manufacturer

	// Parse NumberOfCores (flexible type)
	if cores := parseInterfaceToInt(cpuData.NumberOfCores); cores > 0 {
		cpuInfo.Cores = cores
	}

	// Parse NumberOfLogicalProcessors (flexible type)
	if threads := parseInterfaceToInt(cpuData.NumberOfLogicalProcessors); threads > 0 {
		cpuInfo.Threads = threads
	}

	// Parse MaxClockSpeed (flexible type)
	if clockSpeed := parseInterfaceToFloat64(cpuData.MaxClockSpeed); clockSpeed > 0 {
		cpuInfo.ClockSpeed = clockSpeed
	}

	// Parse L3CacheSize (flexible type)
	if cacheSize := parseInterfaceToInt64(cpuData.L3CacheSize); cacheSize > 0 {
		cpuInfo.CacheSize = cacheSize * 1024 // Convert to bytes
	}

	return cpuInfo, nil
}

func (c *SystemHardwareCollector) collectWindowsMemoryInfo(ctx context.Context) (*MemoryInfo, error) {
	memInfo := &MemoryInfo{}

	// Get total physical memory
	script := `Get-WmiObject -Class Win32_ComputerSystem | Select-Object TotalPhysicalMemory | ConvertTo-Json`
	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query memory info: %v", err)
	}

	var memData struct {
		TotalPhysicalMemory interface{} `json:"TotalPhysicalMemory"` // Can be number or string
	}

	if err := json.Unmarshal(output, &memData); err != nil {
		return nil, fmt.Errorf("failed to parse memory WMI output: %v", err)
	}

	// Parse TotalPhysicalMemory (flexible type)
	if totalBytes := parseInterfaceToInt64(memData.TotalPhysicalMemory); totalBytes > 0 {
		memInfo.TotalGB = float64(totalBytes) / 1024 / 1024 / 1024
	}

	// Get available memory
	script2 := `Get-WmiObject -Class Win32_OperatingSystem | Select-Object FreePhysicalMemory | ConvertTo-Json`
	cmd2 := exec.CommandContext(ctx, "powershell", "-Command", script2)
	if output2, err := cmd2.Output(); err == nil {
		var freeData struct {
			FreePhysicalMemory interface{} `json:"FreePhysicalMemory"` // Can be number or string
		}
		if err := json.Unmarshal(output2, &freeData); err == nil {
			if freeKB := parseInterfaceToInt64(freeData.FreePhysicalMemory); freeKB > 0 {
				memInfo.AvailableGB = float64(freeKB) / 1024 / 1024
			}
		}
	}

	// Calculate used memory and usage percentage
	memInfo.UsedGB = memInfo.TotalGB - memInfo.AvailableGB
	if memInfo.TotalGB > 0 {
		memInfo.UsagePercent = (memInfo.UsedGB / memInfo.TotalGB) * 100
	}

	return memInfo, nil
}

func (c *SystemHardwareCollector) collectWindowsStorageInfo(ctx context.Context) ([]*StorageDevice, error) {
	script := `Get-WmiObject -Class Win32_DiskDrive | Select-Object DeviceID, Model, Size, InterfaceType | ConvertTo-Json`

	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query storage info: %v", err)
	}

	// Handle both single disk and multiple disk scenarios
	var diskDataArray []struct {
		DeviceID      string      `json:"DeviceID"`
		Model         string      `json:"Model"`
		Size          interface{} `json:"Size"` // Can be int or string
		InterfaceType string      `json:"InterfaceType"`
	}

	// First try to unmarshal as array
	if err := json.Unmarshal(output, &diskDataArray); err != nil {
		// If array fails, try as single object
		var singleDiskData struct {
			DeviceID      string      `json:"DeviceID"`
			Model         string      `json:"Model"`
			Size          interface{} `json:"Size"`
			InterfaceType string      `json:"InterfaceType"`
		}
		if singleErr := json.Unmarshal(output, &singleDiskData); singleErr != nil {
			return nil, fmt.Errorf("failed to parse storage WMI output: %v (original array error: %v)", singleErr, err)
		}
		diskDataArray = append(diskDataArray, singleDiskData)
	}

	var devices []*StorageDevice

	for _, disk := range diskDataArray {
		// Parse Size (flexible type)
		sizeBytes := parseInterfaceToInt64(disk.Size)
		sizeGB := float64(sizeBytes) / 1024 / 1024 / 1024

		device := &StorageDevice{
			Device:    disk.DeviceID,
			Model:     disk.Model,
			SizeGB:    sizeGB,
			Interface: disk.InterfaceType,
			Type:      c.getWindowsStorageType(disk.InterfaceType),
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// Helper functions to parse interface{} to different types
func parseInterfaceToInt(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

func parseInterfaceToInt64(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func parseInterfaceToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

func (c *SystemHardwareCollector) getWindowsStorageType(interfaceType string) string {
	switch strings.ToLower(interfaceType) {
	case "ide":
		return "HDD"
	case "sata":
		return "SSD"
	case "scsi":
		return "SSD"
	case "usb":
		return "USB"
	default:
		return "Unknown"
	}
}

func (c *SystemHardwareCollector) collectWindowsSystemInfo(ctx context.Context) (*SystemInfo, error) {
	system := &SystemInfo{
		OS:           "Windows",
		Architecture: runtime.GOARCH,
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		system.Hostname = hostname
	}

	// Get OS version
	script := `Get-WmiObject -Class Win32_OperatingSystem | Select-Object Caption, Version | ConvertTo-Json`
	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	if output, err := cmd.Output(); err == nil {
		var osData struct {
			Caption string `json:"Caption"`
			Version string `json:"Version"`
		}
		if err := json.Unmarshal(output, &osData); err == nil {
			system.OSVersion = fmt.Sprintf("%s (%s)", osData.Caption, osData.Version)
		}
	}

	return system, nil
}

// macOS-specific implementations
func (c *SystemHardwareCollector) collectMacOSCPUInfo(ctx context.Context) (*CPUInfo, error) {
	cpuInfo := &CPUInfo{
		Architecture: runtime.GOARCH,
	}

	// CRITICAL FIX: Add timeout to prevent system_profiler from hanging
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Use system_profiler to get CPU information
	cmd := exec.CommandContext(cmdCtx, "system_profiler", "SPHardwareDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run system_profiler: %v", err)
	}

	var profilerData struct {
		SPHardwareDataType []struct {
			ChipType         string `json:"chip_type"`
			NumberProcessors string `json:"number_processors"`
			ProcessorName    string `json:"processor_name"`
			ProcessorSpeed   string `json:"processor_speed"`
			// Apple Silicon specific fields
			ChipInfo   string `json:"chip_information"`
			TotalCores string `json:"total_number_of_cores"`
		} `json:"SPHardwareDataType"`
	}

	if err := json.Unmarshal(output, &profilerData); err != nil {
		return nil, fmt.Errorf("failed to parse system_profiler output: %v", err)
	}

	if len(profilerData.SPHardwareDataType) > 0 {
		hw := profilerData.SPHardwareDataType[0]

		// Set CPU model/name
		cpuInfo.Model = hw.ProcessorName
		if cpuInfo.Model == "" {
			cpuInfo.Model = hw.ChipType
		}
		if cpuInfo.Model == "" {
			cpuInfo.Model = hw.ChipInfo
		}

		// Parse processor count - try different fields
		if cores, err := strconv.Atoi(hw.NumberProcessors); err == nil && cores > 0 {
			cpuInfo.Cores = cores
			cpuInfo.Threads = cores
		} else if cores, err := strconv.Atoi(hw.TotalCores); err == nil && cores > 0 {
			cpuInfo.Cores = cores
			cpuInfo.Threads = cores
		}

		// Parse processor speed
		if speedStr := hw.ProcessorSpeed; speedStr != "" {
			re := regexp.MustCompile(`(\d+\.?\d*)\s*GHz`)
			if matches := re.FindStringSubmatch(speedStr); len(matches) > 1 {
				if speed, err := strconv.ParseFloat(matches[1], 64); err == nil {
					cpuInfo.ClockSpeed = speed * 1000 // Convert GHz to MHz
				}
			}
		}
	}

	// Fallback: use sysctl to get more CPU information
	if cpuInfo.Cores == 0 {
		if coresStr, err := c.runMacOSSysctl(ctx, "hw.ncpu"); err == nil {
			if cores, err := strconv.Atoi(coresStr); err == nil {
				cpuInfo.Cores = cores
				cpuInfo.Threads = cores
			}
		}
	}

	// Get CPU brand string from sysctl if model is still empty
	if cpuInfo.Model == "" {
		if brand, err := c.runMacOSSysctl(ctx, "machdep.cpu.brand_string"); err == nil {
			cpuInfo.Model = brand
		}
	}

	// Get CPU vendor from sysctl
	if vendor, err := c.runMacOSSysctl(ctx, "machdep.cpu.vendor"); err == nil {
		cpuInfo.Vendor = vendor
	}

	return cpuInfo, nil
}

func (c *SystemHardwareCollector) runMacOSSysctl(ctx context.Context, key string) (string, error) {
	// CRITICAL FIX: Add timeout to prevent sysctl from hanging
	cmdCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sysctl", "-n", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *SystemHardwareCollector) collectMacOSMemoryInfo(ctx context.Context) (*MemoryInfo, error) {
	memInfo := &MemoryInfo{}

	// CRITICAL FIX: Add timeouts to prevent hanging
	cmdCtx1, cancel1 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel1()

	// Get total memory using sysctl
	cmd := exec.CommandContext(cmdCtx1, "sysctl", "-n", "hw.memsize")
	if output, err := cmd.Output(); err == nil {
		if totalBytes, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64); err == nil {
			memInfo.TotalGB = float64(totalBytes) / 1024 / 1024 / 1024
		}
	}

	// CRITICAL FIX: Add timeout for vm_stat
	cmdCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()

	// Get memory pressure using vm_stat
	cmd2 := exec.CommandContext(cmdCtx2, "vm_stat")
	if output, err := cmd2.Output(); err == nil {
		c.parseMacOSVMStat(string(output), memInfo)
	}

	// Calculate usage percentage
	if memInfo.TotalGB > 0 {
		memInfo.UsagePercent = (memInfo.UsedGB / memInfo.TotalGB) * 100
	}

	return memInfo, nil
}

func (c *SystemHardwareCollector) parseMacOSVMStat(vmstatOutput string, memInfo *MemoryInfo) {
	lines := strings.Split(vmstatOutput, "\n")
	pageSize := int64(4096) // Default page size

	for _, line := range lines {
		if strings.Contains(line, "page size of") {
			re := regexp.MustCompile(`page size of (\d+) bytes`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				pageSize, _ = strconv.ParseInt(matches[1], 10, 64)
			}
		} else if strings.Contains(line, "Pages free:") {
			re := regexp.MustCompile(`Pages free:\s+(\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				if freePages, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
					memInfo.AvailableGB = float64(freePages*pageSize) / 1024 / 1024 / 1024
				}
			}
		}
	}

	// Calculate used memory
	memInfo.UsedGB = memInfo.TotalGB - memInfo.AvailableGB
}

func (c *SystemHardwareCollector) collectMacOSStorageInfo(ctx context.Context) ([]*StorageDevice, error) {
	// Use df to get storage information
	cmd := exec.CommandContext(ctx, "df", "-h")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run df: %v", err)
	}

	var devices []*StorageDevice
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 || !strings.HasPrefix(fields[0], "/dev/") {
			continue
		}

		device := fields[0]
		sizeStr := fields[1]
		usedStr := fields[2]
		availStr := fields[3]
		mountPoint := fields[5]

		// Parse sizes (remove unit suffixes)
		sizeGB := c.parseMacOSSize(sizeStr)
		usedGB := c.parseMacOSSize(usedStr)
		availGB := c.parseMacOSSize(availStr)

		usagePercent := float64(0)
		if sizeGB > 0 {
			usagePercent = (usedGB / sizeGB) * 100
		}

		storageDevice := &StorageDevice{
			Device:       device,
			SizeGB:       sizeGB,
			UsedGB:       usedGB,
			AvailableGB:  availGB,
			UsagePercent: usagePercent,
			MountPoint:   mountPoint,
			FileSystem:   "APFS",
			Type:         "SSD",
			Interface:    "Internal",
		}

		devices = append(devices, storageDevice)
	}

	return devices, nil
}

func (c *SystemHardwareCollector) parseMacOSSize(sizeStr string) float64 {
	// Remove unit suffixes and convert to GB
	originalStr := sizeStr
	sizeStr = strings.TrimSuffix(sizeStr, "Gi")
	sizeStr = strings.TrimSuffix(sizeStr, "G")
	sizeStr = strings.TrimSuffix(sizeStr, "Ti")
	sizeStr = strings.TrimSuffix(sizeStr, "T")
	sizeStr = strings.TrimSuffix(sizeStr, "Mi")
	sizeStr = strings.TrimSuffix(sizeStr, "M")

	size, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return 0
	}

	// Convert to GB based on original unit
	if strings.HasSuffix(originalStr, "T") || strings.HasSuffix(originalStr, "Ti") {
		return size * 1024
	} else if strings.HasSuffix(originalStr, "M") || strings.HasSuffix(originalStr, "Mi") {
		return size / 1024
	}

	return size // Assume GB if no unit
}

func (c *SystemHardwareCollector) collectMacOSSystemInfo(ctx context.Context) (*SystemInfo, error) {
	system := &SystemInfo{
		OS:           "macOS",
		Architecture: runtime.GOARCH,
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		system.Hostname = hostname
	}

	// CRITICAL FIX: Add timeouts to prevent hanging
	cmdCtx1, cancel1 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel1()

	// Get OS version using sw_vers
	cmd := exec.CommandContext(cmdCtx1, "sw_vers", "-productVersion")
	if output, err := cmd.Output(); err == nil {
		system.OSVersion = strings.TrimSpace(string(output))
	}

	// CRITICAL FIX: Add timeout for uname
	cmdCtx2, cancel2 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel2()

	// Get kernel version using uname
	cmd2 := exec.CommandContext(cmdCtx2, "uname", "-r")
	if output2, err := cmd2.Output(); err == nil {
		system.Kernel = strings.TrimSpace(string(output2))
	}

	return system, nil
}
