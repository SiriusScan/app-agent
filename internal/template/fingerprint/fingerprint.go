package fingerprint

import (
	"context"
	"os"
	"runtime"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/sysinfo"
)

// HostFingerprint contains basic host identification information.
type HostFingerprint struct {
	OS        string // Operating system (linux, windows, darwin)
	OSVersion string // OS version string
	Hostname  string // System hostname
	PrimaryIP string // Primary non-loopback IPv4 address
	AgentID   string // Agent identifier
}

// CollectBasicFingerprint gathers basic host identification information.
// It uses existing sysinfo utilities and adds hostname collection.
// Returns partial data if some collection steps fail.
func CollectBasicFingerprint(ctx context.Context, cfg *config.AgentConfig) (*HostFingerprint, error) {
	fingerprint := &HostFingerprint{
		OS:        runtime.GOOS,
		OSVersion: sysinfo.GetOSVersion(),
		PrimaryIP: sysinfo.GetPrimaryIP(),
		AgentID:   cfg.AgentID,
	}

	// Get hostname (best effort)
	hostname, err := os.Hostname()
	if err != nil {
		// Log warning but don't fail - use agent ID as fallback
		fingerprint.Hostname = cfg.AgentID
	} else {
		fingerprint.Hostname = hostname
	}

	// Note: We never return an error here - partial data is acceptable
	// This ensures template scans can still report even if fingerprinting partially fails
	return fingerprint, nil
}

