package executor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

const (
	// DefaultStepTimeout is the maximum time for a single step execution
	DefaultStepTimeout = 30 * time.Second
)

// Executor executes templates and produces results
type Executor struct {
	// stepTimeout is the maximum time for a single step
	stepTimeout time.Duration
}

// New creates a new Executor with default settings
func New() *Executor {
	return &Executor{
		stepTimeout: DefaultStepTimeout,
	}
}

// NewWithTimeout creates a new Executor with a custom step timeout
func NewWithTimeout(timeout time.Duration) *Executor {
	return &Executor{
		stepTimeout: timeout,
	}
}

// ExecuteTemplate executes a template and returns the result.
// It filters steps by platform, executes them sequentially, evaluates logic,
// calculates confidence, and builds the final result.
func (e *Executor) ExecuteTemplate(ctx context.Context, template *types.Template) (*types.Result, error) {
	if template == nil {
		return nil, fmt.Errorf("template cannot be nil")
	}

	// Get hostname for result
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	// Filter steps by current platform
	currentPlatform := types.Platform(runtime.GOOS)
	applicableSteps := e.filterStepsByPlatform(template.Detection.Steps, currentPlatform)

	// Execute steps sequentially
	stepResults := make([]types.StepResult, 0, len(applicableSteps))
	var executionErrors []string

	for i, step := range applicableSteps {
		stepResult := e.executeStep(ctx, step, i)
		stepResults = append(stepResults, stepResult)

		// Collect errors but continue execution
		if stepResult.Error != "" {
			executionErrors = append(executionErrors, fmt.Sprintf("step %d (%s): %s", i, step.Type, stepResult.Error))
		}
	}

	// Evaluate detection logic
	matched := e.evaluateLogic(template.Detection.Logic, stepResults)

	// Calculate confidence
	confidence := e.calculateConfidence(template.Detection.Logic, stepResults, applicableSteps)

	// Build final result
	result := &types.Result{
		TemplateID:   template.ID,
		TemplateName: template.Info.Name,
		Severity:     template.Info.Severity,
		Matched:      matched,
		Confidence:   confidence,
		Steps:        stepResults,
		Errors:       executionErrors,
		Timestamp:    time.Now(),
		Host:         hostname,
	}

	return result, nil
}

// filterStepsByPlatform filters detection steps by platform compatibility
func (e *Executor) filterStepsByPlatform(steps []types.DetectionStep, currentPlatform types.Platform) []types.DetectionStep {
	filtered := make([]types.DetectionStep, 0, len(steps))

	for _, step := range steps {
		// If no platforms specified, step applies to all platforms
		if len(step.Platforms) == 0 {
			filtered = append(filtered, step)
			continue
		}

		// Check if current platform is in the list
		for _, platform := range step.Platforms {
			if platform == currentPlatform {
				filtered = append(filtered, step)
				break
			}
		}
	}

	return filtered
}

// executeStep executes a single detection step
func (e *Executor) executeStep(ctx context.Context, step types.DetectionStep, index int) types.StepResult {
	startTime := time.Now()

	// Create step result
	stepResult := types.StepResult{
		Step:     index,
		Type:     step.Type,
		Matched:  false,
		Evidence: nil,
		Error:    "",
		Duration: 0,
	}

	// Get module from registry
	module := registry.Get(step.Type)
	if module == nil {
		stepResult.Error = fmt.Sprintf("module type '%s' not found in registry", step.Type)
		stepResult.Duration = time.Since(startTime)
		return stepResult
	}

	// Create context with timeout for this step
	stepCtx, cancel := context.WithTimeout(ctx, e.stepTimeout)
	defer cancel()

	// Execute module
	moduleResult, err := module.Execute(stepCtx, step.Config)
	stepResult.Duration = time.Since(startTime)

	if err != nil {
		stepResult.Error = fmt.Sprintf("module execution error: %v", err)
		return stepResult
	}

	// Copy result from module
	if moduleResult != nil {
		stepResult.Matched = moduleResult.Matched
		stepResult.Evidence = moduleResult.Evidence
		if moduleResult.Error != "" {
			stepResult.Error = moduleResult.Error
		}
	}

	return stepResult
}

// evaluateLogic evaluates the detection logic (AND/OR) based on step results
func (e *Executor) evaluateLogic(logic types.DetectionLogic, steps []types.StepResult) bool {
	if len(steps) == 0 {
		return false
	}

	// Default to "all" (AND) logic
	if logic == "" {
		logic = types.LogicAll
	}

	switch logic {
	case types.LogicAll:
		// AND logic: all steps must match
		for _, step := range steps {
			// Skip steps with errors
			if step.Error != "" {
				continue
			}
			if !step.Matched {
				return false
			}
		}
		// At least one step must have matched (not just errors)
		hasMatch := false
		for _, step := range steps {
			if step.Error == "" && step.Matched {
				hasMatch = true
				break
			}
		}
		return hasMatch

	case types.LogicAny:
		// OR logic: at least one step must match
		for _, step := range steps {
			if step.Error == "" && step.Matched {
				return true
			}
		}
		return false

	default:
		// Unknown logic, default to false
		return false
	}
}

// calculateConfidence calculates the confidence score based on step weights
func (e *Executor) calculateConfidence(logic types.DetectionLogic, steps []types.StepResult, originalSteps []types.DetectionStep) float64 {
	if len(steps) == 0 {
		return 0.0
	}

	// Default to "all" (AND) logic
	if logic == "" {
		logic = types.LogicAll
	}

	// Collect weights for matched steps (without errors)
	var matchedWeights []float64
	for i, step := range steps {
		if step.Error == "" && step.Matched {
			weight := 1.0 // Default weight
			if i < len(originalSteps) && originalSteps[i].Weight > 0 {
				weight = originalSteps[i].Weight
			}
			matchedWeights = append(matchedWeights, weight)
		}
	}

	// No matched steps = zero confidence
	if len(matchedWeights) == 0 {
		return 0.0
	}

	switch logic {
	case types.LogicAll:
		// AND logic: minimum of matched weights
		minWeight := matchedWeights[0]
		for _, weight := range matchedWeights[1:] {
			if weight < minWeight {
				minWeight = weight
			}
		}
		return minWeight

	case types.LogicAny:
		// OR logic: maximum of matched weights
		maxWeight := matchedWeights[0]
		for _, weight := range matchedWeights[1:] {
			if weight > maxWeight {
				maxWeight = weight
			}
		}
		return maxWeight

	default:
		return 0.0
	}
}

