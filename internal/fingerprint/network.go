package fingerprint

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"time"
)

// NetworkConfigCollector implements lightweight network interface detection
type NetworkConfigCollector struct{}

// NewNetworkConfigCollector creates a new network configuration collector
func NewNetworkConfigCollector() *NetworkConfigCollector {
	return &NetworkConfigCollector{}
}

// GetNetworkConfiguration collects basic network interface information
func (n *NetworkConfigCollector) GetNetworkConfiguration(ctx context.Context) (*NetworkConfiguration, error) {
	// Simple timeout for the entire operation
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	config := &NetworkConfiguration{
		CollectedAt: time.Now(),
		Platform:    Platform(runtime.GOOS),
		Interfaces:  []NetworkInterface{},
		// Remove heavy data collection - we don't need routing tables, DNS, or stats
		RoutingTable: []RouteEntry{},
		DNSServers:   []string{},
		NetworkStats: &NetworkStats{},
	}

	// Use Go's built-in net package - much safer than external commands
	interfaces, err := n.getBasicInterfaces(ctx)
	if err != nil {
		return config, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	config.Interfaces = interfaces
	return config, nil
}

// getBasicInterfaces gets essential interface information using Go's net package
func (n *NetworkConfigCollector) getBasicInterfaces(ctx context.Context) ([]NetworkInterface, error) {
	// Use Go's built-in interface enumeration - no external commands needed
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate interfaces: %w", err)
	}

	interfaces := make([]NetworkInterface, 0, len(netInterfaces))

	for _, iface := range netInterfaces {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return interfaces, ctx.Err()
		default:
		}

		netInterface := NetworkInterface{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			MTU:        iface.MTU,
			Type:       n.determineInterfaceType(iface.Name),
		}

		// Determine interface state
		if iface.Flags&net.FlagUp != 0 {
			netInterface.State = "up"
		} else {
			netInterface.State = "down"
		}

		// Get IP addresses - this is the main data we actually need
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				addrStr := addr.String()
				// Simple IPv4/IPv6 detection
				if ip, _, err := net.ParseCIDR(addrStr); err == nil {
					if ip.To4() != nil {
						netInterface.IPv4Addresses = append(netInterface.IPv4Addresses, addrStr)
					} else {
						netInterface.IPv6Addresses = append(netInterface.IPv6Addresses, addrStr)
					}
				}
			}
		}

		// No statistics collection - we don't need packet counts, errors, etc.
		// This eliminates all the expensive netstat command executions

		interfaces = append(interfaces, netInterface)
	}

	return interfaces, nil
}

// determineInterfaceType determines the interface type based on name patterns
func (n *NetworkConfigCollector) determineInterfaceType(name string) string {
	// Simple pattern matching without external commands
	switch {
	case name == "lo" || name == "lo0":
		return "loopback"
	case len(name) >= 3 && (name[:3] == "eth" || name[:2] == "en"):
		return "ethernet"
	case len(name) >= 4 && name[:4] == "wlan":
		return "wireless"
	case len(name) >= 6 && name[:6] == "docker":
		return "bridge"
	case len(name) >= 3 && (name[:3] == "tun" || name[:3] == "tap"):
		return "tunnel"
	default:
		return "unknown"
	}
}
