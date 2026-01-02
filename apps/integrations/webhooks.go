package integrations

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/models"
)

// WebhookController handles incoming webhooks from integrations
type WebhookController struct{}

// Default conversation timeout in hours
const DefaultConversationTimeoutHours = 12

// Setting key for conversation timeout
const SettingConversationTimeoutHours = "workflow.conversation_timeout_hours"

// =============================================================================
// Slack Webhook Handler
// =============================================================================

// SlackWebhook handles incoming Slack webhook events
func (c WebhookController) SlackWebhook(request *evo.Request) any {
	// Get Slack integration config
	integration, err := models.GetIntegration(models.IntegrationTypeSlack)
	if err != nil || integration.Status != models.IntegrationStatusEnabled {
		log.Warning("Slack webhook received but integration is not enabled")
		return map[string]string{"error": "Integration not enabled"}
	}

	var config models.SlackConfig
	if err := json.Unmarshal([]byte(integration.Config), &config); err != nil {
		log.Error("Failed to parse Slack config:", err)
		return map[string]string{"error": "Invalid configuration"}
	}

	// Get request body as string
	body := request.Body()

	// Verify Slack signature if signing secret is configured
	if config.SigningSecret != "" {
		timestamp := request.Get("X-Slack-Request-Timestamp").String()
		signature := request.Get("X-Slack-Signature").String()

		if !verifySlackSignature(config.SigningSecret, timestamp, body, signature) {
			log.Warning("Invalid Slack signature")
			return map[string]string{"error": "Invalid signature"}
		}
	}

	// Parse the event
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		log.Error("Failed to parse Slack payload:", err)
		return map[string]string{"error": "Invalid payload"}
	}

	// Handle URL verification challenge
	if payloadType, ok := payload["type"].(string); ok && payloadType == "url_verification" {
		challenge := payload["challenge"].(string)
		return map[string]string{"challenge": challenge}
	}

	// Handle event callbacks
	if payloadType, ok := payload["type"].(string); ok && payloadType == "event_callback" {
		event, ok := payload["event"].(map[string]interface{})
		if !ok {
			return map[string]string{"ok": "true"}
		}

		eventType, _ := event["type"].(string)

		// Only process message events
		if eventType == "message" {
			// Ignore bot messages and message changes
			if _, hasSubtype := event["subtype"]; hasSubtype {
				return map[string]string{"ok": "true"}
			}

			// Extract message details
			userID, _ := event["user"].(string)
			text, _ := event["text"].(string)
			channelID, _ := event["channel"].(string)
			ts, _ := event["ts"].(string)

			if userID != "" && text != "" {
				// Process the message
				go processIncomingMessage(
					models.ExternalIDTypeSlack,
					userID,
					text,
					channelID,
					ts,
					nil, // No additional user info from basic webhook
				)
			}
		}
	}

	return map[string]string{"ok": "true"}
}

// verifySlackSignature verifies the Slack request signature
func verifySlackSignature(signingSecret, timestamp, body, signature string) bool {
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// =============================================================================
// Telegram Webhook Handler
// =============================================================================

// TelegramWebhook handles incoming Telegram webhook updates
func (c WebhookController) TelegramWebhook(request *evo.Request) any {
	// Get Telegram integration config
	integration, err := models.GetIntegration(models.IntegrationTypeTelegram)
	if err != nil || integration.Status != models.IntegrationStatusEnabled {
		log.Warning("Telegram webhook received but integration is not enabled")
		return map[string]string{"ok": "true"}
	}

	// Get request body
	body := request.Body()

	// Parse the update
	var update TelegramUpdate
	if err := json.Unmarshal([]byte(body), &update); err != nil {
		log.Error("Failed to parse Telegram update:", err)
		return map[string]string{"ok": "true"}
	}

	// Process message
	if update.Message != nil {
		msg := update.Message

		// Build user info
		userInfo := map[string]interface{}{
			"first_name": msg.From.FirstName,
			"last_name":  msg.From.LastName,
			"username":   msg.From.Username,
		}

		// Get external ID (Telegram user ID)
		externalID := strconv.FormatInt(msg.From.ID, 10)

		// Get chat ID for context
		chatID := strconv.FormatInt(msg.Chat.ID, 10)

		// Get message ID
		messageID := strconv.Itoa(msg.MessageID)

		// Build user name
		userName := msg.From.FirstName
		if msg.From.LastName != "" {
			userName += " " + msg.From.LastName
		}
		if userName == "" && msg.From.Username != "" {
			userName = msg.From.Username
		}
		userInfo["name"] = userName

		// Process the message
		go processIncomingMessage(
			models.ExternalIDTypeTelegram,
			externalID,
			msg.Text,
			chatID,
			messageID,
			userInfo,
		)
	}

	return map[string]string{"ok": "true"}
}

// TelegramUpdate represents a Telegram webhook update
type TelegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	MessageID int          `json:"message_id"`
	From      TelegramUser `json:"from"`
	Chat      TelegramChat `json:"chat"`
	Date      int64        `json:"date"`
	Text      string       `json:"text"`
}

