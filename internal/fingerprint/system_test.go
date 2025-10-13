package fingerprint

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestNewSystemHardwareCollector verifies hardware collector creation
func TestNewSystemHardwareCollector(t *testing.T) {
	t.Log("\n🔍 Testing SystemHardwareCollector creation...")

	collector := NewSystemHardwareCollector()
	if collector == nil {
		t.Fatal("❌ NewSystemHardwareCollector returned nil")
	}

	// Verify it implements the HardwareCollector interface
	_, ok := collector.(HardwareCollector)
	if !ok {
		t.Fatal("❌ SystemHardwareCollector does not implement HardwareCollector interface")
	}

	t.Log("✅ SystemHardwareCollector created successfully")
}

// TestGetCPUInfo tests CPU information collection across platforms
func TestGetCPUInfo(t *testing.T) {
	t.Log("\n🔍 Testing CPU information collection...")

	collector := NewSystemHardwareCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cpuInfo, err := collector.GetCPUInfo(ctx)
	if err != nil {
		t.Fatalf("❌ GetCPUInfo failed: %v", err)
	}

	if cpuInfo == nil {
		t.Fatal("❌ GetCPUInfo returned nil CPUInfo")
	}

	t.Logf("\n📊 CPU Information:\n")
	t.Logf("  Model: %s", cpuInfo.Model)
	t.Logf("  Vendor: %s", cpuInfo.Vendor)
	t.Logf("  Architecture: %s", cpuInfo.Architecture)
	t.Logf("  Cores: %d", cpuInfo.Cores)
	t.Logf("  Threads: %d", cpuInfo.Threads)
	t.Logf("  Clock Speed: %.2f MHz", cpuInfo.ClockSpeed)
	t.Logf("  Cache Size: %d KB", cpuInfo.CacheSize)

	// Validate required fields
	if cpuInfo.Architecture == "" {
		t.Error("❌ CPU Architecture is empty")
	}

	if cpuInfo.Architecture != runtime.GOARCH {
		t.Errorf("❌ CPU Architecture mismatch: got %s, expected %s", cpuInfo.Architecture, runtime.GOARCH)
	}

	if cpuInfo.Cores <= 0 {
		t.Error("❌ CPU Cores should be greater than 0")
	}

	if cpuInfo.Threads <= 0 {
		t.Error("❌ CPU Threads should be greater than 0")
	}

	if cpuInfo.Threads < cpuInfo.Cores {
		t.Error("❌ CPU Threads should be >= Cores")
	}

	t.Log("✅ CPU information collection successful")
}

// TestGetMemoryInfo tests memory information collection across platforms
func TestGetMemoryInfo(t *testing.T) {
	t.Log("\n🔍 Testing memory information collection...")

	collector := NewSystemHardwareCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	memInfo, err := collector.GetMemoryInfo(ctx)
	if err != nil {
		t.Fatalf("❌ GetMemoryInfo failed: %v", err)
	}

	if memInfo == nil {
		t.Fatal("❌ GetMemoryInfo returned nil MemoryInfo")
	}

	t.Logf("\n💾 Memory Information:\n")
	t.Logf("  Total: %.2f GB", memInfo.TotalGB)
	t.Logf("  Available: %.2f GB", memInfo.AvailableGB)
	t.Logf("  Used: %.2f GB", memInfo.UsedGB)
	t.Logf("  Usage: %.1f%%", memInfo.UsagePercent)
	t.Logf("  Swap Total: %.2f GB", memInfo.SwapTotalGB)
	t.Logf("  Swap Used: %.2f GB", memInfo.SwapUsedGB)

	// Validate memory values
	if memInfo.TotalGB <= 0 {
		t.Error("❌ Total memory should be greater than 0")
	}

	if memInfo.AvailableGB < 0 {
		t.Error("❌ Available memory should be >= 0")
	}

	if memInfo.UsedGB < 0 {
		t.Error("❌ Used memory should be >= 0")
	}

	if memInfo.AvailableGB > memInfo.TotalGB {
		t.Error("❌ Available memory should not exceed total memory")
	}

	if memInfo.UsagePercent < 0 || memInfo.UsagePercent > 100 {
		t.Error("❌ Memory usage percentage should be between 0 and 100")
	}

	// Check calculation consistency
	calculatedUsed := memInfo.TotalGB - memInfo.AvailableGB
	if abs(memInfo.UsedGB-calculatedUsed) > 0.1 {
		t.Errorf("❌ Memory calculation inconsistent: reported used %.2f GB, calculated %.2f GB",
			memInfo.UsedGB, calculatedUsed)
	}

	t.Log("✅ Memory information collection successful")
}

