package scan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/shell"
	"github.com/SiriusScan/app-agent/internal/sysinfo"
	"go.uber.org/zap"
)

// ScanCommand implements the internal scan command.
type ScanCommand struct{}

// Ensures ScanCommand implements the Command interface at compile time.
var _ commands.Command = (*ScanCommand)(nil)

func init() {
	// Register this command with the prefix "internal:scan"
	commands.Register("internal:scan", &ScanCommand{})
}

// Execute gathers system inventory based on arguments and OS, returning results as JSON.
func (c *ScanCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (output string, err error) {
	agentInfo.Logger.Info("Executing internal scan command", zap.String("raw_args", args))

	result := ScanResult{
		OSInfo: OSInfo{
			OS:        runtime.GOOS,
			Version:   sysinfo.GetOSVersion(),
			Hostname:  agentInfo.Config.AgentID, // Use agent ID as hostname for now
			PrimaryIP: sysinfo.GetPrimaryIP(),
		},
		Packages:      make([]InstalledPackage, 0),
		CustomResults: make(map[string]CustomScanOutput),
		ScanErrors:    make([]string, 0),
	}

	// --- Argument Parsing ---
	scanPackages := true // Default: Scan packages
	scanCustomScripts := []string{}
	scriptsFlagFound := false // Track if --scripts was explicitly used

	agentInfo.Logger.Debug("Raw args string for parsing", zap.String("args", args))
	argsList := strings.Fields(args) // Split args by space
	agentInfo.Logger.Debug("Args split by space", zap.Strings("argsList", argsList))
	for _, arg := range argsList {
		if strings.HasPrefix(arg, "--scripts=") {
			scriptsFlagFound = true // Mark that the flag was found
			scriptNames := strings.TrimPrefix(arg, "--scripts=")
			if scriptNames != "" {
				scanCustomScripts = strings.Split(scriptNames, ",")
				// Trim whitespace from each script name
				for i := range scanCustomScripts {
					scanCustomScripts[i] = strings.TrimSpace(scanCustomScripts[i])
				}
			}
		}
		// TODO: Add flags for modules, e.g., --modules=packages,patches or --modules=none
		// Example: else if arg == "--modules=none" { scanPackages = false }
	}

	// If --scripts was explicitly provided, disable the default package scan
	// (unless other flags like --modules=packages modify this later)
	if scriptsFlagFound {
		scanPackages = false
		// TODO: If --modules=packages is *also* specified, re-enable scanPackages here.
	}

	agentInfo.Logger.Debug("Scan arguments parsed",
		zap.Bool("scanPackages", scanPackages),
		zap.Strings("final_scanCustomScripts", scanCustomScripts))

	// --- OS Specific Logic ---
	if scanPackages {
		pkgs, err := c.gatherPlatformPackages(ctx, agentInfo, &result) // Pass result to append errors
		if err != nil {
			// Log the top-level error from gather function, detailed errors are in result.ScanErrors
			agentInfo.Logger.Error("Error gathering packages", zap.Error(err))
			result.ScanErrors = append(result.ScanErrors, fmt.Sprintf("Package gathering failed: %v", err))
		} else {
			result.Packages = pkgs
		}
	}

	// --- Custom Script Execution ---
	if len(scanCustomScripts) > 0 {
		agentInfo.Logger.Debug("Calling executeCustomScripts", zap.Strings("scripts", scanCustomScripts))
		c.executeCustomScripts(ctx, agentInfo, &result, scanCustomScripts)
	}

	// --- Marshal Final Result ---
	jsonData, err := json.MarshalIndent(result, "", "  ") // Use indent for readability
	if err != nil {
		agentInfo.Logger.Error("Failed to marshal scan result to JSON", zap.Error(err))
		// Return error here, as we can't form the output
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	agentInfo.Logger.Info("Internal scan command completed")
	output = string(jsonData)
	return output, nil // Overall command success, errors are in the JSON output
}

// gatherPlatformPackages routes to the OS-specific package gathering function.
func (c *ScanCommand) gatherPlatformPackages(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult) ([]InstalledPackage, error) {
	switch runtime.GOOS {
	case "linux":
		return gatherLinuxPackages(ctx, agentInfo, result)
	case "windows":
		return gatherWindowsPackages(ctx, agentInfo, result)
	case "darwin":
		return gatherMacOSPackages(ctx, agentInfo, result)
	default:
		msg := fmt.Sprintf("Package gathering not supported for OS: %s", runtime.GOOS)
		result.ScanErrors = append(result.ScanErrors, msg)
		return nil, fmt.Errorf(msg)
	}
}

// executeCustomScripts handles locating, reading, and executing custom scripts.
func (c *ScanCommand) executeCustomScripts(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult, scriptNames []string) {
	agentInfo.Logger.Debug("Entered executeCustomScripts", zap.Strings("scriptNames", scriptNames))
	scriptBaseDir := getScriptBaseDir() // Get base directory based on OS
	if scriptBaseDir == "" {
		result.ScanErrors = append(result.ScanErrors, fmt.Sprintf("Custom script execution not supported for OS: %s", runtime.GOOS))
		return
	}

	for _, scriptName := range scriptNames {
		if scriptName == "" {
			continue
		}
		scriptPath := filepath.Join(scriptBaseDir, scriptName)
		agentInfo.Logger.Debug("Processing custom script", zap.String("name", scriptName), zap.String("path", scriptPath))

		scriptContentBytes, err := os.ReadFile(scriptPath)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to read custom script file '%s': %v", scriptPath, err)
			agentInfo.Logger.Error("Custom script read error", zap.String("script", scriptName), zap.Error(err))
			result.ScanErrors = append(result.ScanErrors, errMsg)
			// Store error in results for this script
			result.CustomResults[scriptName] = CustomScanOutput{
				ScriptName: scriptName,
				Error:      errMsg,
				ExitCode:   -1, // Indicate agent-level error
			}
			continue // Move to next script
		}
		scriptContent := string(scriptContentBytes)

		var stdout, stderr string
		var exitCode int
		var execErr error

		switch runtime.GOOS {
		case "windows":
			if agentInfo.ScriptingEnabled {
				stdout, stderr, exitCode, execErr = shell.ExecuteScript(ctx, agentInfo.PowerShellPath, scriptContent)
			} else {
				execErr = errors.New("powershell scripting disabled")
				exitCode = -1
			}
		// TODO: Add Linux/macOS execution (e.g., using bash -c or sh -c)
		// case "linux":
		// 	 stdout, stderr, exitCode, execErr = executeBashScript(ctx, scriptContent)
		// case "darwin":
		// 	 stdout, stderr, exitCode, execErr = executeBashScript(ctx, scriptContent)
		default:
			execErr = fmt.Errorf("custom script execution not implemented for OS: %s", runtime.GOOS)
			exitCode = -1
		}

		// Store results
		customResult := CustomScanOutput{
			ScriptName: scriptName,
			StdOut:     stdout,
			StdErr:     stderr,
			ExitCode:   exitCode,
		}
		if execErr != nil {
			// Check if the error is just a non-zero exit code
			var exitErr *exec.ExitError
			if errors.As(execErr, &exitErr) {
				// It's just a non-zero exit, not an execution failure.
				// Log it but don't treat as a scan error or populate the Error field.
				agentInfo.Logger.Info("Custom script finished with non-zero exit code",
					zap.String("script", scriptName),
					zap.Int("exitCode", exitCode),
					zap.String("stderr", stderr))
			} else {
				// Different error occurred (e.g., process start failed)
				customResult.Error = execErr.Error()
				agentInfo.Logger.Error("Custom script execution error", zap.String("script", scriptName), zap.Error(execErr), zap.Int("exitCode", exitCode))
				// Also add to main scan errors for visibility
				result.ScanErrors = append(result.ScanErrors, fmt.Sprintf("Error executing custom script '%s': %v", scriptName, execErr))
			}
		} else {
			agentInfo.Logger.Info("Custom script executed successfully", zap.String("script", scriptName), zap.Int("exitCode", exitCode))
		}
		result.CustomResults[scriptName] = customResult
	}
}

// getScriptBaseDir returns the expected base directory for scan scripts based on OS.
func getScriptBaseDir() string {
	// Assuming scripts are deployed relative to the agent executable
	exePath, err := os.Executable()
	if err != nil {
		return "" // Cannot determine executable path
	}
	exeDir := filepath.Dir(exePath)

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(exeDir, "scripts", "powershell", "windows")
	case "linux":
		return filepath.Join(exeDir, "scripts", "bash", "linux") // Placeholder
	case "darwin":
		return filepath.Join(exeDir, "scripts", "bash", "macos") // Placeholder
	default:
		return ""
	}
}

// Placeholder functions for OS-specific implementations (to be moved to separate files)
/*
func gatherLinuxPackages(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult) ([]InstalledPackage, error) {
	// Implementation for Linux
	return nil, nil
}

func gatherWindowsPackages(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult) ([]InstalledPackage, error) {
	// Implementation for Windows
	return nil, nil
}

func gatherMacOSPackages(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult) ([]InstalledPackage, error) {
	// Implementation for macOS
	return nil, nil
}
*/
