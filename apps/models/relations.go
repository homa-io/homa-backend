package models

import (
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/getevo/restify"
	"github.com/google/uuid"
)

type ConversationAssignment struct {
	ConversationID uint       `gorm:"column:conversation_id;primaryKey;fk:conversations" json:"conversation_id"`
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

	// Relationships
	User       auth.User  `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
	Department Department `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`

	restify.API
}

func (UserDepartment) TableName() string {
	return "user_departments"
}
