package scan

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/fingerprint"
)

// TestScanResultJSONCompatibility verifies backward compatibility with existing JSON structure
func TestScanResultJSONCompatibility(t *testing.T) {
	t.Log("\n🔍 Testing ScanResult JSON backward compatibility...")

	// Create a basic ScanResult with only legacy fields
	legacyResult := ScanResult{
		OSInfo: OSInfo{
			OS:        "linux",
			Version:   "Ubuntu 20.04",
			Hostname:  "test-host",
			PrimaryIP: "192.168.1.100",
		},
		Packages: []InstalledPackage{
			{Name: "nginx", Version: "1.18.0", Source: "dpkg"},
			{Name: "curl", Version: "7.68.0", Source: "dpkg"},
		},
		ScanErrors: []string{"test warning"},
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(legacyResult, "", "  ")
	if err != nil {
		t.Fatalf("❌ Failed to marshal legacy ScanResult: %v", err)
	}

	t.Logf("\n📄 Legacy JSON structure:\n%s", string(jsonData))

	// Unmarshal back to verify structure
	var unmarshaled ScanResult
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("❌ Failed to unmarshal legacy JSON: %v", err)
	}

	// Verify basic fields are preserved
	if unmarshaled.OSInfo.OS != legacyResult.OSInfo.OS {
		t.Errorf("❌ OS mismatch: got %s, want %s", unmarshaled.OSInfo.OS, legacyResult.OSInfo.OS)
	}

	if len(unmarshaled.Packages) != len(legacyResult.Packages) {
		t.Errorf("❌ Package count mismatch: got %d, want %d", len(unmarshaled.Packages), len(legacyResult.Packages))
	}

	// Verify new fields are nil/empty (omitempty behavior)
	if unmarshaled.SystemFingerprint != nil {
		t.Error("❌ SystemFingerprint should be nil for legacy data")
	}

	if len(unmarshaled.EnhancedPackages) != 0 {
		t.Error("❌ EnhancedPackages should be empty for legacy data")
	}

	t.Log("✅ Legacy JSON compatibility verified")
}

