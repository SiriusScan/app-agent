package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/shell"
	"go.uber.org/zap"
)

// gatherWindowsPackages collects installed software information on Windows.
// It uses PowerShell to query the registry's Uninstall keys.
func gatherWindowsPackages(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult) ([]InstalledPackage, error) {
	if !agentInfo.ScriptingEnabled {
		msg := "Scripting disabled or PowerShell not found, cannot gather Windows packages."
		agentInfo.Logger.Warn(msg)
		result.ScanErrors = append(result.ScanErrors, msg)
		return nil, fmt.Errorf(msg)
	}

	agentInfo.Logger.Info("Gathering Windows packages via PowerShell registry query")

	// PowerShell script to get software from registry.
	// Removed Write-Error diagnostics which caused issues.
	script := `
$ErrorActionPreference = 'Stop'
$outputJson = "[]" # Default to empty JSON array

try {
    $items = Get-ItemProperty HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*,
                     HKLM:\Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\* -ErrorAction SilentlyContinue
    
    if ($items -ne $null) {
        $filteredItems = $items | 
            Select-Object DisplayName, DisplayVersion, Publisher, InstallDate | 
            Where-Object {$_.DisplayName -ne $null -and $_.DisplayName -ne '' -and $_.DisplayVersion -ne $null -and $_.DisplayVersion -ne ''}
        
        # Check if $filteredItems is null or empty before converting
        # ConvertTo-Json handles empty collections correctly (outputs []) but this adds safety
        if ($filteredItems -ne $null) { 
            $outputJson = $filteredItems | ConvertTo-Json -Compress -Depth 3
        }
    }
} catch {
    # Log the error to stderr for Go to capture
    Write-Host "Error during package script execution: $($_.Exception.ToString())" -ForegroundColor Red -ErrorVariable scriptError
    # Set a specific exit code? Maybe not needed if stderr is captured.
}

# Always output the JSON string (or the default "[]")
Write-Output $outputJson 
`

	stdout, stderr, exitCode, err := shell.ExecuteScript(ctx, agentInfo.PowerShellPath, script)

	// Log stderr regardless of exit code
	if stderr != "" {
		agentInfo.Logger.Warn("PowerShell script stderr output", zap.String("stderr", stderr)) // Use Warn level for stderr
	}

	if err != nil {
		// Script execution itself failed (e.g., process couldn't start, non-zero exit)
		errMsg := fmt.Sprintf("PowerShell script execution failed (ExitCode: %d): %v. Stderr: %s", exitCode, err, stderr)
		agentInfo.Logger.Error("Windows package gathering failed", zap.Error(err), zap.Int("exitCode", exitCode), zap.String("stderr", stderr))
		result.ScanErrors = append(result.ScanErrors, errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	// Trim whitespace from stdout before checking if empty
	trimmedStdout := strings.TrimSpace(stdout)

	if trimmedStdout == "" {
		// Script executed successfully but produced no output. This might be valid (no packages found).
		agentInfo.Logger.Info("Windows package script returned empty output. Assuming no packages found.", zap.Int("exitCode", exitCode), zap.String("stderr", stderr))
		// Log stderr in case it contains relevant warnings
		if stderr != "" {
			agentInfo.Logger.Warn("PowerShell stderr had content despite success exit code", zap.String("stderr", stderr))
		}
		return make([]InstalledPackage, 0), nil // Return empty list, not an error
	}

	// Define a temporary struct to unmarshal PowerShell output
	type psPackageInfo struct {
		DisplayName    string
		DisplayVersion string
		Publisher      string // Optional, capture if present
		InstallDate    string // Optional, capture if present
	}

	var rawPackages []psPackageInfo
	// Handle cases where PowerShell might return a single object instead of an array
	// by first trying to unmarshal into a slice, and fallback to single object
	if err := json.Unmarshal([]byte(trimmedStdout), &rawPackages); err != nil {
		// Try unmarshalling as a single object
		var singlePackage psPackageInfo
		if singleErr := json.Unmarshal([]byte(trimmedStdout), &singlePackage); singleErr == nil {
			// If single object parsing succeeds, put it into the slice
			if singlePackage.DisplayName != "" {
				rawPackages = append(rawPackages, singlePackage)
			}
		} else {
			// If both slice and single object fail, report the original slice error
			parseErrMsg := fmt.Sprintf("Failed to parse PowerShell JSON output for packages: %v. Output: [%s], Stderr: [%s]", err, trimmedStdout, stderr)
			agentInfo.Logger.Error("Windows package JSON parsing failed", zap.Error(err), zap.String("stdout", trimmedStdout), zap.String("stderr", stderr))
			result.ScanErrors = append(result.ScanErrors, parseErrMsg)
			// Return nil slice but no error, as script executed but parsing failed
			return nil, nil
		}
	}

	// Convert to the standard InstalledPackage format
	packages := make([]InstalledPackage, 0, len(rawPackages))
	for _, rp := range rawPackages {
		// Basic sanitization
		name := strings.TrimSpace(rp.DisplayName)
		version := strings.TrimSpace(rp.DisplayVersion)
		if name != "" && version != "" {
			packages = append(packages, InstalledPackage{
				Name:    name,
				Version: version,
				Source:  "windows_registry", // Indicate the source
			})
		}
	}

	agentInfo.Logger.Info("Successfully gathered Windows packages", zap.Int("count", len(packages)))
	return packages, nil
}
