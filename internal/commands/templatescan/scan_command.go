package templatescan

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/commands/scan" // For package enumeration
	_ "github.com/SiriusScan/app-agent/internal/modules/filecontent" // Register module
	_ "github.com/SiriusScan/app-agent/internal/modules/filehash"    // Register module
	"github.com/SiriusScan/app-agent/internal/template/executor"
	"github.com/SiriusScan/app-agent/internal/template/fingerprint"
	"github.com/SiriusScan/app-agent/internal/template/parser"
	"github.com/SiriusScan/app-agent/internal/template/reporting"
	"github.com/SiriusScan/app-agent/internal/template/storage"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// ScanCommand implements the template scanning command for server integration
type ScanCommand struct{}

// Ensure ScanCommand implements the Command interface at compile time
var _ commands.Command = (*ScanCommand)(nil)

func init() {
	// Register the command with the registry
	commands.Register("internal:template-scan", &ScanCommand{})
}

// Execute runs the template scan command
func (c *ScanCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (output string, err error) {
	agentInfo.Logger.Info("Executing template scan command",
		zap.String("args", args))

	// Parse arguments
	config, err := c.parseArgs(args)
	if err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Discover templates
	var templates []*types.Template
	var discoveryErrors []error

	if config.TemplatePath != "" {
		// Single template (explicit file path)
		template, err := parser.ParseTemplate(config.TemplatePath)
		if err != nil {
			return "", fmt.Errorf("failed to parse template %q: %w", config.TemplatePath, err)
		}
		if err := parser.ValidateTemplate(template); err != nil {
			return "", fmt.Errorf("template %q is invalid: %w", config.TemplatePath, err)
		}
		templates = []*types.Template{template}

		agentInfo.Logger.Info("Running single template",
			zap.String("path", config.TemplatePath))

	} else if config.Directory != "" {
		// Specific directory (bypasses manager, uses direct path)
		agentInfo.Logger.Info("Discovering templates from directory",
			zap.String("directory", config.Directory))

		templates, discoveryErrors = parser.DiscoverTemplatesWithContext(ctx, config.Directory)

		if len(templates) == 0 {
			errMsg := fmt.Sprintf("no valid templates found in %q", config.Directory)
			if len(discoveryErrors) > 0 {
				errMsg += fmt.Sprintf(" (%d discovery errors)", len(discoveryErrors))
				for i, err := range discoveryErrors {
					if i < 3 { // Show first 3 errors
						errMsg += fmt.Sprintf("\n  - %v", err)
					}
				}
				if len(discoveryErrors) > 3 {
					errMsg += fmt.Sprintf("\n  ... and %d more errors", len(discoveryErrors)-3)
				}
			}
			return "", fmt.Errorf(errMsg)
		}

	} else {
		// Use template manager (discovers from all sources: custom > server > builtin)
		agentInfo.Logger.Info("Using template manager for discovery")

		manager, err := storage.NewManager(agentInfo.Logger)
		if err != nil {
			return "", fmt.Errorf("failed to initialize template manager: %w", err)
		}

		templates, err = manager.DiscoverTemplates(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to discover templates: %w", err)
		}

		if len(templates) == 0 {
			return "", fmt.Errorf("no templates available. Try installing templates in %s or use --directory", manager.GetStoragePath())
		}

		agentInfo.Logger.Info("Discovered templates from manager",
			zap.Int("count", len(templates)),
			zap.String("base_dir", manager.GetStoragePath()))
	}

	// Execute templates with worker pool
	startTime := time.Now()

	poolConfig := executor.DefaultWorkerPoolConfig()
	poolConfig.Context = ctx
	poolConfig.Workers = config.Workers
	if config.TimeoutSeconds > 0 {
		poolConfig.PerTemplateTimeout = time.Duration(config.TimeoutSeconds) * time.Second
	}

	agentInfo.Logger.Info("Executing templates",
		zap.Int("count", len(templates)),
		zap.Int("workers", config.Workers))

	results, execErrors := executor.ExecuteTemplatesParallelWithConfig(templates, poolConfig)
	executionTime := time.Since(startTime)

	// Submit to REST API if enabled and we have matched results
	if shouldSubmitToAPI(agentInfo, results) {
		agentInfo.Logger.Info("Submitting template results to REST API")
		go submitTemplateResultsToAPI(ctx, agentInfo, results, executionTime)
	}

	// Build output
	return c.generateOutput(templates, results, discoveryErrors, execErrors, executionTime, config.Format)
}

// parseArgs parses command arguments
func (c *ScanCommand) parseArgs(args string) (*ScanConfig, error) {
	config := &ScanConfig{
		Format:         "json",
		Workers:        runtime.NumCPU(),
		TimeoutSeconds: 300, // 5 minutes default
	}

	if args == "" {
		return config, nil
	}

	parts := strings.Fields(args)
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "--directory", "-d":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("--directory requires a value")
			}
			i++
			config.Directory = parts[i]

		case "--template", "-t":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("--template requires a value")
			}
			i++
			config.TemplatePath = parts[i]

		case "--all":
			// --all flag: use template manager (leave Directory empty)
			// This triggers the template manager code path which discovers
			// templates from all sources: custom > server > builtin
			// Do nothing - keep config.Directory empty

		case "--workers", "-w":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("--workers requires a value")
			}
			i++
			workers, err := strconv.Atoi(parts[i])
			if err != nil {
				return nil, fmt.Errorf("invalid workers value: %w", err)
			}
			if err := executor.ValidateWorkerCount(workers); err != nil {
				return nil, err
			}
			config.Workers = workers

		case "--timeout":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("--timeout requires a value")
			}
			i++
			timeout, err := strconv.Atoi(parts[i])
			if err != nil {
				return nil, fmt.Errorf("invalid timeout value: %w", err)
			}
			config.TimeoutSeconds = timeout

		case "--format", "-f":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("--format requires a value")
			}
			i++
			format := parts[i]
			if format != "json" && format != "text" {
				return nil, fmt.Errorf("invalid format: %s (must be 'json' or 'text')", format)
			}
			config.Format = format

		default:
			return nil, fmt.Errorf("unknown argument: %s", parts[i])
		}
	}

	// Validate that template and directory are not both set
	if config.TemplatePath != "" && config.Directory != "" {
		return nil, fmt.Errorf("cannot specify both --template and --directory")
	}

	return config, nil
}

