package models

// WebhookBroadcaster is a callback function for broadcasting webhooks
// This is set by the webhook app to avoid circular dependencies
var WebhookBroadcaster func(event string, data map[string]any)

// BroadcastWebhook sends a webhook event if a broadcaster is registered
func BroadcastWebhook(event string, data map[string]any) {
	if WebhookBroadcaster != nil {
		WebhookBroadcaster(event, data)
	}
}
