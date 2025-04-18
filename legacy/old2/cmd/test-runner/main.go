package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

// TestConfig defines the configuration for a test scenario
type TestConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description" json:"description"`
	AgentID     string            `yaml:"agent_id" json:"agent_id"`
	MessageType string            `yaml:"message_type" json:"message_type"`
	Args        map[string]string `yaml:"args" json:"args"`
	Timeout     int               `yaml:"timeout" json:"timeout"`
	Validation  struct {
		ExpectedStatus string            `yaml:"expected_status" json:"expected_status"`
		ExpectedData   map[string]string `yaml:"expected_data" json:"expected_data"`
		ValidateKeys   []string          `yaml:"validate_keys" json:"validate_keys"`
	} `yaml:"validation" json:"validation"`
}

// TestSuite defines a collection of test scenarios
type TestSuite struct {
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	Parallel    bool         `yaml:"parallel" json:"parallel"`
	Tests       []TestConfig `yaml:"tests" json:"tests"`
}

// TestResult stores the result of a test execution
type TestResult struct {
	TestName    string            `json:"test_name"`
	CommandID   string            `json:"command_id"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Duration    time.Duration     `json:"duration"`
	Success     bool              `json:"success"`
	Status      string            `json:"status"`
	Data        map[string]string `json:"data,omitempty"`
	Error       string            `json:"error,omitempty"`
	Validations []struct {
		Check   string `json:"check"`
		Success bool   `json:"success"`
		Message string `json:"message"`
	} `json:"validations"`
}

// TestRunner manages the execution of test suites
type TestRunner struct {
	logger        *zap.Logger
	store         store.KVStore
	suitePath     string
	resultPath    string
	testerPath    string
	retrieverPath string
	timeout       time.Duration
	useGRPC       bool
	serverAddr    string
}

// NewTestRunner creates a new test runner
func NewTestRunner(
	logger *zap.Logger,
	store store.KVStore,
	suitePath string,
	resultPath string,
	testerPath string,
	retrieverPath string,
	timeout time.Duration,
	useGRPC bool,
	serverAddr string,
) *TestRunner {
	return &TestRunner{
		logger:        logger,
		store:         store,
		suitePath:     suitePath,
		resultPath:    resultPath,
		testerPath:    testerPath,
		retrieverPath: retrieverPath,
		timeout:       timeout,
		useGRPC:       useGRPC,
		serverAddr:    serverAddr,
	}
}

// Run executes all test suites
func (r *TestRunner) Run(ctx context.Context) error {
	r.logger.Info("Starting test runner",
		zap.String("suite_path", r.suitePath),
		zap.String("result_path", r.resultPath),
		zap.Duration("timeout", r.timeout))

	// Load the test suite
	suite, err := r.loadTestSuite(r.suitePath)
	if err != nil {
		return fmt.Errorf("failed to load test suite: %w", err)
	}

	r.logger.Info("Loaded test suite",
		zap.String("name", suite.Name),
		zap.String("description", suite.Description),
		zap.Int("test_count", len(suite.Tests)))

	// Create results directory if it doesn't exist
	if err := os.MkdirAll(r.resultPath, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Run the test suite
	results, err := r.runTestSuite(ctx, suite)
	if err != nil {
		return fmt.Errorf("failed to run test suite: %w", err)
	}

	// Generate the test report
	if err := r.generateReport(suite, results); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Display summary
	r.displaySummary(results)

	return nil
}

// loadTestSuite loads a test suite from a YAML file
func (r *TestRunner) loadTestSuite(path string) (*TestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &suite, nil
}

// runTestSuite executes all tests in a suite
func (r *TestRunner) runTestSuite(ctx context.Context, suite *TestSuite) ([]TestResult, error) {
	var results []TestResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	r.logger.Info("Running test suite",
		zap.String("name", suite.Name),
		zap.Bool("parallel", suite.Parallel))

	runTest := func(test TestConfig) {
		defer wg.Done()

		r.logger.Info("Starting test",
			zap.String("test", test.Name),
			zap.String("message_type", test.MessageType))

		result, err := r.runTest(ctx, test)
		if err != nil {
			r.logger.Error("Test failed",
				zap.String("test", test.Name),
				zap.Error(err))

			// Create a failure result
			failResult := TestResult{
				TestName:  test.Name,
				StartTime: time.Now(),
				EndTime:   time.Now(),
				Success:   false,
				Error:     err.Error(),
			}

			mu.Lock()
			results = append(results, failResult)
			mu.Unlock()
			return
		}

		mu.Lock()
		results = append(results, result)
		mu.Unlock()

		r.logger.Info("Test completed",
			zap.String("test", test.Name),
			zap.Bool("success", result.Success),
			zap.Duration("duration", result.Duration))
	}

	for _, test := range suite.Tests {
		wg.Add(1)
		if suite.Parallel {
			go runTest(test)
		} else {
			runTest(test)
		}
	}

	wg.Wait()

	return results, nil
}

// runTest executes a single test
func (r *TestRunner) runTest(ctx context.Context, test TestConfig) (TestResult, error) {
	// Initialize result
	result := TestResult{
		TestName:  test.Name,
		StartTime: time.Now(),
		Success:   false,
	}

	// Create a command ID with timestamp and test name
	commandID := fmt.Sprintf("test-%s-%s",
		time.Now().Format("20060102-150405"),
		strings.ReplaceAll(test.Name, " ", "-"))

	// Set the timeout for this test
	timeout := r.timeout
	if test.Timeout > 0 {
		timeout = time.Duration(test.Timeout) * time.Second
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Send the command using the message tester
	if err := r.sendCommand(testCtx, test, commandID); err != nil {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Error = fmt.Sprintf("command send failed: %v", err)
		return result, err
	}

	r.logger.Info("Command sent, waiting for results",
		zap.String("command_id", commandID),
		zap.Duration("timeout", timeout))

	// Implement polling for results with timeout
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	success := false
	var taskResult agent.TaskMessage

	// Poll for results
	for {
		select {
		case <-testCtx.Done():
			if testCtx.Err() == context.DeadlineExceeded {
				result.Error = "test timed out waiting for result"
				r.logger.Warn("Test timed out",
					zap.String("test", test.Name),
					zap.String("command_id", commandID))
			} else {
				result.Error = fmt.Sprintf("test context cancelled: %v", testCtx.Err())
				r.logger.Warn("Test cancelled",
					zap.String("test", test.Name),
					zap.String("command_id", commandID),
					zap.Error(testCtx.Err()))
			}
			goto COMPLETE
		case <-ticker.C:
			// Check for result
			resp, err := r.retrieveResult(testCtx, commandID)
			if err != nil {
				// If the result isn't available yet, keep polling
				if strings.Contains(err.Error(), "key not found") ||
					strings.Contains(err.Error(), "no results found") {
					continue
				}

				result.Error = fmt.Sprintf("failed to retrieve result: %v", err)
				r.logger.Error("Result retrieval failed",
					zap.String("test", test.Name),
					zap.String("command_id", commandID),
					zap.Error(err))
				goto COMPLETE
			}

			// Process the retrieved result
			taskResult = resp

			// Check if the command is complete (not queued or in_progress)
			if taskResult.Status != agent.TaskStatusQueued &&
				taskResult.Status != agent.TaskStatusInProgress {
				success = true
				goto COMPLETE
			}
		}
	}

COMPLETE:
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.CommandID = commandID

	// If we successfully retrieved a result, validate it
	if success {
		result.Status = string(taskResult.Status)
		result.Data = taskResult.Args // In a real implementation, this would be the result data
		result.Error = taskResult.Error

		// Validate the result
		validationResults := r.validateResult(test, taskResult)
		result.Validations = validationResults

		// Check if all validations passed
		allPassed := true
		for _, v := range validationResults {
			if !v.Success {
				allPassed = false
				break
			}
		}

		result.Success = allPassed
	}

	return result, nil
}

// sendCommand sends a command using the message tester
func (r *TestRunner) sendCommand(ctx context.Context, test TestConfig, commandID string) error {
	// Convert args to JSON for passing to the tester
	argsJSON, err := json.Marshal(test.Args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	// Build command for the message tester
	args := []string{
		"-type", test.MessageType,
		"-agent", test.AgentID,
		"-args", string(argsJSON),
	}

	// Add gRPC flag if needed
	if r.useGRPC {
		args = append(args, "-grpc")
		args = append(args, "-addr", r.serverAddr)
	}

	cmd := exec.CommandContext(ctx, r.testerPath, args...)
	cmd.Env = os.Environ()

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("message tester failed: %w, output: %s", err, string(output))
	}

	r.logger.Debug("Message tester output", zap.String("output", string(output)))
	return nil
}

// retrieveResult retrieves a command result using the result retriever
func (r *TestRunner) retrieveResult(ctx context.Context, commandID string) (agent.TaskMessage, error) {
	// In a real implementation, we would use the result retriever to get results
	// For now, we'll use the mock store directly
	key := "task:" + commandID
	resp, err := r.store.GetValue(ctx, key)
	if err != nil {
		return agent.TaskMessage{}, err
	}

	var task agent.TaskMessage
	if err := json.Unmarshal([]byte(resp.Message.Value), &task); err != nil {
		return agent.TaskMessage{}, fmt.Errorf("failed to parse task data: %w", err)
	}

	return task, nil
}

// validateResult validates a command result against the expected values
func (r *TestRunner) validateResult(test TestConfig, result agent.TaskMessage) []struct {
	Check   string `json:"check"`
	Success bool   `json:"success"`
	Message string `json:"message"`
} {
	var validations []struct {
		Check   string `json:"check"`
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	// Validate status
	if test.Validation.ExpectedStatus != "" {
		statusCheck := struct {
			Check   string `json:"check"`
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Check:   "status",
			Success: string(result.Status) == test.Validation.ExpectedStatus,
		}

		if !statusCheck.Success {
			statusCheck.Message = fmt.Sprintf("expected status %s, got %s",
				test.Validation.ExpectedStatus, string(result.Status))
		}

		validations = append(validations, statusCheck)
	}

	// Validate specific data fields
	for key, expectedValue := range test.Validation.ExpectedData {
		dataCheck := struct {
			Check   string `json:"check"`
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Check:   fmt.Sprintf("data.%s", key),
			Success: result.Args[key] == expectedValue,
		}

		if !dataCheck.Success {
			dataCheck.Message = fmt.Sprintf("expected %s=%s, got %s",
				key, expectedValue, result.Args[key])
		}

		validations = append(validations, dataCheck)
	}

	// Validate that required keys exist
	for _, key := range test.Validation.ValidateKeys {
		keyCheck := struct {
			Check   string `json:"check"`
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Check:   fmt.Sprintf("key_exists.%s", key),
			Success: false,
		}

		_, exists := result.Args[key]
		keyCheck.Success = exists

		if !keyCheck.Success {
			keyCheck.Message = fmt.Sprintf("required key %s not found in result", key)
		}

		validations = append(validations, keyCheck)
	}

	return validations
}

// generateReport generates a JSON report of test results
func (r *TestRunner) generateReport(suite *TestSuite, results []TestResult) error {
	// Create results structure
	report := struct {
		SuiteName    string        `json:"suite_name"`
		Description  string        `json:"description"`
		TestCount    int           `json:"test_count"`
		SuccessCount int           `json:"success_count"`
		FailureCount int           `json:"failure_count"`
		StartTime    time.Time     `json:"start_time"`
		EndTime      time.Time     `json:"end_time"`
		Duration     time.Duration `json:"duration"`
		Results      []TestResult  `json:"results"`
	}{
		SuiteName:   suite.Name,
		Description: suite.Description,
		TestCount:   len(results),
		Results:     results,
	}

	// Calculate stats
	report.StartTime = time.Now()
	report.EndTime = time.Now()

	if len(results) > 0 {
		report.StartTime = results[0].StartTime
		report.EndTime = results[0].EndTime

		for _, result := range results {
			if result.StartTime.Before(report.StartTime) {
				report.StartTime = result.StartTime
			}
			if result.EndTime.After(report.EndTime) {
				report.EndTime = result.EndTime
			}
			if result.Success {
				report.SuccessCount++
			} else {
				report.FailureCount++
			}
		}
	}

	report.Duration = report.EndTime.Sub(report.StartTime)

	// Format the filename with timestamp
	filename := fmt.Sprintf("%s/test-report-%s.json",
		r.resultPath, time.Now().Format("20060102-150405"))

	// Write to file
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	r.logger.Info("Test report generated",
		zap.String("file", filename),
		zap.Int("total", report.TestCount),
		zap.Int("success", report.SuccessCount),
		zap.Int("failure", report.FailureCount))

	return nil
}

// displaySummary prints a summary of test results
func (r *TestRunner) displaySummary(results []TestResult) {
	fmt.Println("\n==== TEST RESULTS SUMMARY ====")
	fmt.Printf("Total Tests: %d\n", len(results))

	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	fmt.Printf("Successes: %d\n", successCount)
	fmt.Printf("Failures: %d\n", failureCount)

	if failureCount > 0 {
		fmt.Println("\nFAILED TESTS:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  - %s: %s\n", result.TestName, result.Error)
				if len(result.Validations) > 0 {
					for _, v := range result.Validations {
						if !v.Success {
							fmt.Printf("    * %s: %s\n", v.Check, v.Message)
						}
					}
				}
			}
		}
	}

	fmt.Println("=============================")
}

func main() {
	// Parse flags
	suitePath := flag.String("suite", "", "Path to test suite YAML file")
	resultPath := flag.String("output", "test-results", "Directory for test results")
	testerPath := flag.String("tester", "cmd/agent-message-tester/agent-message-tester", "Path to message tester binary")
	retrieverPath := flag.String("retriever", "cmd/result-retriever/result-retriever", "Path to result retriever binary")
	timeout := flag.Int("timeout", 60, "Global timeout in seconds for each test")
	useGRPC := flag.Bool("grpc", false, "Use gRPC for sending commands")
	serverAddr := flag.String("addr", ":50051", "gRPC server address")

	flag.Parse()

	// Check required flags
	if *suitePath == "" {
		log.Fatalf("Missing required flag: -suite")
	}

	// Set up logger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	logger, _ := config.Build()
	defer logger.Sync()

	// Resolve paths
	absTestPath, err := filepath.Abs(*suitePath)
	if err != nil {
		logger.Fatal("Failed to resolve test suite path", zap.Error(err))
	}

	absResultPath, err := filepath.Abs(*resultPath)
	if err != nil {
		logger.Fatal("Failed to resolve result path", zap.Error(err))
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Shutting down gracefully...")
		cancel()
	}()

	// Create mock storage
	mockStore := store.NewMockStore()

	// Create the test runner
	runner := NewTestRunner(
		logger,
		mockStore,
		absTestPath,
		absResultPath,
		*testerPath,
		*retrieverPath,
		time.Duration(*timeout)*time.Second,
		*useGRPC,
		*serverAddr,
	)

	// Run the tests
	if err := runner.Run(ctx); err != nil {
		logger.Fatal("Test execution failed", zap.Error(err))
	}

	logger.Info("Test execution completed successfully")
}