// TelegramUser represents a Telegram user
type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// TelegramChat represents a Telegram chat
type TelegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// =============================================================================
// WhatsApp Webhook Handler
// =============================================================================

// WhatsAppWebhook handles incoming WhatsApp webhook events
func (c WebhookController) WhatsAppWebhook(request *evo.Request) any {
	// Get WhatsApp integration config
	integration, err := models.GetIntegration(models.IntegrationTypeWhatsApp)
	if err != nil || integration.Status != models.IntegrationStatusEnabled {
		log.Warning("WhatsApp webhook received but integration is not enabled")
		return map[string]string{"error": "Integration not enabled"}
	}

	var config models.WhatsAppConfig
	if err := json.Unmarshal([]byte(integration.Config), &config); err != nil {
		log.Error("Failed to parse WhatsApp config:", err)
		return map[string]string{"error": "Invalid configuration"}
	}

	// Handle webhook verification (GET request)
	if request.Method() == "GET" {
		mode := request.Query("hub.mode").String()
		token := request.Query("hub.verify_token").String()
		challenge := request.Query("hub.challenge").String()

		if mode == "subscribe" && token == config.WebhookVerifyToken {
			log.Info("WhatsApp webhook verified successfully")
			return challenge
		}
		log.Warning("WhatsApp webhook verification failed")
		return map[string]string{"error": "Verification failed"}
	}

	// Handle POST request (incoming messages)
	body := request.Body()

	// Parse the webhook payload
	var payload WhatsAppWebhookPayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		log.Error("Failed to parse WhatsApp payload:", err)
		return map[string]string{"error": "Invalid payload"}
	}

	// Process entries
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			value := change.Value
			if value.Messages == nil {
				continue
			}

			for _, msg := range value.Messages {
				// Only process text messages for now
				if msg.Type != "text" || msg.Text == nil {
					continue
				}

				// Find contact info
				var contactName string
				for _, contact := range value.Contacts {
					if contact.WaID == msg.From {
						if contact.Profile.Name != "" {
							contactName = contact.Profile.Name
						}
						break
					}
				}

				userInfo := map[string]interface{}{
					"name":  contactName,
					"phone": msg.From,
					"wa_id": msg.From,
				}

				// Process the message
				go processIncomingMessage(
					models.ExternalIDTypeWhatsapp,
					msg.From,
					msg.Text.Body,
					value.Metadata.PhoneNumberID,
					msg.ID,
					userInfo,
				)
			}
		}
	}

	return map[string]string{"status": "ok"}
}

// WhatsApp webhook payload structures
type WhatsAppWebhookPayload struct {
	Object string          `json:"object"`
	Entry  []WhatsAppEntry `json:"entry"`
}

type WhatsAppEntry struct {
	ID      string           `json:"id"`
	Changes []WhatsAppChange `json:"changes"`
}

type WhatsAppChange struct {
	Field string              `json:"field"`
	Value WhatsAppChangeValue `json:"value"`
}

type WhatsAppChangeValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         WhatsAppMetadata  `json:"metadata"`
	Contacts         []WhatsAppContact `json:"contacts,omitempty"`
	Messages         []WhatsAppMessage `json:"messages,omitempty"`
}

type WhatsAppMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WhatsAppContact struct {
	WaID    string `json:"wa_id"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
}

type WhatsAppMessage struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      *struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
}

// =============================================================================
// Common Message Processing
// =============================================================================

// processIncomingMessage handles the common logic for processing incoming messages
// from any integration (Slack, Telegram, WhatsApp)
func processIncomingMessage(
	externalIDType string,
	externalIDValue string,
	messageText string,
	channelContext string,
	messageID string,
	userInfo map[string]interface{},
) {
	log.Info("Processing incoming message from %s: user=%s, message=%s",
		externalIDType, externalIDValue, truncateString(messageText, 50))

	// 1. Upsert client
	client, err := upsertClient(externalIDType, externalIDValue, userInfo)
	if err != nil {
		log.Error("Failed to upsert client:", err)
		return
	}

	// 2. Find or create conversation
	conversation, err := findOrCreateConversation(client, externalIDType, channelContext)
	if err != nil {
		log.Error("Failed to find/create conversation:", err)
		return
	}

	// 3. Create message
	message := models.Message{
		ConversationID: conversation.ID,
		ClientID:       &client.ID,
		Body:           messageText,
	}

	if err := db.Create(&message).Error; err != nil {
		log.Error("Failed to create message:", err)
		return
	}

	log.Info("Message created successfully: conversation=%d, message=%d", conversation.ID, message.ID)
}

// upsertClient finds or creates a client based on external ID
func upsertClient(externalIDType, externalIDValue string, userInfo map[string]interface{}) (*models.Client, error) {
	// First, try to find existing client by external ID
	var externalID models.ClientExternalID
	err := db.Where("type = ? AND value = ?", externalIDType, externalIDValue).
		Preload("Client").
		First(&externalID).Error

	if err == nil {
		// Client exists, update info if provided
		client := &externalID.Client

		if userInfo != nil {
			updated := false

			// Update name if provided and different
			if name, ok := userInfo["name"].(string); ok && name != "" && client.Name != name {
				client.Name = name
				updated = true
			}

			if updated {
				if err := db.Save(client).Error; err != nil {
					log.Warning("Failed to update client info:", err)
				}
			}
		}

		return client, nil
	}

	// Client doesn't exist, create new one
	clientName := "Unknown"
	if userInfo != nil {
		if name, ok := userInfo["name"].(string); ok && name != "" {
			clientName = name
		}
	}

	// Build client data from userInfo
	var clientData []byte
	if userInfo != nil {
		clientData, _ = json.Marshal(userInfo)
	}

	client := &models.Client{
		Name: clientName,
		Data: clientData,
	}

	if err := db.Create(client).Error; err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create external ID link
	newExternalID := models.ClientExternalID{
		ClientID: client.ID,
		Type:     externalIDType,
		Value:    externalIDValue,
	}

	if err := db.Create(&newExternalID).Error; err != nil {
		return nil, fmt.Errorf("failed to create external ID: %w", err)
	}

	log.Info("Created new client: id=%s, name=%s, external_type=%s",
		client.ID, client.Name, externalIDType)

	return client, nil
}

// findOrCreateConversation finds an existing conversation within timeout or creates a new one
func findOrCreateConversation(client *models.Client, channelType, channelContext string) (*models.Conversation, error) {
	// Get conversation timeout from settings
	timeoutHours := getConversationTimeoutHours()
	cutoffTime := time.Now().Add(-time.Duration(timeoutHours) * time.Hour)

	// Map external ID type to channel ID
	channelID := mapExternalTypeToChannelID(channelType)

	// Find recent open conversation for this client on this channel
	var conversation models.Conversation
	err := db.Where("client_id = ? AND channel_id = ? AND status NOT IN (?, ?, ?) AND created_at > ?",
		client.ID,
		channelID,
		models.ConversationStatusClosed,
		models.ConversationStatusResolved,
		models.ConversationStatusSpam,
		cutoffTime,
	).Order("created_at DESC").First(&conversation).Error

	if err == nil {
		log.Info("Found existing conversation: id=%d for client=%s", conversation.ID, client.ID)
		return &conversation, nil
	}

	// No existing conversation found, create new one
	// Generate a random secret for the conversation
	secret := generateSecret(32)

	conversation = models.Conversation{
		Title:        fmt.Sprintf("%s Conversation", toTitle(channelType)),
		ClientID:     client.ID,
		ChannelID:    channelID,
		Secret:       secret,
		Status:       models.ConversationStatusNew,
		Priority:     models.ConversationPriorityMedium,
		CustomFields: []byte("{}"),
	}

	// Get default department from settings
	defaultDeptStr := models.GetSettingValue("workflow.default_department", "")
	if defaultDeptStr != "" {
		if deptID, err := strconv.ParseUint(defaultDeptStr, 10, 32); err == nil && deptID > 0 {
			deptIDUint := uint(deptID)
			conversation.DepartmentID = &deptIDUint
		}
	}

	if err := db.Create(&conversation).Error; err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	log.Info("Created new conversation: id=%d for client=%s on channel=%s",
		conversation.ID, client.ID, channelID)

	return &conversation, nil
}

// getConversationTimeoutHours gets the conversation timeout from settings
func getConversationTimeoutHours() int {
	timeoutStr := models.GetSettingValue(SettingConversationTimeoutHours, "")
	if timeoutStr == "" {
		return DefaultConversationTimeoutHours
	}

	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil || timeout < 1 {
		return DefaultConversationTimeoutHours
	}

	return timeout
}

// mapExternalTypeToChannelID maps external ID type to channel ID
func mapExternalTypeToChannelID(externalType string) string {
	switch externalType {
	case models.ExternalIDTypeSlack:
		return "slack"
	case models.ExternalIDTypeTelegram:
		return "telegram"
	case models.ExternalIDTypeWhatsapp:
		return "whatsapp"
	default:
		return "unknown"
	}
}

// generateSecret generates a random secret string
func generateSecret(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// truncateString truncates a string to max length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// toTitle converts string to title case (first letter uppercase)
func toTitle(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// =============================================================================
// Outbound Message Sending
// =============================================================================

// SendSlackMessage sends a message to Slack
func SendSlackMessage(channelID, text string) error {
	integration, err := models.GetIntegration(models.IntegrationTypeSlack)
	if err != nil || integration.Status != models.IntegrationStatusEnabled {
		return fmt.Errorf("Slack integration not enabled")
	}

	var config models.SlackConfig
	if err := json.Unmarshal([]byte(integration.Config), &config); err != nil {
		return fmt.Errorf("invalid Slack config: %w", err)
	}

	// Implementation for sending would go here
	// Using Slack's chat.postMessage API
	log.Info("Would send Slack message to %s: %s", channelID, truncateString(text, 50))
	return nil
}

// SendTelegramMessage sends a message to Telegram
func SendTelegramMessage(chatID, text string) error {
	integration, err := models.GetIntegration(models.IntegrationTypeTelegram)
	if err != nil || integration.Status != models.IntegrationStatusEnabled {
		return fmt.Errorf("Telegram integration not enabled")
	}

	var config models.TelegramConfig
	if err := json.Unmarshal([]byte(integration.Config), &config); err != nil {
		return fmt.Errorf("invalid Telegram config: %w", err)
	}

	if config.BotToken == "" {
		return fmt.Errorf("Telegram bot token not configured")
	}

	// Call Telegram Bot API sendMessage endpoint
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error("Telegram API error: %s", string(body))
		return fmt.Errorf("Telegram API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Info("Sent Telegram message to %s: %s", chatID, truncateString(text, 50))
	return nil
}

// SendWhatsAppMessage sends a message to WhatsApp
func SendWhatsAppMessage(phoneNumber, text string) error {
	integration, err := models.GetIntegration(models.IntegrationTypeWhatsApp)
	if err != nil || integration.Status != models.IntegrationStatusEnabled {
		return fmt.Errorf("WhatsApp integration not enabled")
	}

	var config models.WhatsAppConfig
	if err := json.Unmarshal([]byte(integration.Config), &config); err != nil {
		return fmt.Errorf("invalid WhatsApp config: %w", err)
	}

	// Implementation for sending would go here
	// Using WhatsApp Business API's messages endpoint
	log.Info("Would send WhatsApp message to %s: %s", phoneNumber, truncateString(text, 50))
	return nil
}

// Placeholder for unused uuid import
var _ = uuid.Nil