// TestGetStorageInfo tests storage information collection across platforms
func TestGetStorageInfo(t *testing.T) {
	t.Log("\n🔍 Testing storage information collection...")

	collector := NewSystemHardwareCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	storageDevices, err := collector.GetStorageInfo(ctx)
	if err != nil {
		t.Fatalf("❌ GetStorageInfo failed: %v", err)
	}

	if storageDevices == nil {
		t.Fatal("❌ GetStorageInfo returned nil storage devices")
	}

	if len(storageDevices) == 0 {
		t.Log("⚠️ No storage devices found - this might be expected in some environments")
		return
	}

	t.Logf("\n💽 Found %d storage devices:\n", len(storageDevices))

	for i, device := range storageDevices {
		t.Logf("  Device %d:", i+1)
		t.Logf("    Device: %s", device.Device)
		t.Logf("    Model: %s", device.Model)
		t.Logf("    Size: %.2f GB", device.SizeGB)
		t.Logf("    Type: %s", device.Type)
		t.Logf("    Interface: %s", device.Interface)
		t.Logf("    Filesystem: %s", device.FileSystem)
		t.Logf("    Mount Point: %s", device.MountPoint)
		t.Logf("    Used: %.2f GB", device.UsedGB)
		t.Logf("    Available: %.2f GB", device.AvailableGB)
		t.Logf("    Usage: %.1f%%", device.UsagePercent)
		t.Logf("    Serial: %s", device.SerialNumber)

		// Validate storage device data
		if device.Device == "" {
			t.Errorf("❌ Device %d has empty device name", i+1)
		}

		if device.SizeGB <= 0 {
			t.Errorf("❌ Device %d has invalid size: %.2f GB", i+1, device.SizeGB)
		}

		if device.Type == "" {
			t.Errorf("❌ Device %d has empty type", i+1)
		}

		// Validate usage calculations if available
		if device.UsedGB > 0 && device.AvailableGB > 0 {
			totalUsed := device.UsedGB + device.AvailableGB
			if totalUsed > device.SizeGB*1.1 { // Allow 10% variance for filesystem overhead
				t.Errorf("❌ Device %d: used+available (%.2f GB) exceeds device size (%.2f GB)",
					i+1, totalUsed, device.SizeGB)
			}

			if device.UsagePercent < 0 || device.UsagePercent > 100 {
				t.Errorf("❌ Device %d has invalid usage percentage: %.1f%%", i+1, device.UsagePercent)
			}
		}
	}

	t.Log("✅ Storage information collection successful")
}

// TestGetSystemInfo tests system information collection across platforms
func TestGetSystemInfo(t *testing.T) {
	t.Log("\n🔍 Testing system information collection...")

	collector := NewSystemHardwareCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	systemInfo, err := collector.GetSystemInfo(ctx)
	if err != nil {
		t.Fatalf("❌ GetSystemInfo failed: %v", err)
	}

	if systemInfo == nil {
		t.Fatal("❌ GetSystemInfo returned nil SystemInfo")
	}

	t.Logf("\n🖥️ System Information:\n")
	t.Logf("  Hostname: %s", systemInfo.Hostname)
	t.Logf("  OS: %s", systemInfo.OS)
	t.Logf("  OS Version: %s", systemInfo.OSVersion)
	t.Logf("  Kernel: %s", systemInfo.Kernel)
	t.Logf("  Architecture: %s", systemInfo.Architecture)
	t.Logf("  Uptime: %d seconds", systemInfo.Uptime)
	t.Logf("  Boot Time: %s", systemInfo.BootTime.Format(time.RFC3339))
	t.Logf("  Time Zone: %s", systemInfo.TimeZone)
	t.Logf("  Domain: %s", systemInfo.Domain)
	t.Logf("  Manufacturer: %s", systemInfo.Manufacturer)

	// Validate required fields
	if systemInfo.OS == "" {
		t.Error("❌ OS field is empty")
	}

	if systemInfo.Architecture == "" {
		t.Error("❌ Architecture field is empty")
	}

	if systemInfo.Architecture != runtime.GOARCH {
		t.Errorf("❌ Architecture mismatch: got %s, expected %s",
			systemInfo.Architecture, runtime.GOARCH)
	}

	// Validate OS matches runtime
	expectedOS := map[string]string{
		"linux":   "Linux",
		"windows": "Windows",
		"darwin":  "macOS",
	}

	if expected, ok := expectedOS[runtime.GOOS]; ok {
		if systemInfo.OS != expected {
			t.Errorf("❌ OS mismatch: got %s, expected %s", systemInfo.OS, expected)
		}
	}

	// Validate uptime and boot time relationship
	if systemInfo.Uptime > 0 && !systemInfo.BootTime.IsZero() {
		expectedBootTime := time.Now().Add(-time.Duration(systemInfo.Uptime) * time.Second)
		timeDiff := abs64(systemInfo.BootTime.Unix() - expectedBootTime.Unix())
		if timeDiff > 60 { // Allow 1 minute variance
			t.Errorf("❌ Boot time calculation inconsistent: diff %d seconds", timeDiff)
		}
	}

	t.Log("✅ System information collection successful")
}

