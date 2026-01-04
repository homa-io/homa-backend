package models

import (
	"time"

	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/getevo/restify"
)

// Department status constants
const (
	DepartmentStatusActive    = "active"
	DepartmentStatusSuspended = "suspended"
)

type Department struct {
	ID          uint      `gorm:"column:id;primaryKey" json:"id"`
	Name        string    `gorm:"column:name;size:255;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	Status      string    `gorm:"column:status;size:20;not null;default:'active';check:status IN ('active','suspended')" json:"status"`
	AIAgentID   *uint     `gorm:"column:ai_agent_id;index" json:"ai_agent_id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Conversations []Conversation `gorm:"foreignKey:DepartmentID" json:"conversations,omitempty"`
	Users         []auth.User    `gorm:"many2many:user_departments;foreignKey:ID;joinForeignKey:DepartmentID;references:UserID;joinReferences:UserID" json:"users,omitempty"`
	AIAgent       *AIAgent       `gorm:"foreignKey:AIAgentID;references:ID" json:"ai_agent,omitempty"`

	restify.API
}
