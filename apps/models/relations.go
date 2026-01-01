package models

import (
	"time"

	"github.com/getevo/restify"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
)

type ConversationAssignment struct {
	ID             uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConversationID uint       `gorm:"column:conversation_id;index;fk:conversations" json:"conversation_id"`
	UserID         *uuid.UUID `gorm:"column:user_id;type:char(36);index;fk:users" json:"user_id"`
	DepartmentID   *uint      `gorm:"column:department_id;index;fk:departments" json:"department_id"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`
	User         *auth.User   `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
	Department   *Department  `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`

	restify.API
}

func (ConversationAssignment) TableName() string {
	return "conversation_assignments"
}

type ConversationTag struct {
	ConversationID uint `gorm:"column:conversation_id;primaryKey;fk:conversations" json:"conversation_id"`
	TagID          uint `gorm:"column:tag_id;primaryKey;fk:tags" json:"tag_id"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`
	Tag          Tag          `gorm:"foreignKey:TagID;references:ID" json:"tag,omitempty"`

	restify.API
}

func (ConversationTag) TableName() string {
	return "conversation_tags"
}

type UserDepartment struct {
	UserID       uuid.UUID `gorm:"column:user_id;type:char(36);primaryKey;fk:users" json:"user_id"`
	DepartmentID uint      `gorm:"column:department_id;primaryKey;fk:departments" json:"department_id"`
	Priority     int       `gorm:"column:priority;default:0" json:"priority"`

	// Relationships
	User       auth.User  `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
	Department Department `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`

	restify.API
}

func (UserDepartment) TableName() string {
	return "user_departments"
}

// ConversationReadStatus tracks when each user last read a conversation
type ConversationReadStatus struct {
	UserID         uuid.UUID `gorm:"column:user_id;type:char(36);primaryKey;fk:users" json:"user_id"`
	ConversationID uint      `gorm:"column:conversation_id;primaryKey;fk:conversations" json:"conversation_id"`
	LastReadAt     time.Time `gorm:"column:last_read_at;index" json:"last_read_at"`

	restify.API
}

func (ConversationReadStatus) TableName() string {
	return "conversation_read_status"
}
