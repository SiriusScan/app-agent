package template

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
	"github.com/SiriusScan/app-agent/internal/detect/config"
	"github.com/SiriusScan/app-agent/internal/detect/hash"
	"github.com/SiriusScan/app-agent/internal/detect/registry"
	"go.uber.org/zap"
)

// TemplateExecutor handles template execution and result aggregation
type TemplateExecutor struct {
	logger         *zap.Logger
	parser         *TemplateParser
	hashCalculator *hash.HashCalculator
	registryReader *registry.RegistryReader
	templateDirs   []string
	maxConcurrency int
	timeout        time.Duration
}

// NewTemplateExecutor creates a new template executor instance
func NewTemplateExecutor(logger *zap.Logger, templateDirs []string) *TemplateExecutor {
	if logger == nil {
		logger = zap.NewNop()
	}

	parser := NewTemplateParser(logger, templateDirs)
	hashCalculator := hash.NewHashCalculator(logger)
	registryReader := registry.NewRegistryReader(logger)

	return &TemplateExecutor{
		logger:         logger,
		parser:         parser,
		hashCalculator: hashCalculator,
		registryReader: registryReader,
		templateDirs:   templateDirs,
		maxConcurrency: 5,                // Default concurrency limit
		timeout:        30 * time.Second, // Default timeout
	}
}

// SetConcurrency sets the maximum number of concurrent template executions
func (te *TemplateExecutor) SetConcurrency(maxConcurrency int) {
	te.maxConcurrency = maxConcurrency
}

// SetTimeout sets the maximum execution time for template processing
func (te *TemplateExecutor) SetTimeout(timeout time.Duration) {
	te.timeout = timeout
}

// LoadTemplates loads all templates from configured directories
func (te *TemplateExecutor) LoadTemplates(ctx context.Context) ([]*detect.VulnTemplate, error) {
	te.logger.Info("Loading vulnerability templates",
		zap.Strings("template_dirs", te.templateDirs))

	templates, err := te.parser.LoadTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	te.logger.Info("Templates loaded successfully",
		zap.Int("total_templates", len(templates)))

	return templates, nil
}

// ExecuteTemplate runs vulnerability detection based on a single template
func (te *TemplateExecutor) ExecuteTemplate(ctx context.Context, template *detect.VulnTemplate) (*detect.DetectionResult, error) {
	te.logger.Debug("Executing template",
		zap.String("template_id", template.ID),
		zap.String("template_name", template.Info.Name),
		zap.String("detection_type", string(template.Detection.Type)))

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, te.timeout)
	defer cancel()

	startTime := time.Now()

	// Create detection result
	result := &detect.DetectionResult{
		DetectionID:     generateDetectionID(template.ID),
		Method:          detect.DetectionMethodTemplate,
		SourceID:        template.ID,
		VulnerabilityID: template.Info.CVE,
		Vulnerable:      false,
		Confidence:      0.0,
		Severity:        template.Info.Severity,
		Evidence:        []detect.Evidence{},
		Metadata:        make(map[string]interface{}),
		ExecutedAt:      startTime,
	}

	// Check if template applies to current platform
	if !te.isTemplateApplicableToCurrentPlatform(template) {
		result.Metadata["skipped_reason"] = "template not applicable to current platform"
		result.ExecutionTime = time.Since(startTime)
		te.logger.Debug("Template skipped - not applicable to platform",
			zap.String("template_id", template.ID),
			zap.String("platform", runtime.GOOS))
		return result, nil
	}

	// Execute based on detection type
	switch template.Detection.Type {
	case detect.DetectionTypeFileHash:
		err := te.executeFileHashDetection(execCtx, template, result)
		if err != nil {
			result.Error = err.Error()
		}

	case detect.DetectionTypeRegistry:
		err := te.executeRegistryDetection(execCtx, template, result)
		if err != nil {
			result.Error = err.Error()
		}

	case detect.DetectionTypeConfigFile:
		err := te.executeConfigFileDetection(execCtx, template, result)
		if err != nil {
			result.Error = err.Error()
		}

	default:
		result.Error = fmt.Sprintf("unsupported detection type: %s", template.Detection.Type)
	}

	// Calculate execution time
	result.ExecutionTime = time.Since(startTime)

	// Evaluate conditions and determine vulnerability
	if result.Error == "" {
		te.evaluateConditions(template, result)
	}

	te.logger.Debug("Template execution completed",
		zap.String("template_id", template.ID),
		zap.Bool("vulnerable", result.Vulnerable),
		zap.Float64("confidence", result.Confidence),
		zap.Duration("execution_time", result.ExecutionTime),
		zap.String("error", result.Error))

	return result, nil
}

