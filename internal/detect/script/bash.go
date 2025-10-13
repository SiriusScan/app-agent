package script

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
)

// executeBashScript executes a Bash script with security controls
func (se *ScriptExecutor) executeBashScript(ctx context.Context, script *detect.DetectionScript, args []string) (*ScriptExecutionResult, error) {
	if se.bashPath == "" {
		return nil, fmt.Errorf("Bash interpreter not available")
	}

	se.logger.Debug("Executing Bash script",
		zap.String("script_path", script.Path),
		zap.String("bash_path", se.bashPath),
		zap.Strings("args", args))

	startTime := time.Now()

	// Prepare Bash command with security parameters
	cmdArgs := se.buildBashCommand(script.Path, args)

	// Create command with context
	cmd := exec.CommandContext(ctx, se.bashPath, cmdArgs...)

	// Apply security sandbox if enabled
	if se.enableSandbox {
		se.applyBashSandbox(cmd)
	}

	// Set up secure environment
	cmd.Env = se.createSecureBashEnvironment()

	// Execute command and capture output
	stdout, stderr, exitCode, execErr := se.executeBashCommand(cmd)

	duration := time.Since(startTime)

	result := &ScriptExecutionResult{
		Vulnerable: false,
		Confidence: 0.0,
		Evidence:   []detect.Evidence{},
		Metadata:   make(map[string]interface{}),
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		Duration:   duration,
	}

	if execErr != nil {
		result.Error = execErr.Error()
		se.logger.Error("Bash script execution failed",
			zap.String("script_name", script.Name),
			zap.Error(execErr),
			zap.Int("exit_code", exitCode))
		return result, nil // Return result with error, don't fail the call
	}

	// Parse script output for vulnerability detection results
	if err := se.parseBashOutput(result, stdout, stderr); err != nil {
		se.logger.Warn("Failed to parse Bash script output",
			zap.String("script_name", script.Name),
			zap.Error(err))
		result.Error = fmt.Sprintf("output parsing failed: %v", err)
	}

	se.logger.Debug("Bash script execution completed",
		zap.String("script_name", script.Name),
		zap.Bool("vulnerable", result.Vulnerable),
		zap.Float64("confidence", result.Confidence),
		zap.Duration("duration", duration),
		zap.Int("exit_code", exitCode))

	return result, nil
}

// buildBashCommand constructs Bash command arguments with security controls
func (se *ScriptExecutor) buildBashCommand(scriptPath string, args []string) []string {
	cmdArgs := []string{
		"-e",             // Exit on error
		"-u",             // Treat unset variables as error
		"-o", "pipefail", // Fail if any command in pipe fails
		scriptPath,
	}

	// Add script arguments if provided
	if len(args) > 0 {
		// Escape arguments for shell
		for _, arg := range args {
			escapedArg := se.escapeBashArgument(arg)
			cmdArgs = append(cmdArgs, escapedArg)
		}
	}

	return cmdArgs
}

// escapeBashArgument safely escapes an argument for Bash
func (se *ScriptExecutor) escapeBashArgument(arg string) string {
	// Basic escaping for Bash - wrap in single quotes and escape internal single quotes
	escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// applyBashSandbox applies security restrictions to Bash execution
func (se *ScriptExecutor) applyBashSandbox(cmd *exec.Cmd) {
	// Set working directory to a safe location
	if tempDir := os.TempDir(); tempDir != "" {
		cmd.Dir = tempDir
	}

	// Apply Unix-specific security restrictions
	if runtime.GOOS != "windows" {
		// Set process group for signal handling
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create new process group
		}

		// TODO: Implement additional Unix sandbox restrictions
		// - Use seccomp to restrict system calls
		// - Drop privileges if running as root
		// - Restrict file system access with chroot
		// - Set resource limits (ulimit)

		se.logger.Debug("Applied Unix Bash sandbox restrictions")
	}
}

// createSecureBashEnvironment creates a secure environment for Bash script execution
func (se *ScriptExecutor) createSecureBashEnvironment() []string {
	// Start with minimal environment
	env := []string{
		"PATH=/usr/bin:/bin:/usr/sbin:/sbin", // Restricted PATH
		"HOME=" + os.TempDir(),               // Safe home directory
		"SHELL=" + se.bashPath,               // Explicit shell
	}

	// Add essential Unix environment variables
	if runtime.GOOS != "windows" {
		unixVars := []string{"USER", "LOGNAME", "TERM"}
		for _, varName := range unixVars {
			if value := os.Getenv(varName); value != "" {
				env = append(env, varName+"="+value)
			}
		}

		// Add safe defaults if not set
		if os.Getenv("USER") == "" {
			env = append(env, "USER=sirius-agent")
		}
		if os.Getenv("TERM") == "" {
			env = append(env, "TERM=xterm")
		}
	}

	// Block potentially dangerous environment variables
	// (IFS, PS1, PS2, etc. are not included to prevent shell injection)

	return env
}