// TestEnhancedScanResultJSON tests the full enhanced structure
func TestEnhancedScanResultJSON(t *testing.T) {
	t.Log("\n🔍 Testing enhanced ScanResult JSON structure...")

	now := time.Now()
	installDate := now.Add(-30 * 24 * time.Hour) // 30 days ago

	// Create enhanced ScanResult with all new fields
	enhancedResult := ScanResult{
		OSInfo: OSInfo{
			OS:        "linux",
			Version:   "Ubuntu 22.04",
			Hostname:  "enhanced-host",
			PrimaryIP: "10.0.1.50",
		},
		Packages: []InstalledPackage{
			{Name: "basic-pkg", Version: "1.0.0", Source: "dpkg"},
		},
		EnhancedPackages: []EnhancedPackageInfo{
			{
				Name:            "apache2",
				Version:         "2.4.52-1ubuntu4.7",
				Source:          "dpkg",
				Architecture:    "amd64",
				Description:     "Apache HTTP Server",
				Publisher:       "Ubuntu Developers",
				InstallDate:     &installDate,
				SizeBytes:       2048576,
				SizeMB:          2.0,
				CPE:             "cpe:2.3:a:apache:http_server:2.4.52:*:*:*:*:*:*:*",
				Dependencies:    []string{"libc6", "libssl3"},
				InstallLocation: "/usr/sbin/apache2",
				Signed:          true,
				SignatureValid:  true,
			},
		},
		SystemFingerprint: &fingerprint.SystemFingerprint{
			CollectedAt: now,
			Platform:    fingerprint.PlatformLinux,
			Hardware: &fingerprint.HardwareInfo{
				CPU: &fingerprint.CPUInfo{
					Model:        "Intel Core i7-9700K",
					Vendor:       "Intel",
					Cores:        8,
					Threads:      8,
					Architecture: "x86_64",
					ClockSpeed:   3600.0,
				},
				Memory: &fingerprint.MemoryInfo{
					TotalGB:      16.0,
					AvailableGB:  8.5,
					UsedGB:       7.5,
					UsagePercent: 46.875,
				},
				CollectedAt: now,
			},
			Network: &fingerprint.NetworkInfo{
				Interfaces: []*fingerprint.NetworkInterface{
					{
						Name:          "eth0",
						MAC:           "00:1B:44:11:3A:B7",
						IPv4Addresses: []string{"10.0.1.50"},
						IPv6Addresses: []string{"fe80::21b:44ff:fe11:3ab7"},
						Status:        "up",
						Type:          "ethernet",
						Speed:         1000,
						MTU:           1500,
					},
				},
				DNS: &fingerprint.DNSConfig{
					Servers:       []string{"8.8.8.8", "8.8.4.4"},
					SearchDomains: []string{"example.com"},
					Domain:        "example.com",
					Hostname:      "enhanced-host",
				},
				CollectedAt: now,
			},
			CollectionDuration: 5 * time.Second,
		},
		CertificateInventory: &fingerprint.CertificateInfo{
			SystemCertificates: []*fingerprint.CertificateDetails{
				{
					Subject:              "CN=Test Certificate",
					Issuer:               "CN=Test CA",
					SerialNumber:         "123456789",
					NotBefore:            now.Add(-365 * 24 * time.Hour),
					NotAfter:             now.Add(365 * 24 * time.Hour),
					Fingerprint:          "sha256:abcdef123456789",
					FingerprintAlgorithm: "SHA256",
					KeyUsage:             []string{"digital_signature", "key_encipherment"},
					Store:                "system",
					StoreLocation:        "machine",
				},
			},
			CollectedAt: now,
		},
		ServiceInformation: &fingerprint.ServiceInfo{
			Services: []*fingerprint.ServiceDetails{
				{
					Name:        "apache2",
					DisplayName: "Apache HTTP Server",
					Status:      "running",
					StartType:   "auto",
					ProcessID:   1234,
					BinaryPath:  "/usr/sbin/apache2",
					Description: "The Apache HTTP Server",
					Version:     "2.4.52",
				},
			},
			CollectedAt: now,
		},
	}

	// Set scan metadata
	enhancedResult.SetScanMetadata("1.2.0", "linux", "enhanced-host", "--enhanced", []string{"packages", "fingerprint", "certificates", "services"})
	enhancedResult.CompleteScanMetadata()
	enhancedResult.AddCollectionError("certificates", "Some certificates could not be accessed")

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(enhancedResult, "", "  ")
	if err != nil {
		t.Fatalf("❌ Failed to marshal enhanced ScanResult: %v", err)
	}

	t.Logf("\n📄 Enhanced JSON structure preview (first 1000 chars):\n%.1000s...", string(jsonData))

	// Unmarshal back to verify structure
	var unmarshaled ScanResult
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("❌ Failed to unmarshal enhanced JSON: %v", err)
	}

	// Verify enhanced fields are preserved
	if unmarshaled.SystemFingerprint == nil {
		t.Error("❌ SystemFingerprint should not be nil")
	}

	if len(unmarshaled.EnhancedPackages) != 1 {
		t.Errorf("❌ Expected 1 enhanced package, got %d", len(unmarshaled.EnhancedPackages))
	}

	if unmarshaled.CertificateInventory == nil {
		t.Error("❌ CertificateInventory should not be nil")
	}

	if unmarshaled.ServiceInformation == nil {
		t.Error("❌ ServiceInformation should not be nil")
	}

	// Verify enhanced package details
	if len(unmarshaled.EnhancedPackages) > 0 {
		pkg := unmarshaled.EnhancedPackages[0]
		if pkg.CPE != "cpe:2.3:a:apache:http_server:2.4.52:*:*:*:*:*:*:*" {
			t.Errorf("❌ CPE mismatch: got %s", pkg.CPE)
		}
		if pkg.SizeMB != 2.0 {
			t.Errorf("❌ Size mismatch: got %f, want 2.0", pkg.SizeMB)
		}
		if len(pkg.Dependencies) != 2 {
			t.Errorf("❌ Expected 2 dependencies, got %d", len(pkg.Dependencies))
		}
	}

	// Verify system fingerprint
	if unmarshaled.SystemFingerprint != nil {
		if unmarshaled.SystemFingerprint.Hardware == nil {
			t.Error("❌ Hardware info should not be nil")
		}
		if unmarshaled.SystemFingerprint.Network == nil {
			t.Error("❌ Network info should not be nil")
		}
		if unmarshaled.SystemFingerprint.Hardware != nil && unmarshaled.SystemFingerprint.Hardware.CPU != nil {
			cpu := unmarshaled.SystemFingerprint.Hardware.CPU
			if cpu.Cores != 8 {
				t.Errorf("❌ CPU cores mismatch: got %d, want 8", cpu.Cores)
			}
		}
	}

	// Verify scan metadata
	if unmarshaled.ScanMetadata == nil {
		t.Error("❌ ScanMetadata should not be nil")
	} else {
		if unmarshaled.ScanMetadata.AgentVersion != "1.2.0" {
			t.Errorf("❌ Agent version mismatch: got %s, want 1.2.0", unmarshaled.ScanMetadata.AgentVersion)
		}
		if len(unmarshaled.ScanMetadata.ScanModules) != 4 {
			t.Errorf("❌ Expected 4 scan modules, got %d", len(unmarshaled.ScanMetadata.ScanModules))
		}
		if len(unmarshaled.ScanMetadata.CollectionErrors) == 0 {
			t.Error("❌ Collection errors should be present")
		}
	}

	t.Log("✅ Enhanced JSON structure verified")
}

