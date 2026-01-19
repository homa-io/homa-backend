package jobs

import (
	"context"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
)

// Job names as constants for consistency
const (
	JobSendCSATEmails       = "send_csat_emails"
	JobCalculateMetrics     = "calculate_metrics"
	JobCloseUnresponded     = "close_unresponded_tickets"
	JobArchiveOldTickets    = "archive_old_tickets"
	JobDeleteOldTickets     = "delete_old_tickets"
	JobCleanupJobExecutions = "cleanup_job_executions"
	// JobFetchEmailMessages is defined in email_fetch.go
)

// RegisterAllJobs registers all background jobs with the registry
func RegisterAllJobs() {
	registry := GetRegistry()

	// Send CSAT emails for closed conversations
	registry.Register(JobDefinition{
		Name:           JobSendCSATEmails,
		Description:    "Send CSAT survey emails and conversation summaries for recently closed conversations",
		TimeoutSeconds: 600, // 10 minutes
		Handler:        handleSendCSATEmails,
	})

	// Calculate daily metrics
	registry.Register(JobDefinition{
		Name:           JobCalculateMetrics,
		Description:    "Calculate and store daily metrics for reporting",
		TimeoutSeconds: 1800, // 30 minutes
		Handler:        handleCalculateMetrics,
	})

	// Close unresponded tickets
	registry.Register(JobDefinition{
		Name:           JobCloseUnresponded,
		Description:    "Close tickets that have had no response within the configured time period",
		TimeoutSeconds: 900, // 15 minutes
		Handler:        handleCloseUnresponded,
	})

	// Archive old tickets
	registry.Register(JobDefinition{
		Name:           JobArchiveOldTickets,
		Description:    "Archive resolved tickets older than the configured retention period",
		TimeoutSeconds: 1800, // 30 minutes
		Handler:        handleArchiveOldTickets,
	})

	// Delete very old tickets
	registry.Register(JobDefinition{
		Name:           JobDeleteOldTickets,
		Description:    "Permanently delete archived tickets older than the configured deletion period",
		TimeoutSeconds: 3600, // 60 minutes
		Handler:        handleDeleteOldTickets,
	})

	// Cleanup old job execution records
	registry.Register(JobDefinition{
		Name:           JobCleanupJobExecutions,
		Description:    "Clean up job execution history older than 7 days",
		TimeoutSeconds: 300, // 5 minutes
		Handler:        handleCleanupJobExecutions,
	})

	// Register email fetch job (defined in email_fetch.go)
	RegisterEmailFetchJob()

	log.Info("[jobs] Registered %d jobs", registry.Count())
}

// Job handlers

// CSATResult is the result of the CSAT email job
type CSATResult struct {
	EmailsSent int `json:"emails_sent"`
	Skipped    int `json:"skipped"`
}

func handleSendCSATEmails(ctx context.Context) (interface{}, error) {
	log.Info("[%s] Starting CSAT email job", JobSendCSATEmails)

	result := CSATResult{
		EmailsSent: 0,
		Skipped:    0,
	}

	// TODO: Implement CSAT email sending
	// 1. Query conversations closed since last run (or last X hours) that haven't received CSAT
	// 2. For each conversation:
	//    - Generate conversation summary
	//    - Send CSAT survey email
	//    - Mark conversation as CSAT sent
	// 3. Track count in result

	log.Info("[%s] CSAT email job completed: %d sent, %d skipped",
		JobSendCSATEmails, result.EmailsSent, result.Skipped)
	return result, nil
}

// MetricsResult is the result of the metrics calculation job
type MetricsResult struct {
	MetricsCalculated int    `json:"metrics_calculated"`
	Date              string `json:"date"`
}

func handleCalculateMetrics(ctx context.Context) (interface{}, error) {
	log.Info("[%s] Starting metrics calculation job", JobCalculateMetrics)

	result := MetricsResult{
		MetricsCalculated: 0,
		Date:              "",
	}

	// TODO: Implement metrics calculation
	// 1. Calculate metrics for yesterday:
	//    - Total conversations
	//    - Average response time
	//    - Resolution rate
	//    - CSAT scores
	//    - Agent performance metrics
	// 2. Store in metrics table
	// 3. Track in result

	log.Info("[%s] Metrics calculation job completed: %d metrics",
		JobCalculateMetrics, result.MetricsCalculated)
	return result, nil
}

