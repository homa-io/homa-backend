package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/restify"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/nats"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Conversation status constants
const (
	ConversationStatusNew          = "new"
	ConversationStatusWaitForAgent = "wait_for_agent"
	ConversationStatusInProgress   = "in_progress"
	ConversationStatusWaitForUser  = "wait_for_user"
	ConversationStatusOnHold       = "on_hold"
	ConversationStatusResolved     = "resolved"
	ConversationStatusClosed       = "closed"
	ConversationStatusUnresolved   = "unresolved"
	ConversationStatusSpam         = "spam"
)

// Conversation priority constants
const (
	ConversationPriorityLow    = "low"
	ConversationPriorityMedium = "medium"
	ConversationPriorityHigh   = "high"
	ConversationPriorityUrgent = "urgent"
)

type Conversation struct {
	ID           uint           `gorm:"column:id;primaryKey" json:"id"`
	Title        string         `gorm:"column:title;size:255;not null" json:"title"`
	ClientID     uuid.UUID      `gorm:"column:client_id;type:char(36);not null;index;fk:clients" json:"client_id"`
	DepartmentID *uint          `gorm:"column:department_id;index;fk:departments" json:"department_id"`
	ChannelID    string         `gorm:"column:channel_id;size:50;not null;index;fk:channels" json:"channel_id"`
	ExternalID   *string        `gorm:"column:external_id;size:255;index" json:"external_id"`
	Secret       string         `gorm:"column:secret;size:32;not null" json:"secret"` // Exposed in JSON - conversation_id + secret acts as credentials
	Status          string         `gorm:"column:status;size:50;not null;check:status IN ('new','wait_for_agent','in_progress','wait_for_user','on_hold','resolved','closed','unresolved','spam')" json:"status"`
	Priority        string         `gorm:"column:priority;size:50;not null;check:priority IN ('low','medium','high','urgent')" json:"priority"`
	CustomFields    datatypes.JSON `gorm:"column:custom_fields;type:json" json:"custom_fields"`
	IP              *string        `gorm:"column:ip;size:45" json:"ip"`
	Browser         *string        `gorm:"column:browser;size:255" json:"browser"`
	OperatingSystem *string        `gorm:"column:operating_system;size:255" json:"operating_system"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	ClosedAt        *time.Time     `gorm:"column:closed_at" json:"closed_at"`

	// Relationships
	Client      Client                   `gorm:"foreignKey:ClientID;references:ID" json:"client,omitempty"`
	Department  *Department              `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`
	Channel     Channel                  `gorm:"foreignKey:ChannelID;references:ID" json:"channel,omitempty"`
	Messages    []Message                `gorm:"foreignKey:ConversationID;references:ID" json:"messages,omitempty"`
	Tags        []Tag                    `gorm:"many2many:conversation_tags;foreignKey:ID;joinForeignKey:ConversationID;references:ID;joinReferences:TagID" json:"tags,omitempty"`
	Assignments []ConversationAssignment `gorm:"foreignKey:ConversationID;references:ID" json:"assignments,omitempty"`

	restify.API
}

type Message struct {
	ID              uint       `gorm:"column:id;primaryKey" json:"id"`
	ConversationID  uint       `gorm:"column:conversation_id;not null;index;fk:conversations" json:"conversation_id"`
	UserID          *uuid.UUID `gorm:"column:user_id;type:char(36);index;fk:users" json:"user_id"`
	ClientID        *uuid.UUID `gorm:"column:client_id;type:char(36);index;fk:clients" json:"client_id"`
	Body            string     `gorm:"column:body;type:text;not null" json:"body"`
	IsSystemMessage bool       `gorm:"column:is_system_message;default:0" json:"is_system_message"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`
	User         *auth.User   `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
	Client       *Client      `gorm:"foreignKey:ClientID;references:ID" json:"client,omitempty"`

	restify.API
}

type Tag struct {
	ID   uint   `gorm:"column:id;primaryKey" json:"id"`
	Name string `gorm:"column:name;size:100;uniqueIndex;not null" json:"name"`

	// Relationships
	Conversations []Conversation `gorm:"many2many:conversation_tags;foreignKey:ID;joinForeignKey:TagID;references:ID;joinReferences:ConversationID" json:"conversations,omitempty"`

	restify.API
}

// ToWebhookData creates a clean conversation map for webhook payloads
// excluding empty relationships and only including client with external IDs
func (c *Conversation) ToWebhookData() map[string]any {
	data := map[string]any{
		"id":               c.ID,
		"title":            c.Title,
		"client_id":        c.ClientID,
		"department_id":    c.DepartmentID,
		"channel_id":       c.ChannelID,
		"external_id":      c.ExternalID,
		"status":           c.Status,
		"priority":         c.Priority,
		"custom_fields":    c.CustomFields,
		"ip":               c.IP,
		"browser":          c.Browser,
		"operating_system": c.OperatingSystem,
		"created_at":       c.CreatedAt,
		"updated_at":       c.UpdatedAt,
		"closed_at":        c.ClosedAt,
	}

	// Include client if loaded (non-zero ID)
	if c.Client.ID != uuid.Nil {
		clientData := map[string]any{
			"id":         c.Client.ID,
			"name":       c.Client.Name,
			"avatar":     c.Client.Avatar,
			"data":       c.Client.Data,
			"language":   c.Client.Language,
			"timezone":   c.Client.Timezone,
			"created_at": c.Client.CreatedAt,
			"updated_at": c.Client.UpdatedAt,
		}
		// Include external IDs if loaded
		if len(c.Client.ExternalIDs) > 0 {
			clientData["external_ids"] = c.Client.ExternalIDs
		}
		data["client"] = clientData
	}

	return data
}