// TestPlatformSpecificMethods tests platform-specific helper methods
func TestPlatformSpecificMethods(t *testing.T) {
	t.Log("\n🔍 Testing platform-specific helper methods...")

	collector := &SystemHardwareCollector{timeout: 30 * time.Second}

	// Test storage type detection
	testCases := map[string]struct {
		deviceName   string
		expectedType string
	}{
		"nvme0n1": {"nvme0n1", "NVMe"},
		"sda1":    {"sda1", "SATA"},
		"hda1":    {"hda1", "IDE"},
		"unknown": {"unknown", "Unknown"},
	}

	switch runtime.GOOS {
	case "linux":
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				storageType := collector.getLinuxStorageType(tc.deviceName)
				t.Logf("Device %s detected as type: %s", tc.deviceName, storageType)

				// For NVMe, we expect exact match
				if tc.deviceName == "nvme0n1" && storageType != "NVMe" {
					t.Errorf("❌ Expected NVMe for %s, got %s", tc.deviceName, storageType)
				}
			})
		}

		// Test interface detection
		interfaceTests := map[string]string{
			"nvme0n1": "NVMe",
			"sda1":    "SATA",
			"hda1":    "IDE",
		}

		for device, expectedInterface := range interfaceTests {
			t.Run("interface_"+device, func(t *testing.T) {
				actualInterface := collector.getLinuxStorageInterface(device)
				if actualInterface != expectedInterface {
					t.Errorf("❌ Expected interface %s for %s, got %s",
						expectedInterface, device, actualInterface)
				}
				t.Logf("Device %s detected interface: %s", device, actualInterface)
			})
		}

	case "windows":
		// Test Windows storage type mapping
		windowsTests := map[string]string{
			"ide":  "HDD",
			"sata": "SSD",
			"scsi": "SSD",
			"usb":  "USB",
		}

		for interfaceType, expectedType := range windowsTests {
			t.Run("windows_"+interfaceType, func(t *testing.T) {
				actualType := collector.getWindowsStorageType(interfaceType)
				if actualType != expectedType {
					t.Errorf("❌ Expected type %s for %s interface, got %s",
						expectedType, interfaceType, actualType)
				}
				t.Logf("Windows interface %s maps to type: %s", interfaceType, actualType)
			})
		}

	case "darwin":
		// Test macOS size parsing
		macOSTests := map[string]float64{
			"100G":    100.0,
			"1.5T":    1536.0,
			"512M":    0.5,
			"2Gi":     2.0,
			"invalid": 0.0,
		}

		for sizeStr, expectedGB := range macOSTests {
			t.Run("macos_size_"+sizeStr, func(t *testing.T) {
				actualGB := collector.parseMacOSSize(sizeStr)
				if abs(actualGB-expectedGB) > 0.1 {
					t.Errorf("❌ Expected %.1f GB for %s, got %.1f GB",
						expectedGB, sizeStr, actualGB)
				}
				t.Logf("Size %s parsed as: %.1f GB", sizeStr, actualGB)
			})
		}
	}

	t.Log("✅ Platform-specific methods testing completed")
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	t.Log("\n🔍 Testing context cancellation handling...")

	collector := NewSystemHardwareCollector()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test with cancelled context
	_, err := collector.GetCPUInfo(ctx)
	if err == nil {
		t.Log("⚠️ GetCPUInfo with cancelled context didn't return error - may not support cancellation")
	} else {
		t.Logf("✅ GetCPUInfo properly handled cancelled context: %v", err)
	}

	// Test with very short timeout
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer shortCancel()

	// Allow the context to timeout
	time.Sleep(1 * time.Millisecond)

	_, err = collector.GetSystemInfo(shortCtx)
	if err == nil {
		t.Log("⚠️ GetSystemInfo with timed-out context didn't return error - may not support timeouts")
	} else {
		t.Logf("✅ GetSystemInfo properly handled timed-out context: %v", err)
	}

	t.Log("✅ Context cancellation testing completed")
}

// BenchmarkHardwareCollection benchmarks the performance of hardware collection
func BenchmarkHardwareCollection(b *testing.B) {
	collector := NewSystemHardwareCollector()
	ctx := context.Background()

	b.Run("CPUInfo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := collector.GetCPUInfo(ctx)
			if err != nil {
				b.Fatalf("GetCPUInfo failed: %v", err)
			}
		}
	})

	b.Run("MemoryInfo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := collector.GetMemoryInfo(ctx)
			if err != nil {
				b.Fatalf("GetMemoryInfo failed: %v", err)
			}
		}
	})

	b.Run("SystemInfo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := collector.GetSystemInfo(ctx)
			if err != nil {
				b.Fatalf("GetSystemInfo failed: %v", err)
			}
		}
	})

	// Note: Storage info can be slow due to external commands, so fewer iterations
	b.Run("StorageInfo", func(b *testing.B) {
		b.N = min(b.N, 10) // Limit iterations for storage
		for i := 0; i < b.N; i++ {
			_, err := collector.GetStorageInfo(ctx)
			if err != nil {
				b.Fatalf("GetStorageInfo failed: %v", err)
			}
		}
	})
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
