package jobs

import (
	"sync"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
)

const (
	// RetentionDays is the number of days to keep job execution logs
	RetentionDays = 7
	// CleanupInterval is how often to check for old logs
	CleanupInterval = 24 * time.Hour
)

var (
	lastCleanup   time.Time
	cleanupMutex  sync.Mutex
)

// CleanupOldLogs removes job execution logs older than RetentionDays
// It's called lazily and only runs once per day
func CleanupOldLogs() {
	cleanupMutex.Lock()
	defer cleanupMutex.Unlock()

	// Only run cleanup once per day
	if time.Since(lastCleanup) < CleanupInterval {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -RetentionDays)

	result := db.Where("created_at < ?", cutoff).Delete(&JobExecution{})
	if result.Error != nil {
		log.Error("[jobs] Failed to cleanup old logs: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Info("[jobs] Cleaned up %d job execution logs older than %d days",
			result.RowsAffected, RetentionDays)
	}

	lastCleanup = time.Now()
}

// ForceCleanupOldLogs runs cleanup regardless of the last cleanup time
func ForceCleanupOldLogs() (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -RetentionDays)

	result := db.Where("created_at < ?", cutoff).Delete(&JobExecution{})
	if result.Error != nil {
		return 0, result.Error
	}

	cleanupMutex.Lock()
	lastCleanup = time.Now()
	cleanupMutex.Unlock()

	return result.RowsAffected, nil
}

// GetExecutionHistory returns recent job executions
func GetExecutionHistory(jobName string, limit int) ([]JobExecution, error) {
	var executions []JobExecution

	query := db.Model(&JobExecution{}).Order("started_at DESC")

	if jobName != "" {
		query = query.Where("job_name = ?", jobName)
	}

	if limit > 0 {
		query = query.Limit(limit)
	} else {
		query = query.Limit(100) // Default limit
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, err
	}

	return executions, nil
}

// GetLastExecution returns the most recent execution for a job
func GetLastExecution(jobName string) (*JobExecution, error) {
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
