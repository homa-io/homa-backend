package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WebhookPayload represents the webhook data structure
type WebhookPayload struct {
	Event     string         `json:"event"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// WebhookLog represents what we save to file
type WebhookLog struct {
	ReceivedAt time.Time              `json:"received_at"`
	Event      string                 `json:"event"`
	Timestamp  string                 `json:"timestamp"`
	Data       map[string]any         `json:"data"`
	Headers    map[string]string      `json:"headers"`
	Signature  string                 `json:"signature"`
	Verified   bool                   `json:"signature_verified"`
	RawPayload string                 `json:"raw_payload"`
}

const (
	// Test secret - should match the webhook secret in database
	TestSecret = "test-secret-key-123"
	LogDir     = "_webhook_test/logs"
)

func main() {
	// Create logs directory
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		log.Fatal("Failed to create logs directory:", err)
	}

	// Setup HTTP handler
	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/", handleRoot)

	port := ":9000"
	fmt.Println("🚀 Webhook Test Server Started")
	fmt.Println("================================")
	fmt.Printf("📡 Listening on http://localhost%s\n", port)
	fmt.Printf("📝 Webhook endpoint: http://localhost%s/webhook\n", port)
	fmt.Printf("📁 Logs directory: %s\n", LogDir)
	fmt.Printf("🔑 Test secret: %s\n", TestSecret)
	fmt.Println("================================")
	fmt.Println("\nWaiting for webhooks...\n")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Webhook Test Server</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .status { background: #e8f5e9; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .endpoint { background: #f5f5f5; padding: 10px; border-radius: 4px; font-family: monospace; }
        h1 { color: #2e7d32; }
        code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>🚀 Webhook Test Server</h1>
    <div class="status">
        <h2>✅ Server is Running</h2>
        <p><strong>Webhook Endpoint:</strong></p>
        <div class="endpoint">POST http://localhost:9000/webhook</div>
        <p><strong>Test Secret:</strong> <code>test-secret-key-123</code></p>
        <p><strong>Logs Directory:</strong> <code>_webhook_test/logs/</code></p>
    </div>

    <h2>📋 Test Instructions</h2>
    <ol>
        <li>Create a webhook in Homa with URL: <code>http://localhost:9000/webhook</code></li>
        <li>Set the secret to: <code>test-secret-key-123</code></li>
        <li>Create a ticket or trigger an event</li>
        <li>Check the logs directory for received webhooks</li>
    </ol>

    <h2>🔍 Recent Webhooks</h2>
    <p>Check <code>_webhook_test/logs/</code> directory for webhook logs</p>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Error reading body: %v\n", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Parse webhook payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("❌ Error parsing JSON: %v\n", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Verify signature if present
	signature := r.Header.Get("X-Webhook-Signature")
	verified := false
	if signature != "" {
		verified = verifySignature(body, signature, TestSecret)
	}

	// Create log entry
	webhookLog := WebhookLog{
		ReceivedAt: time.Now(),
		Event:      payload.Event,
		Timestamp:  payload.Timestamp,
		Data:       payload.Data,
		Headers:    headers,
		Signature:  signature,
		Verified:   verified,
		RawPayload: string(body),
	}

	// Save to file
	if err := saveWebhookLog(webhookLog); err != nil {
		log.Printf("❌ Error saving log: %v\n", err)
	}

	// Log to console
	logToConsole(webhookLog)

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "success",
		"message":  "Webhook received and logged",
		"event":    payload.Event,
		"verified": verified,
	})
}

func verifySignature(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return signature == expectedSignature
}

func saveWebhookLog(webhookLog WebhookLog) error {
	// Create filename with timestamp and event
	timestamp := webhookLog.ReceivedAt.Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.json", timestamp, webhookLog.Event)
	filepath := filepath.Join(LogDir, filename)

	// Convert to pretty JSON
	jsonData, err := json.MarshalIndent(webhookLog, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return err
	}

	fmt.Printf("💾 Saved to: %s\n", filepath)
	return nil
}

func logToConsole(webhookLog WebhookLog) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("📨 WEBHOOK RECEIVED: %s\n", webhookLog.Event)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("⏰ Received At: %s\n", webhookLog.ReceivedAt.Format(time.RFC3339))
	fmt.Printf("📍 Event: %s\n", webhookLog.Event)
	fmt.Printf("🕐 Event Timestamp: %s\n", webhookLog.Timestamp)

	if webhookLog.Signature != "" {
		if webhookLog.Verified {
			fmt.Printf("✅ Signature: VERIFIED\n")
		} else {
			fmt.Printf("❌ Signature: FAILED\n")
		}
	} else {
		fmt.Printf("⚠️  Signature: NOT PROVIDED\n")
	}

	fmt.Println("\n📋 Headers:")
	for key, value := range webhookLog.Headers {
		if key == "X-Webhook-Signature" || key == "X-Webhook-Event" || key == "X-Webhook-Id" || key == "User-Agent" {
			fmt.Printf("   %s: %s\n", key, value)
		}
	}

	fmt.Println("\n📦 Data:")
	dataJSON, _ := json.MarshalIndent(webhookLog.Data, "   ", "  ")
	fmt.Printf("   %s\n", string(dataJSON))

	fmt.Println(strings.Repeat("=", 80))
}
