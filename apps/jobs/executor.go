package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/redis"
)

const (
	lockPrefix     = "job:lock:"
	lockExtraTime  = 30 * time.Second // Extra time added to lock TTL beyond job timeout
)

// Executor handles job execution with Redis-based locking
type Executor struct {
	registry *Registry
	mu       sync.Mutex
}

var (
	executor     *Executor
	executorOnce sync.Once
)

// GetExecutor returns the singleton executor instance
func GetExecutor() *Executor {
	executorOnce.Do(func() {
		executor = &Executor{
			registry: GetRegistry(),
		}
	})
	return executor
}

// Execute runs a job by name with Redis locking
func (e *Executor) Execute(jobName string) (*JobTriggerResponse, error) {
	// Get job definition
	job, exists := e.registry.Get(jobName)
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobName)
	}

	// Try to acquire lock
	lockKey := lockPrefix + jobName
	lockTTL := job.Timeout() + lockExtraTime

	if !e.acquireLock(lockKey, lockTTL) {
		// Job is already running
		return &JobTriggerResponse{
			JobName: jobName,
			Status:  JobStatusSkipped,
			Error:   "job is already running",
		}, nil
	}

	// Create execution record
	execution := &JobExecution{
		JobName:   jobName,
		Status:    JobStatusRunning,
		StartedAt: time.Now(),
	}

	if err := db.Create(execution).Error; err != nil {
		e.releaseLock(lockKey)
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	log.Info("[jobs] Starting job: %s (execution: %d)", jobName, execution.ID)

	// Execute the job
	response := e.executeJob(job, execution, lockKey)

	return response, nil
}

// executeJob runs the job handler and updates the execution record
func (e *Executor) executeJob(job *JobDefinition, execution *JobExecution, lockKey string) *JobTriggerResponse {
	defer e.releaseLock(lockKey)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), job.Timeout())
	defer cancel()

	// Channel for job result
	type jobResult struct {
		result interface{}
		err    error
	}
	resultChan := make(chan jobResult, 1)

	// Run job in goroutine
	go func() {
		result, err := job.Handler(ctx)
		resultChan <- jobResult{result: result, err: err}
	}()

	// Wait for completion or timeout
	var result interface{}
	var jobErr error

	select {
	case res := <-resultChan:
		result = res.result
		jobErr = res.err
	case <-ctx.Done():
		jobErr = ctx.Err()
	}

	// Update execution record
	now := time.Now()
	execution.CompletedAt = &now
	execution.DurationMs = now.Sub(execution.StartedAt).Milliseconds()

	response := &JobTriggerResponse{
		JobName:     job.Name,
		ExecutionID: execution.ID,
		DurationMs:  execution.DurationMs,
	}

	if jobErr != nil {
		execution.Status = JobStatusFailed
		execution.Error = jobErr.Error()
		response.Status = JobStatusFailed
		response.Error = jobErr.Error()
		log.Error("[jobs] Job %s failed: %v", job.Name, jobErr)
	} else {
		execution.Status = JobStatusCompleted
		response.Status = JobStatusCompleted
		response.Result = result

		// Store result as JSON
		if result != nil {
			if resultJSON, err := json.Marshal(result); err == nil {
				execution.Result = string(resultJSON)
			}
		}

		log.Info("[jobs] Job %s completed (execution: %d, duration: %dms)",
			job.Name, execution.ID, execution.DurationMs)
	}

	// Save execution record
	if err := db.Save(execution).Error; err != nil {
		log.Error("[jobs] Failed to update execution record: %v", err)
	}

	return response
}

// IsRunning checks if a job is currently running (has a lock)
func (e *Executor) IsRunning(jobName string) bool {
	if !redis.IsAvailable() {
		return false
	}

	lockKey := lockPrefix + jobName
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	exists, err := redis.Client.Exists(ctx, lockKey).Result()
	if err != nil {
		log.Error("[jobs] Redis error checking lock: %v", err)
		return false
	}

	return exists > 0
}

// acquireLock attempts to acquire a Redis lock for a job
func (e *Executor) acquireLock(lockKey string, ttl time.Duration) bool {
	if !redis.IsAvailable() {
		// If Redis is unavailable, use local mutex as fallback
		// This won't work across multiple instances but prevents local double-execution
		e.mu.Lock()
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// SET key value NX EX seconds
	// NX = only set if not exists
	// EX = expire after seconds
	result, err := redis.Client.SetNX(ctx, lockKey, time.Now().Unix(), ttl).Result()
	if err != nil {
		log.Error("[jobs] Redis error acquiring lock: %v", err)
		return false
	}

	return result
}

// releaseLock releases a Redis lock for a job
func (e *Executor) releaseLock(lockKey string) {
	if !redis.IsAvailable() {
		e.mu.Unlock()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := redis.Client.Del(ctx, lockKey).Err(); err != nil {
		log.Error("[jobs] Redis error releasing lock: %v", err)
	}
}

// GetJobsInfo returns information about all registered jobs
func (e *Executor) GetJobsInfo() []JobInfo {
	jobs := e.registry.List()
	infos := make([]JobInfo, 0, len(jobs))

	for _, job := range jobs {
		infos = append(infos, JobInfo{
			Name:           job.Name,
			Description:    job.Description,
			TimeoutSeconds: job.TimeoutSeconds,
			IsRunning:      e.IsRunning(job.Name),
		})
	}

	return infos
}
