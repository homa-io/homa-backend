package models

import (
	"time"

	"github.com/getevo/restify"
)

// Webhook represents a webhook subscription
type Webhook struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	URL         string    `gorm:"size:500;not null" json:"url"`
	Secret      string    `gorm:"size:255" json:"-"` // Hidden from JSON responses for security
	Enabled     bool      `gorm:"default:1" json:"enabled"`
	Description string    `gorm:"type:text" json:"description,omitempty"`

	// Event subscriptions - boolean flags for each event type
	EventAll                     bool `gorm:"default:0" json:"event_all"`
	EventConversationCreated     bool `gorm:"default:0" json:"event_conversation_created"`
	EventConversationUpdated     bool `gorm:"default:0" json:"event_conversation_updated"`
	EventConversationStatusChange bool `gorm:"default:0" json:"event_conversation_status_change"`
	EventConversationClosed      bool `gorm:"default:0" json:"event_conversation_closed"`
	EventConversationAssigned    bool `gorm:"default:0" json:"event_conversation_assigned"`
	EventMessageCreated          bool `gorm:"default:0" json:"event_message_created"`
	EventClientCreated           bool `gorm:"default:0" json:"event_client_created"`
	EventClientUpdated           bool `gorm:"default:0" json:"event_client_updated"`
	EventUserCreated             bool `gorm:"default:0" json:"event_user_created"`
	EventUserUpdated             bool `gorm:"default:0" json:"event_user_updated"`

	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	restify.API
}

// IsSubscribedTo checks if the webhook is subscribed to a specific event
func (w *Webhook) IsSubscribedTo(event string) bool {
	// If subscribed to all events, return true
	if w.EventAll {
		return true
	}

	// Test events always pass through
	if event == WebhookEventWebhookTest {
		return true
	}

	// Check specific event subscription
	switch event {
	case WebhookEventConversationCreated:
		return w.EventConversationCreated
	case WebhookEventConversationUpdated:
		return w.EventConversationUpdated
	case WebhookEventConversationStatusChange:
		return w.EventConversationStatusChange
	case WebhookEventConversationClosed:
		return w.EventConversationClosed
	case WebhookEventConversationAssigned:
		return w.EventConversationAssigned
	case WebhookEventMessageCreated:
		return w.EventMessageCreated
	case WebhookEventClientCreated:
		return w.EventClientCreated
	case WebhookEventClientUpdated:
		return w.EventClientUpdated
	case WebhookEventUserCreated:
		return w.EventUserCreated
	case WebhookEventUserUpdated:
		return w.EventUserUpdated
	default:
		return false
	}
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	WebhookID uint      `gorm:"not null;index;fk:webhooks" json:"webhook_id"`
	Event     string    `gorm:"size:100;not null" json:"event"`
	Success   bool      `gorm:"not null" json:"success"`

	// Request details for debugging
	RequestURL     string `gorm:"size:500" json:"request_url,omitempty"`
	RequestBody    string `gorm:"type:text" json:"request_body,omitempty"`
	RequestHeaders string `gorm:"type:text" json:"request_headers,omitempty"`

	// Response details
	StatusCode int    `gorm:"default:0" json:"status_code"`
	Response   string `gorm:"type:text" json:"response,omitempty"`

	// Duration in milliseconds
	DurationMs int64     `gorm:"default:0" json:"duration_ms"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	Webhook Webhook `gorm:"foreignKey:WebhookID;references:ID" json:"webhook,omitempty"`

	restify.API
}

// WebhookEvents defines available webhook event types
const (
	WebhookEventConversationCreated      = "conversation.created"
	WebhookEventConversationUpdated      = "conversation.updated"
	WebhookEventConversationStatusChange = "conversation.status_changed"
	WebhookEventConversationClosed       = "conversation.closed"
	WebhookEventConversationAssigned     = "conversation.assigned"
	WebhookEventMessageCreated           = "message.created"
	WebhookEventClientCreated            = "client.created"
	WebhookEventClientUpdated            = "client.updated"
	WebhookEventUserCreated              = "user.created"
	WebhookEventUserUpdated              = "user.updated"
	WebhookEventWebhookTest              = "webhook.test"
	WebhookEventAll                      = "*"
)
