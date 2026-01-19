package conversation

import (
	"strings"
	"time"
	"unicode"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/models"
)

// getUnreadCountForConversation returns the number of unread messages for a user in a conversation
func getUnreadCountForConversation(userID uuid.UUID, conversationID uint) int64 {
	// Get user's last read timestamp for this conversation
	var readStatus models.ConversationReadStatus
	err := db.Where("user_id = ? AND conversation_id = ?", userID, conversationID).First(&readStatus).Error

	var count int64
	if err != nil {
		// No read status means all messages are unread (excluding user's own messages)
		db.Model(&models.Message{}).
			Where("conversation_id = ? AND user_id != ? AND user_id IS NOT NULL", conversationID, userID).
			Count(&count)
	} else {
		// Count messages after last read time (excluding user's own messages)
		db.Model(&models.Message{}).
			Where("conversation_id = ? AND created_at > ? AND user_id != ? AND user_id IS NOT NULL", conversationID, readStatus.LastReadAt, userID).
			Count(&count)
	}
	return count
}

// getTotalUnreadCount returns the total unread count across all conversations for a user
func getTotalUnreadCount(userID uuid.UUID) int64 {
	var total int64

	// Get all conversations where the user is assigned
	var conversationIDs []uint
	db.Model(&models.ConversationAssignment{}).
		Where("user_id = ?", userID).
		Pluck("conversation_id", &conversationIDs)

	for _, convID := range conversationIDs {
		total += getUnreadCountForConversation(userID, convID)
	}

	return total
}

// markConversationAsRead updates or creates a read status record for a user/conversation
func markConversationAsRead(userID uuid.UUID, conversationID uint) error {
	readStatus := models.ConversationReadStatus{
		UserID:         userID,
		ConversationID: conversationID,
		LastReadAt:     time.Now(),
	}

	// Use upsert: update if exists, create if not
	return db.Save(&readStatus).Error
}

// isValidAttributeName checks if attribute name is valid (alphanumeric and underscore only)
func isValidAttributeName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// getInitials extracts initials from a name
func getInitials(name string) string {
	if len(name) == 0 {
		return ""
	}
	parts := strings.Fields(name)
	var initials string
	if len(parts) >= 2 {
		initials = string(parts[0][0]) + string(parts[1][0])
	} else if len(parts) == 1 && len(parts[0]) > 0 {
		initials = string(parts[0][0])
	}
	return strings.ToUpper(initials)
}
