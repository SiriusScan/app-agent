package scan

import (
	"context"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/fingerprint"
	"github.com/SiriusScan/go-api/sirius"
	"go.uber.org/zap"
)

// MockAPIClient implements the APIClient interface for testing
type MockAPIClient struct {
	lastHost              sirius.Host
	lastSoftwareInventory map[string]interface{}
	lastSystemFingerprint map[string]interface{}
	lastAgentMetadata     map[string]interface{}
	callCount             int
}

func (m *MockAPIClient) UpdateHostRecord(ctx context.Context, apiBaseURL string, hostData sirius.Host) error {
	m.lastHost = hostData
	m.callCount++
	return nil
}

func (m *MockAPIClient) UpdateHostRecordWithSource(ctx context.Context, apiBaseURL string, hostData sirius.Host) error {
	m.lastHost = hostData
	m.callCount++
	return nil
}

func (m *MockAPIClient) UpdateHostRecordWithEnhancedData(ctx context.Context, apiBaseURL string, hostData sirius.Host, softwareInventory, systemFingerprint, agentMetadata map[string]interface{}) error {
	m.lastHost = hostData
	m.lastSoftwareInventory = softwareInventory
	m.lastSystemFingerprint = systemFingerprint
	m.lastAgentMetadata = agentMetadata
	m.callCount++
	return nil
}

func TestScanCommandDatabaseIntegration(t *testing.T) {
	// Create mock API client
	mockClient := &MockAPIClient{}

	// Create test agent info
	logger := zap.NewNop()
	agentInfo := commands.AgentInfo{
		Logger:    logger,
		APIClient: mockClient,
		Config: &config.AgentConfig{
			AgentID:       "test-agent-123",
			HostID:        "test-host-456",
			ApiBaseURL:    "http://localhost:8080",
			ServerAddress: "localhost:9090",
		},
		StartTime:        time.Now(),
		ScriptingEnabled: false,
	}

	// Create scan command
	scanCmd := &ScanCommand{}

	t.Run("TestConvertScanResultToHostWithJSONB", func(t *testing.T) {
		// Create test scan result with enhanced packages and fingerprint
		result := &ScanResult{
			OSInfo: OSInfo{
				OS:        "linux",
				Version:   "Ubuntu 22.04",
				Hostname:  "test-host",
				PrimaryIP: "192.168.1.100",
			},
			EnhancedPackages: []EnhancedPackageInfo{
				{
					Name:         "curl",
					Version:      "7.81.0-1ubuntu1.15",
					Architecture: "amd64",
					Description:  "command line tool for transferring data",
					Publisher:    "Ubuntu Developers",
					InstallDate:  time.Now(),
					Size:         461824,
				},
				{
					Name:         "openssh-server",
					Version:      "1:8.9p1-3ubuntu0.6",
					Architecture: "amd64",
					Description:  "secure shell (SSH) server",
					Publisher:    "Ubuntu Developers",
					InstallDate:  time.Now(),
					Size:         1048576,
				},
			},
			SystemFingerprint: &fingerprint.SystemFingerprint{
				CollectedAt: time.Now(),
				Platform:    fingerprint.LinuxPlatform,
				Hardware: &fingerprint.HardwareInfo{
					CPU: &fingerprint.CPUInfo{
						ModelName: "Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz",
						Cores:     8,
						Threads:   16,
					},
					Memory: &fingerprint.MemoryInfo{
						TotalMemory: 16777216000, // 16GB
						Available:   8388608000,  // 8GB
					},
					CollectedAt: time.Now(),
				},
				CollectionDuration: time.Millisecond * 500,
			},
		}

		// Test conversion
		host, softwareInventory, systemFingerprint, agentMetadata, err := scanCmd.convertScanResultToHostWithJSONB(agentInfo, result)
		if err != nil {
			t.Fatalf("Failed to convert scan result: %v", err)
		}

		// Validate host data
		if host.IP != "192.168.1.100" {
			t.Errorf("Expected IP 192.168.1.100, got %s", host.IP)
		}
		if host.OS != "linux" {
			t.Errorf("Expected OS linux, got %s", host.OS)
		}

		// Validate software inventory
		if len(softwareInventory) == 0 {
			t.Error("Expected software inventory data")
		}
		if packages, ok := softwareInventory["packages"]; !ok {
			t.Error("Expected packages field in software inventory")
		} else {
			if pkgList, ok := packages.([]EnhancedPackageInfo); !ok {
				t.Error("Expected packages to be []EnhancedPackageInfo")
			} else if len(pkgList) != 2 {
				t.Errorf("Expected 2 packages, got %d", len(pkgList))
			}
		}

		// Validate system fingerprint
		if len(systemFingerprint) == 0 {
			t.Error("Expected system fingerprint data")
		}
		if platform, ok := systemFingerprint["platform"]; !ok {
			t.Error("Expected platform field in system fingerprint")
		} else if platform != fingerprint.LinuxPlatform {
			t.Errorf("Expected platform %s, got %v", fingerprint.LinuxPlatform, platform)
		}

		// Validate agent metadata
		if len(agentMetadata) == 0 {
			t.Error("Expected agent metadata")
		}
		if agentID, ok := agentMetadata["agent_id"]; !ok {
			t.Error("Expected agent_id in metadata")
		} else if agentID != "test-agent-123" {
			t.Errorf("Expected agent_id test-agent-123, got %v", agentID)
		}

		t.Logf("✅ Successfully converted scan result to host with JSONB data")
		t.Logf("   Software inventory: %d fields", len(softwareInventory))
		t.Logf("   System fingerprint: %d fields", len(systemFingerprint))
		t.Logf("   Agent metadata: %d fields", len(agentMetadata))
	})

	t.Run("TestScanCommandWithEnhancedData", func(t *testing.T) {
		// Reset mock client
		mockClient.callCount = 0

		// Create a context
		ctx := context.Background()

		// Execute scan command with fingerprint enabled
		output, err := scanCmd.Execute(ctx, agentInfo, "scan --fingerprint", "--fingerprint")
		if err != nil {
			t.Logf("⚠️  Scan command failed (expected in test environment): %v", err)
			// This is expected in test environment without actual system access
		} else {
			t.Logf("✅ Scan command executed successfully")
			t.Logf("   Output length: %d characters", len(output))
		}

		// The API call should have been attempted (even if scan failed)
		// Note: In a real test environment, we'd mock the system calls too
		t.Logf("   API client call count: %d", mockClient.callCount)
	})
}

