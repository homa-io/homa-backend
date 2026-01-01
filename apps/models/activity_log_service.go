package models

import (
	"encoding/json"
	"fmt"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
)

// LogActivity creates a new activity log entry asynchronously
func LogActivity(entry ActivityLogEntry) {
	go func() {
		if err := createActivityLog(entry); err != nil {
			log.Error("Failed to create activity log: %v", err)
		}
	}()
}

// LogActivitySync creates a new activity log entry synchronously
func LogActivitySync(entry ActivityLogEntry) error {
	return createActivityLog(entry)
}

// createActivityLog is the internal function that actually creates the log entry
func createActivityLog(entry ActivityLogEntry) error {
	activityLog := ActivityLog{
		EntityType: entry.EntityType,
		EntityID:   entry.EntityID,
		Action:     entry.Action,
		UserID:     entry.UserID,
	}

	// Convert old values to JSON
	if entry.OldValues != nil {
		oldValuesJSON, err := json.Marshal(entry.OldValues)
		if err == nil {
			activityLog.OldValues = oldValuesJSON
		}
	}

	// Convert new values to JSON
	if entry.NewValues != nil {
		newValuesJSON, err := json.Marshal(entry.NewValues)
		if err == nil {
			activityLog.NewValues = newValuesJSON
		}
	}

	// Convert metadata to JSON
	if entry.Metadata != nil {
		metadataJSON, err := json.Marshal(entry.Metadata)
		if err == nil {
			activityLog.Metadata = metadataJSON
		}
	}

	// Set IP address if provided
	if entry.IPAddress != "" {
		activityLog.IPAddress = &entry.IPAddress
	}

	// Set user agent if provided
	if entry.UserAgent != "" {
		activityLog.UserAgent = &entry.UserAgent
	}

	return db.Create(&activityLog).Error
}

// LogConversationCreate logs a conversation creation
func LogConversationCreate(conversationID uint, userID *uuid.UUID, newValues map[string]any, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityConversation,
		EntityID:   fmt.Sprintf("%d", conversationID),
		Action:     ActionCreate,
		UserID:     userID,
		NewValues:  newValues,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogConversationUpdate logs a conversation update
func LogConversationUpdate(conversationID uint, userID *uuid.UUID, oldValues, newValues map[string]any, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityConversation,
		EntityID:   fmt.Sprintf("%d", conversationID),
		Action:     ActionUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogConversationStatusChange logs a conversation status change
func LogConversationStatusChange(conversationID uint, userID *uuid.UUID, oldStatus, newStatus string, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityConversation,
		EntityID:   fmt.Sprintf("%d", conversationID),
		Action:     ActionStatusChange,
		UserID:     userID,
		OldValues:  map[string]any{"status": oldStatus},
		NewValues:  map[string]any{"status": newStatus},
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogConversationAssign logs a user or department assignment to a conversation
func LogConversationAssign(conversationID uint, userID *uuid.UUID, assignedUserID *uuid.UUID, assignedDeptID *uint, ip, userAgent string) {
	metadata := map[string]any{}
	if assignedUserID != nil {
		metadata["assigned_user_id"] = assignedUserID.String()
	}
	if assignedDeptID != nil {
		metadata["assigned_department_id"] = *assignedDeptID
	}

	LogActivity(ActivityLogEntry{
		EntityType: EntityConversation,
		EntityID:   fmt.Sprintf("%d", conversationID),
		Action:     ActionAssign,
		UserID:     userID,
		Metadata:   metadata,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogUserLogin logs a user login event
func LogUserLogin(userID uuid.UUID, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityUser,
		EntityID:   userID.String(),
		Action:     ActionLogin,
		UserID:     &userID,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogUserLogout logs a user logout event
func LogUserLogout(userID uuid.UUID, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityUser,
		EntityID:   userID.String(),
		Action:     ActionLogout,
		UserID:     &userID,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogArticleCreate logs an article creation
func LogArticleCreate(articleID uint, userID *uuid.UUID, title string, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityArticle,
		EntityID:   fmt.Sprintf("%d", articleID),
		Action:     ActionCreate,
		UserID:     userID,
		NewValues:  map[string]any{"title": title},
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogArticleUpdate logs an article update
func LogArticleUpdate(articleID uint, userID *uuid.UUID, oldValues, newValues map[string]any, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityArticle,
		EntityID:   fmt.Sprintf("%d", articleID),
		Action:     ActionUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogSettingChange logs a setting change
func LogSettingChange(key string, userID *uuid.UUID, oldValue, newValue any, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntitySetting,
		EntityID:   key,
		Action:     ActionUpdate,
		UserID:     userID,
		OldValues:  map[string]any{"value": oldValue},
		NewValues:  map[string]any{"value": newValue},
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogWebhookCreate logs a webhook creation
func LogWebhookCreate(webhookID uint, userID *uuid.UUID, name string, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityWebhook,
		EntityID:   fmt.Sprintf("%d", webhookID),
		Action:     ActionCreate,
		UserID:     userID,
		NewValues:  map[string]any{"name": name},
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogWebhookUpdate logs a webhook update
func LogWebhookUpdate(webhookID uint, userID *uuid.UUID, oldValues, newValues map[string]any, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityWebhook,
		EntityID:   fmt.Sprintf("%d", webhookID),
		Action:     ActionUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// LogWebhookDelete logs a webhook deletion
func LogWebhookDelete(webhookID uint, userID *uuid.UUID, name string, ip, userAgent string) {
	LogActivity(ActivityLogEntry{
		EntityType: EntityWebhook,
		EntityID:   fmt.Sprintf("%d", webhookID),
		Action:     ActionDelete,
		UserID:     userID,
		OldValues:  map[string]any{"name": name},
		IPAddress:  ip,
		UserAgent:  userAgent,
	})
}

// GetActivityLogs retrieves activity logs with filtering and pagination
func GetActivityLogs(entityType, entityID, action string, userID *uuid.UUID, limit, offset int) ([]ActivityLog, int64, error) {
	var logs []ActivityLog
	var total int64

	query := db.Model(&ActivityLog{})

	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if entityID != "" {
		query = query.Where("entity_id = ?", entityID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if userID != nil {
		query = query.Where("user_id = ?", userID)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Set defaults for pagination
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	// Get logs with user info
	err := query.
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, total, err
}

// GetEntityActivityLogs retrieves all activity logs for a specific entity
func GetEntityActivityLogs(entityType, entityID string, limit int) ([]ActivityLog, error) {
	var logs []ActivityLog

	if limit <= 0 {
		limit = 50
	}

	err := db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	return logs, err
}
