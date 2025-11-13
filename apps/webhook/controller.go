package webhook

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

type Controller struct{}

// TestWebhook sends a test payload to the webhook
func (c Controller) TestWebhook(request *evo.Request) any {
	// Check admin authentication
	user := request.User().(*auth.User)
	if user.Anonymous() || user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrUnauthorized)
	}

	webhookID := request.Param("id").String()
	var webhook models.Webhook

	if err := db.First(&webhook, webhookID).Error; err != nil {
		return response.NotFound(request, "Webhook not found")
	}

	// Send test webhook
	if err := SendWebhook(&webhook, "webhook.test", map[string]any{
		"message":    "This is a test webhook",
		"webhook_id": webhook.ID,
		"timestamp":  "now",
	}); err != nil {
		return response.InternalError(request, "Failed to send test webhook: "+err.Error())
	}

	return response.OK(map[string]any{
		"message": "Test webhook sent successfully",
	})
}
