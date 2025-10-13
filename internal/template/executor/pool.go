package executor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/SiriusScan/app-agent/internal/template/types"
)

const (
	// DefaultWorkerCount is the default number of workers for parallel execution
	DefaultWorkerCount = 0 // 0 means use runtime.NumCPU()

	// MinWorkerCount is the minimum number of workers allowed
	MinWorkerCount = 1

	// MaxWorkerCount is the maximum number of workers allowed
	MaxWorkerCount = 50

	// DefaultPerTemplateTimeout is the maximum time for a single template
	DefaultPerTemplateTimeout = 5 * time.Minute
)

// WorkerPoolConfig configures the worker pool behavior
type WorkerPoolConfig struct {
	// Workers is the number of concurrent workers (0 = runtime.NumCPU())
	Workers int

	// PerTemplateTimeout is the maximum time for a single template
	PerTemplateTimeout time.Duration

	// Context allows cancellation of the entire pool
	Context context.Context
}

// DefaultWorkerPoolConfig returns the default worker pool configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		Workers:            DefaultWorkerCount,
		PerTemplateTimeout: DefaultPerTemplateTimeout,
		Context:            context.Background(),
	}
}

// job represents a template to be executed
type job struct {
	template *types.Template
	index    int // Original index for maintaining order
}

// result represents the result of executing a template
type result struct {
	result *types.Result
	index  int
	err    error
}

// ExecuteTemplatesParallel executes multiple templates in parallel using a worker pool.
// Returns results in the same order as the input templates.
func ExecuteTemplatesParallel(templates []*types.Template, workers int) ([]*types.Result, []error) {
	config := DefaultWorkerPoolConfig()
	config.Workers = workers
	return ExecuteTemplatesParallelWithConfig(templates, config)
}

// ExecuteTemplatesParallelWithConfig executes templates with custom configuration.
func ExecuteTemplatesParallelWithConfig(templates []*types.Template, config WorkerPoolConfig) ([]*types.Result, []error) {
	if len(templates) == 0 {
		return nil, nil
	}

	// Determine worker count
	workerCount := config.Workers
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	if workerCount < MinWorkerCount {
		workerCount = MinWorkerCount
	}
	if workerCount > MaxWorkerCount {
		workerCount = MaxWorkerCount
	}

	// Don't use more workers than templates
	if workerCount > len(templates) {
		workerCount = len(templates)
	}

	// Create channels
	jobs := make(chan job, len(templates))
	results := make(chan result, len(templates))

	// Create executor for workers to use
	exec := New()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(i, &wg, jobs, results, exec, config)
	}

	// Send jobs
	for i, template := range templates {
		jobs <- job{template: template, index: i}
	}
	close(jobs)

	// Wait for workers to finish in a goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	resultSlice := make([]*types.Result, len(templates))
	var errors []error

	for res := range results {
		if res.err != nil {
			errors = append(errors, fmt.Errorf("template %d: %w", res.index, res.err))
			continue
		}
		resultSlice[res.index] = res.result
	}

	return resultSlice, errors
}

// worker processes jobs from the jobs channel
func worker(id int, wg *sync.WaitGroup, jobs <-chan job, results chan<- result, exec *Executor, config WorkerPoolConfig) {
	defer wg.Done()
	defer func() {
		// Panic recovery
		if r := recover(); r != nil {
			// Log the panic but don't crash the entire pool
			err := fmt.Errorf("worker %d panicked: %v", id, r)
			// Try to send error result if possible
			select {
			case results <- result{err: err, index: -1}:
			default:
				// Channel might be closed, ignore
			}
		}
	}()

	for job := range jobs {
		// Check for context cancellation
		select {
		case <-config.Context.Done():
			results <- result{
				err:   fmt.Errorf("cancelled: %w", config.Context.Err()),
				index: job.index,
			}
			continue
		default:
		}

		// Create context with per-template timeout
		ctx, cancel := context.WithTimeout(config.Context, config.PerTemplateTimeout)

		// Execute template
		res, err := exec.ExecuteTemplate(ctx, job.template)

		cancel() // Clean up context

		// Send result
		results <- result{
			result: res,
			index:  job.index,
			err:    err,
		}
	}
}

// ValidateWorkerCount validates the worker count is within acceptable range
func ValidateWorkerCount(workers int) error {
	if workers < MinWorkerCount || workers > MaxWorkerCount {
		return fmt.Errorf("worker count must be between %d and %d (got %d)", MinWorkerCount, MaxWorkerCount, workers)
	}
	return nil
}

