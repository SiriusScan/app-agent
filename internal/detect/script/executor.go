package script

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"go.uber.org/zap"
)

// ScriptExecutor implements the ScriptEngine interface with security controls
type ScriptExecutor struct {
	logger           *zap.Logger
	powershellPath   string
	bashPath         string
	pythonPath       string
	scriptDirs       []string
	loadedScripts    map[string]*detect.DetectionScript
	scriptsMutex     sync.RWMutex
	maxConcurrency   int
	defaultTimeout   time.Duration
	enableSandbox    bool
	allowedPlatforms []string
}

// NewScriptExecutor creates a new script executor instance
func NewScriptExecutor(logger *zap.Logger, scriptDirs []string) *ScriptExecutor {
	if logger == nil {
		logger = zap.NewNop()
	}

	executor := &ScriptExecutor{
		logger:           logger,
		scriptDirs:       scriptDirs,
		loadedScripts:    make(map[string]*detect.DetectionScript),
		maxConcurrency:   3, // Conservative default for security
		defaultTimeout:   30 * time.Second,
		enableSandbox:    true, // Always enable sandbox by default
		allowedPlatforms: []string{"windows", "linux", "darwin"},
	}

	// Detect available script interpreters
	executor.detectInterpreters()

	return executor
}

// detectInterpreters locates available script interpreters on the system
func (se *ScriptExecutor) detectInterpreters() {
	// Detect PowerShell
	powershellPaths := []string{
		"pwsh",           // PowerShell Core
		"powershell",     // Windows PowerShell
		"powershell.exe", // Windows PowerShell with extension
	}

	for _, path := range powershellPaths {
		if se.commandExists(path) {
			se.powershellPath = path
			se.logger.Info("PowerShell interpreter found", zap.String("path", path))
			break
		}
	}

	// Detect Bash
	bashPaths := []string{
		"bash",
		"/bin/bash",
		"/usr/bin/bash",
	}

	for _, path := range bashPaths {
		if se.commandExists(path) {
			se.bashPath = path
			se.logger.Info("Bash interpreter found", zap.String("path", path))
			break
		}
	}

	// Detect Python
	pythonPaths := []string{
		"python3",
		"python",
		"/usr/bin/python3",
		"/usr/bin/python",
	}

	for _, path := range pythonPaths {
		if se.commandExists(path) {
			se.pythonPath = path
			se.logger.Info("Python interpreter found", zap.String("path", path))
			break
		}
	}
}

// commandExists checks if a command is available in the system PATH
func (se *ScriptExecutor) commandExists(cmd string) bool {
	// Handle absolute paths
	if filepath.IsAbs(cmd) {
		_, err := os.Stat(cmd)
		return err == nil
	}

	// Check in PATH
	_, err := os.Stat(cmd)
	if err == nil {
		return true
	}

	// Try with common extensions on Windows
	if runtime.GOOS == "windows" {
		for _, ext := range []string{".exe", ".bat", ".cmd"} {
			if _, err := os.Stat(cmd + ext); err == nil {
				return true
			}
		}
	}

	return false
}

// SetConcurrency sets the maximum number of concurrent script executions
func (se *ScriptExecutor) SetConcurrency(maxConcurrency int) {
	if maxConcurrency > 0 && maxConcurrency <= 10 { // Safety limit
		se.maxConcurrency = maxConcurrency
	}
}

// SetTimeout sets the default script execution timeout
func (se *ScriptExecutor) SetTimeout(timeout time.Duration) {
	if timeout > 0 && timeout <= 5*time.Minute { // Safety limit
		se.defaultTimeout = timeout
	}
}

// SetSandboxEnabled controls whether script sandboxing is enabled
func (se *ScriptExecutor) SetSandboxEnabled(enabled bool) {
	se.enableSandbox = enabled
	se.logger.Info("Script sandboxing configuration updated", zap.Bool("enabled", enabled))
}