// generateOutput generates command output in the requested format
func (c *ScanCommand) generateOutput(
	templates []*types.Template,
	results []*types.Result,
	discoveryErrors []error,
	execErrors []error,
	executionTime time.Duration,
	format string,
) (string, error) {
	if format == "text" {
		return c.generateTextOutput(templates, results, discoveryErrors, execErrors, executionTime)
	}
	return c.generateJSONOutput(templates, results, discoveryErrors, execErrors, executionTime)
}

// generateJSONOutput generates JSON formatted output
func (c *ScanCommand) generateJSONOutput(
	templates []*types.Template,
	results []*types.Result,
	discoveryErrors []error,
	execErrors []error,
	executionTime time.Duration,
) (string, error) {
	matchedCount := 0
	for _, result := range results {
		if result != nil && result.Matched {
			matchedCount++
		}
	}

	// Convert errors to strings
	var discoveryErrorStrings []string
	for _, err := range discoveryErrors {
		discoveryErrorStrings = append(discoveryErrorStrings, err.Error())
	}
	var execErrorStrings []string
	for _, err := range execErrors {
		execErrorStrings = append(execErrorStrings, err.Error())
	}

	output := struct {
		Summary struct {
			TotalTemplates  int   `json:"total_templates"`
			Matched         int   `json:"matched"`
			ExecutionTimeMs int64 `json:"execution_time_ms"`
			Workers         int   `json:"workers"`
		} `json:"summary"`
		Results         []*types.Result `json:"results"`
		DiscoveryErrors []string        `json:"discovery_errors,omitempty"`
		ExecutionErrors []string        `json:"execution_errors,omitempty"`
	}{}

	output.Summary.TotalTemplates = len(templates)
	output.Summary.Matched = matchedCount
	output.Summary.ExecutionTimeMs = executionTime.Milliseconds()
	output.Summary.Workers = runtime.NumCPU()
	output.Results = results
	output.DiscoveryErrors = discoveryErrorStrings
	output.ExecutionErrors = execErrorStrings

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), nil
}

