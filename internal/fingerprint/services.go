package fingerprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ServiceCollectorImpl implements ServiceCollector interface
type ServiceCollectorImpl struct {
	logger  *zap.Logger
	timeout time.Duration
}

// NewServiceCollector creates a new service collector
func NewServiceCollector(logger *zap.Logger) ServiceCollector {
	return &ServiceCollectorImpl{
		logger:  logger,
		timeout: 30 * time.Second,
	}
}

// GetRunningServices retrieves running services
func (sc *ServiceCollectorImpl) GetRunningServices(ctx context.Context) ([]*ServiceDetails, error) {
	sc.logger.Info("Starting service enumeration", zap.String("platform", runtime.GOOS))

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, sc.timeout)
	defer cancel()

	var services []*ServiceDetails
	var err error

	switch runtime.GOOS {
	case "windows":
		services, err = sc.collectWindowsServices(timeoutCtx)
	case "linux":
		services, err = sc.collectLinuxServices(timeoutCtx)
	case "darwin":
		services, err = sc.collectMacOSServices(timeoutCtx)
	default:
		return nil, fmt.Errorf("service collection not supported on platform: %s", runtime.GOOS)
	}

	if err != nil {
		sc.logger.Error("Service collection failed", zap.Error(err))
		return nil, fmt.Errorf("failed to collect services: %w", err)
	}

	sc.logger.Info("Service enumeration completed", zap.Int("total_services", len(services)))
	return services, nil
}

// GetRunningProcesses retrieves running processes
func (sc *ServiceCollectorImpl) GetRunningProcesses(ctx context.Context) ([]*ProcessDetails, error) {
	// For now, return empty slice - can be implemented later
	return []*ProcessDetails{}, nil
}

// GetStartupPrograms retrieves startup programs
func (sc *ServiceCollectorImpl) GetStartupPrograms(ctx context.Context) ([]*StartupProgram, error) {
	// For now, return empty slice - can be implemented later
	return []*StartupProgram{}, nil
}

// GetInstalledSoftware retrieves installed software list
func (sc *ServiceCollectorImpl) GetInstalledSoftware(ctx context.Context) ([]*InstalledSoftware, error) {
	// For now, return empty slice - can be implemented later
	return []*InstalledSoftware{}, nil
}

// collectWindowsServices gathers Windows service information
func (sc *ServiceCollectorImpl) collectWindowsServices(ctx context.Context) ([]*ServiceDetails, error) {
	var services []*ServiceDetails

	// Get services with detailed information using PowerShell
	psScript := `
		Get-Service | ForEach-Object {
			$service = $_
			$process = $null
			
			# Try to get process information if service is running
			if ($service.Status -eq 'Running') {
				try {
					$process = Get-WmiObject -Class Win32_Service -Filter "Name='$($service.ServiceName)'" | 
						Select-Object ProcessId, StartName, Description, PathName
				} catch {}
			}
			
			$result = @{
				'Name' = $service.ServiceName
				'DisplayName' = $service.DisplayName
				'Status' = $service.Status.ToString()
				'StartType' = $service.StartType.ToString()
				'ProcessID' = if ($process.ProcessId) { $process.ProcessId } else { 0 }
				'Description' = if ($process.Description) { $process.Description } else { '' }
				'BinaryPath' = if ($process.PathName) { $process.PathName } else { '' }
			}
			
			$result | ConvertTo-Json -Compress
		}
	`

	cmd := exec.CommandContext(ctx, "powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		sc.logger.Error("Failed to execute PowerShell service query", zap.Error(err))
		return sc.collectWindowsServicesFallback(ctx)
	}

	// Parse JSON output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var serviceData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &serviceData); err != nil {
			sc.logger.Debug("Failed to parse service JSON", zap.String("line", line), zap.Error(err))
			continue
		}

		service := &ServiceDetails{
			Name:        getString(serviceData, "Name"),
			DisplayName: getString(serviceData, "DisplayName"),
			Status:      getString(serviceData, "Status"),
			StartType:   getString(serviceData, "StartType"),
			Description: getString(serviceData, "Description"),
			BinaryPath:  getString(serviceData, "BinaryPath"),
		}

		if pid := getFloat64(serviceData, "ProcessID"); pid > 0 {
			service.ProcessID = int(pid)
		}

		services = append(services, service)
	}

	sc.logger.Debug("Collected Windows services", zap.Int("count", len(services)))
	return services, nil
}