// CloseTicketsResult is the result of the close unresponded tickets job
type CloseTicketsResult struct {
	ChatsClosed  int `json:"chats_closed"`
	EmailsClosed int `json:"emails_closed"`
}

func handleCloseUnresponded(ctx context.Context) (interface{}, error) {
	log.Info("[%s] Starting close unresponded tickets job", JobCloseUnresponded)

	result := CloseTicketsResult{
		ChatsClosed:  0,
		EmailsClosed: 0,
	}

	now := time.Now()

	// Get settings for chat and email
	chatEnabled, chatAfterHours := GetCloseChatSettings()
	emailEnabled, emailAfterHours := GetCloseEmailSettings()

	// Close web conversations with inbox-specific timeouts
	inboxes, err := models.GetEnabledInboxes()
	if err != nil {
		log.Error("[%s] Failed to get inboxes: %v", JobCloseUnresponded, err)
	} else {
		for _, inbox := range inboxes {
			// Skip if inbox has no timeout configured
			if inbox.ConversationTimeout <= 0 {
				continue
			}

			inboxCutoff := now.Add(-time.Duration(inbox.ConversationTimeout) * time.Hour)

			var webChatsToClose []models.Conversation
			err := db.Where("channel_id = ?", "web").
				Where("inbox_id = ?", inbox.ID).
				Where("status = ?", models.ConversationStatusWaitForUser).
				Where("updated_at < ?", inboxCutoff).
				Find(&webChatsToClose).Error

			if err != nil {
				log.Error("[%s] Failed to query web chats for inbox %d: %v", JobCloseUnresponded, inbox.ID, err)
				continue
			}

			for _, conv := range webChatsToClose {
				select {
				case <-ctx.Done():
					log.Warning("[%s] Job cancelled", JobCloseUnresponded)
					return result, ctx.Err()
				default:
				}

				closedAt := now
				err := db.Model(&models.Conversation{}).
					Where("id = ?", conv.ID).
					Updates(map[string]interface{}{
						"status":    models.ConversationStatusClosed,
						"closed_at": closedAt,
					}).Error

				if err != nil {
					log.Error("[%s] Failed to close web chat %d: %v", JobCloseUnresponded, conv.ID, err)
					continue
				}

				models.CreateActionMessage(conv.ID, nil, "", "Auto-closed due to inactivity (Inbox: "+inbox.Name+")")
				result.ChatsClosed++
			}
		}
	}

	// Close other chat conversations (telegram, whatsapp, slack) using global settings
	// Also handle web chats without inbox
	if chatEnabled && chatAfterHours > 0 {
		chatCutoff := now.Add(-time.Duration(chatAfterHours) * time.Hour)

		// Non-web chat channels
		otherChatChannels := []string{"telegram", "whatsapp", "slack"}
		var otherChatsToClose []models.Conversation
		err := db.Where("channel_id IN ?", otherChatChannels).
			Where("status = ?", models.ConversationStatusWaitForUser).
			Where("updated_at < ?", chatCutoff).
			Find(&otherChatsToClose).Error

		if err != nil {
			log.Error("[%s] Failed to query other chats to close: %v", JobCloseUnresponded, err)
		} else {
			for _, conv := range otherChatsToClose {
				select {
				case <-ctx.Done():
					log.Warning("[%s] Job cancelled", JobCloseUnresponded)
					return result, ctx.Err()
				default:
				}

				closedAt := now
				err := db.Model(&models.Conversation{}).
					Where("id = ?", conv.ID).
					Updates(map[string]interface{}{
						"status":    models.ConversationStatusClosed,
						"closed_at": closedAt,
					}).Error

				if err != nil {
					log.Error("[%s] Failed to close chat %d: %v", JobCloseUnresponded, conv.ID, err)
					continue
				}

				models.CreateActionMessage(conv.ID, nil, "", "Auto-closed due to inactivity")
				result.ChatsClosed++
			}
		}

		// Web chats without inbox (inbox_id is NULL) - use global chat settings
		var webNoInboxChats []models.Conversation
		err = db.Where("channel_id = ?", "web").
			Where("inbox_id IS NULL").
			Where("status = ?", models.ConversationStatusWaitForUser).
			Where("updated_at < ?", chatCutoff).
			Find(&webNoInboxChats).Error

		if err != nil {
			log.Error("[%s] Failed to query web chats without inbox: %v", JobCloseUnresponded, err)
		} else {
			for _, conv := range webNoInboxChats {
				select {
				case <-ctx.Done():
					log.Warning("[%s] Job cancelled", JobCloseUnresponded)
					return result, ctx.Err()
				default:
				}

				closedAt := now
				err := db.Model(&models.Conversation{}).
					Where("id = ?", conv.ID).
					Updates(map[string]interface{}{
						"status":    models.ConversationStatusClosed,
						"closed_at": closedAt,
					}).Error

				if err != nil {
					log.Error("[%s] Failed to close web chat %d: %v", JobCloseUnresponded, conv.ID, err)
					continue
				}

				models.CreateActionMessage(conv.ID, nil, "", "Auto-closed due to inactivity")
				result.ChatsClosed++
			}
		}
	}

	// Close email conversations
	if emailEnabled && emailAfterHours > 0 {
		emailCutoff := now.Add(-time.Duration(emailAfterHours) * time.Hour)

		var emailsToClose []models.Conversation
		err := db.Where("channel_id = ?", "email").
			Where("status = ?", models.ConversationStatusWaitForUser).
			Where("updated_at < ?", emailCutoff).
			Find(&emailsToClose).Error

		if err != nil {
			log.Error("[%s] Failed to query emails to close: %v", JobCloseUnresponded, err)
		} else {
			for _, conv := range emailsToClose {
				select {
				case <-ctx.Done():
					log.Warning("[%s] Job cancelled", JobCloseUnresponded)
					return result, ctx.Err()
				default:
				}

				closedAt := now
				err := db.Model(&models.Conversation{}).
					Where("id = ?", conv.ID).
					Updates(map[string]interface{}{
						"status":    models.ConversationStatusClosed,
						"closed_at": closedAt,
					}).Error

				if err != nil {
					log.Error("[%s] Failed to close email %d: %v", JobCloseUnresponded, conv.ID, err)
					continue
				}

				models.CreateActionMessage(conv.ID, nil, "", "Auto-closed due to inactivity")
				result.EmailsClosed++
			}
		}
	}

	log.Info("[%s] Close unresponded tickets job completed: %d chats closed, %d emails closed",
		JobCloseUnresponded, result.ChatsClosed, result.EmailsClosed)
	return result, nil
}

