package jobs

import (
	"context"
	"time"
)

// JobStatus represents the status of a job execution
type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusSkipped   JobStatus = "skipped" // When job is already running
)

// JobExecution tracks the execution history of background jobs
type JobExecution struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	JobName     string     `gorm:"size:100;not null;index:idx_job_name_started,priority:1" json:"job_name"`
	Status      JobStatus  `gorm:"size:20;not null;default:running" json:"status"`
	Result      string     `gorm:"type:text" json:"result,omitempty"`
	Error       string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt   time.Time  `gorm:"not null;index:idx_job_name_started,priority:2" json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DurationMs  int64      `gorm:"default:0" json:"duration_ms"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the table name for JobExecution
func (JobExecution) TableName() string {
	return "job_executions"
}

// JobDefinition defines a background job
type JobDefinition struct {
	Name           string        // Unique job name
	Description    string        // Human-readable description
	TimeoutSeconds int           // Maximum execution time in seconds
	Handler        JobHandler    // Function to execute
}

// Timeout returns the timeout as time.Duration
func (j *JobDefinition) Timeout() time.Duration {
	if j.TimeoutSeconds <= 0 {
		return 5 * time.Minute // Default 5 minutes
	}
	return time.Duration(j.TimeoutSeconds) * time.Second
}

// JobHandler is the function signature for job handlers
// Returns a result (any JSON-serializable value) and an error
type JobHandler func(ctx context.Context) (result interface{}, err error)

// JobInfo is the API response for job listing
type JobInfo struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	IsRunning      bool   `json:"is_running"`
	TriggerURL     string `json:"trigger_url"`
}

// JobTriggerResponse is the API response for job trigger
type JobTriggerResponse struct {
	JobName     string      `json:"job_name"`
	ExecutionID uint        `json:"execution_id"`
	Status      JobStatus   `json:"status"`
	Result      interface{} `json:"result,omitempty"`
	Error       string      `json:"error,omitempty"`
	DurationMs  int64       `json:"duration_ms"`
}

// JobListResponse is the API response for listing jobs
type JobListResponse struct {
	Jobs []JobInfo `json:"jobs"`
}
