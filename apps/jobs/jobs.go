package jobs

import (
	"time"

	"github.com/getevo/evo/v2/lib/log"
)

// Job names as constants for consistency
const (
	JobSendCSATEmails       = "send_csat_emails"
	JobCalculateMetrics     = "calculate_metrics"
	JobCloseUnresponded     = "close_unresponded_tickets"
	JobArchiveOldTickets    = "archive_old_tickets"
	JobDeleteOldTickets     = "delete_old_tickets"
	JobCleanupJobExecutions = "cleanup_job_executions"
)

// RegisterAllJobs registers all background jobs with the scheduler
func RegisterAllJobs(s *Scheduler) {
	// Send CSAT emails for closed conversations
	// Runs every 5 minutes
	s.RegisterJob(JobDefinition{
		Name:        JobSendCSATEmails,
		Description: "Send CSAT survey emails and conversation summaries for recently closed conversations",
		Schedule:    "0 */5 * * * *", // Every 5 minutes
		Timeout:     10 * time.Minute,
		Handler:     handleSendCSATEmails,
		Enabled:     true,
	})

	// Calculate daily metrics
	// Runs at 2:00 AM every day
	s.RegisterJob(JobDefinition{
		Name:        JobCalculateMetrics,
		Description: "Calculate and store daily metrics for reporting",
		Schedule:    "0 0 2 * * *", // 2:00 AM daily
		Timeout:     30 * time.Minute,
		Handler:     handleCalculateMetrics,
		Enabled:     true,
	})

	// Close unresponded tickets
	// Runs every hour
	s.RegisterJob(JobDefinition{
		Name:        JobCloseUnresponded,
		Description: "Close tickets that have had no response within the configured time period",
		Schedule:    "0 0 * * * *", // Every hour at :00
		Timeout:     15 * time.Minute,
		Handler:     handleCloseUnresponded,
		Enabled:     true,
	})

	// Archive old tickets
	// Runs at 3:00 AM every day
	s.RegisterJob(JobDefinition{
		Name:        JobArchiveOldTickets,
		Description: "Archive resolved tickets older than the configured retention period",
		Schedule:    "0 0 3 * * *", // 3:00 AM daily
		Timeout:     30 * time.Minute,
		Handler:     handleArchiveOldTickets,
		Enabled:     true,
	})

	// Delete very old tickets
	// Runs at 4:00 AM every Sunday
	s.RegisterJob(JobDefinition{
		Name:        JobDeleteOldTickets,
		Description: "Permanently delete archived tickets older than the configured deletion period",
		Schedule:    "0 0 4 * * 0", // 4:00 AM on Sundays
		Timeout:     60 * time.Minute,
		Handler:     handleDeleteOldTickets,
		Enabled:     true,
	})

	// Cleanup old job execution records
	// Runs at 5:00 AM every day
	s.RegisterJob(JobDefinition{
		Name:        JobCleanupJobExecutions,
		Description: "Clean up job execution history older than 30 days",
		Schedule:    "0 0 5 * * *", // 5:00 AM daily
		Timeout:     5 * time.Minute,
		Handler:     handleCleanupJobExecutions,
		Enabled:     true,
	})
}

// Job handlers - implement these with actual logic

func handleSendCSATEmails(ctx *JobContext) error {
	log.Info("[%s] Starting CSAT email job", ctx.JobName)

	// TODO: Implement CSAT email sending
	// 1. Query conversations closed since last run (or last X hours) that haven't received CSAT
	// 2. For each conversation:
	//    - Generate conversation summary
	//    - Send CSAT survey email
	//    - Mark conversation as CSAT sent
	// 3. Track count in ctx.IncrementProcessed()

	log.Info("[%s] CSAT email job completed", ctx.JobName)
	return nil
}

func handleCalculateMetrics(ctx *JobContext) error {
	log.Info("[%s] Starting metrics calculation job", ctx.JobName)

	// TODO: Implement metrics calculation
	// 1. Calculate metrics for yesterday:
	//    - Total conversations
	//    - Average response time
	//    - Resolution rate
	//    - CSAT scores
	//    - Agent performance metrics
	// 2. Store in metrics table
	// 3. Track in ctx.IncrementProcessed()

	log.Info("[%s] Metrics calculation job completed", ctx.JobName)
	return nil
}

func handleCloseUnresponded(ctx *JobContext) error {
	log.Info("[%s] Starting close unresponded tickets job", ctx.JobName)

	// TODO: Implement auto-close logic
	// 1. Get auto-close settings (time period, status conditions)
	// 2. Find tickets matching criteria:
	//    - Status = waiting_for_customer
	//    - Last message from agent > X days ago
	//    - No response from customer
	// 3. Close each ticket with auto-close note
	// 4. Track in ctx.IncrementProcessed()

	log.Info("[%s] Close unresponded tickets job completed", ctx.JobName)
	return nil
}

func handleArchiveOldTickets(ctx *JobContext) error {
	log.Info("[%s] Starting archive old tickets job", ctx.JobName)

	// TODO: Implement archive logic
	// 1. Get archive settings (retention period, status conditions)
	// 2. Find resolved/closed tickets older than retention period
	// 3. Move to archive (set archived = true, archived_at = now)
	// 4. Track in ctx.IncrementProcessed()

	log.Info("[%s] Archive old tickets job completed", ctx.JobName)
	return nil
}

func handleDeleteOldTickets(ctx *JobContext) error {
	log.Info("[%s] Starting delete old tickets job", ctx.JobName)

	// TODO: Implement permanent deletion logic
	// 1. Get deletion settings (retention period for archived tickets)
	// 2. Find archived tickets older than deletion period
	// 3. Permanently delete (with related messages, attachments)
	// 4. Track in ctx.IncrementProcessed()

	log.Info("[%s] Delete old tickets job completed", ctx.JobName)
	return nil
}

func handleCleanupJobExecutions(ctx *JobContext) error {
	log.Info("[%s] Starting job execution cleanup", ctx.JobName)

	scheduler := GetScheduler()
	if scheduler == nil {
		return nil
	}

	// Clean up executions older than 30 days
	deleted, err := scheduler.CleanupOldExecutions(30 * 24 * time.Hour)
	if err != nil {
		return err
	}

	ctx.IncrementProcessed(int(deleted))
	log.Info("[%s] Cleaned up %d old job execution records", ctx.JobName, deleted)
	return nil
}
