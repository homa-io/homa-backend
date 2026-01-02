package models

import (
	"time"

	"github.com/getevo/restify"
)

// ConversationSummary stores AI-generated summaries for conversations
// Version tracks the message count when the summary was generated
// Language stores which language the summary is in (allows multiple summaries per conversation)
type ConversationSummary struct {
	ID             uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConversationID uint      `gorm:"column:conversation_id;not null;index;fk:conversations" json:"conversation_id"`
	Language       string    `gorm:"column:language;size:10;not null;default:'en';index" json:"language"`
	Summary        string    `gorm:"column:summary;type:text;not null" json:"summary"`
	KeyPoints      string    `gorm:"column:key_points;type:text" json:"key_points"` // JSON array of key points
	Version        int       `gorm:"column:version;not null;default:0" json:"version"` // Number of messages when generated
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`

	restify.API
}

func (ConversationSummary) TableName() string {
	return "conversation_summaries"
}
