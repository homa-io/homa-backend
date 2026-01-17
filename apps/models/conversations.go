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

// Language detection function - set by the ai package to avoid circular imports
var DetectMessageLanguage func(text string) string

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

// Message type constants
const (
	MessageTypeMessage = "message"
	MessageTypeAction  = "action"
)

// Conversation priority constants
const (
	ConversationPriorityLow    = "low"
	ConversationPriorityMedium = "medium"
	ConversationPriorityHigh   = "high"
	ConversationPriorityUrgent = "urgent"
)

// Outbound messaging functions - set by the integrations package to avoid circular imports
var (
	SendTelegramMessage func(chatID, text string) error
	SendWhatsAppMessage func(phoneNumber, text string) error
	SendSlackMessage    func(channelID, text string) error
)

// AI Agent processing function - set by the ai package to avoid circular imports
var ProcessIncomingMessage func(message *Message) error

type Conversation struct {
	ID           uint           `gorm:"column:id;primaryKey" json:"id"`
	Title        string         `gorm:"column:title;size:255;not null" json:"title"`
	ClientID     uuid.UUID      `gorm:"column:client_id;type:char(36);not null;index;fk:clients" json:"client_id"`
	DepartmentID *uint          `gorm:"column:department_id;index;fk:departments" json:"department_id"`
	ChannelID    string         `gorm:"column:channel_id;size:50;not null;index;fk:channels" json:"channel_id"`
	ExternalID   *string        `gorm:"column:external_id;size:255;index" json:"external_id"`
	Secret       string         `gorm:"column:secret;size:32;not null" json:"-"` // Hidden from JSON - only returned on creation via CreateConversationResponse
	Status          string         `gorm:"column:status;size:50;not null;index;check:status IN ('new','wait_for_agent','in_progress','wait_for_user','on_hold','resolved','closed','unresolved','spam')" json:"status"`
	Priority        string         `gorm:"column:priority;size:50;not null;index;check:priority IN ('low','medium','high','urgent')" json:"priority"`
	HandleByBot     bool           `gorm:"column:handle_by_bot;default:1" json:"handle_by_bot"`
	CustomFields    datatypes.JSON `gorm:"column:custom_fields;type:json" json:"custom_fields"`
	IP              *string        `gorm:"column:ip;size:45" json:"ip"`
	Browser         *string        `gorm:"column:browser;size:255" json:"browser"`
	OperatingSystem *string        `gorm:"column:operating_system;size:255" json:"operating_system"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime;index" json:"updated_at"`
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
	Type            string     `gorm:"column:type;type:enum('message','action');default:'message';not null" json:"type"`
	Language        string     `gorm:"column:language;size:10" json:"language"`
	IsSystemMessage bool       `gorm:"column:is_system_message;default:0" json:"is_system_message"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation,omitempty"`
	User         *auth.User   `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
	Client       *Client      `gorm:"foreignKey:ClientID;references:ID" json:"client,omitempty"`

	restify.API
}

// CreateActionMessage creates an action message for conversation activity logs
// actorName: name of the person/system performing the action (empty for system actions)
// conversationID: the conversation this action belongs to
// userID: optional user ID if action was performed by a user (nil for system actions)
// action: the action description with variables in quotes, e.g. 'switched status to "Closed"'
func CreateActionMessage(conversationID uint, userID *uuid.UUID, actorName string, action string) error {
	var body string
	if actorName != "" {
		body = fmt.Sprintf("%s %s", actorName, action)
	} else {
		body = action
	}

	message := Message{
		ConversationID:  conversationID,
		UserID:          userID,
		Body:            body,
		Type:            MessageTypeAction,
		IsSystemMessage: true,
	}

	if err := db.Create(&message).Error; err != nil {
		log.Error("Failed to create action message: ", err)
		return err
	}

	return nil
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
		"handle_by_bot":    c.HandleByBot,
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
	// Check if department changed and auto-assign department users
	if tx.Statement.Changed("DepartmentID") && c.DepartmentID != nil {
		go c.assignDepartmentUsers(tx)
	}

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

// assignDepartmentUsers automatically assigns all users from the conversation's department
// This is called when a conversation is created or updated with a department_id
func (c *Conversation) assignDepartmentUsers(tx *gorm.DB) {
	if c.DepartmentID == nil {
		return
	}

	// Get all users assigned to this department
	var userDepartments []UserDepartment
	if err := db.Where("department_id = ?", *c.DepartmentID).Find(&userDepartments).Error; err != nil {
		log.Error("Failed to get department users for auto-assignment: %v", err)
		return
	}

	if len(userDepartments) == 0 {
		return
	}

	// Get existing user assignments for this conversation
	var existingAssignments []ConversationAssignment
	if err := db.Where("conversation_id = ? AND user_id IS NOT NULL", c.ID).Find(&existingAssignments).Error; err != nil {
		log.Error("Failed to get existing assignments: %v", err)
		return
	}

	// Create a map of existing user assignments
	existingUserIDs := make(map[string]bool)
	for _, a := range existingAssignments {
		if a.UserID != nil {
			existingUserIDs[a.UserID.String()] = true
		}
	}

	// Create assignments for users not already assigned
	for _, ud := range userDepartments {
		if existingUserIDs[ud.UserID.String()] {
			continue // Skip if already assigned
		}

		assignment := ConversationAssignment{
			ConversationID: c.ID,
			UserID:         &ud.UserID,
			DepartmentID:   c.DepartmentID,
		}

		if err := db.Create(&assignment).Error; err != nil {
			log.Error("Failed to auto-assign user %s to conversation %d: %v", ud.UserID, c.ID, err)
		} else {
			log.Info("Auto-assigned user %s to conversation %d from department %d", ud.UserID, c.ID, *c.DepartmentID)
		}
	}
}

// GORM Hooks for Message

// BeforeCreate hook - detect message language before saving
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	// Skip language detection for system messages or if already set
	if m.IsSystemMessage || m.Language != "" {
		return nil
	}

	// Detect language if the detection function is available
	if DetectMessageLanguage != nil && m.Body != "" {
		m.Language = DetectMessageLanguage(m.Body)
	}

	// If detection failed (empty result) and this is a customer message,
	// use the conversation's dominant language as fallback
	if m.Language == "" && m.ClientID != nil && m.ConversationID > 0 {
		// Get the most common language from recent customer messages in this conversation
		var dominantLang string
		tx.Raw(`
			SELECT language FROM messages
			WHERE conversation_id = ?
			AND client_id IS NOT NULL
			AND language IS NOT NULL
			AND language != ''
			ORDER BY created_at DESC
			LIMIT 1
		`, m.ConversationID).Scan(&dominantLang)

		if dominantLang != "" {
			m.Language = dominantLang
		}
	}

	return nil
}

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
			// Preload user info for agent messages
			if m.User == nil {
				var msg Message
				if err := db.Preload("User").First(&msg, m.ID).Error; err == nil && msg.User != nil {
					m.User = msg.User
				}
			}
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

	// Send outbound message to external channels (Telegram, WhatsApp, etc.)
	// Only for agent messages (UserID is set, not ClientID)
	log.Info("Message.AfterCreate: ID=%d, UserID=%v, ClientID=%v, IsSystem=%v", m.ID, m.UserID, m.ClientID, m.IsSystemMessage)
	if m.UserID != nil && !m.IsSystemMessage {
		log.Info("Message.AfterCreate: Agent message detected, will check bot handling for conv %d", m.ConversationID)
		go m.sendToExternalChannel()

		// Auto-disable bot handling when a human agent sends a message
		go m.checkAndDisableBotHandling()
	}

	// Process incoming customer messages with AI agent
	// Only for customer messages (ClientID is set, not UserID)
	if m.ClientID != nil && m.UserID == nil && !m.IsSystemMessage {
		if ProcessIncomingMessage != nil {
			go func() {
				if err := ProcessIncomingMessage(m); err != nil {
					log.Error("AI agent processing failed for message %d: %v", m.ID, err)
				}
			}()
		}
	}

	return nil
}

// checkAndDisableBotHandling disables bot handling when a human agent sends a message
func (m *Message) checkAndDisableBotHandling() {
	log.Info("checkAndDisableBotHandling called for message %d, UserID: %v", m.ID, m.UserID)

	if m.UserID == nil {
		log.Info("checkAndDisableBotHandling: UserID is nil, skipping")
		return
	}

	// Get the user to check if they are a bot
	var user auth.User
	if err := db.Where("id = ?", m.UserID.String()).First(&user).Error; err != nil {
		log.Warning("Failed to get user for bot handling check: %v", err)
		return
	}

	log.Info("checkAndDisableBotHandling: User %s (type: %s) sent message", user.DisplayName, user.Type)

	// If the user is a bot, don't disable bot handling
	if user.Type == auth.UserTypeBot {
		log.Info("checkAndDisableBotHandling: User is a bot, skipping")
		return
	}

	// Get the conversation
	var conversation Conversation
	if err := db.First(&conversation, m.ConversationID).Error; err != nil {
		log.Warning("Failed to get conversation for bot handling check: %v", err)
		return
	}

	// If bot handling is already disabled, nothing to do
	if !conversation.HandleByBot {
		log.Info("checkAndDisableBotHandling: Bot handling already disabled for conversation %d", m.ConversationID)
		return
	}

	// Disable bot handling
	if err := db.Model(&Conversation{}).Where("id = ?", m.ConversationID).Update("handle_by_bot", false).Error; err != nil {
		log.Error("Failed to disable bot handling for conversation %d: %v", m.ConversationID, err)
		return
	}

	log.Info("Bot handling disabled for conversation %d because human agent %s sent a message", m.ConversationID, user.DisplayName)
}

// sendToExternalChannel sends the message to the appropriate external channel
func (m *Message) sendToExternalChannel() {
	// Fetch the conversation with client and their external IDs
	var conversation Conversation
	if err := db.Preload("Client").Preload("Client.ExternalIDs").First(&conversation, m.ConversationID).Error; err != nil {
		log.Error("Failed to fetch conversation for outbound message: %v", err)
		return
	}

	// Check the channel and send accordingly
	switch conversation.ChannelID {
	case "telegram":
		// Find the Telegram chat ID from the client's external IDs
		var telegramChatID string
		for _, extID := range conversation.Client.ExternalIDs {
			if extID.Type == ExternalIDTypeTelegram {
				telegramChatID = extID.Value
				break
			}
		}
		if telegramChatID == "" {
			log.Warning("No Telegram chat ID found for client %s", conversation.ClientID)
			return
		}
		if err := SendTelegramMessage(telegramChatID, m.Body); err != nil {
			log.Error("Failed to send Telegram message: %v", err)
		}

	case "whatsapp":
		// Find the WhatsApp phone number from the client's external IDs
		var whatsappPhone string
		for _, extID := range conversation.Client.ExternalIDs {
			if extID.Type == ExternalIDTypeWhatsapp {
				whatsappPhone = extID.Value
				break
			}
		}
		if whatsappPhone == "" {
			log.Warning("No WhatsApp phone found for client %s", conversation.ClientID)
			return
		}
		if err := SendWhatsAppMessage(whatsappPhone, m.Body); err != nil {
			log.Error("Failed to send WhatsApp message: %v", err)
		}

	case "slack":
		// Find the Slack user/channel ID from the client's external IDs
		var slackID string
		for _, extID := range conversation.Client.ExternalIDs {
			if extID.Type == ExternalIDTypeSlack {
				slackID = extID.Value
				break
			}
		}
		if slackID == "" {
			log.Warning("No Slack ID found for client %s", conversation.ClientID)
			return
		}
		if err := SendSlackMessage(slackID, m.Body); err != nil {
			log.Error("Failed to send Slack message: %v", err)
		}

	default:
		// For web chat and other channels, no outbound sending needed
		// The dashboard handles the realtime updates via WebSocket
	}
}