// TestEnhancedPackageInfoConversion tests conversion between package types
func TestEnhancedPackageInfoConversion(t *testing.T) {
	t.Log("\n🔍 Testing package type conversion...")

	installDate := time.Now().Add(-7 * 24 * time.Hour)

	// Test EnhancedPackageInfo -> InstalledPackage conversion
	enhanced := EnhancedPackageInfo{
		Name:         "test-package",
		Version:      "1.2.3",
		Source:       "dpkg",
		Architecture: "amd64",
		Description:  "Test package description",
		InstallDate:  &installDate,
		CPE:          "cpe:2.3:a:test:package:1.2.3:*:*:*:*:*:*:*",
		Dependencies: []string{"libc6", "libssl3"},
		SizeBytes:    1048576,
		SizeMB:       1.0,
	}

	basic := enhanced.ToInstalledPackage()
	if basic.Name != enhanced.Name {
		t.Errorf("❌ Name conversion failed: got %s, want %s", basic.Name, enhanced.Name)
	}
	if basic.Version != enhanced.Version {
		t.Errorf("❌ Version conversion failed: got %s, want %s", basic.Version, enhanced.Version)
	}
	if basic.Source != enhanced.Source {
		t.Errorf("❌ Source conversion failed: got %s, want %s", basic.Source, enhanced.Source)
	}

	// Test InstalledPackage -> EnhancedPackageInfo conversion
	basicPkg := InstalledPackage{
		Name:    "simple-pkg",
		Version: "2.0.0",
		Source:  "rpm",
	}

	enhancedFromBasic := FromInstalledPackage(basicPkg)
	if enhancedFromBasic.Name != basicPkg.Name {
		t.Errorf("❌ Name conversion failed: got %s, want %s", enhancedFromBasic.Name, basicPkg.Name)
	}
	if enhancedFromBasic.Version != basicPkg.Version {
		t.Errorf("❌ Version conversion failed: got %s, want %s", enhancedFromBasic.Version, basicPkg.Version)
	}
	if enhancedFromBasic.Source != basicPkg.Source {
		t.Errorf("❌ Source conversion failed: got %s, want %s", enhancedFromBasic.Source, basicPkg.Source)
	}

	t.Log("✅ Package type conversion verified")
}

