package models

import (
	"github.com/getevo/restify"
	"gorm.io/datatypes"
	"time"
)

type Channel struct {
	ID            string         `gorm:"column:id;primaryKey;size:50" json:"id"`
	Name          string         `gorm:"column:name;size:255;not null" json:"name"`
	Logo          *string        `gorm:"column:logo;size:500" json:"logo"`
	Configuration datatypes.JSON `gorm:"column:configuration;type:json" json:"configuration"`
	Enabled       bool           `gorm:"column:enabled;default:1" json:"enabled"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Conversations []Conversation `gorm:"foreignKey:ChannelID;references:ID" json:"conversations,omitempty"`

	restify.API
}
