package integrations

import (
	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/apps/integrations/email"
	"github.com/iesreza/homa-backend/apps/models"
)

type App struct{}

func (a App) Register() error {
	// Initialize outbound messaging functions to avoid circular imports
	models.SendTelegramMessage = SendTelegramMessage
	models.SendWhatsAppMessage = SendWhatsAppMessage
	models.SendSlackMessage = SendSlackMessage

	// Initialize email reply function
	email.RegisterSendEmailReply()

	return nil
}

func (a App) Router() error {
	var controller WebhookController

	// Integration webhook endpoints (no authentication required - these receive external callbacks)
	// Slack webhook
	evo.Post("/api/integrations/webhooks/slack", controller.SlackWebhook)

	// Telegram webhook
	evo.Post("/api/integrations/webhooks/telegram", controller.TelegramWebhook)

	// WhatsApp webhook (supports both GET for verification and POST for messages)
	evo.Get("/api/integrations/webhooks/whatsapp", controller.WhatsAppWebhook)
	evo.Post("/api/integrations/webhooks/whatsapp", controller.WhatsAppWebhook)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "integrations"
}