// ExecuteTemplates runs multiple templates with concurrency control
func (te *TemplateExecutor) ExecuteTemplates(ctx context.Context, templates []*detect.VulnTemplate) ([]*detect.DetectionResult, error) {
	te.logger.Info("Starting batch template execution",
		zap.Int("templates", len(templates)),
		zap.Int("max_concurrency", te.maxConcurrency))

	// Filter templates for current platform
	platformTemplates := te.parser.FilterTemplatesByPlatform(templates, runtime.GOOS)

	te.logger.Info("Templates filtered by platform",
		zap.Int("total_templates", len(templates)),
		zap.Int("applicable_templates", len(platformTemplates)),
		zap.String("platform", runtime.GOOS))

	if len(platformTemplates) == 0 {
		te.logger.Warn("No templates applicable to current platform")
		return []*detect.DetectionResult{}, nil
	}

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, te.maxConcurrency)
	results := make([]*detect.DetectionResult, len(platformTemplates))
	errors := make([]error, len(platformTemplates))
	done := make(chan bool, len(platformTemplates))

	// Execute templates concurrently
	for i, template := range platformTemplates {
		go func(idx int, tmpl *detect.VulnTemplate) {
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
				done <- true
			}()

			// Execute template
			result, err := te.ExecuteTemplate(ctx, tmpl)
			results[idx] = result
			errors[idx] = err
		}(i, template)
	}

	// Wait for all executions to complete
	for i := 0; i < len(platformTemplates); i++ {
		<-done
	}

	// Check for execution errors
	var executionErrors []string
	for i, err := range errors {
		if err != nil {
			executionErrors = append(executionErrors,
				fmt.Sprintf("template %s: %v", platformTemplates[i].ID, err))
		}
	}

	if len(executionErrors) > 0 {
		te.logger.Warn("Some templates had execution errors",
			zap.Strings("errors", executionErrors))
	}

	te.logger.Info("Batch template execution completed",
		zap.Int("executed", len(results)),
		zap.Int("errors", len(executionErrors)))

	return results, nil
}

// executeFileHashDetection performs file hash-based vulnerability detection
func (te *TemplateExecutor) executeFileHashDetection(ctx context.Context, template *detect.VulnTemplate, result *detect.DetectionResult) error {
	te.logger.Debug("Executing file hash detection", zap.String("template_id", template.ID))

	// Get hash algorithm from template
	algorithm := detect.HashAlgorithmSHA256 // Default
	if template.Detection.Method != "" {
		algorithm = detect.HashAlgorithm(template.Detection.Method)
	}

	// Convert targets to hash targets
	var hashTargets []detect.HashTarget
	for _, target := range template.Detection.Targets {
		// Skip if not applicable to current platform
		if !te.isTargetApplicableToCurrentPlatform(target) {
			continue
		}

		hashTargets = append(hashTargets, detect.HashTarget{
			Path:            target.Path,
			ExpectedHash:    target.Hash,
			Algorithm:       algorithm,
			Description:     target.Description,
			VulnerabilityID: template.Info.CVE,
		})
	}

	if len(hashTargets) == 0 {
		return fmt.Errorf("no applicable hash targets for current platform")
	}

	// Perform batch hash verification
	hashResults, err := te.hashCalculator.BatchHashCheck(hashTargets)
	if err != nil {
		return fmt.Errorf("hash verification failed: %w", err)
	}

	// Process hash results into evidence
	for _, hashResult := range hashResults {
		evidence := detect.Evidence{
			Type:        detect.EvidenceTypeFileHash,
			Location:    hashResult.Target.Path,
			Expected:    hashResult.Target.ExpectedHash,
			Actual:      hashResult.ActualHash,
			Description: hashResult.Target.Description,
			Context: map[string]interface{}{
				"algorithm":   string(hashResult.Target.Algorithm),
				"file_exists": hashResult.FileExists,
				"matches":     hashResult.Matches,
				"checked_at":  hashResult.CheckedAt,
			},
		}

		if hashResult.Error != "" {
			evidence.Context["error"] = hashResult.Error
		}

		result.Evidence = append(result.Evidence, evidence)
	}

	// Store hash results in metadata
	result.Metadata["hash_results"] = hashResults
	result.Metadata["algorithm"] = string(algorithm)
	result.Metadata["targets_checked"] = len(hashTargets)

	return nil
}