// TestScanResultHelperMethods tests all helper methods
func TestScanResultHelperMethods(t *testing.T) {
	t.Log("\n🔍 Testing ScanResult helper methods...")

	result := &ScanResult{}

	// Test initial state
	if result.HasSystemFingerprint() {
		t.Error("❌ Should not have system fingerprint initially")
	}
	if result.HasCertificateInventory() {
		t.Error("❌ Should not have certificate inventory initially")
	}
	if result.HasServiceInformation() {
		t.Error("❌ Should not have service information initially")
	}

	// Test AddPackage method
	basicPkg := InstalledPackage{Name: "test1", Version: "1.0", Source: "dpkg"}
	result.AddPackage(basicPkg)

	if len(result.Packages) != 1 {
		t.Errorf("❌ Expected 1 basic package, got %d", len(result.Packages))
	}
	if len(result.EnhancedPackages) != 1 {
		t.Errorf("❌ Expected 1 enhanced package, got %d", len(result.EnhancedPackages))
	}

	// Test AddEnhancedPackage method
	enhancedPkg := EnhancedPackageInfo{
		Name:        "test2",
		Version:     "2.0",
		Source:      "rpm",
		CPE:         "cpe:2.3:a:test:test2:2.0:*:*:*:*:*:*:*",
		Description: "Test package 2",
	}
	result.AddEnhancedPackage(enhancedPkg)

	if len(result.Packages) != 2 {
		t.Errorf("❌ Expected 2 basic packages, got %d", len(result.Packages))
	}
	if len(result.EnhancedPackages) != 2 {
		t.Errorf("❌ Expected 2 enhanced packages, got %d", len(result.EnhancedPackages))
	}

	// Test GetPackageList method
	packageList := result.GetPackageList()
	if len(packageList) != 2 {
		t.Errorf("❌ Expected 2 packages from GetPackageList, got %d", len(packageList))
	}

	// Add fingerprinting data and test helper methods
	result.SystemFingerprint = &fingerprint.SystemFingerprint{
		Platform: fingerprint.PlatformLinux,
	}
	result.CertificateInventory = &fingerprint.CertificateInfo{
		CollectedAt: time.Now(),
	}
	result.ServiceInformation = &fingerprint.ServiceInfo{
		CollectedAt: time.Now(),
	}

	if !result.HasSystemFingerprint() {
		t.Error("❌ Should have system fingerprint after setting")
	}
	if !result.HasCertificateInventory() {
		t.Error("❌ Should have certificate inventory after setting")
	}
	if !result.HasServiceInformation() {
		t.Error("❌ Should have service information after setting")
	}

	// Test scan metadata methods
	result.SetScanMetadata("1.0.0", "linux", "test-host", "--test", []string{"packages"})
	if result.ScanMetadata == nil {
		t.Error("❌ ScanMetadata should be set")
	}
	if result.ScanMetadata.AgentVersion != "1.0.0" {
		t.Errorf("❌ Agent version mismatch: got %s, want 1.0.0", result.ScanMetadata.AgentVersion)
	}

	// Test adding collection errors
	result.AddCollectionError("test-module", "test error message")
	if len(result.ScanMetadata.CollectionErrors) == 0 {
		t.Error("❌ Collection errors should be present")
	}
	if len(result.ScanMetadata.CollectionErrors["test-module"]) != 1 {
		t.Error("❌ Should have 1 error for test-module")
	}

	// Test completing scan metadata
	startTime := result.ScanMetadata.ScanStartTime
	time.Sleep(10 * time.Millisecond) // Small delay to ensure duration > 0
	result.CompleteScanMetadata()

	if result.ScanMetadata.ScanEndTime.Before(startTime) {
		t.Error("❌ End time should be after start time")
	}
	if result.ScanMetadata.ScanDuration <= 0 {
		t.Error("❌ Scan duration should be positive")
	}

	t.Log("✅ All helper methods verified")
}

