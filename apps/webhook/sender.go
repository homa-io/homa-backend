package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
)

// WebhookPayload represents the structure of data sent to webhooks
type WebhookPayload struct {
	Event     string         `json:"event"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// SendWebhook sends a webhook notification to all registered webhooks for the given event
func SendWebhook(webhook *models.Webhook, event string, data map[string]any) error {
	// Check if webhook is enabled
	if !webhook.Enabled {
		return nil
	}

	// Check if webhook is subscribed to this event
	if !webhook.IsSubscribedTo(event) {
		return nil
	}

	// Create payload
	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Homa-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", event)
	req.Header.Set("X-Webhook-ID", fmt.Sprintf("%d", webhook.ID))

	// Add HMAC signature if secret is set
	if webhook.Secret != "" {
		signature := generateHMACSignature(jsonData, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// Log failed delivery
		logWebhookDelivery(webhook.ID, event, false, err.Error())
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	var responseMessage string
	if !success {
		responseMessage = fmt.Sprintf("received status code: %d", resp.StatusCode)
	}

	// Log delivery
	logWebhookDelivery(webhook.ID, event, success, responseMessage)

	if !success {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

// BroadcastWebhook sends a webhook event to all registered webhooks
func BroadcastWebhook(event string, data map[string]any) {
	var webhooks []models.Webhook
	if err := db.Where("enabled = ?", true).Find(&webhooks).Error; err != nil {
		log.Error("Failed to fetch webhooks for broadcast:", err)
		return
	}

	for _, webhook := range webhooks {
		// Send webhook asynchronously
		go func(w models.Webhook) {
			if err := SendWebhook(&w, event, data); err != nil {
				log.Error("Failed to send webhook to", w.URL, ":", err)
			}
		}(webhook)
	}
}

// generateHMACSignature creates an HMAC-SHA256 signature for the payload
func generateHMACSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// logWebhookDelivery logs webhook delivery attempts
func logWebhookDelivery(webhookID uint, event string, success bool, message string) {
	delivery := models.WebhookDelivery{
		WebhookID: webhookID,
		Event:     event,
		Success:   success,
		Response:  message,
	}

	if err := db.Create(&delivery).Error; err != nil {
		log.Error("Failed to log webhook delivery:", err)
	}
}
