// Package versioncmd implements a module for extracting version information from command output.
package versioncmd

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
)

// CommandVersionModule implements the Module interface for version detection via command execution.
type CommandVersionModule struct{}

// Execute runs a command, captures output, and extracts version information.
//
// Config fields:
//   - command ([]string, required): Command to execute as array [cmd, arg1, arg2, ...]
//   - regex (string, required): Regular expression to extract version from output
//   - exit_code (int, optional): Expected exit code. If not specified, any exit code is accepted.
func (m *CommandVersionModule) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	// Create a context with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Extract configuration
	commandArray, regex, expectedExitCode, err := m.parseConfig(config)
	if err != nil {
		return &modules.Result{
			Matched: false,
			Error:   err.Error(),
		}, nil
	}

	// Validate command array
	if len(commandArray) == 0 {
		return &modules.Result{
			Matched: false,
			Error:   "command array is empty",
		}, nil
	}

	// Execute command - SECURITY: Use exec.Command directly, NO shell interpretation
	cmdName := commandArray[0]
	cmdArgs := commandArray[1:]
	cmd := exec.CommandContext(execCtx, cmdName, cmdArgs...)

	// Capture stdout and stderr
	stdout, err := cmd.Output()
	var stderr []byte
	if exitErr, ok := err.(*exec.ExitError); ok {
		stderr = exitErr.Stderr
	}

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			return &modules.Result{
				Matched: false,
				Error:   "command execution timed out",
			}, nil
		} else {
			// Command not found or other execution error
			return &modules.Result{
				Matched: false,
				Error:   fmt.Sprintf("command execution failed: %v", err),
			}, nil
		}
	}

	// Verify exit code if specified
	if expectedExitCode != nil && exitCode != *expectedExitCode {
		evidence := map[string]interface{}{
			"command":       commandArray,
			"exit_code":     exitCode,
			"expected_code": *expectedExitCode,
			"stdout":        string(stdout),
			"stderr":        string(stderr),
		}
		return &modules.Result{
			Matched:  false,
			Evidence: evidence,
		}, nil
	}

	// Combine stdout and stderr for regex matching
	output := string(stdout) + string(stderr)

	// Apply regex to extract version
	re, err := regexp.Compile(regex)
	if err != nil {
		return &modules.Result{
			Matched: false,
			Error:   fmt.Sprintf("invalid regex pattern: %v", err),
		}, nil
	}

	matches := re.FindStringSubmatch(output)
	var extractedVersion string
	matched := false

	if len(matches) > 0 {
		matched = true
		if len(matches) > 1 {
			// Use first capture group if available
			extractedVersion = matches[1]
		} else {
			// Use entire match if no capture groups
			extractedVersion = matches[0]
		}
	}

	// Build evidence
	evidence := map[string]interface{}{
		"command":          commandArray,
		"exit_code":        exitCode,
		"stdout":           string(stdout),
		"stderr":           string(stderr),
		"regex":            regex,
		"matched_version":  extractedVersion,
		"full_output_size": len(output),
	}

	return &modules.Result{
		Matched:  matched,
		Evidence: evidence,
	}, nil
}

// parseConfig extracts configuration values from the config map.
func (m *CommandVersionModule) parseConfig(config modules.StepConfig) ([]string, string, *int, error) {
	// Extract command (required, array)
	commandArray := config.GetStringSlice("command")
	if len(commandArray) == 0 {
		return nil, "", nil, fmt.Errorf("missing required field: command")
	}

	// Extract regex (required, string)
	regex := config.GetString("regex")
	if regex == "" {
		return nil, "", nil, fmt.Errorf("missing required field: regex")
	}

	// Extract exit_code (optional, int)
	var expectedExitCode *int
	if exitCodeRaw, ok := config["exit_code"]; ok {
		switch v := exitCodeRaw.(type) {
		case int:
			expectedExitCode = &v
		case float64:
			// JSON unmarshaling often produces float64 for numbers
			code := int(v)
			expectedExitCode = &code
		default:
			return nil, "", nil, fmt.Errorf("exit_code must be an integer")
		}
	}

	return commandArray, regex, expectedExitCode, nil
}

// init registers the CommandVersion module in the global registry
func init() {
	descriptor := modules.Descriptor{
		Type:        "version_cmd",
		Name:        "Command Version Extractor",
		Description: "Executes a command and extracts version information using regex",
		Version:     "1.0.0",
		Author:      "Sirius Scan",
		SupportedOS: []string{"linux", "darwin", "windows"},
		ConfigDocs: map[string]string{
			"command":   "Command to execute (array of strings: [\"command\", \"arg1\", \"arg2\"])",
			"regex":     "Regular expression to extract version from output",
			"exit_code": "Expected exit code (optional, any exit code accepted if not specified)",
		},
	}

	// Register module
	if err := registry.Register(&CommandVersionModule{}, descriptor); err != nil {
		panic(fmt.Sprintf("failed to register version_cmd module: %v", err))
	}
}