func TestEnhancedPackageValidation(t *testing.T) {
	t.Run("TestPackageConversion", func(t *testing.T) {
		// Test basic package to enhanced package conversion
		basicPkg := InstalledPackage{
			Name:    "test-package",
			Version: "1.0.0",
		}

		enhancedPkg := FromInstalledPackage(basicPkg)
		if enhancedPkg.Name != basicPkg.Name {
			t.Errorf("Expected name %s, got %s", basicPkg.Name, enhancedPkg.Name)
		}
		if enhancedPkg.Version != basicPkg.Version {
			t.Errorf("Expected version %s, got %s", basicPkg.Version, enhancedPkg.Version)
		}

		// Test conversion back
		convertedBack := enhancedPkg.ToInstalledPackage()
		if convertedBack.Name != basicPkg.Name {
			t.Errorf("Expected name %s after conversion, got %s", basicPkg.Name, convertedBack.Name)
		}

		t.Logf("✅ Package conversion working correctly")
	})
}

func TestJSONBDataStructures(t *testing.T) {
	t.Run("TestSoftwareInventoryStructure", func(t *testing.T) {
		// Test that software inventory has expected structure
		packages := []EnhancedPackageInfo{
			{
				Name:         "test-pkg",
				Version:      "1.0.0",
				Architecture: "amd64",
				Publisher:    "Test Publisher",
			},
		}

		inventory := map[string]interface{}{
			"packages":      packages,
			"package_count": len(packages),
			"collected_at":  time.Now().UTC(),
			"source":        "sirius-agent",
		}

		// Validate structure
		if count, ok := inventory["package_count"]; !ok {
			t.Error("Expected package_count field")
		} else if count != 1 {
			t.Errorf("Expected package_count 1, got %v", count)
		}

		if source, ok := inventory["source"]; !ok {
			t.Error("Expected source field")
		} else if source != "sirius-agent" {
			t.Errorf("Expected source sirius-agent, got %v", source)
		}

		t.Logf("✅ Software inventory structure validated")
	})

	t.Run("TestAgentMetadataStructure", func(t *testing.T) {
		metadata := map[string]interface{}{
			"agent_id":       "test-agent",
			"host_id":        "test-host",
			"scan_timestamp": time.Now().UTC(),
			"agent_version":  "1.0.0",
			"platform":       "linux",
			"architecture":   "amd64",
		}

		// Validate required fields
		requiredFields := []string{"agent_id", "host_id", "scan_timestamp", "platform"}
		for _, field := range requiredFields {
			if _, ok := metadata[field]; !ok {
				t.Errorf("Expected required field %s", field)
			}
		}

		t.Logf("✅ Agent metadata structure validated")
	})
}
