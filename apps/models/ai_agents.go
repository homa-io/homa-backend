package models

import (
	"time"

	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/getevo/restify"
	"gorm.io/datatypes"
)

// AI Agent status constants
const (
	AIAgentStatusActive   = "active"
	AIAgentStatusInactive = "inactive"
)

// AI Agent tone constants
const (
	AIAgentToneFormal     = "formal"
	AIAgentToneCasual     = "casual"
	AIAgentToneDetailed   = "detailed"
	AIAgentTonePrecise    = "precise"
	AIAgentToneEmpathetic = "empathetic"
	AIAgentToneTechnical  = "technical"
)

type AIAgent struct {
	ID                uint           `gorm:"column:id;primaryKey" json:"id"`
	Name              string         `gorm:"column:name;size:255;not null" json:"name"`
	BotID             string         `gorm:"column:bot_id;size:255;not null;index" json:"bot_id"`
	HandoverEnabled   bool           `gorm:"column:handover_enabled;not null;default:0" json:"handover_enabled"`
	HandoverUserID    *string        `gorm:"column:handover_user_id;size:255;index" json:"handover_user_id"`         // Deprecated: use HandoverUserIDs
	HandoverUserIDs   datatypes.JSON `gorm:"column:handover_user_ids;type:json" json:"handover_user_ids"`            // JSON array of user IDs
	MultiLanguage     bool           `gorm:"column:multi_language;not null;default:1" json:"multi_language"`
	InternetAccess    bool      `gorm:"column:internet_access;not null;default:0" json:"internet_access"`
	Tone              string    `gorm:"column:tone;size:50;not null;default:'casual'" json:"tone"`
	UseKnowledgeBase  bool      `gorm:"column:use_knowledge_base;not null;default:1" json:"use_knowledge_base"`
	UnitConversion    bool      `gorm:"column:unit_conversion;not null;default:1" json:"unit_conversion"`
	Instructions      string    `gorm:"column:instructions;type:text" json:"instructions"`
	GreetingMessage   string    `gorm:"column:greeting_message;type:text" json:"greeting_message"`
	MaxResponseLength int       `gorm:"column:max_response_length;not null;default:0" json:"max_response_length"`
	ContextWindow     int       `gorm:"column:context_window;not null;default:10" json:"context_window"`
	BlockedTopics     string    `gorm:"column:blocked_topics;type:text" json:"blocked_topics"`
	MaxToolCalls      int       `gorm:"column:max_tool_calls;not null;default:5" json:"max_tool_calls"`
	CollectUserInfo       bool      `gorm:"column:collect_user_info;not null;default:0" json:"collect_user_info"`
	CollectUserInfoFields string    `gorm:"column:collect_user_info_fields;type:text" json:"collect_user_info_fields"`
	HumorLevel        int       `gorm:"column:humor_level;not null;default:50" json:"humor_level"`
	UseEmojis         bool      `gorm:"column:use_emojis;not null;default:0" json:"use_emojis"`
	FormalityLevel    int       `gorm:"column:formality_level;not null;default:50" json:"formality_level"`
	PriorityDetection bool      `gorm:"column:priority_detection;not null;default:0" json:"priority_detection"`
	AutoTagging       bool      `gorm:"column:auto_tagging;not null;default:0" json:"auto_tagging"`
	Status            string    `gorm:"column:status;size:20;not null;default:'active';check:status IN ('active','inactive')" json:"status"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Bot          *auth.User `gorm:"foreignKey:BotID;references:UserID" json:"bot,omitempty"`
	HandoverUser *auth.User `gorm:"foreignKey:HandoverUserID;references:UserID" json:"handover_user,omitempty"`

	restify.API
}

// TableName specifies the table name for GORM
func (AIAgent) TableName() string {
	return "ai_agents"
}