// LoadScripts discovers and loads all detection scripts from configured directories
func (se *ScriptExecutor) LoadScripts(ctx context.Context) error {
	se.logger.Info("Loading detection scripts", zap.Strings("script_dirs", se.scriptDirs))

	se.scriptsMutex.Lock()
	defer se.scriptsMutex.Unlock()

	// Clear previously loaded scripts
	se.loadedScripts = make(map[string]*detect.DetectionScript)

	totalLoaded := 0
	totalErrors := 0

	for _, scriptDir := range se.scriptDirs {
		if _, err := os.Stat(scriptDir); os.IsNotExist(err) {
			se.logger.Warn("Script directory does not exist", zap.String("dir", scriptDir))
			continue
		}

		err := filepath.Walk(scriptDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				se.logger.Error("Error walking script directory", zap.String("path", path), zap.Error(err))
				return nil // Continue walking
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Check file extension to determine script type
			if se.isScriptFile(path) {
				script, loadErr := se.loadScript(path)
				if loadErr != nil {
					se.logger.Error("Failed to load script", zap.String("path", path), zap.Error(loadErr))
					totalErrors++
					return nil // Continue loading other scripts
				}

				se.loadedScripts[script.Name] = script
				totalLoaded++
				se.logger.Debug("Script loaded successfully",
					zap.String("name", script.Name),
					zap.String("path", script.Path),
					zap.String("language", string(script.Language)))
			}

			return nil
		})

		if err != nil {
			se.logger.Error("Failed to walk script directory", zap.String("dir", scriptDir), zap.Error(err))
			totalErrors++
		}
	}

	se.logger.Info("Script loading completed",
		zap.Int("loaded", totalLoaded),
		zap.Int("errors", totalErrors),
		zap.Int("total_scripts", len(se.loadedScripts)))

	return nil
}

// isScriptFile determines if a file is a recognized script type
func (se *ScriptExecutor) isScriptFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	scriptExtensions := map[string]detect.ScriptLanguage{
		".ps1": detect.ScriptLanguagePowerShell,
		".sh":  detect.ScriptLanguageBash,
		".py":  detect.ScriptLanguagePython,
	}

	_, isScript := scriptExtensions[ext]
	return isScript
}

// loadScript loads and validates a single script file
func (se *ScriptExecutor) loadScript(scriptPath string) (*detect.DetectionScript, error) {
	// Read script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}

	// Calculate checksum
	hasher := sha256.New()
	hasher.Write(content)
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// Determine script language
	ext := strings.ToLower(filepath.Ext(scriptPath))
	var language detect.ScriptLanguage
	switch ext {
	case ".ps1":
		language = detect.ScriptLanguagePowerShell
	case ".sh":
		language = detect.ScriptLanguageBash
	case ".py":
		language = detect.ScriptLanguagePython
	default:
		return nil, fmt.Errorf("unsupported script extension: %s", ext)
	}

	// Extract metadata from script content
	metadata, err := se.extractScriptMetadata(content, language)
	if err != nil {
		se.logger.Warn("Failed to extract script metadata, using defaults",
			zap.String("script", scriptPath), zap.Error(err))
		metadata = &ScriptMetadata{
			Name:        filepath.Base(scriptPath),
			Description: "Auto-detected script",
			Author:      "unknown",
			Version:     "1.0",
		}
	}

	script := &detect.DetectionScript{
		Name:               metadata.Name,
		Path:               scriptPath,
		Platform:           se.determinePlatform(language),
		Language:           language,
		VulnerabilityID:    metadata.VulnerabilityID,
		Severity:           metadata.Severity,
		Description:        metadata.Description,
		Author:             metadata.Author,
		Version:            metadata.Version,
		Timeout:            se.getScriptTimeout(metadata.Timeout),
		RequiredPrivileges: metadata.RequiredPrivileges,
		Parameters:         metadata.Parameters,
		Checksum:           checksum,
		LoadedAt:           time.Now(),
	}

	return script, nil
}

