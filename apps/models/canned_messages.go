package models

import (
	"github.com/getevo/restify"
	"time"
)

type CannedMessage struct {
	ID        uint      `gorm:"column:id;primaryKey" json:"id"`
	Title     string    `gorm:"column:title;size:255;not null" json:"title"`
	Message   string    `gorm:"column:message;type:text;not null" json:"message"`
	Shortcut  *string   `gorm:"column:shortcut;size:50;uniqueIndex" json:"shortcut"`
	IsActive  bool      `gorm:"column:is_active;default:1" json:"is_active"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	restify.API
}
