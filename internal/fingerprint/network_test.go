package fingerprint

import (
	"context"
	"net"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewNetworkConfigCollector(t *testing.T) {
	t.Log("\n🔍 Testing NetworkConfigCollector creation...")

	collector := NewNetworkConfigCollector()
	if collector == nil {
		t.Fatal("❌ NetworkConfigCollector creation failed - returned nil")
	}

	t.Log("✅ NetworkConfigCollector created successfully")
}

func TestGetNetworkConfiguration(t *testing.T) {
	t.Log("\n🔍 Testing network configuration collection...")

	collector := NewNetworkConfigCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := collector.GetNetworkConfiguration(ctx)
	if err != nil {
		t.Fatalf("❌ Network configuration collection failed: %v", err)
	}

	if config == nil {
		t.Fatal("❌ Network configuration is nil")
	}

	t.Log("\n🌐 Network Configuration:")
	t.Logf("  Platform: %s", config.Platform)
	t.Logf("  Collected At: %s", config.CollectedAt.Format(time.RFC3339))
	t.Logf("  Number of Interfaces: %d", len(config.Interfaces))
	t.Logf("  Number of Routes: %d", len(config.RoutingTable))
	t.Logf("  Number of DNS Servers: %d", len(config.DNSServers))

	// Display interface details
	for i, iface := range config.Interfaces {
		t.Logf("  Interface %d:", i+1)
		t.Logf("    Name: %s", iface.Name)
		t.Logf("    Type: %s", iface.Type)
		t.Logf("    State: %s", iface.State)
		t.Logf("    MAC: %s", iface.MACAddress)
		t.Logf("    MTU: %d", iface.MTU)
		t.Logf("    IPv4 Addresses: %v", iface.IPv4Addresses)
		t.Logf("    IPv6 Addresses: %v", iface.IPv6Addresses)

		if iface.Statistics != nil {
			t.Logf("    Statistics:")
			t.Logf("      Bytes Received: %d", iface.Statistics.BytesReceived)
			t.Logf("      Bytes Transmitted: %d", iface.Statistics.BytesTransmitted)
			t.Logf("      Packets Received: %d", iface.Statistics.PacketsReceived)
			t.Logf("      Packets Transmitted: %d", iface.Statistics.PacketsTransmitted)
		}
	}

	// Display routing table
	if len(config.RoutingTable) > 0 {
		t.Log("  Routing Table (first 5 entries):")
		for i, route := range config.RoutingTable {
			if i >= 5 {
				break
			}
			t.Logf("    Route %d: %s -> %s via %s", i+1, route.Destination, route.Gateway, route.Interface)
		}
	}

	// Display DNS servers
	if len(config.DNSServers) > 0 {
		t.Logf("  DNS Servers: %v", config.DNSServers)
	}

	// Display network statistics
	if config.NetworkStats != nil {
		t.Log("  Network Statistics:")
		t.Logf("    Active Connections: %d", config.NetworkStats.ActiveConnections)
		t.Logf("    Listening Ports: %d", config.NetworkStats.ListeningPorts)
	}

	// Validation checks
	if config.Platform == "" {
		t.Error("❌ Platform should not be empty")
	}

	if config.CollectedAt.IsZero() {
		t.Error("❌ CollectedAt should not be zero")
	}

	if len(config.Interfaces) == 0 {
		t.Error("❌ Should have at least one network interface")
	}

	// Check that we have at least a loopback interface
	hasLoopback := false
	for _, iface := range config.Interfaces {
		if iface.Type == "loopback" || iface.Name == "lo" || iface.Name == "lo0" {
			hasLoopback = true
			break
		}
	}
	if !hasLoopback {
		t.Log("⚠️ No loopback interface found - this might be expected on some systems")
	}

	t.Log("✅ Network configuration collection successful")
}

func TestNetworkInterfaceValidation(t *testing.T) {
	t.Log("\n🔍 Testing network interface validation...")

	collector := NewNetworkConfigCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := collector.GetNetworkConfiguration(ctx)
	if err != nil {
		t.Fatalf("❌ Network configuration collection failed: %v", err)
	}

	validInterfaces := 0
	for _, iface := range config.Interfaces {
		// Validate interface name
		if iface.Name == "" {
			t.Errorf("❌ Interface has empty name")
			continue
		}

		// Validate interface state
		if iface.State != "up" && iface.State != "down" {
			t.Errorf("❌ Interface %s has invalid state: %s", iface.Name, iface.State)
		}

		// Validate MTU (should be positive)
		if iface.MTU <= 0 {
			t.Logf("⚠️ Interface %s has invalid MTU: %d", iface.Name, iface.MTU)
		}

		// Validate MAC address format (if present)
		if iface.MACAddress != "" && !isValidMACAddress(iface.MACAddress) {
			t.Errorf("❌ Interface %s has invalid MAC address: %s", iface.Name, iface.MACAddress)
		}

		// Validate IP addresses (basic format check)
		for _, ipv4 := range iface.IPv4Addresses {
			if !isValidIPv4CIDR(ipv4) {
				t.Errorf("❌ Interface %s has invalid IPv4 address: %s", iface.Name, ipv4)
			}
		}

		for _, ipv6 := range iface.IPv6Addresses {
			if !isValidIPv6CIDR(ipv6) {
				t.Errorf("❌ Interface %s has invalid IPv6 address: %s", iface.Name, ipv6)
			}
		}

		validInterfaces++
	}

	t.Logf("✅ Validated %d network interfaces", validInterfaces)
}

func TestRoutingTableValidation(t *testing.T) {
	t.Log("\n🔍 Testing routing table validation...")

	collector := NewNetworkConfigCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := collector.GetNetworkConfiguration(ctx)
	if err != nil {
		t.Fatalf("❌ Network configuration collection failed: %v", err)
	}

	validRoutes := 0
	for _, route := range config.RoutingTable {
		// Validate destination
		if route.Destination == "" {
			t.Errorf("❌ Route has empty destination")
			continue
		}

		// Validate gateway (can be empty for local routes)
		if route.Gateway != "" && !isValidIP(route.Gateway) {
			t.Errorf("❌ Route has invalid gateway: %s", route.Gateway)
		}

		// Validate interface
		if route.Interface == "" {
			t.Errorf("❌ Route has empty interface")
		}

		validRoutes++
	}

	t.Logf("✅ Validated %d routing table entries", validRoutes)
}

func TestDNSServerValidation(t *testing.T) {
	t.Log("\n🔍 Testing DNS server validation...")

	collector := NewNetworkConfigCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := collector.GetNetworkConfiguration(ctx)
	if err != nil {
		t.Fatalf("❌ Network configuration collection failed: %v", err)
	}

	validDNSServers := 0
	for _, dns := range config.DNSServers {
		if dns == "" {
			t.Errorf("❌ Empty DNS server entry")
			continue
		}

		if !isValidIP(dns) {
			t.Errorf("❌ Invalid DNS server IP: %s", dns)
			continue
		}

		validDNSServers++
	}

	t.Logf("✅ Validated %d DNS servers", validDNSServers)
}

func TestNetworkPlatformSpecificMethods(t *testing.T) {
	t.Log("\n🔍 Testing network platform-specific helper methods...")

	collector := NewNetworkConfigCollector()

	// Test interface type determination
	testCases := []struct {
		name     string
		expected string
	}{
		{"eth0", "ethernet"},
		{"en0", "ethernet"},
		{"wlan0", "wireless"},
		{"wi-fi", "wireless"},
		{"lo", "loopback"},
		{"lo0", "loopback"},
		{"tun0", "tunnel"},
		{"tap0", "tunnel"},
		{"docker0", "bridge"},
		{"br-1234", "bridge"},
		{"unknown123", "unknown"},
	}

	for _, tc := range testCases {
		result := collector.determineInterfaceType(tc.name)
		if result != tc.expected {
			t.Errorf("❌ Interface type for %s: expected %s, got %s", tc.name, tc.expected, result)
		} else {
			t.Logf("✅ Interface type for %s: %s", tc.name, result)
		}
	}

	// Test hex netmask to CIDR conversion (macOS specific)
	if runtime.GOOS == "darwin" {
		hexCases := []struct {
			hex      string
			expected int
		}{
			{"ffffff00", 24},
			{"ffff0000", 16},
			{"ff000000", 8},
			{"ffffffff", 32},
		}

		for _, tc := range hexCases {
			result := collector.hexNetmaskToCIDR(tc.hex)
			if result != tc.expected {
				t.Errorf("❌ Hex netmask %s: expected /%d, got /%d", tc.hex, tc.expected, result)
			} else {
				t.Logf("✅ Hex netmask %s: /%d", tc.hex, result)
			}
		}
	}

	t.Log("✅ Platform-specific methods testing completed")
}

func TestNetworkContextCancellation(t *testing.T) {
	t.Log("\n🔍 Testing network context cancellation handling...")

	collector := NewNetworkConfigCollector()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := collector.GetNetworkConfiguration(ctx)
	if err == nil {
		t.Log("⚠️ GetNetworkConfiguration with cancelled context didn't return error - may not support cancellation")
	} else {
		t.Logf("✅ GetNetworkConfiguration properly handled cancelled context: %v", err)
	}

	// Test with timeout context
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err = collector.GetNetworkConfiguration(ctx)
	if err == nil {
		t.Log("⚠️ GetNetworkConfiguration with timed-out context didn't return error - may not support timeouts")
	}

	t.Log("✅ Context cancellation testing completed")
}

func BenchmarkNetworkConfiguration(b *testing.B) {
	collector := NewNetworkConfigCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.GetNetworkConfiguration(ctx)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkInterfaceDetection(b *testing.B) {
	collector := NewNetworkConfigCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch runtime.GOOS {
		case "linux":
			_, err := collector.getLinuxInterfaces(ctx)
			if err != nil {
				b.Fatalf("Linux interface detection failed: %v", err)
			}
		case "darwin":
			_, err := collector.getMacOSInterfaces(ctx)
			if err != nil {
				b.Fatalf("macOS interface detection failed: %v", err)
			}
		case "windows":
			_, err := collector.getWindowsInterfaces(ctx)
			if err != nil {
				b.Fatalf("Windows interface detection failed: %v", err)
			}
		}
	}
}

// Helper functions for validation

func isValidMACAddress(mac string) bool {
	// Basic MAC address format validation (XX:XX:XX:XX:XX:XX)
	if len(mac) != 17 {
		return false
	}

	for i, char := range mac {
		if i%3 == 2 {
			if char != ':' {
				return false
			}
		} else {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}

	return true
}

func isValidIPv4CIDR(cidr string) bool {
	// Basic IPv4 CIDR validation
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return false
	}

	// Validate IP part
	ip := net.ParseIP(parts[0])
	if ip == nil || ip.To4() == nil {
		return false
	}

	// Validate prefix length
	prefix, err := strconv.Atoi(parts[1])
	if err != nil || prefix < 0 || prefix > 32 {
		return false
	}

	return true
}

func isValidIPv6CIDR(cidr string) bool {
	// Basic IPv6 CIDR validation
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return false
	}

	// Validate IP part (remove zone identifier if present)
	ipPart := strings.Split(parts[0], "%")[0]
	ip := net.ParseIP(ipPart)
	if ip == nil || ip.To4() != nil {
		return false
	}

	// Validate prefix length
	prefix, err := strconv.Atoi(parts[1])
	if err != nil || prefix < 0 || prefix > 128 {
		return false
	}

	return true
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
