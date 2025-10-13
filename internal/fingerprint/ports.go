package fingerprint

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// PortCollectorImpl implements port and connection scanning
type PortCollectorImpl struct {
	logger  *zap.Logger
	timeout time.Duration
}

// NewPortCollector creates a new port collector
func NewPortCollector(logger *zap.Logger) *PortCollectorImpl {
	return &PortCollectorImpl{
		logger:  logger,
		timeout: 30 * time.Second,
	}
}

// CollectOpenPorts gathers information about open ports and listening services
func (pc *PortCollectorImpl) CollectOpenPorts(ctx context.Context) ([]*NetworkConnection, error) {
	pc.logger.Info("Starting port and connection enumeration", zap.String("platform", runtime.GOOS))

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, pc.timeout)
	defer cancel()

	var connections []*NetworkConnection
	var err error

	switch runtime.GOOS {
	case "windows":
		connections, err = pc.collectWindowsPorts(timeoutCtx)
	case "linux":
		connections, err = pc.collectLinuxPorts(timeoutCtx)
	case "darwin":
		connections, err = pc.collectMacOSPorts(timeoutCtx)
	default:
		return nil, fmt.Errorf("port collection not supported on platform: %s", runtime.GOOS)
	}

	if err != nil {
		pc.logger.Error("Port collection failed", zap.Error(err))
		return nil, fmt.Errorf("failed to collect ports: %w", err)
	}

	pc.logger.Info("Port enumeration completed",
		zap.Int("total_connections", len(connections)),
		zap.Int("listening_ports", pc.countListeningPorts(connections)))

	return connections, nil
}

// collectWindowsPorts gathers Windows port information using netstat
func (pc *PortCollectorImpl) collectWindowsPorts(ctx context.Context) ([]*NetworkConnection, error) {
	var connections []*NetworkConnection

	// Use netstat with process information
	cmd := exec.CommandContext(ctx, "netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		pc.logger.Error("Failed to execute netstat command", zap.Error(err))
		return nil, fmt.Errorf("netstat command failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i < 2 { // Skip header lines
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			connection := pc.parseWindowsNetstatLine(fields)
			if connection != nil {
				connections = append(connections, connection)
			}
		}
	}

	// Try to get process names for PIDs
	pc.enrichWithProcessNames(ctx, connections)

	pc.logger.Debug("Collected Windows ports", zap.Int("count", len(connections)))
	return connections, nil
}

// parseWindowsNetstatLine parses a single netstat output line
func (pc *PortCollectorImpl) parseWindowsNetstatLine(fields []string) *NetworkConnection {
	if len(fields) < 4 {
		return nil
	}

	protocol := strings.ToUpper(fields[0])
	localAddr := fields[1]
	remoteAddr := fields[2]
	state := ""
	pidIndex := 3

	// For TCP connections, there's a state field
	if protocol == "TCP" && len(fields) >= 5 {
		state = fields[3]
		pidIndex = 4
	}

	// Parse local address and port
	localIP, localPort := pc.parseAddress(localAddr)
	if localPort == 0 {
		return nil
	}

	// Parse remote address and port
	remoteIP, remotePort := pc.parseAddress(remoteAddr)

	// Parse PID
	pid := 0
	if pidIndex < len(fields) {
		if pidVal, err := strconv.Atoi(fields[pidIndex]); err == nil {
			pid = pidVal
		}
	}

	return &NetworkConnection{
		LocalAddress:  localIP,
		LocalPort:     localPort,
		RemoteAddress: remoteIP,
		RemotePort:    remotePort,
		Protocol:      protocol,
		State:         state,
		ProcessID:     pid,
	}
}

// collectLinuxPorts gathers Linux port information using netstat or ss
func (pc *PortCollectorImpl) collectLinuxPorts(ctx context.Context) ([]*NetworkConnection, error) {
	var connections []*NetworkConnection

	// Try ss first (modern systems), then fall back to netstat
	if ssConnections, err := pc.collectLinuxPortsSS(ctx); err == nil {
		connections = ssConnections
	} else {
		pc.logger.Debug("ss command failed, trying netstat", zap.Error(err))
		if netstatConnections, err := pc.collectLinuxPortsNetstat(ctx); err == nil {
			connections = netstatConnections
		} else {
			return nil, fmt.Errorf("both ss and netstat failed: %w", err)
		}
	}

	pc.logger.Debug("Collected Linux ports", zap.Int("count", len(connections)))
	return connections, nil
}