// generateTextOutput generates human-readable text output
func (c *ScanCommand) generateTextOutput(
	templates []*types.Template,
	results []*types.Result,
	discoveryErrors []error,
	execErrors []error,
	executionTime time.Duration,
) (string, error) {
	var output strings.Builder

	output.WriteString("🔍 Template Scan Results\n")
	output.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Summary
	matchedCount := 0
	for _, result := range results {
		if result != nil && result.Matched {
			matchedCount++
		}
	}

	output.WriteString(fmt.Sprintf("📊 Summary:\n"))
	output.WriteString(fmt.Sprintf("  Total Templates: %d\n", len(templates)))
	output.WriteString(fmt.Sprintf("  Matched: %d\n", matchedCount))
	output.WriteString(fmt.Sprintf("  Execution Time: %v\n\n", executionTime.Round(time.Millisecond)))

	// Discovery Errors
	if len(discoveryErrors) > 0 {
		output.WriteString(fmt.Sprintf("⚠️  Discovery Errors (%d):\n", len(discoveryErrors)))
		for i, err := range discoveryErrors {
			output.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
		}
		output.WriteString("\n")
	}

	// Execution Errors
	if len(execErrors) > 0 {
		output.WriteString(fmt.Sprintf("⚠️  Execution Errors (%d):\n", len(execErrors)))
		for i, err := range execErrors {
			output.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
		}
		output.WriteString("\n")
	}

	// Results
	output.WriteString("📋 Template Results:\n\n")
	for i, result := range results {
		if result == nil {
			output.WriteString(fmt.Sprintf("[%d] ❓ Unknown result\n", i+1))
			continue
		}

		status := "❌"
		if result.Matched {
			status = "✅"
		}

		output.WriteString(fmt.Sprintf("[%d] %s %s (ID: %s)\n", i+1, status, result.TemplateName, result.TemplateID))
		output.WriteString(fmt.Sprintf("    Severity: %s | Confidence: %.2f\n", result.Severity, result.Confidence))

		if result.Matched && len(result.Steps) > 0 {
			output.WriteString(fmt.Sprintf("    Matched Steps: %d/%d\n", countMatchedSteps(result), len(result.Steps)))
		}

		if len(result.Errors) > 0 {
			output.WriteString(fmt.Sprintf("    Errors: %s\n", strings.Join(result.Errors, ", ")))
		}
		output.WriteString("\n")
	}

	output.WriteString(strings.Repeat("=", 50) + "\n")
	if matchedCount > 0 {
		output.WriteString("⚠️  Vulnerabilities detected!\n")
	} else {
		output.WriteString("✅ No vulnerabilities detected.\n")
	}

	return output.String(), nil
}

// countMatchedSteps counts how many steps matched in a result
func countMatchedSteps(result *types.Result) int {
	count := 0
	for _, step := range result.Steps {
		if step.Matched {
			count++
		}
	}
	return count
}

// ScanConfig holds configuration for the scan command
type ScanConfig struct {
	Directory      string
	TemplatePath   string
	Workers        int
	TimeoutSeconds int
	Format         string
}