// determinePlatform determines the target platform for a script language
func (se *ScriptExecutor) determinePlatform(language detect.ScriptLanguage) string {
	switch language {
	case detect.ScriptLanguagePowerShell:
		return "windows" // PowerShell is primarily for Windows
	case detect.ScriptLanguageBash:
		return "linux" // Bash is primarily for Unix-like systems
	case detect.ScriptLanguagePython:
		return "cross-platform" // Python works everywhere
	default:
		return runtime.GOOS
	}
}

// getScriptTimeout determines the timeout for a script
func (se *ScriptExecutor) getScriptTimeout(metadataTimeout time.Duration) time.Duration {
	if metadataTimeout > 0 && metadataTimeout <= 5*time.Minute {
		return metadataTimeout
	}
	return se.defaultTimeout
}

// ExecuteScript runs a detection script with security sandbox
func (se *ScriptExecutor) ExecuteScript(ctx context.Context, script *detect.DetectionScript, args []string) (*detect.DetectionResult, error) {
	se.logger.Info("Executing detection script",
		zap.String("script_name", script.Name),
		zap.String("language", string(script.Language)),
		zap.Strings("args", args))

	startTime := time.Now()

	// Create detection result
	result := &detect.DetectionResult{
		DetectionID:     generateDetectionID(script.Name),
		Method:          detect.DetectionMethodScript,
		SourceID:        script.Name,
		VulnerabilityID: script.VulnerabilityID,
		Vulnerable:      false,
		Confidence:      0.0,
		Severity:        script.Severity,
		Evidence:        []detect.Evidence{},
		Metadata:        make(map[string]interface{}),
		ExecutedAt:      startTime,
	}

	// Validate script before execution
	if err := se.ValidateScript(script); err != nil {
		result.Error = fmt.Sprintf("script validation failed: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result, nil // Return result with error, don't fail the call
	}

	// Create execution context with timeout
	timeout := script.Timeout
	if timeout == 0 {
		timeout = se.defaultTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute script based on language
	var scriptResult *ScriptExecutionResult
	var err error

	switch script.Language {
	case detect.ScriptLanguagePowerShell:
		scriptResult, err = se.executePowerShellScript(execCtx, script, args)
	case detect.ScriptLanguageBash:
		scriptResult, err = se.executeBashScript(execCtx, script, args)
	case detect.ScriptLanguagePython:
		scriptResult, err = se.executePythonScript(execCtx, script, args)
	default:
		err = fmt.Errorf("unsupported script language: %s", script.Language)
	}

	if err != nil {
		result.Error = err.Error()
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	// Process script execution results
	if scriptResult != nil {
		result.Vulnerable = scriptResult.Vulnerable
		result.Confidence = scriptResult.Confidence
		result.Evidence = scriptResult.Evidence

		// Store execution metadata
		result.Metadata["exit_code"] = scriptResult.ExitCode
		result.Metadata["stdout_length"] = len(scriptResult.Stdout)
		result.Metadata["stderr_length"] = len(scriptResult.Stderr)
		result.Metadata["execution_duration"] = scriptResult.Duration
		result.Metadata["script_version"] = script.Version
		result.Metadata["script_author"] = script.Author

		// Transfer enhanced vulnerability metadata from script output
		for key, value := range scriptResult.Metadata {
			result.Metadata[key] = value
		}

		if scriptResult.Error != "" {
			result.Error = scriptResult.Error
		}
	}

	result.ExecutionTime = time.Since(startTime)

	se.logger.Info("Script execution completed",
		zap.String("script_name", script.Name),
		zap.Bool("vulnerable", result.Vulnerable),
		zap.Float64("confidence", result.Confidence),
		zap.Duration("execution_time", result.ExecutionTime),
		zap.String("error", result.Error))

	return result, nil
}

// ValidateScript checks script permissions and safety
func (se *ScriptExecutor) ValidateScript(script *detect.DetectionScript) error {
	// Check if script file exists
	if _, err := os.Stat(script.Path); os.IsNotExist(err) {
		return fmt.Errorf("script file does not exist: %s", script.Path)
	}

	// Check if script is readable
	file, err := os.Open(script.Path)
	if err != nil {
		return fmt.Errorf("cannot read script file: %w", err)
	}
	file.Close()

	// Validate script platform compatibility
	if !se.isPlatformSupported(script.Platform) {
		return fmt.Errorf("script platform '%s' not supported on %s", script.Platform, runtime.GOOS)
	}

	// Check if required interpreter is available
	switch script.Language {
	case detect.ScriptLanguagePowerShell:
		if se.powershellPath == "" {
			return fmt.Errorf("PowerShell interpreter not available")
		}
	case detect.ScriptLanguageBash:
		if se.bashPath == "" {
			return fmt.Errorf("Bash interpreter not available")
		}
	case detect.ScriptLanguagePython:
		if se.pythonPath == "" {
			return fmt.Errorf("Python interpreter not available")
		}
	}

	// Validate script checksum if provided
	if script.Checksum != "" {
		content, err := os.ReadFile(script.Path)
		if err != nil {
			return fmt.Errorf("failed to read script for checksum validation: %w", err)
		}

		hasher := sha256.New()
		hasher.Write(content)
		actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

		if actualChecksum != script.Checksum {
			return fmt.Errorf("script checksum mismatch: expected %s, got %s", script.Checksum, actualChecksum)
		}
	}

	return nil
}

// isPlatformSupported checks if the script platform is supported
func (se *ScriptExecutor) isPlatformSupported(platform string) bool {
	if platform == "cross-platform" {
		return true
	}

	currentPlatform := runtime.GOOS

	// Handle platform aliases
	if platform == "unix" && (currentPlatform == "linux" || currentPlatform == "darwin") {
		return true
	}

	return platform == currentPlatform
}

// ListScripts returns all loaded scripts
func (se *ScriptExecutor) ListScripts() []*detect.DetectionScript {
	se.scriptsMutex.RLock()
	defer se.scriptsMutex.RUnlock()

	scripts := make([]*detect.DetectionScript, 0, len(se.loadedScripts))
	for _, script := range se.loadedScripts {
		scripts = append(scripts, script)
	}

	return scripts
}

// GetScript retrieves a specific script by name
func (se *ScriptExecutor) GetScript(name string) (*detect.DetectionScript, bool) {
	se.scriptsMutex.RLock()
	defer se.scriptsMutex.RUnlock()

	script, exists := se.loadedScripts[name]
	return script, exists
}

// GetScriptsByPlatform returns scripts for a specific platform
func (se *ScriptExecutor) GetScriptsByPlatform(platform string) []*detect.DetectionScript {
	se.scriptsMutex.RLock()
	defer se.scriptsMutex.RUnlock()

	var scripts []*detect.DetectionScript
	for _, script := range se.loadedScripts {
		if script.Platform == platform || script.Platform == "cross-platform" {
			scripts = append(scripts, script)
		}
	}

	return scripts
}

// generateDetectionID creates a unique ID for detection execution
func generateDetectionID(scriptName string) string {
	return fmt.Sprintf("script-%s-%d", scriptName, time.Now().Unix())
}

// ScriptExecutionResult represents the result of script execution
type ScriptExecutionResult struct {
	Vulnerable bool                   `json:"vulnerable"`
	Confidence float64                `json:"confidence"`
	Evidence   []detect.Evidence      `json:"evidence"`
	Metadata   map[string]interface{} `json:"metadata"`
	Stdout     string                 `json:"stdout"`
	Stderr     string                 `json:"stderr"`
	ExitCode   int                    `json:"exit_code"`
	Duration   time.Duration          `json:"duration"`
	Error      string                 `json:"error,omitempty"`
}