// executeRegistryDetection performs Windows registry-based vulnerability detection
func (te *TemplateExecutor) executeRegistryDetection(ctx context.Context, template *detect.VulnTemplate, result *detect.DetectionResult) error {
	te.logger.Debug("Executing registry detection", zap.String("template_id", template.ID))

	// Check if registry operations are available
	if !te.registryReader.IsWindowsRegistryAvailable() {
		te.logger.Warn("Registry detection not available",
			zap.String("platform", runtime.GOOS),
			zap.Bool("powershell_available", te.registryReader.IsWindowsRegistryAvailable()))

		result.Metadata["skipped_reason"] = "Registry detection not available on this platform"
		result.Metadata["platform"] = runtime.GOOS
		result.Metadata["powershell_available"] = te.registryReader.IsWindowsRegistryAvailable()
		return nil
	}

	// Convert template registry keys to registry targets
	var registryTargets []detect.RegistryTarget
	for _, key := range template.Detection.Keys {
		target := detect.RegistryTarget{
			Path:        key.Path,
			Value:       key.Value,
			Pattern:     key.Pattern,
			Description: key.Description,
		}
		registryTargets = append(registryTargets, target)
	}

	if len(registryTargets) == 0 {
		return fmt.Errorf("no registry targets defined in template")
	}

	// Perform batch registry checks
	registryResults, err := te.registryReader.BatchCheckRegistryKeys(ctx, registryTargets)
	if err != nil {
		return fmt.Errorf("registry check failed: %w", err)
	}

	// Process registry results into evidence
	for _, regResult := range registryResults {
		evidence := detect.Evidence{
			Type:        detect.EvidenceTypeRegistryKey,
			Location:    regResult.Target.Path,
			Expected:    regResult.Target.Pattern,
			Description: regResult.Target.Description,
			Context: map[string]interface{}{
				"key_exists":      regResult.KeyExists,
				"value_exists":    regResult.ValueExists,
				"value_data":      regResult.ValueData,
				"value_type":      regResult.ValueType,
				"pattern_matches": regResult.PatternMatches,
				"checked_at":      regResult.CheckedAt,
				"processing_time": regResult.ProcessingTime,
			},
		}

		// Set actual value for comparison
		if regResult.ValueExists {
			evidence.Actual = regResult.ValueData
		} else {
			evidence.Actual = "<value not found>"
		}

		// Include any errors
		if regResult.Error != "" {
			evidence.Context["error"] = regResult.Error
		}

		result.Evidence = append(result.Evidence, evidence)
	}

	// Store registry results in metadata
	result.Metadata["detection_type"] = "registry"
	result.Metadata["registry_keys"] = len(registryTargets)
	result.Metadata["registry_results"] = registryResults
	result.Metadata["platform"] = runtime.GOOS
	result.Metadata["powershell_available"] = te.registryReader.IsWindowsRegistryAvailable()

	te.logger.Debug("Registry detection completed",
		zap.String("template_id", template.ID),
		zap.Int("targets_checked", len(registryTargets)),
		zap.Int("evidence_collected", len(result.Evidence)))

	return nil
}