// TestJSONFieldPresence tests that JSON fields are present/absent as expected
func TestJSONFieldPresence(t *testing.T) {
	t.Log("\n🔍 Testing JSON field presence with omitempty...")

	// Test with minimal data (should omit most fields)
	minimal := ScanResult{
		OSInfo: OSInfo{
			OS:       "linux",
			Hostname: "minimal",
		},
	}

	jsonData, err := json.Marshal(minimal)
	if err != nil {
		t.Fatalf("❌ Failed to marshal minimal ScanResult: %v", err)
	}

	jsonStr := string(jsonData)
	t.Logf("\n📄 Minimal JSON: %s", jsonStr)

	// Verify omitempty behavior - these fields should NOT be present
	omittedFields := []string{
		"systemFingerprint",
		"enhancedPackages",
		"certificateInventory",
		"serviceInformation",
		"scanMetadata",
		"packages",
		"scanErrors",
	}

	for _, field := range omittedFields {
		if contains(jsonStr, field) {
			t.Errorf("❌ Field '%s' should be omitted in minimal JSON", field)
		}
	}

	// These fields should be present
	requiredFields := []string{"osInfo", "os", "hostname"}
	for _, field := range requiredFields {
		if !contains(jsonStr, field) {
			t.Errorf("❌ Field '%s' should be present in minimal JSON", field)
		}
	}

	t.Log("✅ JSON field presence verified")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkScanResultSerialization benchmarks JSON serialization performance
func BenchmarkScanResultSerialization(b *testing.B) {
	// Create a moderately complex ScanResult
	result := ScanResult{
		OSInfo: OSInfo{
			OS:        "linux",
			Version:   "Ubuntu 22.04",
			Hostname:  "benchmark-host",
			PrimaryIP: "192.168.1.100",
		},
		Packages:         make([]InstalledPackage, 100),
		EnhancedPackages: make([]EnhancedPackageInfo, 100),
	}

	// Fill with test data
	for i := 0; i < 100; i++ {
		result.Packages[i] = InstalledPackage{
			Name:    "package-" + string(rune(i)),
			Version: "1.0.0",
			Source:  "dpkg",
		}
		result.EnhancedPackages[i] = EnhancedPackageInfo{
			Name:    "enhanced-package-" + string(rune(i)),
			Version: "2.0.0",
			Source:  "dpkg",
			CPE:     "cpe:2.3:a:test:pkg:2.0.0:*:*:*:*:*:*:*",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestCompleteWorkflow tests a complete scan workflow
func TestCompleteWorkflow(t *testing.T) {
	t.Log("\n🔍 Testing complete scan workflow...")

	// Initialize scan result
	result := &ScanResult{
		OSInfo: OSInfo{
			OS:        "linux",
			Version:   "Ubuntu 22.04",
			Hostname:  "workflow-test",
			PrimaryIP: "172.16.1.10",
		},
		Packages:         make([]InstalledPackage, 0),
		EnhancedPackages: make([]EnhancedPackageInfo, 0),
		ScanErrors:       make([]string, 0),
	}

	// Set initial scan metadata
	result.SetScanMetadata("1.2.0", "linux", "workflow-test", "--full-scan", []string{"packages", "fingerprint"})

	// Simulate package discovery
	packages := []EnhancedPackageInfo{
		{
			Name:           "openssh-server",
			Version:        "1:8.9p1-3ubuntu0.4",
			Source:         "dpkg",
			Architecture:   "amd64",
			Description:    "OpenSSH secure shell daemon",
			CPE:            "cpe:2.3:a:openbsd:openssh:8.9p1:*:*:*:*:*:*:*",
			Dependencies:   []string{"libc6", "libssl3"},
			SizeBytes:      458752,
			SizeMB:         0.44,
			Signed:         true,
			SignatureValid: true,
		},
		{
			Name:        "nginx",
			Version:     "1.18.0-6ubuntu14.4",
			Source:      "dpkg",
			Description: "HTTP and reverse proxy server",
			CPE:         "cpe:2.3:a:nginx:nginx:1.18.0:*:*:*:*:*:*:*",
			SizeBytes:   1048576,
			SizeMB:      1.0,
		},
	}

	for _, pkg := range packages {
		result.AddEnhancedPackage(pkg)
	}

	// Simulate system fingerprinting
	result.SystemFingerprint = &fingerprint.SystemFingerprint{
		CollectedAt: time.Now(),
		Platform:    fingerprint.PlatformLinux,
		Hardware: &fingerprint.HardwareInfo{
			CPU: &fingerprint.CPUInfo{
				Model:        "AMD Ryzen 7 5800X",
				Cores:        8,
				Threads:      16,
				Architecture: "x86_64",
			},
			CollectedAt: time.Now(),
		},
		CollectionDuration: 3 * time.Second,
	}

	// Simulate some collection errors
	result.AddCollectionError("certificates", "Certificate store access denied")
	result.AddCollectionError("services", "Some service information unavailable")

	// Complete the scan
	result.CompleteScanMetadata()

	// Verify final result
	if len(result.EnhancedPackages) != 2 {
		t.Errorf("❌ Expected 2 enhanced packages, got %d", len(result.EnhancedPackages))
	}

	if len(result.Packages) != 2 {
		t.Errorf("❌ Expected 2 basic packages, got %d", len(result.Packages))
	}

	if !result.HasSystemFingerprint() {
		t.Error("❌ Should have system fingerprint")
	}

	if result.ScanMetadata == nil {
		t.Error("❌ Should have scan metadata")
	}

	if result.ScanMetadata.ScanDuration <= 0 {
		t.Error("❌ Scan duration should be positive")
	}

	// Test JSON serialization of complete result
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("❌ Failed to serialize complete workflow result: %v", err)
	}

	t.Logf("\n📊 Complete workflow result size: %d bytes", len(jsonData))

	// Verify we can deserialize
	var deserialized ScanResult
	if err := json.Unmarshal(jsonData, &deserialized); err != nil {
		t.Fatalf("❌ Failed to deserialize complete workflow result: %v", err)
	}

	// Verify key data preserved
	if len(deserialized.EnhancedPackages) != 2 {
		t.Error("❌ Enhanced packages not preserved in serialization")
	}

	if deserialized.SystemFingerprint == nil {
		t.Error("❌ System fingerprint not preserved in serialization")
	}

	t.Log("✅ Complete workflow test passed")
}