// ArchiveTicketsResult is the result of the archive old tickets job
type ArchiveTicketsResult struct {
	TicketsArchived int `json:"tickets_archived"`
}

func handleArchiveOldTickets(ctx context.Context) (interface{}, error) {
	log.Info("[%s] Starting archive old tickets job", JobArchiveOldTickets)

	result := ArchiveTicketsResult{
		TicketsArchived: 0,
	}

	// Get archive settings
	enabled, afterDays := GetArchiveSettings()
	if !enabled || afterDays <= 0 {
		log.Info("[%s] Archive is disabled or after_days is 0, skipping", JobArchiveOldTickets)
		return result, nil
	}

	now := time.Now()
	cutoff := now.AddDate(0, 0, -afterDays)

	// Find conversations that:
	// 1. Status is closed or resolved
	// 2. closed_at is before cutoff time
	// 3. Not already archived (status != 'archived')
	var conversationsToArchive []models.Conversation
	err := db.Where("status IN ?", []string{models.ConversationStatusClosed, models.ConversationStatusResolved}).
		Where("closed_at IS NOT NULL").
		Where("closed_at < ?", cutoff).
		Find(&conversationsToArchive).Error

	if err != nil {
		log.Error("[%s] Failed to query conversations to archive: %v", JobArchiveOldTickets, err)
		return result, err
	}

	log.Info("[%s] Found %d conversations to archive", JobArchiveOldTickets, len(conversationsToArchive))

	for _, conv := range conversationsToArchive {
		select {
		case <-ctx.Done():
			log.Warning("[%s] Job cancelled", JobArchiveOldTickets)
			return result, ctx.Err()
		default:
		}

		err := db.Model(&models.Conversation{}).
			Where("id = ?", conv.ID).
			Update("status", models.ConversationStatusArchived).Error

		if err != nil {
			log.Error("[%s] Failed to archive conversation %d: %v", JobArchiveOldTickets, conv.ID, err)
			continue
		}

		// Create action message
		models.CreateActionMessage(conv.ID, nil, "", "Auto-archived due to retention policy")
		result.TicketsArchived++
	}

	log.Info("[%s] Archive old tickets job completed: %d archived",
		JobArchiveOldTickets, result.TicketsArchived)
	return result, nil
}