// executeConfigFileDetection performs configuration file pattern-based vulnerability detection
func (te *TemplateExecutor) executeConfigFileDetection(ctx context.Context, template *detect.VulnTemplate, result *detect.DetectionResult) error {
	te.logger.Debug("Executing config file detection", zap.String("template_id", template.ID))

	// Import the config package
	configReader := config.NewConfigFileReader(te.logger)

	// Process each config file target
	for _, fileTarget := range template.Detection.Files {
		// Analyze configuration file
		configResult, err := configReader.AnalyzeConfigFile(ctx, fileTarget)
		if err != nil {
			te.logger.Error("Config file analysis failed",
				zap.String("file", fileTarget.Path),
				zap.Error(err))
			continue
		}

		// Convert config results to evidence
		evidence := detect.Evidence{
			Type:        detect.EvidenceTypeFileContent,
			Location:    configResult.FilePath,
			Description: fmt.Sprintf("Config file analysis: %s", configResult.FilePath),
			Context: map[string]interface{}{
				"file_exists":     configResult.FileExists,
				"file_size":       configResult.FileSize,
				"total_lines":     configResult.TotalLines,
				"matching_lines":  configResult.MatchingLines,
				"processing_time": configResult.ProcessingTime,
			},
		}

		if configResult.Error != "" {
			evidence.Context["error"] = configResult.Error
		} else {
			// Add pattern match details
			if len(configResult.PatternMatches) > 0 {
				evidence.Expected = fmt.Sprintf("%d patterns", len(fileTarget.Patterns))
				evidence.Actual = fmt.Sprintf("%d matches found", len(configResult.PatternMatches))

				// Include match details in context
				matches := make([]map[string]interface{}, len(configResult.PatternMatches))
				for i, match := range configResult.PatternMatches {
					matches[i] = map[string]interface{}{
						"line_number":  match.LineNumber,
						"pattern":      match.Pattern,
						"description":  match.Description,
						"line_content": match.Line,
					}
				}
				evidence.Context["pattern_matches"] = matches
			}
		}

		result.Evidence = append(result.Evidence, evidence)
	}

	// Store config results in metadata
	result.Metadata["detection_type"] = "config_file"
	result.Metadata["config_files"] = len(template.Detection.Files)

	return nil
}

// evaluateConditions evaluates template conditions and determines vulnerability status
func (te *TemplateExecutor) evaluateConditions(template *detect.VulnTemplate, result *detect.DetectionResult) {
	te.logger.Debug("Evaluating conditions",
		zap.String("template_id", template.ID),
		zap.Int("conditions", len(template.Detection.Conditions)))

	conditionsMet := 0
	totalConditions := len(template.Detection.Conditions)

	for _, condition := range template.Detection.Conditions {
		met := te.evaluateCondition(condition, result)
		if met {
			conditionsMet++
		}
	}

	// Calculate confidence based on conditions met
	if totalConditions > 0 {
		result.Confidence = float64(conditionsMet) / float64(totalConditions)
	}

	// Determine vulnerability (require all conditions to be met for high confidence)
	result.Vulnerable = conditionsMet == totalConditions && result.Confidence >= 0.8

	te.logger.Debug("Conditions evaluation completed",
		zap.String("template_id", template.ID),
		zap.Int("conditions_met", conditionsMet),
		zap.Int("total_conditions", totalConditions),
		zap.Float64("confidence", result.Confidence),
		zap.Bool("vulnerable", result.Vulnerable))
}

// evaluateCondition evaluates a single condition
func (te *TemplateExecutor) evaluateCondition(condition detect.DetectionCondition, result *detect.DetectionResult) bool {
	switch condition.Type {
	case detect.ConditionTypeFileExists:
		return te.evaluateFileExistsCondition(condition, result)
	case detect.ConditionTypeHashMatch:
		return te.evaluateHashMatchCondition(condition, result)
	case detect.ConditionTypeFileExecutable:
		return te.evaluateFileExecutableCondition(condition, result)
	case detect.ConditionTypeKeyExists:
		return te.evaluateKeyExistsCondition(condition, result)
	case detect.ConditionTypeValueMatchesPattern:
		return te.evaluateValueMatchesPatternCondition(condition, result)
	case detect.ConditionTypePatternFound:
		return te.evaluatePatternFoundCondition(condition, result)
	default:
		te.logger.Warn("Unsupported condition type", zap.String("type", string(condition.Type)))
		return false
	}
}

// evaluateFileExistsCondition checks if files referenced in evidence exist
func (te *TemplateExecutor) evaluateFileExistsCondition(condition detect.DetectionCondition, result *detect.DetectionResult) bool {
	expectedExists := true
	if condition.Value != nil {
		if val, ok := condition.Value.(bool); ok {
			expectedExists = val
		}
	}

	for _, evidence := range result.Evidence {
		if evidence.Type == detect.EvidenceTypeFileHash {
			if fileExists, ok := evidence.Context["file_exists"].(bool); ok {
				if fileExists == expectedExists {
					return true
				}
			}
		}
	}
	return false
}

// evaluateHashMatchCondition checks if any hash matches were found
func (te *TemplateExecutor) evaluateHashMatchCondition(condition detect.DetectionCondition, result *detect.DetectionResult) bool {
	expectedMatch := true
	if condition.Value != nil {
		if val, ok := condition.Value.(bool); ok {
			expectedMatch = val
		}
	}

	for _, evidence := range result.Evidence {
		if evidence.Type == detect.EvidenceTypeFileHash {
			if matches, ok := evidence.Context["matches"].(bool); ok {
				if matches == expectedMatch {
					return true
				}
			}
		}
	}
	return false
}

