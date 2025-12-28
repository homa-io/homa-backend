package webhook

import (
	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/apps/admin"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
)

type App struct {
}

func (a App) Register() error {
	// Set the webhook broadcaster callback in models package
	models.WebhookBroadcaster = BroadcastWebhook
	// Set the webhook sender callback for direct sending
	models.WebhookSender = SendWebhook
	// Set the user webhook broadcaster callback in auth package
	auth.UserWebhookBroadcaster = BroadcastWebhookWithData
	return nil
}

func (a App) Router() error {
	var controller Controller

	// Register restify routes for webhooks at /api/admin/webhooks
	// Note: Restify API is embedded in Webhook model, routes are auto-generated

	// Apply admin authentication to webhook routes
	evo.Use("/api/admin/webhooks", admin.AdminAuthMiddleware)

	// Custom endpoint for testing webhooks
	evo.Post("/api/admin/webhooks/:id/test", controller.TestWebhook)

	return nil
}

func (a App) WhenReady() error {
	// Handle CLI commands
	GenerateMockWebhook()
	return nil
}

func (a App) Name() string {
	return "webhook"
}
