package scan

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/SiriusScan/app-agent/internal/commands"
	"go.uber.org/zap"
)

var errBrewNotFound = errors.New("homebrew (brew) command not found")

// gatherMacOSPackages collects installed software information on macOS.
// Currently, it primarily targets packages installed via Homebrew.
func gatherMacOSPackages(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult) ([]InstalledPackage, error) {
	agentInfo.Logger.Info("Gathering macOS packages (via Homebrew)")

	brewPath, err := exec.LookPath("brew")
	if err != nil {
		msg := "Homebrew (brew) command not found in PATH, cannot list brew packages."
		agentInfo.Logger.Warn(msg)
		result.ScanErrors = append(result.ScanErrors, msg)
		// Don't return an error for the whole scan if brew isn't installed
		// Consider adding system_profiler lookup here later
		return make([]InstalledPackage, 0), nil
	}
	agentInfo.Logger.Info("Found Homebrew", zap.String("path", brewPath))

	cmd := exec.CommandContext(ctx, brewPath, "list", "--versions")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("brew list failed (Stderr: %s): %v", stderr.String(), err)
		agentInfo.Logger.Error("Homebrew package gathering failed", zap.Error(err), zap.String("stderr", stderr.String()))
		result.ScanErrors = append(result.ScanErrors, errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	packages := make([]InstalledPackage, 0)
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Output format is typically "packageName version1 version2 ..."
		parts := strings.Fields(line) // Split by whitespace
		if len(parts) >= 2 {
			name := parts[0]
			version := parts[len(parts)-1] // Assume last part is the primary/latest version
			packages = append(packages, InstalledPackage{
				Name:    name,
				Version: version,
				Source:  "brew",
			})
		} else if len(parts) == 1 {
			// Handle case where version might not be present (less common with --versions)
			packages = append(packages, InstalledPackage{
				Name:    parts[0],
				Version: "(unknown)",
				Source:  "brew",
			})
		}
	}

	if err := scanner.Err(); err != nil {
		errMsg := fmt.Sprintf("error scanning brew output: %v", err)
		agentInfo.Logger.Error("Failed to scan brew output", zap.Error(err))
		result.ScanErrors = append(result.ScanErrors, errMsg)
		// Return packages found so far, but report the scanning error
		return packages, nil
	}

	agentInfo.Logger.Info("Successfully gathered macOS packages via Homebrew", zap.Int("count", len(packages)))
	return packages, nil
}
