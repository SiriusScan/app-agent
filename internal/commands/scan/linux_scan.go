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

var errPackageManagerNotFound = errors.New("no supported package manager (dpkg or rpm) found")

// gatherLinuxPackages collects installed software information on Linux.
// It detects dpkg or rpm and queries the installed packages.
func gatherLinuxPackages(ctx context.Context, agentInfo commands.AgentInfo, result *ScanResult) ([]InstalledPackage, error) {
	agentInfo.Logger.Info("Gathering Linux packages")
	var pkgs []InstalledPackage
	var err error
	var source string

	// Check for dpkg
	dpkgPath, dpkgErr := exec.LookPath("dpkg")
	if dpkgErr == nil {
		agentInfo.Logger.Info("Found dpkg package manager", zap.String("path", dpkgPath))
		pkgs, err = runDpkgQuery(ctx, agentInfo)
		source = "dpkg"
	} else {
		// Check for rpm
		rpmPath, rpmErr := exec.LookPath("rpm")
		if rpmErr == nil {
			agentInfo.Logger.Info("Found rpm package manager", zap.String("path", rpmPath))
			pkgs, err = runRpmQuery(ctx, agentInfo)
			source = "rpm"
		} else {
			// No supported manager found
			err = errPackageManagerNotFound
			agentInfo.Logger.Error("Could not find dpkg or rpm", zap.Error(dpkgErr), zap.Error(rpmErr))
		}
	}

	if err != nil {
		errMsg := fmt.Sprintf("Failed to query packages using %s: %v", source, err)
		agentInfo.Logger.Error("Linux package gathering failed", zap.Error(err))
		result.ScanErrors = append(result.ScanErrors, errMsg)
		return nil, err // Return the specific error
	}

	// Add source information
	for i := range pkgs {
		pkgs[i].Source = source
	}

	agentInfo.Logger.Info("Successfully gathered Linux packages", zap.Int("count", len(pkgs)), zap.String("source", source))
	return pkgs, nil
}

// runDpkgQuery executes dpkg-query to list installed packages.
func runDpkgQuery(ctx context.Context, agentInfo commands.AgentInfo) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Package}\t${Version}\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("dpkg-query failed (Stderr: %s): %w", stderr.String(), err)
	}

	return parseTabSeparatedPackages(stdout.String())
}

// runRpmQuery executes rpm to list installed packages.
func runRpmQuery(ctx context.Context, agentInfo commands.AgentInfo) ([]InstalledPackage, error) {
	cmd := exec.CommandContext(ctx, "rpm", "-qa", "--qf", "%{NAME}\t%{VERSION}\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rpm query failed (Stderr: %s): %w", stderr.String(), err)
	}

	return parseTabSeparatedPackages(stdout.String())
}

// parseTabSeparatedPackages parses the common output format from dpkg/rpm queries.
func parseTabSeparatedPackages(output string) ([]InstalledPackage, error) {
	packages := make([]InstalledPackage, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			version := strings.TrimSpace(parts[1])
			if name != "" && version != "" {
				packages = append(packages, InstalledPackage{
					Name:    name,
					Version: version,
					// Source added by caller
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning package output: %w", err)
	}
	return packages, nil
}