// GORM Hooks for Conversation

// AfterCreate hook - broadcast conversation creation to NATS and webhooks
func (c *Conversation) AfterCreate(tx *gorm.DB) error {
	// Broadcast to NATS
	go func() {
		subject := fmt.Sprintf("conversation.%d", c.ID)
		data, _ := json.Marshal(map[string]interface{}{
			"event":        "conversation.created",
			"conversation": c,
		})
		if err := nats.Publish(subject, data); err != nil {
			log.Error("Failed to publish conversation.created to NATS: %v", err)
		}
	}()

	// Trigger webhook with clean conversation data
	go func() {
		// Fetch conversation with client and client's external IDs
		var conversation Conversation
		if err := db.Preload("Client").Preload("Client.ExternalIDs").First(&conversation, c.ID).Error; err == nil {
			BroadcastWebhook(WebhookEventConversationCreated, map[string]any{
				"conversation": conversation.ToWebhookData(),
			})
		} else {
			// Fallback with just the conversation
			BroadcastWebhook(WebhookEventConversationCreated, map[string]any{
				"conversation": c.ToWebhookData(),
			})
		}
	}()

	return nil
}

// AfterUpdate hook - broadcast conversation update to NATS and webhooks
func (c *Conversation) AfterUpdate(tx *gorm.DB) error {
	// Broadcast to NATS
	go func() {
		subject := fmt.Sprintf("conversation.%d", c.ID)
		data, _ := json.Marshal(map[string]interface{}{
			"event":        "conversation.updated",
			"conversation": c,
		})
		if err := nats.Publish(subject, data); err != nil {
			log.Error("Failed to publish conversation.updated to NATS: %v", err)
		}
	}()

	// Fetch full conversation with client for webhooks
	go func() {
		var conversation Conversation
		if err := db.Preload("Client").Preload("Client.ExternalIDs").First(&conversation, c.ID).Error; err != nil {
			// Fallback to original conversation if fetch fails
			conversation = *c
		}

		convData := conversation.ToWebhookData()

		// Check if status changed
		if tx.Statement.Changed("Status") {
			// Get old value from select clause
			var oldConv Conversation
			if err := tx.Session(&gorm.Session{}).Clauses(clause.Returning{}).Where("id = ?", c.ID).First(&oldConv).Error; err == nil {
				if oldConv.Status != c.Status {
					BroadcastWebhook(WebhookEventConversationStatusChange, map[string]any{
						"conversation": convData,
						"old_status":   oldConv.Status,
						"new_status":   c.Status,
					})
				}
			}

			// Check if conversation is closed
			if c.Status == ConversationStatusClosed {
				BroadcastWebhook(WebhookEventConversationClosed, map[string]any{
					"conversation": convData,
				})
			}
		}

		// Trigger general update webhook with clean conversation data
		BroadcastWebhook(WebhookEventConversationUpdated, map[string]any{
			"conversation": convData,
		})
	}()

	return nil
}

// GORM Hooks for Message

// AfterCreate hook - broadcast message creation to NATS and webhooks
func (m *Message) AfterCreate(tx *gorm.DB) error {
	// Broadcast to NATS
	go func() {
		subject := fmt.Sprintf("conversation.%d", m.ConversationID)

		// Include sender information for client-side filtering
		var senderID string
		var senderType string
		if m.UserID != nil {
			senderID = m.UserID.String()
			senderType = "agent"
		} else if m.ClientID != nil {
			senderID = m.ClientID.String()
			senderType = "client"
		}

		data, _ := json.Marshal(map[string]interface{}{
			"event":       "message.created",
			"message":     m,
			"sender_id":   senderID,
			"sender_type": senderType,
		})
		if err := nats.Publish(subject, data); err != nil {
			log.Error("Failed to publish message.created to NATS: %v", err)
		}
	}()

	// Trigger webhook with message and conversation (with client)
	go func() {
		// Create clean message map without nested relationships
		messageData := map[string]any{
			"id":                m.ID,
			"conversation_id":   m.ConversationID,
			"user_id":           m.UserID,
			"client_id":         m.ClientID,
			"body":              m.Body,
			"is_system_message": m.IsSystemMessage,
			"created_at":        m.CreatedAt,
		}

		// Fetch the full conversation with client and client's external IDs for the webhook
		var conversation Conversation
		if err := db.Preload("Client").Preload("Client.ExternalIDs").First(&conversation, m.ConversationID).Error; err == nil {
			BroadcastWebhook(WebhookEventMessageCreated, map[string]any{
				"message":      messageData,
				"conversation": conversation.ToWebhookData(),
			})
		} else {
			// Fallback if conversation fetch fails
			BroadcastWebhook(WebhookEventMessageCreated, map[string]any{
				"message": messageData,
			})
		}
	}()

	return nil
}