// executeBashCommand executes a Bash command and captures all output
func (se *ScriptExecutor) executeBashCommand(cmd *exec.Cmd) (stdout, stderr string, exitCode int, err error) {
	// Create pipes for separate stdout and stderr capture
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", "", -1, fmt.Errorf("failed to start command: %w", err)
	}

	// Read output from pipes
	stdoutBytes := make([]byte, 0, 4096)
	stderrBytes := make([]byte, 0, 4096)

	// Read stdout
	stdoutBuffer := make([]byte, 1024)
	for {
		n, err := stdoutPipe.Read(stdoutBuffer)
		if n > 0 {
			stdoutBytes = append(stdoutBytes, stdoutBuffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Read stderr
	stderrBuffer := make([]byte, 1024)
	for {
		n, err := stderrPipe.Read(stderrBuffer)
		if n > 0 {
			stderrBytes = append(stderrBytes, stderrBuffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Wait for command to complete
	err = cmd.Wait()

	stdout = string(stdoutBytes)
	stderr = string(stderrBytes)

	// Get exit code
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1 // Unknown error
		}
	} else {
		exitCode = 0
	}

	return stdout, stderr, exitCode, err
}

// parseBashOutput parses Bash script output for vulnerability results
func (se *ScriptExecutor) parseBashOutput(result *ScriptExecutionResult, stdout, stderr string) error {
	// Try to parse as JSON first
	if jsonResult, err := se.parseJSONOutput(stdout); err == nil {
		se.applyJSONResult(result, jsonResult)
		return nil
	}

	// Try to parse structured text output
	if err := se.parseStructuredTextOutput(result, stdout); err == nil {
		return nil
	}

	// Try to parse Bash-specific output formats
	if err := se.parseBashSpecificOutput(result, stdout, stderr); err == nil {
		return nil
	}

	// Fallback: Analyze output for vulnerability indicators
	se.analyzeOutputForVulnerabilities(result, stdout, stderr)

	return nil
}

// parseBashSpecificOutput parses Bash-specific output formats
func (se *ScriptExecutor) parseBashSpecificOutput(result *ScriptExecutionResult, stdout, stderr string) error {
	lines := strings.Split(stdout, "\n")
	found := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for common Bash output patterns
		if strings.HasPrefix(line, "VULNERABLE=") {
			value := strings.TrimPrefix(line, "VULNERABLE=")
			if value == "true" || value == "1" || value == "yes" {
				result.Vulnerable = true
				found = true
			} else if value == "false" || value == "0" || value == "no" {
				result.Vulnerable = false
				found = true
			}
		}

		if strings.HasPrefix(line, "CONFIDENCE=") {
			value := strings.TrimPrefix(line, "CONFIDENCE=")
			if confidence, err := strconv.ParseFloat(value, 64); err == nil {
				if confidence >= 0.0 && confidence <= 1.0 {
					result.Confidence = confidence
					found = true
				}
			}
		}

		if strings.HasPrefix(line, "EVIDENCE=") {
			evidenceStr := strings.TrimPrefix(line, "EVIDENCE=")
			if evidence := se.parseBashEvidence(evidenceStr); evidence != nil {
				result.Evidence = append(result.Evidence, *evidence)
				found = true
			}
		}

		// Look for exit codes that indicate vulnerabilities
		if strings.HasPrefix(line, "EXIT_CODE=") {
			value := strings.TrimPrefix(line, "EXIT_CODE=")
			if exitCode, err := strconv.Atoi(value); err == nil {
				// Convention: exit code 2 = vulnerable, 0 = safe, 1 = error
				if exitCode == 2 {
					result.Vulnerable = true
					result.Confidence = 0.8 // High confidence for explicit exit code
					found = true
				}
			}
		}
	}

	if found {
		return nil
	}

	return fmt.Errorf("no Bash-specific output patterns found")
}

// parseBashEvidence parses evidence from Bash script output
func (se *ScriptExecutor) parseBashEvidence(evidenceStr string) *detect.Evidence {
	// Try to parse as JSON
	var evidenceData map[string]interface{}
	if err := json.Unmarshal([]byte(evidenceStr), &evidenceData); err == nil {
		evidence := detect.Evidence{
			Context: make(map[string]interface{}),
		}

		if typeStr, ok := evidenceData["type"].(string); ok {
			evidence.Type = detect.EvidenceType(typeStr)
		} else {
			evidence.Type = detect.EvidenceTypeScriptOutput
		}

		if location, ok := evidenceData["location"].(string); ok {
			evidence.Location = location
		}

		if expected, ok := evidenceData["expected"].(string); ok {
			evidence.Expected = expected
		}

		if actual, ok := evidenceData["actual"].(string); ok {
			evidence.Actual = actual
		}

		if description, ok := evidenceData["description"].(string); ok {
			evidence.Description = description
		}

		if context, ok := evidenceData["context"].(map[string]interface{}); ok {
			evidence.Context = context
		}

		return &evidence
	}

	// Parse simple key=value format
	parts := strings.SplitN(evidenceStr, ":", 2)
	if len(parts) == 2 {
		evidence := detect.Evidence{
			Type:        detect.EvidenceTypeScriptOutput,
			Location:    strings.TrimSpace(parts[0]),
			Description: strings.TrimSpace(parts[1]),
			Context:     make(map[string]interface{}),
		}

		return &evidence
	}

	return nil
}

// executePythonScript executes a Python script with security controls
func (se *ScriptExecutor) executePythonScript(ctx context.Context, script *detect.DetectionScript, args []string) (*ScriptExecutionResult, error) {
	if se.pythonPath == "" {
		return nil, fmt.Errorf("Python interpreter not available")
	}

	se.logger.Debug("Executing Python script",
		zap.String("script_path", script.Path),
		zap.String("python_path", se.pythonPath),
		zap.Strings("args", args))

	startTime := time.Now()

	// Prepare Python command
	cmdArgs := []string{script.Path}
	cmdArgs = append(cmdArgs, args...) // Python handles arguments natively

	// Create command with context
	cmd := exec.CommandContext(ctx, se.pythonPath, cmdArgs...)

	// Apply security sandbox if enabled
	if se.enableSandbox {
		se.applyPythonSandbox(cmd)
	}

	// Set up secure environment
	cmd.Env = se.createSecurePythonEnvironment()

	// Execute command and capture output
	stdout, stderr, exitCode, execErr := se.executeBashCommand(cmd) // Reuse Bash execution logic

	duration := time.Since(startTime)

	result := &ScriptExecutionResult{
		Vulnerable: false,
		Confidence: 0.0,
		Evidence:   []detect.Evidence{},
		Metadata:   make(map[string]interface{}),
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		Duration:   duration,
	}

	if execErr != nil {
		result.Error = execErr.Error()
		se.logger.Error("Python script execution failed",
			zap.String("script_name", script.Name),
			zap.Error(execErr),
			zap.Int("exit_code", exitCode))
		return result, nil
	}

	// Parse script output (reuse existing parsing logic)
	if err := se.parseBashOutput(result, stdout, stderr); err != nil {
		se.logger.Warn("Failed to parse Python script output",
			zap.String("script_name", script.Name),
			zap.Error(err))
		result.Error = fmt.Sprintf("output parsing failed: %v", err)
	}

	se.logger.Debug("Python script execution completed",
		zap.String("script_name", script.Name),
		zap.Bool("vulnerable", result.Vulnerable),
		zap.Float64("confidence", result.Confidence),
		zap.Duration("duration", duration),
		zap.Int("exit_code", exitCode))

	return result, nil
}

// applyPythonSandbox applies security restrictions to Python execution
func (se *ScriptExecutor) applyPythonSandbox(cmd *exec.Cmd) {
	// Set working directory to a safe location
	if tempDir := os.TempDir(); tempDir != "" {
		cmd.Dir = tempDir
	}

	// Apply similar restrictions as Bash
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create new process group
		}
		se.logger.Debug("Applied Unix Python sandbox restrictions")
	}
}

// createSecurePythonEnvironment creates a secure environment for Python script execution
func (se *ScriptExecutor) createSecurePythonEnvironment() []string {
	// Start with minimal environment
	env := []string{
		"PATH=/usr/bin:/bin:/usr/sbin:/sbin", // Restricted PATH
		"HOME=" + os.TempDir(),               // Safe home directory
		"PYTHONDONTWRITEBYTECODE=1",          // Don't create .pyc files
		"PYTHONUNBUFFERED=1",                 // Unbuffered output
	}

	// Add essential environment variables
	if runtime.GOOS != "windows" {
		unixVars := []string{"USER", "LOGNAME", "TERM"}
		for _, varName := range unixVars {
			if value := os.Getenv("USER"); value != "" {
				env = append(env, varName+"="+value)
			}
		}
	}

	return env
}
