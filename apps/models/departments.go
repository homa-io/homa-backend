package models

import (
	"time"

	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/getevo/restify"
)

type Department struct {
	ID          uint      `gorm:"column:id;primaryKey" json:"id"`
	Name        string    `gorm:"column:name;size:255;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	Conversations []Conversation `gorm:"foreignKey:DepartmentID" json:"conversations,omitempty"`
	Users         []auth.User    `gorm:"many2many:user_departments;foreignKey:ID;joinForeignKey:DepartmentID;references:UserID;joinReferences:UserID" json:"users,omitempty"`

	restify.API
}