// collectWindowsServicesFallback uses basic Windows commands
func (sc *ServiceCollectorImpl) collectWindowsServicesFallback(ctx context.Context) ([]*ServiceDetails, error) {
	var services []*ServiceDetails

	// Use sc query as fallback
	cmd := exec.CommandContext(ctx, "sc", "query", "state=", "all")
	output, err := cmd.Output()
	if err != nil {
		return services, fmt.Errorf("failed to execute sc query: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var currentService *ServiceDetails

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SERVICE_NAME:") {
			if currentService != nil {
				services = append(services, currentService)
			}
			currentService = &ServiceDetails{
				Name: strings.TrimSpace(strings.TrimPrefix(line, "SERVICE_NAME:")),
			}
		} else if strings.HasPrefix(line, "DISPLAY_NAME:") && currentService != nil {
			currentService.DisplayName = strings.TrimSpace(strings.TrimPrefix(line, "DISPLAY_NAME:"))
		} else if strings.HasPrefix(line, "STATE") && currentService != nil {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentService.Status = parts[3]
			}
		}
	}

	if currentService != nil {
		services = append(services, currentService)
	}

	return services, nil
}

// collectLinuxServices gathers Linux service information
func (sc *ServiceCollectorImpl) collectLinuxServices(ctx context.Context) ([]*ServiceDetails, error) {
	var services []*ServiceDetails

	// Try systemctl first (systemd systems)
	if systemdServices, err := sc.collectSystemdServices(ctx); err == nil {
		services = append(services, systemdServices...)
	} else {
		sc.logger.Debug("systemctl not available, trying alternatives", zap.Error(err))
	}

	return services, nil
}

// collectSystemdServices uses systemctl to get service information
func (sc *ServiceCollectorImpl) collectSystemdServices(ctx context.Context) ([]*ServiceDetails, error) {
	var services []*ServiceDetails

	// Get all systemd services
	cmd := exec.CommandContext(ctx, "systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl command failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 && strings.HasSuffix(fields[0], ".service") {
			serviceName := strings.TrimSuffix(fields[0], ".service")
			status := "unknown"
			if len(fields) >= 3 {
				status = fields[2] // ACTIVE field
			}

			service := &ServiceDetails{
				Name:   serviceName,
				Status: status,
			}

			// Try to get more details for active services
			if status == "active" {
				if details := sc.getSystemdServiceDetails(ctx, fields[0]); details != nil {
					service.ProcessID = details.ProcessID
					service.BinaryPath = details.BinaryPath
					service.Description = details.Description
				}
			}

			services = append(services, service)
		}
	}

	return services, nil
}

// getSystemdServiceDetails gets detailed information for a specific service
func (sc *ServiceCollectorImpl) getSystemdServiceDetails(ctx context.Context, serviceName string) *ServiceDetails {
	cmd := exec.CommandContext(ctx, "systemctl", "show", serviceName, "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	details := &ServiceDetails{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "MainPID":
				if pid, err := strconv.Atoi(value); err == nil && pid > 0 {
					details.ProcessID = pid
				}
			case "ExecStart":
				details.BinaryPath = value
			case "Description":
				details.Description = value
			}
		}
	}

	return details
}

// collectMacOSServices gathers macOS service information
func (sc *ServiceCollectorImpl) collectMacOSServices(ctx context.Context) ([]*ServiceDetails, error) {
	var services []*ServiceDetails

	// Use launchctl to get services
	cmd := exec.CommandContext(ctx, "launchctl", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("launchctl command failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid := 0
			if fields[0] != "-" {
				pid, _ = strconv.Atoi(fields[0])
			}

			status := "stopped"
			if pid > 0 {
				status = "running"
			}

			service := &ServiceDetails{
				Name:      fields[2],
				Status:    status,
				ProcessID: pid,
			}

			services = append(services, service)
		}
	}

	return services, nil
}

// Helper functions
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0
}
