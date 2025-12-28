package models

// WebhookBroadcaster is a callback function for broadcasting webhooks
// This is set by the webhook app to avoid circular dependencies
var WebhookBroadcaster func(event string, data map[string]any)

// WebhookSender is a callback function for sending to a specific webhook
// This is set by the webhook app to avoid circular dependencies
var WebhookSender func(webhook *Webhook, event string, data map[string]any) error

// BroadcastWebhook sends a webhook event if a broadcaster is registered
func BroadcastWebhook(event string, data map[string]any) {
	if WebhookBroadcaster != nil {
		WebhookBroadcaster(event, data)
	}
}

// SendToWebhook sends to a specific webhook if a sender is registered
func SendToWebhook(webhook *Webhook, event string, data map[string]any) error {
	if WebhookSender != nil {
		return WebhookSender(webhook, event, data)
	}
	return nil
}
