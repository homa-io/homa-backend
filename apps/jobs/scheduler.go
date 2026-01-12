package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Scheduler manages background job scheduling and execution
type Scheduler struct {
	cron      *cron.Cron
	locks     *LockManager
	jobs      map[string]*JobDefinition
	mu        sync.RWMutex
	isRunning bool
}

var (
	scheduler *Scheduler
	once      sync.Once
)

// GetScheduler returns the singleton scheduler instance
func GetScheduler() *Scheduler {
	return scheduler
}

// NewScheduler creates a new job scheduler
func NewScheduler(locks *LockManager) *Scheduler {
	once.Do(func() {
		scheduler = &Scheduler{
			cron: cron.New(cron.WithSeconds(), cron.WithChain(
				cron.Recover(cron.DefaultLogger),
			)),
			locks: locks,
			jobs:  make(map[string]*JobDefinition),
		}
	})
	return scheduler
}

// RegisterJob registers a new job with the scheduler
func (s *Scheduler) RegisterJob(job JobDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !job.Enabled {
		log.Info("Job %s is disabled, skipping registration", job.Name)
		return nil
	}

	// Store job definition
	s.jobs[job.Name] = &job

	// Add to cron scheduler
	_, err := s.cron.AddFunc(job.Schedule, func() {
		s.runJob(job.Name)
	})
	if err != nil {
		return err
	}

	log.Info("Registered job: %s (schedule: %s)", job.Name, job.Schedule)
	return nil
}

// runJob executes a job with distributed locking
func (s *Scheduler) runJob(jobName string) {
	s.mu.RLock()
	job, exists := s.jobs[jobName]
	s.mu.RUnlock()

	if !exists {
		log.Error("Job not found: %s", jobName)
		return
	}

	// Try to acquire lock
	if !s.locks.TryLock(jobName) {
		log.Debug("Job %s is already running on another instance, skipping", jobName)
		return
	}
	defer s.locks.Unlock(jobName)

	// Create execution record
	executionID := uuid.New()
	execution := &JobExecution{
		ID:         executionID,
		JobName:    jobName,
		InstanceID: s.locks.GetInstanceID(),
		Status:     JobStatusRunning,
		StartedAt:  time.Now(),
	}

	if err := db.Create(execution).Error; err != nil {
		log.Error("Failed to create job execution record: %v", err)
		return
	}

	log.Info("Starting job: %s (execution: %s)", jobName, executionID)

	// Create job context
	ctx := NewJobContext(jobName, executionID)

	// Run with timeout if specified
	var jobErr error
	if job.Timeout > 0 {
		jobErr = s.runWithTimeout(ctx, job.Handler, job.Timeout)
	} else {
		jobErr = job.Handler(ctx)
	}

	// Update execution record
	now := time.Now()
	execution.CompletedAt = &now
	execution.DurationMs = now.Sub(execution.StartedAt).Milliseconds()
	execution.RecordsProcessed = ctx.GetProcessed()

	if jobErr != nil {
		execution.Status = JobStatusFailed
		execution.Error = jobErr.Error()
		log.Error("Job %s failed: %v", jobName, jobErr)
	} else {
		execution.Status = JobStatusCompleted
		log.Info("Job %s completed successfully (processed: %d, duration: %dms)",
			jobName, execution.RecordsProcessed, execution.DurationMs)
	}

	// Store metadata if any
	if len(ctx.GetMetadata()) > 0 {
		if metadataJSON, err := json.Marshal(ctx.GetMetadata()); err == nil {
			execution.Metadata = string(metadataJSON)
		}
	}

	if err := db.Save(execution).Error; err != nil {
		log.Error("Failed to update job execution record: %v", err)
	}
}

// runWithTimeout executes a job handler with a timeout
func (s *Scheduler) runWithTimeout(jobCtx *JobContext, handler JobHandler, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- handler(jobCtx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return
	}

	s.cron.Start()
	s.isRunning = true
	log.Info("Job scheduler started with %d jobs", len(s.jobs))
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.isRunning = false
	log.Info("Job scheduler stopped")
}

// RunNow triggers immediate execution of a job
func (s *Scheduler) RunNow(jobName string) error {
	s.mu.RLock()
	_, exists := s.jobs[jobName]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	go s.runJob(jobName)
	return nil
}

// GetJobs returns all registered job definitions
func (s *Scheduler) GetJobs() map[string]*JobDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*JobDefinition)
	for k, v := range s.jobs {
		result[k] = v
	}
	return result
}

// GetRecentExecutions returns recent job executions
func (s *Scheduler) GetRecentExecutions(jobName string, limit int) ([]JobExecution, error) {
	var executions []JobExecution

	query := db.Model(&JobExecution{}).Order("started_at DESC").Limit(limit)
	if jobName != "" {
		query = query.Where("job_name = ?", jobName)
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, err
	}

	return executions, nil
}

// GetLastExecution returns the most recent execution for a job
func (s *Scheduler) GetLastExecution(jobName string) (*JobExecution, error) {
	var execution JobExecution

	err := db.Model(&JobExecution{}).
		Where("job_name = ?", jobName).
		Order("started_at DESC").
		First(&execution).Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

// CleanupOldExecutions removes execution records older than the specified duration
func (s *Scheduler) CleanupOldExecutions(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	result := db.Where("started_at < ?", cutoff).Delete(&JobExecution{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
