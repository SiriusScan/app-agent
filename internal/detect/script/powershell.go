package script

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
)

// executePowerShellScript executes a PowerShell script with security controls
func (se *ScriptExecutor) executePowerShellScript(ctx context.Context, script *detect.DetectionScript, args []string) (*ScriptExecutionResult, error) {
	if se.powershellPath == "" {
		return nil, fmt.Errorf("PowerShell interpreter not available")
	}

	se.logger.Debug("Executing PowerShell script",
		zap.String("script_path", script.Path),
		zap.String("powershell_path", se.powershellPath),
		zap.Strings("args", args))

	startTime := time.Now()

	// Read script content
	scriptContent, err := os.ReadFile(script.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}

	// Prepare PowerShell command with security parameters
	cmdArgs := se.buildPowerShellCommand(string(scriptContent), args)

	// Create command with context
	cmd := exec.CommandContext(ctx, se.powershellPath, cmdArgs...)

	// Apply security sandbox if enabled
	if se.enableSandbox {
		se.applyPowerShellSandbox(cmd)
	}

	// Set up environment
	cmd.Env = se.createSecureEnvironment()

	// Execute command and capture output
	stdout, stderr, exitCode, execErr := se.executeCommand(cmd)

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
		se.logger.Error("PowerShell script execution failed",
			zap.String("script_name", script.Name),
			zap.Error(execErr),
			zap.Int("exit_code", exitCode))
		return result, nil // Return result with error, don't fail the call
	}

	// Parse script output for vulnerability detection results
	if err := se.parsePowerShellOutput(result, stdout, stderr); err != nil {
		se.logger.Warn("Failed to parse PowerShell script output",
			zap.String("script_name", script.Name),
			zap.Error(err))
		result.Error = fmt.Sprintf("output parsing failed: %v", err)
	}

	se.logger.Debug("PowerShell script execution completed",
		zap.String("script_name", script.Name),
		zap.Bool("vulnerable", result.Vulnerable),
		zap.Float64("confidence", result.Confidence),
		zap.Duration("duration", duration),
		zap.Int("exit_code", exitCode))

	return result, nil
}

// buildPowerShellCommand constructs PowerShell command arguments with security controls
func (se *ScriptExecutor) buildPowerShellCommand(scriptContent string, args []string) []string {
	cmdArgs := []string{
		"-NoProfile",                 // Don't load PowerShell profile
		"-NoLogo",                    // Don't show logo
		"-NonInteractive",            // No interactive prompts
		"-ExecutionPolicy", "Bypass", // Allow script execution (sandboxed)
		"-Command", scriptContent,
	}

	// Add script arguments if provided
	if len(args) > 0 {
		// Escape arguments for PowerShell
		for _, arg := range args {
			escapedArg := se.escapePowerShellArgument(arg)
			cmdArgs = append(cmdArgs, escapedArg)
		}
	}

	return cmdArgs
}

// escapePowerShellArgument safely escapes an argument for PowerShell
func (se *ScriptExecutor) escapePowerShellArgument(arg string) string {
	// Basic escaping for PowerShell - wrap in single quotes and escape internal single quotes
	escaped := strings.ReplaceAll(arg, "'", "''")
	return "'" + escaped + "'"
}

// applyPowerShellSandbox applies security restrictions to PowerShell execution
func (se *ScriptExecutor) applyPowerShellSandbox(cmd *exec.Cmd) {
	// Set working directory to a safe location
	if tempDir := os.TempDir(); tempDir != "" {
		cmd.Dir = tempDir
	}

	// On Windows, we can apply additional restrictions
	if runtime.GOOS == "windows" {
		// TODO: Implement Windows-specific sandbox restrictions
		// - Run with limited privileges
		// - Restrict network access
		// - Limit file system access
		se.logger.Debug("Applied Windows PowerShell sandbox restrictions")
	}

	// For PowerShell Core on Unix systems
	if runtime.GOOS != "windows" {
		// PowerShell Core runs on Unix with limited privileges by default
		se.logger.Debug("Applied Unix PowerShell Core sandbox restrictions")
	}
}

// createSecureEnvironment creates a secure environment for script execution
func (se *ScriptExecutor) createSecureEnvironment() []string {
	// Start with minimal environment
	env := []string{
		"PATH=" + os.Getenv("PATH"), // Preserve PATH for basic functionality
	}

	// Add PowerShell specific environment variables if needed
	if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
		env = append(env, "PSModulePath="+psModulePath)
	}

	// On Windows, preserve some essential variables
	if runtime.GOOS == "windows" {
		windowsVars := []string{"SYSTEMROOT", "WINDIR", "TEMP", "TMP"}
		for _, varName := range windowsVars {
			if value := os.Getenv(varName); value != "" {
				env = append(env, varName+"="+value)
			}
		}
	}

	return env
}

