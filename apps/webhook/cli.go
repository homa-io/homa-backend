package webhook

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/getevo/evo/v2/lib/args"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/iesreza/homa-backend/apps/models"
)

// GenerateMockWebhook creates a mock webhook for testing
func GenerateMockWebhook() {
	if args.Get("--generate-webhook") == "" {
		return
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Get URL from arguments or use default mock URL
	url := args.Get("--url")
	if url == "" {
		url = "https://webhook.site/unique-id-here" // Replace with actual webhook.site URL for testing
	}

	// Generate random webhook configuration
	webhookTypes := []string{
		"Slack Integration",
		"Discord Bot",
		"Zapier Automation",
		"Custom API Integration",
		"Email Notification Service",
		"Analytics Platform",
	}

	webhookType := webhookTypes[rand.Intn(len(webhookTypes))]

	webhook := models.Webhook{
		Name:        fmt.Sprintf("%s - %d", webhookType, rand.Intn(1000)),
		URL:         url,
		Secret:      generateRandomSecret(),
		Enabled:     true,
		Description: fmt.Sprintf("Auto-generated mock webhook for testing - %s", time.Now().Format(time.RFC3339)),
	}

	// Randomly subscribe to events
	webhook.EventConversationCreated = rand.Intn(2) == 1
	webhook.EventConversationUpdated = rand.Intn(2) == 1
	webhook.EventConversationStatusChange = rand.Intn(2) == 1
	webhook.EventConversationClosed = rand.Intn(2) == 1
	webhook.EventConversationAssigned = rand.Intn(2) == 1
	webhook.EventMessageCreated = rand.Intn(2) == 1
	webhook.EventClientCreated = rand.Intn(2) == 1
	webhook.EventClientUpdated = rand.Intn(2) == 1
	webhook.EventUserCreated = rand.Intn(2) == 1
	webhook.EventUserUpdated = rand.Intn(2) == 1

	// Ensure at least one event is subscribed
	if !webhook.EventConversationCreated && !webhook.EventConversationUpdated &&
		!webhook.EventConversationStatusChange && !webhook.EventConversationClosed &&
		!webhook.EventConversationAssigned && !webhook.EventMessageCreated &&
		!webhook.EventClientCreated && !webhook.EventClientUpdated &&
		!webhook.EventUserCreated && !webhook.EventUserUpdated {
		webhook.EventConversationCreated = true
	}

	// 20% chance to subscribe to all events
	if rand.Intn(5) == 0 {
		webhook.EventAll = true
	}

	// Create webhook in database
	if err := db.Create(&webhook).Error; err != nil {
		fmt.Printf("❌ Failed to create mock webhook: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Mock webhook created successfully!")
	fmt.Printf("   ID: %d\n", webhook.ID)
	fmt.Printf("   Name: %s\n", webhook.Name)
	fmt.Printf("   URL: %s\n", webhook.URL)
	fmt.Printf("   Secret: %s\n", webhook.Secret)
	fmt.Printf("   Enabled: %v\n", webhook.Enabled)
	fmt.Println("\n📋 Event Subscriptions:")
	fmt.Printf("   All Events: %v\n", webhook.EventAll)
	fmt.Printf("   Conversation Created: %v\n", webhook.EventConversationCreated)
	fmt.Printf("   Conversation Updated: %v\n", webhook.EventConversationUpdated)
	fmt.Printf("   Conversation Status Change: %v\n", webhook.EventConversationStatusChange)
	fmt.Printf("   Conversation Closed: %v\n", webhook.EventConversationClosed)
	fmt.Printf("   Conversation Assigned: %v\n", webhook.EventConversationAssigned)
	fmt.Printf("   Message Created: %v\n", webhook.EventMessageCreated)
	fmt.Printf("   Client Created: %v\n", webhook.EventClientCreated)
	fmt.Printf("   Client Updated: %v\n", webhook.EventClientUpdated)
	fmt.Printf("   User Created: %v\n", webhook.EventUserCreated)
	fmt.Printf("   User Updated: %v\n", webhook.EventUserUpdated)

	// Test sending the webhook if --send flag is provided
	if args.Get("--send") != "" {
		fmt.Println("\n📤 Sending test webhook...")

		testData := map[string]any{
			"test":       true,
			"message":    "This is a mock test webhook",
			"webhook_id": webhook.ID,
			"timestamp":  time.Now().Format(time.RFC3339),
			"random_data": map[string]any{
				"ticket_id":   rand.Intn(1000),
				"client_id":   fmt.Sprintf("client-%d", rand.Intn(100)),
				"status":      []string{"new", "in_progress", "closed"}[rand.Intn(3)],
				"priority":    []string{"low", "medium", "high", "urgent"}[rand.Intn(4)],
			},
		}

		if err := SendWebhook(&webhook, "webhook.test", testData); err != nil {
			fmt.Printf("❌ Failed to send test webhook: %v\n", err)
		} else {
			fmt.Println("✅ Test webhook sent successfully!")
			fmt.Println("\n💡 Check your webhook URL to see the received payload")
		}
	}

	os.Exit(0)
}

// generateRandomSecret creates a random secret key
func generateRandomSecret() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = charset[rand.Intn(len(charset))]
	}
	return string(secret)
}
