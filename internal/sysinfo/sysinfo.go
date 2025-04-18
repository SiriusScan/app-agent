package sysinfo

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// GetPrimaryIP attempts to find a non-loopback IPv4 address for the host.
// It iterates through network interfaces and returns the first suitable address found.
// Returns an empty string if no suitable IP is found or an error occurs.
func GetPrimaryIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "" // Could log error here
	}

	for _, iface := range interfaces {
		// Skip down interfaces and loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue // Could log error here
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // Not an IPv4 address
			}

			// Found a non-loopback IPv4 address
			return ip.String()
		}
	}

	return "" // No suitable IP found
}

// GetOSVersion attempts to determine the operating system version in a cross-platform way.
// It returns a string representation of the OS version or an empty string on failure.
func GetOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxOSVersion()
	case "darwin":
		return getMacOSVersion()
	case "windows":
		return getWindowsOSVersion()
	default:
		return runtime.GOOS // Fallback to just the OS name
	}
}

func getLinuxOSVersion() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		// Fallback if os-release is not available
		return runCommandOutput("uname", "-sr")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var prettyName, versionID string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			prettyName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `" `)
			break // Prefer PRETTY_NAME
		} else if strings.HasPrefix(line, "VERSION=") {
			versionID = strings.Trim(strings.TrimPrefix(line, "VERSION="), `" `)
		}
	}

	if prettyName != "" {
		return prettyName
	} else if versionID != "" {
		return versionID
	}

	// Fallback if fields not found in os-release
	return runCommandOutput("uname", "-sr")
}

func getMacOSVersion() string {
	return runCommandOutput("sw_vers", "-productVersion")
}

func getWindowsOSVersion() string {
	// 'ver' is an internal cmd command, needs 'cmd /c'
	output := runCommandOutput("cmd", "/c", "ver")
	// Example output: Microsoft Windows [Version 10.0.19045.4170]
	// We'll return the whole line for now, trimmed.
	return strings.TrimSpace(output)
}

// runCommandOutput executes a command and returns its trimmed stdout or an empty string.
func runCommandOutput(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return "" // Could log error here
	}
	return strings.TrimSpace(stdout.String())
}
