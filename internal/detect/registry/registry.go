package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
)

// RegistryReader handles Windows registry queries and analysis
type RegistryReader struct {
	logger         *zap.Logger
	timeout        time.Duration
	powershellPath string
}

// NewRegistryReader creates a new registry reader instance
func NewRegistryReader(logger *zap.Logger) *RegistryReader {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Determine PowerShell path based on platform
	powershellPath := "powershell.exe"
	if runtime.GOOS != "windows" {
		// For non-Windows systems, try to find PowerShell Core (pwsh)
		if path, err := exec.LookPath("pwsh"); err == nil {
			powershellPath = path
		} else {
			powershellPath = "" // Will cause registry operations to fail gracefully
		}
	}

	return &RegistryReader{
		logger:         logger,
		timeout:        30 * time.Second,
		powershellPath: powershellPath,
	}
}

// SetTimeout sets the maximum execution time for registry operations
func (rr *RegistryReader) SetTimeout(timeout time.Duration) {
	rr.timeout = timeout
}

// CheckRegistryKey verifies if a registry key exists and optionally checks its value
func (rr *RegistryReader) CheckRegistryKey(ctx context.Context, target detect.RegistryTarget) (*RegistryResult, error) {
	rr.logger.Debug("Checking registry key",
		zap.String("path", target.Path),
		zap.String("value", target.Value))

	startTime := time.Now()

	result := &RegistryResult{
		Target:      target,
		KeyExists:   false,
		ValueExists: false,
		CheckedAt:   startTime,
	}

	// Check if we're on Windows or have PowerShell available
	if rr.powershellPath == "" {
		result.Error = "PowerShell not available for registry operations"
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, rr.timeout)
	defer cancel()

	// Check if key exists
	keyExists, err := rr.checkKeyExists(execCtx, target.Path)
	if err != nil {
		result.Error = err.Error()
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}
	result.KeyExists = keyExists

	if !keyExists {
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	// If specific value is requested, check it
	if target.Value != "" {
		valueResult, err := rr.getRegistryValue(execCtx, target.Path, target.Value)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.ValueExists = valueResult.Exists
			result.ValueData = valueResult.Data
			result.ValueType = valueResult.Type

			// Apply pattern matching if specified
			if target.Pattern != "" && valueResult.Data != "" {
				matches, err := rr.matchPattern(target.Pattern, valueResult.Data)
				if err != nil {
					result.Error = fmt.Sprintf("pattern matching failed: %v", err)
				} else {
					result.PatternMatches = matches
				}
			}
		}
	}

	result.ProcessingTime = time.Since(startTime)

	rr.logger.Debug("Registry key check completed",
		zap.String("path", target.Path),
		zap.Bool("key_exists", result.KeyExists),
		zap.Bool("value_exists", result.ValueExists),
		zap.Duration("processing_time", result.ProcessingTime))

	return result, nil
}

// BatchCheckRegistryKeys checks multiple registry keys
func (rr *RegistryReader) BatchCheckRegistryKeys(ctx context.Context, targets []detect.RegistryTarget) ([]*RegistryResult, error) {
	rr.logger.Info("Starting batch registry key check",
		zap.Int("targets", len(targets)))

	results := make([]*RegistryResult, len(targets))

	for i, target := range targets {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result, err := rr.CheckRegistryKey(ctx, target)
		if err != nil {
			// Create error result
			result = &RegistryResult{
				Target:    target,
				Error:     err.Error(),
				CheckedAt: time.Now(),
			}
		}
		results[i] = result
	}

	rr.logger.Info("Batch registry key check completed",
		zap.Int("total", len(results)),
		zap.Int("successful", rr.countSuccessfulResults(results)),
		zap.Int("errors", rr.countErrorResults(results)))

	return results, nil
}

// checkKeyExists uses PowerShell to check if a registry key exists
func (rr *RegistryReader) checkKeyExists(ctx context.Context, keyPath string) (bool, error) {
	// PowerShell script to check if registry key exists
	script := fmt.Sprintf(`
		try {
			$key = Test-Path -Path "Registry::%s" -ErrorAction Stop
			Write-Output $key
		} catch {
			Write-Output $false
		}
	`, keyPath)

	output, err := rr.executePowerShellScript(ctx, script)
	if err != nil {
		return false, fmt.Errorf("failed to check registry key existence: %w", err)
	}

	return strings.TrimSpace(output) == "True", nil
}

// getRegistryValue retrieves a specific registry value
func (rr *RegistryReader) getRegistryValue(ctx context.Context, keyPath, valueName string) (*RegistryValue, error) {
	// PowerShell script to get registry value with type information
	script := fmt.Sprintf(`
		try {
			$key = Get-ItemProperty -Path "Registry::%s" -Name "%s" -ErrorAction Stop
			$value = $key."%s"
			$type = (Get-Item -Path "Registry::%s").GetValueKind("%s")
			
			$result = @{
				"exists" = $true
				"data" = $value.ToString()
				"type" = $type.ToString()
			}
			$result | ConvertTo-Json -Compress
		} catch {
			$result = @{
				"exists" = $false
				"data" = ""
				"type" = ""
			}
			$result | ConvertTo-Json -Compress
		}
	`, keyPath, valueName, valueName, keyPath, valueName)

	output, err := rr.executePowerShellScript(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry value: %w", err)
	}

	var result RegistryValue
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("failed to parse registry value result: %w", err)
	}

	return &result, nil
}

// matchPattern applies regex pattern matching to registry value data
func (rr *RegistryReader) matchPattern(pattern, data string) (bool, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return regex.MatchString(data), nil
}

// executePowerShellScript executes a PowerShell script and returns output
func (rr *RegistryReader) executePowerShellScript(ctx context.Context, script string) (string, error) {
	if rr.powershellPath == "" {
		return "", fmt.Errorf("PowerShell not available")
	}

	// Create PowerShell command
	args := []string{
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-Command", script,
	}

	cmd := exec.CommandContext(ctx, rr.powershellPath, args...)

	// Execute and capture output
	output, err := cmd.Output()
	if err != nil {
		// Try to get more details from stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("PowerShell execution failed: %v, stderr: %s", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("PowerShell execution failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Helper functions

func (rr *RegistryReader) countSuccessfulResults(results []*RegistryResult) int {
	count := 0
	for _, result := range results {
		if result.Error == "" {
			count++
		}
	}
	return count
}

func (rr *RegistryReader) countErrorResults(results []*RegistryResult) int {
	count := 0
	for _, result := range results {
		if result.Error != "" {
			count++
		}
	}
	return count
}

// IsWindowsRegistryAvailable checks if Windows registry operations are available
func (rr *RegistryReader) IsWindowsRegistryAvailable() bool {
	return rr.powershellPath != "" && (runtime.GOOS == "windows" || rr.powershellPath == "pwsh")
}

// GetStats returns registry reader statistics
func (rr *RegistryReader) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"platform":        runtime.GOOS,
		"powershell_path": rr.powershellPath,
		"available":       rr.IsWindowsRegistryAvailable(),
		"timeout_seconds": rr.timeout.Seconds(),
	}
}