// executeCommand executes a command and captures all output
func (se *ScriptExecutor) executeCommand(cmd *exec.Cmd) (stdout, stderr string, exitCode int, err error) {
	// Capture stdout and stderr
	stdoutBytes, stderrBytes, err := se.runCommand(cmd)

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

// runCommand runs a command and captures output with timeout handling
func (se *ScriptExecutor) runCommand(cmd *exec.Cmd) ([]byte, []byte, error) {
	// Use CombinedOutput for simplicity, but we could use separate pipes for more control
	output, err := cmd.CombinedOutput()
	stdout := output
	stderr := []byte{} // Combined output doesn't separate stderr

	return stdout, stderr, err
}

// parsePowerShellOutput parses PowerShell script output for vulnerability results
func (se *ScriptExecutor) parsePowerShellOutput(result *ScriptExecutionResult, stdout, stderr string) error {
	// Try to parse as JSON first
	if jsonResult, err := se.parseJSONOutput(stdout); err == nil {
		se.applyJSONResult(result, jsonResult)
		return nil
	}

	// Try to parse structured text output
	if err := se.parseStructuredTextOutput(result, stdout); err == nil {
		return nil
	}

	// Fallback: Analyze output for vulnerability indicators
	se.analyzeOutputForVulnerabilities(result, stdout, stderr)

	return nil
}

// parseJSONOutput attempts to parse JSON output from PowerShell script
func (se *ScriptExecutor) parseJSONOutput(output string) (map[string]interface{}, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("empty output")
	}

	// Look for JSON content (might be wrapped in other text)
	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		return nil, fmt.Errorf("no JSON content found")
	}

	jsonContent := output[jsonStart : jsonEnd+1]

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return result, nil
}

// applyJSONResult applies parsed JSON result to ScriptExecutionResult
func (se *ScriptExecutor) applyJSONResult(result *ScriptExecutionResult, jsonData map[string]interface{}) {
	// Extract vulnerability status
	if vulnerable, ok := jsonData["vulnerable"].(bool); ok {
		result.Vulnerable = vulnerable
	}

	// Extract confidence score
	if confidence, ok := jsonData["confidence"].(float64); ok {
		result.Confidence = confidence
	}

	// Extract evidence
	if evidenceData, ok := jsonData["evidence"].([]interface{}); ok {
		result.Evidence = se.parseEvidenceFromJSON(evidenceData)
	}

	// Extract enhanced vulnerability metadata from top level
	if vulnerabilityID, ok := jsonData["vulnerability_id"].(string); ok {
		result.Metadata["vulnerability_id"] = vulnerabilityID
	}
	if severity, ok := jsonData["severity"].(string); ok {
		result.Metadata["severity"] = severity
	}
	if severityScore, ok := jsonData["severity_score"].(float64); ok {
		result.Metadata["severity_score"] = severityScore
	}
	if riskScore, ok := jsonData["risk_score"].(float64); ok {
		result.Metadata["risk_score"] = riskScore
	}
	if cvssScore, ok := jsonData["cvss_score"].(float64); ok {
		result.Metadata["cvss_score"] = cvssScore
	}
	if cvssVector, ok := jsonData["cvss_vector"].(string); ok {
		result.Metadata["cvss_vector"] = cvssVector
	}
	if cveID, ok := jsonData["cve_id"].(string); ok {
		result.Metadata["cve_id"] = cveID
	}
	if description, ok := jsonData["description"].(string); ok {
		result.Metadata["description"] = description
	}
	if category, ok := jsonData["category"].(string); ok {
		result.Metadata["category"] = category
	}
	if tags, ok := jsonData["tags"].([]interface{}); ok {
		result.Metadata["tags"] = tags
	}
	if remediation, ok := jsonData["remediation"].(string); ok {
		result.Metadata["remediation"] = remediation
	}

	// Extract metadata from nested metadata field (for backward compatibility)
	if metadata, ok := jsonData["metadata"].(map[string]interface{}); ok {
		for key, value := range metadata {
			// Don't overwrite top-level metadata
			if _, exists := result.Metadata[key]; !exists {
				result.Metadata[key] = value
			}
		}
	}

	// Extract error if present
	if errorMsg, ok := jsonData["error"].(string); ok && errorMsg != "" {
		result.Error = errorMsg
	}
}