// collectLinuxPortsSS uses ss command to get port information
func (pc *PortCollectorImpl) collectLinuxPortsSS(ctx context.Context) ([]*NetworkConnection, error) {
	var connections []*NetworkConnection

	// Use ss to get listening and established connections
	cmd := exec.CommandContext(ctx, "ss", "-tulpn")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ss command failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			connection := pc.parseLinuxSSLine(fields)
			if connection != nil {
				connections = append(connections, connection)
			}
		}
	}

	return connections, nil
}

// parseLinuxSSLine parses a single ss output line
func (pc *PortCollectorImpl) parseLinuxSSLine(fields []string) *NetworkConnection {
	if len(fields) < 5 {
		return nil
	}

	protocol := strings.ToUpper(fields[0])
	state := fields[1]
	localAddr := fields[4]

	// Parse local address and port
	localIP, localPort := pc.parseAddress(localAddr)
	if localPort == 0 {
		return nil
	}

	connection := &NetworkConnection{
		LocalAddress: localIP,
		LocalPort:    localPort,
		Protocol:     protocol,
		State:        state,
	}

	// Try to extract process information from the last field
	if len(fields) >= 6 {
		processInfo := fields[len(fields)-1]
		if strings.Contains(processInfo, "pid=") {
			// Extract PID and process name from "users:(("nginx",pid=1234,fd=6))"
			if pid, processName := pc.parseLinuxProcessInfo(processInfo); pid > 0 {
				connection.ProcessID = pid
				connection.ProcessName = processName
			}
		}
	}

	return connection
}

// parseLinuxProcessInfo extracts PID and process name from ss output
func (pc *PortCollectorImpl) parseLinuxProcessInfo(processInfo string) (int, string) {
	// Example: users:(("nginx",pid=1234,fd=6))
	if !strings.Contains(processInfo, "pid=") {
		return 0, ""
	}

	var pid int
	var processName string

	// Extract process name
	if start := strings.Index(processInfo, `("`); start != -1 {
		start += 2
		if end := strings.Index(processInfo[start:], `"`); end != -1 {
			processName = processInfo[start : start+end]
		}
	}

	// Extract PID
	if start := strings.Index(processInfo, "pid="); start != -1 {
		start += 4
		if end := strings.Index(processInfo[start:], ","); end != -1 {
			if pidVal, err := strconv.Atoi(processInfo[start : start+end]); err == nil {
				pid = pidVal
			}
		}
	}

	return pid, processName
}

// collectLinuxPortsNetstat uses netstat as fallback
func (pc *PortCollectorImpl) collectLinuxPortsNetstat(ctx context.Context) ([]*NetworkConnection, error) {
	var connections []*NetworkConnection

	cmd := exec.CommandContext(ctx, "netstat", "-tulpn")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netstat command failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i < 2 { // Skip header lines
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 6 {
			connection := pc.parseLinuxNetstatLine(fields)
			if connection != nil {
				connections = append(connections, connection)
			}
		}
	}

	return connections, nil
}

// parseLinuxNetstatLine parses a single netstat output line
func (pc *PortCollectorImpl) parseLinuxNetstatLine(fields []string) *NetworkConnection {
	if len(fields) < 6 {
		return nil
	}

	protocol := strings.ToUpper(fields[0])
	localAddr := fields[3]
	state := fields[5]

	// Parse local address and port
	localIP, localPort := pc.parseAddress(localAddr)
	if localPort == 0 {
		return nil
	}

	connection := &NetworkConnection{
		LocalAddress: localIP,
		LocalPort:    localPort,
		Protocol:     protocol,
		State:        state,
	}

	// Extract process information if available
	if len(fields) >= 7 {
		processInfo := fields[6]
		if pid, processName := pc.parseLinuxNetstatProcessInfo(processInfo); pid > 0 {
			connection.ProcessID = pid
			connection.ProcessName = processName
		}
	}

	return connection
}

