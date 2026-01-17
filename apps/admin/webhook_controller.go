package admin

import (
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/pagination"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/gorm"
)

// ========================
// WEBHOOK MANAGEMENT APIs
// ========================

// ListWebhooks returns all webhooks
func (c Controller) ListWebhooks(request *evo.Request) any {
	var webhooks []models.Webhook

	err := db.Order("id DESC").Find(&webhooks).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(webhooks)
}

// GetWebhook returns a single webhook by ID
func (c Controller) GetWebhook(request *evo.Request) any {
	id := request.Param("id").String()
	var webhook models.Webhook

	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(webhook)
}

// CreateWebhook creates a new webhook
func (c Controller) CreateWebhook(request *evo.Request) any {
	var webhook models.Webhook

	if err := request.BodyParser(&webhook); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Validate required fields
	if webhook.Name == "" {
		return response.BadRequest(request, "Name is required")
	}
	if webhook.URL == "" {
		return response.BadRequest(request, "URL is required")
	}

	// Create the webhook
	err := db.Create(&webhook).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(webhook)
}

// UpdateWebhook updates an existing webhook
func (c Controller) UpdateWebhook(request *evo.Request) any {
	id := request.Param("id").String()

	var webhook models.Webhook
	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Parse update data
	var updateData map[string]any
	if err := request.BodyParser(&updateData); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Update the webhook
	err = db.Model(&webhook).Updates(updateData).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload the webhook
	db.First(&webhook, id)

	return response.OK(webhook)
}

// DeleteWebhook deletes a webhook
func (c Controller) DeleteWebhook(request *evo.Request) any {
	id := request.Param("id").String()

	var webhook models.Webhook
	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Delete associated deliveries first
	db.Where("webhook_id = ?", id).Delete(&models.WebhookDelivery{})

	// Delete the webhook
	err = db.Delete(&webhook).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Webhook deleted successfully"})
}

// TestWebhook sends a test webhook
func (c Controller) TestWebhook(request *evo.Request) any {
	id := request.Param("id").String()

	var webhook models.Webhook
	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Get the event type from query param, default to webhook.test
	eventType := request.Query("event").String()
	if eventType == "" {
		eventType = models.WebhookEventWebhookTest
	}

	// Build test data based on event type
	testData := buildTestDataForEvent(eventType)

	// Send directly to this specific webhook
	if models.WebhookSender != nil {
		// Force enable for test purposes
		webhook.Enabled = true
		err := models.SendToWebhook(&webhook, eventType, testData)
		if err != nil {
			return response.OK(map[string]any{
				"success": false,
				"message": "Test webhook failed: " + err.Error(),
				"event":   eventType,
			})
		}
	} else {
		return response.OK(map[string]any{
			"success": false,
			"message": "Webhook sender not initialized",
			"event":   eventType,
		})
	}

	return response.OK(map[string]any{
		"success": true,
		"message": "Test webhook sent for event: " + eventType,
		"event":   eventType,
	})
}

// buildTestDataForEvent creates sample test data based on event type
func buildTestDataForEvent(eventType string) map[string]any {
	switch eventType {
	case models.WebhookEventConversationCreated,
		models.WebhookEventConversationUpdated,
		models.WebhookEventConversationClosed:
		return map[string]any{
			"conversation": map[string]any{
				"id":         1,
				"client_id":  "550e8400-e29b-41d4-a716-446655440000",
				"status":     "open",
				"priority":   "normal",
				"subject":    "Test Conversation Subject",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
	case models.WebhookEventConversationStatusChange:
		return map[string]any{
			"conversation": map[string]any{
				"id":         1,
				"client_id":  "550e8400-e29b-41d4-a716-446655440000",
				"status":     "pending",
				"priority":   "normal",
				"subject":    "Test Conversation Subject",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:40:00Z",
			},
			"old_status": "open",
			"new_status": "pending",
		}
	case models.WebhookEventConversationAssigned:
		return map[string]any{
			"conversation": map[string]any{
				"id":          1,
				"client_id":   "550e8400-e29b-41d4-a716-446655440000",
				"status":      "open",
				"priority":    "normal",
				"assigned_to": "550e8400-e29b-41d4-a716-446655440001",
				"subject":     "Test Conversation Subject",
				"created_at":  "2024-01-15T10:30:00Z",
				"updated_at":  "2024-01-15T10:30:00Z",
			},
			"assigned_to": "Agent Name",
			"assigned_by": "Admin User",
		}
	case models.WebhookEventMessageCreated:
		return map[string]any{
			"message": map[string]any{
				"id":              1,
				"conversation_id": 1,
				"client_id":       "550e8400-e29b-41d4-a716-446655440000",
				"body":            "This is a test message body for webhook testing.",
				"content_type":    "text",
				"created_at":      "2024-01-15T10:30:00Z",
			},
			"conversation": map[string]any{
				"id":         1,
				"client_id":  "550e8400-e29b-41d4-a716-446655440000",
				"status":     "open",
				"priority":   "normal",
				"subject":    "Test Conversation Subject",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
	case models.WebhookEventClientCreated,
		models.WebhookEventClientUpdated:
		return map[string]any{
			"client": map[string]any{
				"id":         "550e8400-e29b-41d4-a716-446655440000",
				"name":       "Test Client",
				"data":       map[string]any{"email": "test@example.com", "phone": "+1234567890"},
				"language":   "en",
				"timezone":   "UTC",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
	case models.WebhookEventUserCreated,
		models.WebhookEventUserUpdated:
		return map[string]any{
			"user": map[string]any{
				"id":           "550e8400-e29b-41d4-a716-446655440001",
				"name":         "Test",
				"last_name":    "Agent",
				"display_name": "Test Agent",
				"email":        "agent@example.com",
				"type":         "agent",
				"status":       "active",
				"created_at":   "2024-01-15T10:30:00Z",
				"updated_at":   "2024-01-15T10:30:00Z",
			},
		}
	default: // webhook.test
		return map[string]any{
			"message":    "This is a test webhook payload",
			"test":       true,
			"webhook_id": 1,
		}
	}
}

// ListWebhookDeliveries returns webhook delivery logs
func (c Controller) ListWebhookDeliveries(request *evo.Request) any {
	var deliveries []models.WebhookDelivery

	query := db.Model(&models.WebhookDelivery{})

	// Filter by webhook_id if provided
	if webhookID := request.Query("webhook_id").String(); webhookID != "" {
		query = query.Where("webhook_id = ?", webhookID)
	}

	// Filter by success if provided
	if success := request.Query("success").String(); success != "" {
		query = query.Where("success = ?", success == "true")
	}

	// Filter by event if provided (supports comma-separated list for multiple events)
	if event := request.Query("event").String(); event != "" {
		events := strings.Split(event, ",")
		if len(events) == 1 {
			query = query.Where("event = ?", event)
		} else {
			query = query.Where("event IN (?)", events)
		}
	}

	// Order by most recent first
	query = query.Order("id DESC")

	p, err := pagination.New(query, request, &deliveries, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(deliveries, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// GetWebhookDelivery returns a single webhook delivery by ID
func (c Controller) GetWebhookDelivery(request *evo.Request) any {
	id := request.Param("id").String()
	var delivery models.WebhookDelivery

	err := db.Preload("Webhook").First(&delivery, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook delivery not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(delivery)
}
