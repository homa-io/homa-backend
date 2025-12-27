package models

import (
	"github.com/getevo/restify"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

// Client external ID type constants
const (
	ExternalIDTypeEmail    = "email"
	ExternalIDTypePhone    = "phone"
	ExternalIDTypeWhatsapp = "whatsapp"
	ExternalIDTypeSlack    = "slack"
	ExternalIDTypeTelegram = "telegram"
	ExternalIDTypeWeb      = "web"
	ExternalIDTypeChat     = "chat"
)

type Client struct {
	ID        uuid.UUID      `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	Name      string         `gorm:"column:name;size:255;not null" json:"name"`
	Avatar    *string        `gorm:"column:avatar;size:500" json:"avatar"`
	Data      datatypes.JSON `gorm:"column:data;type:json" json:"data"`
	Language  *string        `gorm:"column:language;size:10" json:"language"`
	Timezone  *string        `gorm:"column:timezone;size:50" json:"timezone"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	ExternalIDs   []ClientExternalID `gorm:"foreignKey:ClientID;references:ID" json:"external_ids,omitempty"`
	Conversations []Conversation     `gorm:"foreignKey:ClientID;references:ID" json:"conversations,omitempty"`
	Messages      []Message          `gorm:"foreignKey:ClientID;references:ID" json:"messages,omitempty"`

	restify.API
}

type ClientExternalID struct {
	ID        uint      `gorm:"column:id;primaryKey" json:"id"`
	ClientID  uuid.UUID `gorm:"column:client_id;type:char(36);not null;index;fk:clients" json:"client_id"`
	Type      string    `gorm:"column:type;size:50;not null;check:type IN ('email','phone','whatsapp','slack','telegram','web','chat')" json:"type"`
	Value     string    `gorm:"column:value;size:255;not null" json:"value"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	Client Client `gorm:"foreignKey:ClientID;references:ID" json:"client,omitempty"`

	restify.API
}

// BeforeCreate hook to generate UUID for Client and set default values
func (c *Client) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New()

	// Set default timezone to UTC if not provided
	if c.Timezone == nil {
		utc := "UTC"
		c.Timezone = &utc
	}

	return nil
}

func (ClientExternalID) TableName() string {
	return "client_external_ids"
}
