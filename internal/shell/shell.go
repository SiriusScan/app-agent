package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	// Assuming logger can be passed or is global - for now, use fmt
)

// FindPowerShell attempts to find the path to the PowerShell executable.
// It checks the system PATH for 'pwsh' (Linux/macOS) or 'powershell.exe' (Windows).
// Returns the full path, a boolean indicating if found, and any error encountered during the search.
func FindPowerShell() (executablePath string, found bool, err error) {
	var psExecutable string
	if runtime.GOOS == "windows" {
		psExecutable = "powershell.exe"
	} else {
		// On Unix-like systems, PowerShell Core is typically named 'pwsh'
		psExecutable = "pwsh"
	}

	executablePath, err = exec.LookPath(psExecutable)
	if err != nil {
		// If LookPath fails (e.g., command not found), return false
		// We return the original error in case it's something else (e.g., permission issue)
		return "", false, err
	}

	return executablePath, true, nil
}

// ExecuteScript runs a given PowerShell script using the specified executable.
// It passes the script content via standard input to avoid command line injection issues.
// It captures and returns the standard output, standard error, the script's exit code,
// and any error encountered during execution.
func ExecuteScript(ctx context.Context, executablePath string, scriptContent string) (stdout string, stderr string, exitCode int, err error) {
	// Use flags for non-interactive execution, but allow profile loading
	cmdArgs := []string{"-NonInteractive", "-Command", "-"}
	fmt.Printf("[DEBUG] Executing PowerShell: %s %v\n", executablePath, cmdArgs) // Temporary Debug Print
	cmd := exec.CommandContext(ctx, executablePath, cmdArgs...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	stdinPipe, pipeErr := cmd.StdinPipe()
	if pipeErr != nil {
		err = fmt.Errorf("failed to get stdin pipe: %w", pipeErr)
		fmt.Printf("[ERROR] %v\n", err) // Temporary Debug Print
		return "", "", -1, err
	}

	// Write script async
	go func() {
		defer stdinPipe.Close()
		_, _ = io.WriteString(stdinPipe, scriptContent)
	}()

	fmt.Printf("[DEBUG] Running command for script: %s...\n", scriptContent[:min(len(scriptContent), 50)]) // Temporary Debug Print
	runErr := cmd.Run()

	rawStdout := stdoutBuf.String()
	rawStderr := stderrBuf.String()
	fmt.Printf("[DEBUG] Command Run Error: %v\n", runErr) // Temporary Debug Print
	fmt.Printf("[DEBUG] Raw Stdout: [%s]\n", rawStdout)   // Temporary Debug Print
	fmt.Printf("[DEBUG] Raw Stderr: [%s]\n", rawStderr)   // Temporary Debug Print

	stdout = rawStdout // Assign raw output first
	stderr = rawStderr

	if runErr != nil {
		err = runErr // Store the primary error
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Could be context cancelled, path error etc.
			exitCode = -1                                               // Indicate non-exit error
			err = fmt.Errorf("powershell cmd.Run() failed: %w", runErr) // Wrap for clarity
		}
		fmt.Printf("[DEBUG] Processed Run Error: ExitCode=%d, Err=%v\n", exitCode, err) // Temporary Debug Print
		// Return immediately if cmd.Run() failed, preserving stdout/stderr captured so far
		return stdout, stderr, exitCode, err
	}

	// Success
	exitCode = 0
	err = nil
	fmt.Printf("[DEBUG] Command completed successfully. ExitCode=0\n") // Temporary Debug Print
	return stdout, stderr, exitCode, err
}

// Helper function (add if not already present)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