// DeleteTicketsResult is the result of the delete old tickets job
type DeleteTicketsResult struct {
	TicketsDeleted  int `json:"tickets_deleted"`
	MessagesDeleted int `json:"messages_deleted"`
}

func handleDeleteOldTickets(ctx context.Context) (interface{}, error) {
	log.Info("[%s] Starting delete old tickets job", JobDeleteOldTickets)

	result := DeleteTicketsResult{
		TicketsDeleted:  0,
		MessagesDeleted: 0,
	}

	// Get delete archived settings
	enabled, afterDays := GetDeleteArchivedSettings()
	if !enabled || afterDays <= 0 {
		log.Info("[%s] Delete archived is disabled or after_days is 0, skipping", JobDeleteOldTickets)
		return result, nil
	}

	now := time.Now()
	cutoff := now.AddDate(0, 0, -afterDays)

	// Find archived conversations older than deletion period
	// Use closed_at since archived conversations were closed before being archived
	var conversationsToDelete []models.Conversation
	err := db.Where("status = ?", models.ConversationStatusArchived).
		Where("closed_at IS NOT NULL").
		Where("closed_at < ?", cutoff).
		Find(&conversationsToDelete).Error

	if err != nil {
		log.Error("[%s] Failed to query conversations to delete: %v", JobDeleteOldTickets, err)
		return result, err
	}

	log.Info("[%s] Found %d conversations to permanently delete", JobDeleteOldTickets, len(conversationsToDelete))

	for _, conv := range conversationsToDelete {
		select {
		case <-ctx.Done():
			log.Warning("[%s] Job cancelled", JobDeleteOldTickets)
			return result, ctx.Err()
		default:
		}

		// Delete related data in order (to respect foreign keys)
		// 1. Delete messages
		var msgCount int64
		db.Model(&models.Message{}).Where("conversation_id = ?", conv.ID).Count(&msgCount)
		if err := db.Unscoped().Where("conversation_id = ?", conv.ID).Delete(&models.Message{}).Error; err != nil {
			log.Error("[%s] Failed to delete messages for conversation %d: %v", JobDeleteOldTickets, conv.ID, err)
			continue
		}
		result.MessagesDeleted += int(msgCount)

		// 2. Delete conversation_tags (many-to-many join table)
		if err := db.Exec("DELETE FROM conversation_tags WHERE conversation_id = ?", conv.ID).Error; err != nil {
			log.Error("[%s] Failed to delete tags for conversation %d: %v", JobDeleteOldTickets, conv.ID, err)
		}

		// 3. Delete conversation assignments
		if err := db.Unscoped().Where("conversation_id = ?", conv.ID).Delete(&models.ConversationAssignment{}).Error; err != nil {
			log.Error("[%s] Failed to delete assignments for conversation %d: %v", JobDeleteOldTickets, conv.ID, err)
		}

		// 4. Delete the conversation itself (hard delete)
		if err := db.Unscoped().Delete(&conv).Error; err != nil {
			log.Error("[%s] Failed to delete conversation %d: %v", JobDeleteOldTickets, conv.ID, err)
			continue
		}

		result.TicketsDeleted++
		log.Debug("[%s] Deleted conversation %d with %d messages", JobDeleteOldTickets, conv.ID, msgCount)
	}

	log.Info("[%s] Delete old tickets job completed: %d tickets deleted, %d messages deleted",
		JobDeleteOldTickets, result.TicketsDeleted, result.MessagesDeleted)
	return result, nil
}

// CleanupResult is the result of the cleanup job executions job
type CleanupResult struct {
	LogsDeleted int64 `json:"logs_deleted"`
}

func handleCleanupJobExecutions(ctx context.Context) (interface{}, error) {
	log.Info("[%s] Starting job execution cleanup", JobCleanupJobExecutions)

	deleted, err := ForceCleanupOldLogs()
	if err != nil {
		return nil, err
	}

	result := CleanupResult{
		LogsDeleted: deleted,
	}

	log.Info("[%s] Cleaned up %d old job execution records", JobCleanupJobExecutions, deleted)
	return result, nil
}
