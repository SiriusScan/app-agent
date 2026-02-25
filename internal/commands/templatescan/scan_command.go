package templatescan

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/SiriusScan/app-agent/internal/apiclient"
	"github.com/SiriusScan/app-agent/internal/commands"
	"github.com/SiriusScan/app-agent/internal/commands/scan" // For package enumeration
	_ "github.com/SiriusScan/app-agent/internal/modules/filecontent" // Register module
	_ "github.com/SiriusScan/app-agent/internal/modules/filehash"    // Register module
	"github.com/SiriusScan/app-agent/internal/output"
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
func (c *ScanCommand) Execute(ctx context.Context, agentInfo commands.AgentInfo, commandString string, args string) (string, error) {
	agentInfo.Logger.Info("Executing template scan command",
		zap.String("args", args))

	// Parse arguments
	config, err := c.parseArgs(args)
	if err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if config.ScanID != "" {
		agentInfo.Logger.Info("Coordinated scan mode",
			zap.String("scan_id", config.ScanID))
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
			return "", fmt.Errorf("%s", errMsg)
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

	// Build output using the output package
	outputStr, err := c.generateOutput(results, discoveryErrors, execErrors, executionTime, config)
	if err != nil {
		return "", err
	}

	// If this is a coordinated scan, wrap the output with scan metadata
	// so the agent server can correlate results to the unified scan
	if config.ScanID != "" {
		// Count matched results for the metadata
		matchedCount := 0
		for _, r := range results {
			if r != nil && r.Matched {
				matchedCount++
			}
		}

		agentInfo.Logger.Info("Coordinated scan complete",
			zap.String("scan_id", config.ScanID),
			zap.Int("matched_templates", matchedCount),
			zap.Int("total_templates", len(results)))
	}

	return outputStr, nil
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
			if !output.IsValidFormat(format) {
				return nil, fmt.Errorf("invalid format: %s (valid: %s)", format, strings.Join(output.AvailableStrings(), ", "))
			}
			config.Format = format

		case "--templates":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("--templates requires a value")
			}
			i++
			// Template filter: comma-separated template names (not used in config yet, reserved for future)

		default:
			// Check for --key=value format
			if strings.HasPrefix(parts[i], "--scan-id=") {
				config.ScanID = strings.TrimPrefix(parts[i], "--scan-id=")
			} else if strings.HasPrefix(parts[i], "--workers=") {
				w, err := strconv.Atoi(strings.TrimPrefix(parts[i], "--workers="))
				if err != nil {
					return nil, fmt.Errorf("invalid workers value: %w", err)
				}
				config.Workers = w
			} else if strings.HasPrefix(parts[i], "--timeout=") {
				t, err := strconv.Atoi(strings.TrimPrefix(parts[i], "--timeout="))
				if err != nil {
					return nil, fmt.Errorf("invalid timeout value: %w", err)
				}
				config.TimeoutSeconds = t
			} else if strings.HasPrefix(parts[i], "--templates=") {
				// Template filter: comma-separated template names (reserved for future)
			} else {
				return nil, fmt.Errorf("unknown argument: %s", parts[i])
			}
		}
	}

	// Validate that template and directory are not both set
	if config.TemplatePath != "" && config.Directory != "" {
		return nil, fmt.Errorf("cannot specify both --template and --directory")
	}

	return config, nil
}

// generateOutput generates command output using the output package formatters
func (c *ScanCommand) generateOutput(
	results []*types.Result,
	discoveryErrors []error,
	execErrors []error,
	executionTime time.Duration,
	config *ScanConfig,
) (string, error) {
	// Get the appropriate formatter
	formatter, err := output.GetByString(config.Format)
	if err != nil {
		// Fall back to JSON if format not found
		formatter = output.MustGet(output.FormatJSON)
	}

	// Create summary
	summary := output.NewScanSummary(results, executionTime, config.Workers)

	// Format output
	return formatter.FormatScanResults(results, summary)
}

// ScanConfig holds configuration for the scan command
type ScanConfig struct {
	Directory      string
	TemplatePath   string
	Workers        int
	TimeoutSeconds int
	Format         string
	ScanID         string // Coordinated scan ID for unified scan results
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

	// Don't submit if service API key is not configured
	if !apiclient.ServiceAPIKeyConfigured() {
		agentInfo.Logger.Warn("Skipping API submission: service API key is not configured",
			zap.String("api_base_url", agentInfo.Config.ApiBaseURL),
			zap.Strings("accepted_env_vars", apiclient.ServiceAPIKeyEnvNames()))
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

	// 5. Build software inventory (if we have packages)
	var softwareInventory map[string]interface{}
	if len(packages) > 0 {
		// Convert packages to the expected format
		packageList := make([]map[string]interface{}, 0, len(packages))
		for _, pkg := range packages {
			packageList = append(packageList, map[string]interface{}{
				"name":    pkg.Name,
				"version": pkg.Version,
				"source":  pkg.Source,
			})
		}

		softwareInventory = map[string]interface{}{
			"packages":      packageList,
			"package_count": len(packages),
			"collected_at":  time.Now().Format(time.RFC3339),
			"source":        "sirius-agent",
		}

		agentInfo.Logger.Debug("Built software inventory",
			zap.Int("package_count", len(packages)))
	}

	// 6. Build agent metadata
	agentMetadata := reporting.BuildAgentMetadata(results, executionTime)
	if len(packages) > 0 {
		agentMetadata["package_count"] = len(packages)
		agentMetadata["has_software_inventory"] = true
	}

	// 7. Submit to API with enhanced data
	apiCtx := context.Background() // Use background context for async call

	// Use enhanced API if we have JSONB data to send
	if len(softwareInventory) > 0 || len(agentMetadata) > 0 {
		err = agentInfo.APIClient.UpdateHostRecordWithEnhancedData(
			apiCtx,
			agentInfo.Config.ApiBaseURL,
			hostData,
			softwareInventory,
			nil, // systemFingerprint (not collected yet)
			agentMetadata,
		)
	} else {
		// Fallback to basic API
		err = agentInfo.APIClient.UpdateHostRecord(apiCtx, agentInfo.Config.ApiBaseURL, hostData)
	}

	submissionTime := time.Since(startTime)

	if err != nil {
		agentInfo.Logger.Error("Failed to submit template results to API",
			zap.Error(err),
			zap.Duration("submission_time", submissionTime),
			zap.Int("vulnerabilities", len(vulns)),
			zap.Int("packages", len(packages)))
	} else {
		agentInfo.Logger.Info("Successfully submitted template results to API",
			zap.Int("vulnerabilities", len(vulns)),
			zap.Int("packages", len(packages)),
			zap.Bool("enhanced_data", len(softwareInventory) > 0 || len(agentMetadata) > 0),
			zap.Duration("submission_time", submissionTime),
			zap.String("host_id", agentInfo.Config.HostID),
			zap.String("agent_id", agentInfo.Config.AgentID))
	}
}
