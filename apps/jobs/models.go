package jobs

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of a job execution
type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// JobExecution tracks the execution history of background jobs
type JobExecution struct {
	ID               uuid.UUID  `gorm:"type:char(36);primaryKey" json:"id"`
	JobName          string     `gorm:"size:100;not null;index:idx_job_started,priority:1" json:"job_name"`
	InstanceID       string     `gorm:"size:100;not null" json:"instance_id"`
	Status           JobStatus  `gorm:"size:20;not null;default:running" json:"status"`
	StartedAt        time.Time  `gorm:"not null;index:idx_job_started,priority:2" json:"started_at"`
	CompletedAt      *time.Time `gorm:"" json:"completed_at"`
	DurationMs       int64      `gorm:"default:0" json:"duration_ms"`
	RecordsProcessed int        `gorm:"default:0" json:"records_processed"`
	Error            string     `gorm:"type:text" json:"error,omitempty"`
	Metadata         string     `gorm:"type:json" json:"metadata,omitempty"`
}

// TableName returns the table name for JobExecution
func (JobExecution) TableName() string {
	return "job_executions"
}

// JobDefinition defines a scheduled job
type JobDefinition struct {
	Name        string        // Unique job name
	Description string        // Human-readable description
	Schedule    string        // Cron expression
	Timeout     time.Duration // Maximum execution time
	Handler     JobHandler    // Function to execute
	Enabled     bool          // Whether the job is enabled
}

// JobHandler is the function signature for job handlers
type JobHandler func(ctx *JobContext) error

// JobContext provides context and utilities to job handlers
type JobContext struct {
	JobName    string
	ExecutionID uuid.UUID
	StartedAt  time.Time
	processed  int
	metadata   map[string]interface{}
}

// NewJobContext creates a new job context
func NewJobContext(jobName string, executionID uuid.UUID) *JobContext {
	return &JobContext{
		JobName:     jobName,
		ExecutionID: executionID,
		StartedAt:   time.Now(),
		metadata:    make(map[string]interface{}),
	}
}

// IncrementProcessed increments the records processed counter
func (ctx *JobContext) IncrementProcessed(count int) {
	ctx.processed += count
}

// GetProcessed returns the number of records processed
func (ctx *JobContext) GetProcessed() int {
	return ctx.processed
}

// SetMetadata sets a metadata value
func (ctx *JobContext) SetMetadata(key string, value interface{}) {
	ctx.metadata[key] = value
}

// GetMetadata returns the metadata map
func (ctx *JobContext) GetMetadata() map[string]interface{} {
	return ctx.metadata
}
