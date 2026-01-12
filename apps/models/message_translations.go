package models

import (
	"time"

	"github.com/getevo/restify"
)

// Translation type constants
const (
	TranslationTypeIncoming = "incoming"
	TranslationTypeOutgoing = "outgoing"
)

// ConversationMessageTranslation stores translated messages for multilingual support
type ConversationMessageTranslation struct {
	ID             uint      `gorm:"column:id;primaryKey" json:"id"`
	ConversationID uint      `gorm:"column:conversation_id;not null;index" json:"conversation_id"`
	MessageID      uint      `gorm:"column:message_id;not null;index" json:"message_id"`
	FromLang       string    `gorm:"column:from_lang;size:10;not null" json:"from_lang"`
	ToLang         string    `gorm:"column:to_lang;size:10;not null" json:"to_lang"`
	Content        string    `gorm:"column:content;type:text;not null" json:"content"`
	Type           string    `gorm:"column:type;type:enum('incoming','outgoing');not null" json:"type"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`
	Message      Message      `gorm:"foreignKey:MessageID;references:ID" json:"message,omitempty"`

	restify.API
}

func (ConversationMessageTranslation) TableName() string {
	return "conversation_message_translations"
}

// TranslationRequest represents a request to translate messages
type TranslationRequest struct {
	MessageIDs []uint `json:"message_ids" validate:"required,min=1"`
	ToLang     string `json:"to_lang" validate:"required"`
}

// TranslationResponse represents a translated message
type TranslationResponse struct {
	MessageID          uint   `json:"message_id"`
	OriginalContent    string `json:"original_content"`
	TranslatedContent  string `json:"translated_content"`
	FromLang           string `json:"from_lang"`
	ToLang             string `json:"to_lang"`
	Type               string `json:"type"`
	IsTranslated       bool   `json:"is_translated"`
}

// BatchTranslationResponse represents the response for batch translation
type BatchTranslationResponse struct {
	Translations []TranslationResponse `json:"translations"`
}
