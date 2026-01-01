package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
)

// Retry configuration
const (
	MaxRetries     = 5
	InitialBackoff = 1 * time.Second
	MaxBackoff     = 60 * time.Second
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

	// Pretty print the body for logging
	var prettyBody bytes.Buffer
	json.Indent(&prettyBody, jsonData, "", "  ")

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

	// Add Authorization header if secret is set
	if webhook.Secret != "" {
		req.Header.Set("Authorization", webhook.Secret)
	}

	// Capture headers for logging
	headersJSON := formatHeaders(req.Header)

	// Send request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	durationMs := time.Since(startTime).Milliseconds()

	if err != nil {
		// Log failed delivery with request details
		logWebhookDeliveryFull(webhook.ID, event, false, webhook.URL, prettyBody.String(), headersJSON, 0, err.Error(), durationMs)
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, _ := io.ReadAll(resp.Body)
	responseText := string(respBody)
	if len(responseText) > 2000 {
		responseText = responseText[:2000] + "... (truncated)"
	}

	// Check response status
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	// Log delivery with full details
	logWebhookDeliveryFull(webhook.ID, event, success, webhook.URL, prettyBody.String(), headersJSON, resp.StatusCode, responseText, durationMs)

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

// BroadcastWebhookWithData broadcasts a webhook event with data that may already contain nested structures
// This is used for user webhooks where we pass the sanitized data directly
func BroadcastWebhookWithData(event string, data map[string]any) {
	BroadcastWebhook(event, data)
}

// formatHeaders converts http.Header to JSON string for logging
func formatHeaders(headers http.Header) string {
	headerMap := make(map[string]string)
	for key, values := range headers {
		headerMap[key] = strings.Join(values, ", ")
	}
	jsonBytes, _ := json.Marshal(headerMap)
	return string(jsonBytes)
}

// logWebhookDeliveryFull logs webhook delivery attempts with full request details
func logWebhookDeliveryFull(webhookID uint, event string, success bool, url, body, headers string, statusCode int, response string, durationMs int64) {
	delivery := models.WebhookDelivery{
		WebhookID:      webhookID,
		Event:          event,
		Success:        success,
		RequestURL:     url,
		RequestBody:    body,
		RequestHeaders: headers,
		StatusCode:     statusCode,
		Response:       response,
		DurationMs:     durationMs,
	}

	if err := db.Create(&delivery).Error; err != nil {
		log.Error("Failed to log webhook delivery:", err)
	}
}

// logWebhookDelivery logs webhook delivery attempts (simplified version for backward compatibility)
func logWebhookDelivery(webhookID uint, event string, success bool, message string) {
	logWebhookDeliveryFull(webhookID, event, success, "", "", "", 0, message, 0)
}

// SendWebhookWithRetry sends a webhook with exponential backoff retry logic
// Retries up to MaxRetries times on failure with exponential backoff
func SendWebhookWithRetry(webhook *models.Webhook, event string, data map[string]any) error {
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		err := SendWebhook(webhook, event, data)
		if err == nil {
			if attempt > 0 {
				log.Info("Webhook succeeded on retry %d for %s to %s", attempt, event, webhook.URL)
			}
			return nil
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == MaxRetries {
			break
		}

		// Calculate backoff with exponential delay
		backoff := calculateBackoff(attempt)
		log.Warning("Webhook failed (attempt %d/%d) for %s to %s: %v. Retrying in %v",
			attempt+1, MaxRetries+1, event, webhook.URL, err, backoff)

		time.Sleep(backoff)
	}

	log.Error("Webhook failed after %d attempts for %s to %s: %v",
		MaxRetries+1, event, webhook.URL, lastErr)
	return fmt.Errorf("webhook failed after %d retries: %w", MaxRetries, lastErr)
}

// calculateBackoff calculates exponential backoff with jitter
func calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: 1s, 2s, 4s, 8s, 16s (capped at MaxBackoff)
	backoff := float64(InitialBackoff) * math.Pow(2, float64(attempt))
	if backoff > float64(MaxBackoff) {
		backoff = float64(MaxBackoff)
	}
	return time.Duration(backoff)
}

// BroadcastWebhookWithRetry sends a webhook event to all registered webhooks with retry logic
func BroadcastWebhookWithRetry(event string, data map[string]any) {
	var webhooks []models.Webhook
	if err := db.Where("enabled = ?", true).Find(&webhooks).Error; err != nil {
		log.Error("Failed to fetch webhooks for broadcast:", err)
		return
	}

	for _, webhook := range webhooks {
		// Send webhook asynchronously with retry
		go func(w models.Webhook) {
			if err := SendWebhookWithRetry(&w, event, data); err != nil {
				log.Error("Failed to send webhook (with retries) to", w.URL, ":", err)
			}
		}(webhook)
	}
}
