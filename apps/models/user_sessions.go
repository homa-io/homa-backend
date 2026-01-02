package models

import (
	"time"

	"github.com/getevo/restify"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"gorm.io/datatypes"
)

// UserSession tracks individual browser sessions with tab support
// Active status is determined by last_activity timestamp (active if < 5 min ago)
type UserSession struct {
	ID           uint           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID       uuid.UUID      `gorm:"column:user_id;type:char(36);not null;index;fk:users" json:"user_id"`
	SessionID    string         `gorm:"column:session_id;size:36;not null;index" json:"session_id"` // Browser session (shared across tabs)
	TabID        *string        `gorm:"column:tab_id;size:36;index" json:"tab_id,omitempty"`        // Individual tab ID
	IPAddress    string         `gorm:"column:ip_address;size:45;not null" json:"ip_address"`
	UserAgent    string         `gorm:"column:user_agent;size:500" json:"user_agent"`
	DeviceInfo   datatypes.JSON `gorm:"column:device_info;type:json" json:"device_info,omitempty"`
	StartedAt    time.Time      `gorm:"column:started_at;not null;index" json:"started_at"`
	LastActivity time.Time      `gorm:"column:last_activity;not null;index" json:"last_activity"`

	// Relationships
	User *auth.User `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`

	restify.API
}

func (UserSession) TableName() string {
	return "user_sessions"
}

// UserDailyActivity aggregates daily activity for a user
type UserDailyActivity struct {
	ID                 uint           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID             uuid.UUID      `gorm:"column:user_id;type:char(36);not null;index;fk:users" json:"user_id"`
	ActivityDate       string         `gorm:"column:activity_date;type:date;not null;index" json:"activity_date"` // YYYY-MM-DD
	TotalActiveSeconds int            `gorm:"column:total_active_seconds;not null;default:0" json:"total_active_seconds"`
	ActivePeriods      datatypes.JSON `gorm:"column:active_periods;type:json" json:"active_periods"` // [{from: timestamp, to: timestamp}, ...]
	FirstActivity      time.Time      `gorm:"column:first_activity;not null" json:"first_activity"`
	LastActivity       time.Time      `gorm:"column:last_activity;not null" json:"last_activity"`
	SessionCount       int            `gorm:"column:session_count;not null;default:1" json:"session_count"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	User *auth.User `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`

	restify.API
}

func (UserDailyActivity) TableName() string {
	return "user_daily_activity"
}

// ActivityPeriod represents a single active period
type ActivityPeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// UserActivitySummary provides aggregated activity statistics
type UserActivitySummary struct {
	TotalHours              float64 `json:"total_hours"`
	TotalDays               int     `json:"total_days"`
	AverageHoursPerDay      float64 `json:"average_hours_per_day"`
	ConversationsResponded  int     `json:"conversations_responded"`
	CurrentStreak           int     `json:"current_streak"`
	LongestStreak           int     `json:"longest_streak"`
}
