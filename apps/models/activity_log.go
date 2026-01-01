package models

import (
	"time"

	"github.com/getevo/restify"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"gorm.io/datatypes"
)

// Activity log action constants
const (
	ActionCreate       = "create"
	ActionUpdate       = "update"
	ActionDelete       = "delete"
	ActionStatusChange = "status_change"
	ActionAssign       = "assign"
	ActionUnassign     = "unassign"
	ActionLogin        = "login"
	ActionLogout       = "logout"
	ActionView         = "view"
)

// Activity log entity type constants
const (
	EntityConversation = "conversation"
	EntityClient       = "client"
	EntityUser         = "user"
	EntityDepartment   = "department"
	EntityTag          = "tag"
	EntityWebhook      = "webhook"
	EntityArticle      = "article"
	EntityCategory     = "category"
	EntitySetting      = "setting"
	EntityMessage      = "message"
)

// ActivityLog tracks all changes to entities in the system
type ActivityLog struct {
	ID         uint           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EntityType string         `gorm:"column:entity_type;size:50;not null;index" json:"entity_type"`
	EntityID   string         `gorm:"column:entity_id;size:255;not null;index" json:"entity_id"`
	Action     string         `gorm:"column:action;size:50;not null;index" json:"action"`
	UserID     *uuid.UUID     `gorm:"column:user_id;type:char(36);index;fk:users" json:"user_id"`
	OldValues  datatypes.JSON `gorm:"column:old_values;type:json" json:"old_values,omitempty"`
	NewValues  datatypes.JSON `gorm:"column:new_values;type:json" json:"new_values,omitempty"`
	Metadata   datatypes.JSON `gorm:"column:metadata;type:json" json:"metadata,omitempty"`
	IPAddress  *string        `gorm:"column:ip_address;size:45" json:"ip_address,omitempty"`
	UserAgent  *string        `gorm:"column:user_agent;size:500" json:"user_agent,omitempty"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`

	// Relationships
	User *auth.User `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`

	restify.API
}

func (ActivityLog) TableName() string {
	return "activity_logs"
}

// ActivityLogEntry is a helper struct for creating activity log entries
type ActivityLogEntry struct {
	EntityType string
	EntityID   string
	Action     string
	UserID     *uuid.UUID
	OldValues  map[string]any
	NewValues  map[string]any
	Metadata   map[string]any
	IPAddress  string
	UserAgent  string
}