// parseLinuxNetstatProcessInfo extracts PID and process name from netstat output
func (pc *PortCollectorImpl) parseLinuxNetstatProcessInfo(processInfo string) (int, string) {
	// Example: "1234/nginx"
	if !strings.Contains(processInfo, "/") {
		return 0, ""
	}

	parts := strings.Split(processInfo, "/")
	if len(parts) != 2 {
		return 0, ""
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, ""
	}

	return pid, parts[1]
}

// collectMacOSPorts gathers macOS port information using netstat
func (pc *PortCollectorImpl) collectMacOSPorts(ctx context.Context) ([]*NetworkConnection, error) {
	var connections []*NetworkConnection

	// Use netstat on macOS
	cmd := exec.CommandContext(ctx, "netstat", "-an")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netstat command failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i < 2 { // Skip header lines
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			connection := pc.parseMacOSNetstatLine(fields)
			if connection != nil {
				connections = append(connections, connection)
			}
		}
	}

	pc.logger.Debug("Collected macOS ports", zap.Int("count", len(connections)))
	return connections, nil
}

// parseMacOSNetstatLine parses a single macOS netstat output line
func (pc *PortCollectorImpl) parseMacOSNetstatLine(fields []string) *NetworkConnection {
	if len(fields) < 3 {
		return nil
	}

	protocol := strings.ToUpper(fields[0])
	localAddr := fields[3]

	// Parse local address and port
	localIP, localPort := pc.parseAddress(localAddr)
	if localPort == 0 {
		return nil
	}

	connection := &NetworkConnection{
		LocalAddress: localIP,
		LocalPort:    localPort,
		Protocol:     protocol,
	}

	// Set state if available
	if len(fields) >= 6 {
		connection.State = fields[5]
	}

	return connection
}

// parseAddress parses "IP:Port" or "[IPv6]:Port" format
func (pc *PortCollectorImpl) parseAddress(addr string) (string, int) {
	if addr == "" {
		return "", 0
	}

	// Handle IPv6 addresses with brackets
	if strings.HasPrefix(addr, "[") {
		// IPv6 format: [::1]:8080
		if closeBracket := strings.Index(addr, "]"); closeBracket != -1 {
			ip := addr[1:closeBracket]
			if colon := strings.Index(addr[closeBracket:], ":"); colon != -1 {
				portStr := addr[closeBracket+colon+1:]
				if port, err := strconv.Atoi(portStr); err == nil {
					return ip, port
				}
			}
			return ip, 0
		}
		return "", 0
	}

	// Handle IPv4 and simple formats
	if lastColon := strings.LastIndex(addr, ":"); lastColon != -1 {
		ip := addr[:lastColon]
		portStr := addr[lastColon+1:]

		// Handle special addresses
		if ip == "*" || ip == "0.0.0.0" {
			ip = "0.0.0.0"
		}

		if port, err := strconv.Atoi(portStr); err == nil {
			return ip, port
		}
	}

	return addr, 0
}

// enrichWithProcessNames tries to get process names for Windows connections
func (pc *PortCollectorImpl) enrichWithProcessNames(ctx context.Context, connections []*NetworkConnection) {
	if runtime.GOOS != "windows" {
		return
	}

	// Use tasklist to get process names
	cmd := exec.CommandContext(ctx, "tasklist", "/fo", "csv")
	output, err := cmd.Output()
	if err != nil {
		pc.logger.Debug("Failed to get process names", zap.Error(err))
		return
	}

	// Parse tasklist output and build PID -> Name map
	processMap := make(map[int]string)
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}

		// Parse CSV line
		fields := strings.Split(line, ",")
		if len(fields) >= 2 {
			// Remove quotes from fields
			name := strings.Trim(fields[0], `"`)
			pidStr := strings.Trim(fields[1], `"`)

			if pid, err := strconv.Atoi(pidStr); err == nil {
				processMap[pid] = name
			}
		}
	}

	// Update connections with process names
	for _, conn := range connections {
		if conn.ProcessID > 0 {
			if name, exists := processMap[conn.ProcessID]; exists {
				conn.ProcessName = name
			}
		}
	}
}

// countListeningPorts counts the number of listening ports
func (pc *PortCollectorImpl) countListeningPorts(connections []*NetworkConnection) int {
	count := 0
	for _, conn := range connections {
		state := strings.ToLower(conn.State)
		if state == "listen" || state == "listening" || state == "" {
			count++
		}
	}
	return count
}