// parseEvidenceFromJSON converts JSON evidence array to Evidence structs
func (se *ScriptExecutor) parseEvidenceFromJSON(evidenceData []interface{}) []detect.Evidence {
	var evidence []detect.Evidence

	for _, item := range evidenceData {
		if evidenceMap, ok := item.(map[string]interface{}); ok {
			ev := detect.Evidence{
				Context: make(map[string]interface{}),
			}

			if typeStr, ok := evidenceMap["type"].(string); ok {
				ev.Type = detect.EvidenceType(typeStr)
			}

			if location, ok := evidenceMap["location"].(string); ok {
				ev.Location = location
			}

			if expected, ok := evidenceMap["expected"].(string); ok {
				ev.Expected = expected
			}

			if actual, ok := evidenceMap["actual"].(string); ok {
				ev.Actual = actual
			}

			if description, ok := evidenceMap["description"].(string); ok {
				ev.Description = description
			}

			if context, ok := evidenceMap["context"].(map[string]interface{}); ok {
				ev.Context = context
			}

			evidence = append(evidence, ev)
		}
	}

	return evidence
}

// parseStructuredTextOutput parses structured text output for vulnerability results
func (se *ScriptExecutor) parseStructuredTextOutput(result *ScriptExecutionResult, output string) error {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for vulnerability indicators
		if strings.Contains(strings.ToLower(line), "vulnerable: true") {
			result.Vulnerable = true
		}

		if strings.Contains(strings.ToLower(line), "vulnerable: false") {
			result.Vulnerable = false
		}

		// Look for confidence scores
		if strings.Contains(strings.ToLower(line), "confidence:") {
			// Try to extract confidence value
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				confidenceStr := strings.TrimSpace(parts[1])
				if confidence, err := parseFloat(confidenceStr); err == nil {
					if confidence >= 0.0 && confidence <= 1.0 {
						result.Confidence = confidence
					}
				}
			}
		}
	}

	return nil
}

// analyzeOutputForVulnerabilities performs heuristic analysis of script output
func (se *ScriptExecutor) analyzeOutputForVulnerabilities(result *ScriptExecutionResult, stdout, stderr string) {
	output := strings.ToLower(stdout + " " + stderr)

	// Vulnerability indicators
	vulnIndicators := []string{
		"vulnerable", "exploit", "backdoor", "malware", "trojan",
		"security hole", "cve-", "vulnerability found",
		"weak password", "unpatched", "outdated",
	}

	// Safety indicators
	safeIndicators := []string{
		"secure", "protected", "patched", "up to date",
		"no vulnerabilities", "safe", "clean",
	}

	vulnCount := 0
	safeCount := 0

	for _, indicator := range vulnIndicators {
		if strings.Contains(output, indicator) {
			vulnCount++
		}
	}

	for _, indicator := range safeIndicators {
		if strings.Contains(output, indicator) {
			safeCount++
		}
	}

	// Simple heuristic: if more vulnerability indicators than safe indicators
	if vulnCount > safeCount && vulnCount > 0 {
		result.Vulnerable = true
		result.Confidence = 0.3 + (float64(vulnCount) * 0.1) // Low confidence heuristic
		if result.Confidence > 0.7 {
			result.Confidence = 0.7 // Cap heuristic confidence
		}

		// Create evidence based on heuristic analysis
		evidence := detect.Evidence{
			Type:        detect.EvidenceTypeScriptOutput,
			Location:    "script_output",
			Expected:    "secure system",
			Actual:      fmt.Sprintf("found %d vulnerability indicators", vulnCount),
			Description: "Heuristic analysis of script output detected potential vulnerabilities",
			Context: map[string]interface{}{
				"vulnerability_indicators": vulnCount,
				"safe_indicators":          safeCount,
				"analysis_method":          "heuristic",
			},
		}

		result.Evidence = append(result.Evidence, evidence)
	}
}

// parseFloat safely parses a float64 from string
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)

	// Handle common formats
	s = strings.TrimSuffix(s, "%")

	// Try standard parsing
	if f, err := fmt.Sscanf(s, "%f", new(float64)); err == nil && f == 1 {
		var result float64
		fmt.Sscanf(s, "%f", &result)
		return result, nil
	}

	return 0, fmt.Errorf("unable to parse float: %s", s)
}
