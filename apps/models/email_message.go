package models

import (
	"time"

	"github.com/getevo/evo/v2/lib/db"
)

// EmailMessage tracks processed emails to prevent duplicate processing.
type EmailMessage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	IntegrationType string   `gorm:"size:50;not null;index" json:"integration_type"` // smtp, gmail, outlook
	MessageID      string    `gorm:"size:500;not null;uniqueIndex" json:"message_id"` // Email Message-ID header
	ConversationID uint      `gorm:"index" json:"conversation_id,omitempty"`
	MessageRecordID uint     `gorm:"index" json:"message_record_id,omitempty"` // Reference to messages table
	Subject        string    `gorm:"size:500" json:"subject"`
	FromEmail      string    `gorm:"size:255;index" json:"from_email"`
	FromName       string    `gorm:"size:255" json:"from_name"`
	ToEmail        string    `gorm:"size:255" json:"to_email"`
	InReplyTo      string    `gorm:"size:500" json:"in_reply_to,omitempty"`
	References     string    `gorm:"type:text" json:"references,omitempty"` // Space-separated Message-IDs
	Direction      string    `gorm:"size:20;not null" json:"direction"` // inbound, outbound
	ProcessedAt    time.Time `gorm:"autoCreateTime" json:"processed_at"`
	ReceivedAt     time.Time `json:"received_at"`
}

// TableName returns the table name.
func (EmailMessage) TableName() string {
	return "email_messages"
}

// EmailMessageExists checks if an email with the given Message-ID has been processed.
func EmailMessageExists(messageID string) (bool, error) {
	var count int64
	err := db.Model(&EmailMessage{}).Where("message_id = ?", messageID).Count(&count).Error
	return count > 0, err
}

// GetEmailMessageByMessageID retrieves an email record by Message-ID.
func GetEmailMessageByMessageID(messageID string) (*EmailMessage, error) {
	var email EmailMessage
	err := db.Where("message_id = ?", messageID).First(&email).Error
	return &email, err
}

// GetConversationByInReplyTo finds a conversation by checking In-Reply-To header.
func GetConversationByInReplyTo(inReplyTo string) (*Conversation, error) {
	var emailMsg EmailMessage
	err := db.Where("message_id = ?", inReplyTo).First(&emailMsg).Error
	if err != nil {
		return nil, err
	}

	var conv Conversation
	err = db.First(&conv, emailMsg.ConversationID).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetConversationByEmailThread finds a conversation by checking References headers.
func GetConversationByEmailThread(references []string) (*Conversation, error) {
	if len(references) == 0 {
		return nil, nil
	}

	var emailMsg EmailMessage
	err := db.Where("message_id IN ?", references).
		Order("processed_at DESC").
		First(&emailMsg).Error
	if err != nil {
		return nil, err
	}

	var conv Conversation
	err = db.First(&conv, emailMsg.ConversationID).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetConversationByEmailAndSubject finds an open conversation by email and cleaned subject.
func GetConversationByEmailAndSubject(emailAddr, subject string, channel string) (*Conversation, error) {
	// First, find the client by email in external IDs
	var extID ClientExternalID
	err := db.Where("type = ? AND value = ?", ExternalIDTypeEmail, emailAddr).First(&extID).Error
	if err != nil {
		return nil, err
	}

	// Find an open conversation with matching subject (removing Re:, Fwd:, etc.)
	var conv Conversation
	err = db.Where("client_id = ?", extID.ClientID).
		Where("channel_id = ?", channel).
		Where("status NOT IN ?", []string{ConversationStatusClosed, ConversationStatusArchived}).
		Where("title = ?", subject).
		Order("updated_at DESC").
		First(&conv).Error

	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// CreateEmailMessage creates a new email tracking record.
func CreateEmailMessage(msg *EmailMessage) error {
	return db.Create(msg).Error
}

// GetLastProcessedTime returns the time of the last processed email for an integration.
func GetLastProcessedTime(integrationType string) (time.Time, error) {
	var email EmailMessage
	err := db.Where("integration_type = ?", integrationType).
		Where("direction = ?", "inbound").
		Order("received_at DESC").
		First(&email).Error

	if err != nil {
		// Return a time 24 hours ago if no emails found
		return time.Now().Add(-24 * time.Hour), nil
	}

	return email.ReceivedAt, nil
}

// CleanupOldEmailMessages deletes email tracking records older than the specified days.
func CleanupOldEmailMessages(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := db.Where("processed_at < ?", cutoff).Delete(&EmailMessage{})
	return result.RowsAffected, result.Error
}