// shouldSubmitToAPI checks if we should submit results to the REST API
func shouldSubmitToAPI(agentInfo commands.AgentInfo, results []*types.Result) bool {
	// Don't submit if API client is not available
	if agentInfo.APIClient == nil {
		return false
	}

	// Don't submit if API base URL is not configured
	if agentInfo.Config.ApiBaseURL == "" {
		return false
	}

	// Don't submit if no results
	if len(results) == 0 {
		return false
	}

	// Only submit if at least one template matched
	for _, result := range results {
		if result != nil && result.Matched {
			return true
		}
	}

	return false
}

// submitTemplateResultsToAPI submits template scan results to the REST API
// This runs asynchronously so it doesn't block the command response
func submitTemplateResultsToAPI(
	ctx context.Context,
	agentInfo commands.AgentInfo,
	results []*types.Result,
	executionTime time.Duration,
) {
	startTime := time.Now()

	// 1. Collect host fingerprint
	fp, err := fingerprint.CollectBasicFingerprint(ctx, agentInfo.Config)
	if err != nil {
		agentInfo.Logger.Warn("Failed to collect host fingerprint, using partial data",
			zap.Error(err))
		// Continue with partial fingerprint (never nil)
	}

	agentInfo.Logger.Debug("Collected host fingerprint",
		zap.String("hostname", fp.Hostname),
		zap.String("os", fp.OS),
		zap.String("os_version", fp.OSVersion),
		zap.String("ip", fp.PrimaryIP))

	// 2. Optional: Collect software packages (enhances reporting)
	// This reuses the existing scan package code for package enumeration
	var packages []scan.InstalledPackage
	if agentInfo.ScriptingEnabled || runtime.GOOS != "windows" {
		agentInfo.Logger.Debug("Collecting software packages for enhanced reporting")
		
		// Create a minimal ScanResult for package gathering
		scanResult := &scan.ScanResult{
			ScanErrors: make([]string, 0),
		}
		
		// Use the platform-specific package gathering
		switch runtime.GOOS {
		case "linux":
			packages, _ = scan.GatherLinuxPackages(ctx, agentInfo, scanResult)
		case "darwin":
			packages, _ = scan.GatherMacOSPackages(ctx, agentInfo, scanResult)
		case "windows":
			if agentInfo.ScriptingEnabled {
				packages, _ = scan.GatherWindowsPackages(ctx, agentInfo, scanResult)
			}
		}
		
		if len(packages) > 0 {
			agentInfo.Logger.Debug("Collected software packages",
				zap.Int("package_count", len(packages)))
		}
	}

	// 3. Convert template results to vulnerabilities
	vulns := reporting.ConvertTemplateResultsToVulnerabilities(results)
	agentInfo.Logger.Debug("Converted template results to vulnerabilities",
		zap.Int("vulnerability_count", len(vulns)))

	// 4. Build sirius.Host data
	hostData := reporting.BuildHostData(fp, vulns)

	// 5. Add software inventory if we collected packages
	// Note: This would go into the enhanced API call if we had it
	// For now, we'll just include it in agent_metadata
	agentMetadata := reporting.BuildAgentMetadata(results, executionTime)
	if len(packages) > 0 {
		agentMetadata["package_count"] = len(packages)
		agentMetadata["has_software_inventory"] = true
	}

	// 6. Submit to API
	apiCtx := context.Background() // Use background context for async call
	err = agentInfo.APIClient.UpdateHostRecord(apiCtx, agentInfo.Config.ApiBaseURL, hostData)

	submissionTime := time.Since(startTime)

	if err != nil {
		agentInfo.Logger.Error("Failed to submit template results to API",
			zap.Error(err),
			zap.Duration("submission_time", submissionTime),
			zap.Int("vulnerabilities", len(vulns)))
	} else {
		agentInfo.Logger.Info("Successfully submitted template results to API",
			zap.Int("vulnerabilities", len(vulns)),
			zap.Duration("submission_time", submissionTime),
			zap.String("host_id", agentInfo.Config.HostID),
			zap.String("agent_id", agentInfo.Config.AgentID))
	}
}