// evaluateFileExecutableCondition checks if files have executable permissions
func (te *TemplateExecutor) evaluateFileExecutableCondition(condition detect.DetectionCondition, result *detect.DetectionResult) bool {
	expectedExecutable := true
	if condition.Value != nil {
		if val, ok := condition.Value.(bool); ok {
			expectedExecutable = val
		}
	}

	// Check file permissions for files in evidence
	for _, evidence := range result.Evidence {
		if evidence.Type == detect.EvidenceTypeFileHash {
			if fileExists, ok := evidence.Context["file_exists"].(bool); ok && fileExists {
				// Get file info to check executable status
				fileInfo, err := te.hashCalculator.GetFileInfo(evidence.Location)
				if err == nil && fileInfo.IsExecutable == expectedExecutable {
					return true
				}
			}
		}
	}
	return false
}

// Helper functions

// isTemplateApplicableToCurrentPlatform checks if template applies to current platform
func (te *TemplateExecutor) isTemplateApplicableToCurrentPlatform(template *detect.VulnTemplate) bool {
	return te.parser.isTemplateApplicableToPlatform(template, runtime.GOOS)
}

// isTargetApplicableToCurrentPlatform checks if target applies to current platform
func (te *TemplateExecutor) isTargetApplicableToCurrentPlatform(target detect.DetectionTarget) bool {
	if len(target.Platform) == 0 {
		return true // No platform restriction
	}

	currentPlatform := runtime.GOOS
	for _, platform := range target.Platform {
		if platform == currentPlatform {
			return true
		}
	}
	return false
}

// generateDetectionID generates a unique detection ID
func generateDetectionID(templateID string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", templateID, timestamp)
}

// evaluateKeyExistsCondition checks if registry keys exist
func (te *TemplateExecutor) evaluateKeyExistsCondition(condition detect.DetectionCondition, result *detect.DetectionResult) bool {
	expectedExists := true
	if condition.Value != nil {
		if val, ok := condition.Value.(bool); ok {
			expectedExists = val
		}
	}

	for _, evidence := range result.Evidence {
		if evidence.Type == detect.EvidenceTypeRegistryKey {
			if keyExists, ok := evidence.Context["key_exists"].(bool); ok {
				if keyExists == expectedExists {
					return true
				}
			}
		}
	}
	return false
}

// evaluateValueMatchesPatternCondition checks if registry values match patterns
func (te *TemplateExecutor) evaluateValueMatchesPatternCondition(condition detect.DetectionCondition, result *detect.DetectionResult) bool {
	expectedMatch := true
	if condition.Value != nil {
		if val, ok := condition.Value.(bool); ok {
			expectedMatch = val
		}
	}

	for _, evidence := range result.Evidence {
		if evidence.Type == detect.EvidenceTypeRegistryKey {
			if patternMatches, ok := evidence.Context["pattern_matches"].(bool); ok {
				if patternMatches == expectedMatch {
					return true
				}
			}
		}
	}
	return false
}

// evaluatePatternFoundCondition checks if configuration file patterns were found
func (te *TemplateExecutor) evaluatePatternFoundCondition(condition detect.DetectionCondition, result *detect.DetectionResult) bool {
	expectedFound := true
	if condition.Value != nil {
		if val, ok := condition.Value.(bool); ok {
			expectedFound = val
		}
	}

	for _, evidence := range result.Evidence {
		if evidence.Type == detect.EvidenceTypeFileContent {
			if patternMatches, ok := evidence.Context["pattern_matches"].([]map[string]interface{}); ok {
				hasPatterns := len(patternMatches) > 0
				if hasPatterns == expectedFound {
					return true
				}
			}
		}
	}
	return false
}

// GetStats returns execution statistics
func (te *TemplateExecutor) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["max_concurrency"] = te.maxConcurrency
	stats["timeout"] = te.timeout.String()
	stats["template_dirs"] = te.templateDirs
	stats["platform"] = runtime.GOOS

	// Add registry reader stats
	if te.registryReader != nil {
		stats["registry"] = te.registryReader.GetStats()
	}

	return stats
}
