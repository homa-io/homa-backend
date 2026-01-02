// Package telegram provides the Telegram Bot integration driver.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/iesreza/homa-backend/apps/integrations/drivers"
	"github.com/iesreza/homa-backend/apps/models"
)

const (
	// TypeID is the unique identifier for this integration.
	TypeID = "telegram"
	// DisplayName is the human-readable name.
	DisplayName = "Telegram"
)

// Driver implements the drivers.Driver interface for Telegram.
type Driver struct{}

// New creates a new Telegram driver instance.
func New() *Driver {
	return &Driver{}
}

// Type returns the unique identifier for this integration type.
func (d *Driver) Type() string {
	return TypeID
}

// Name returns the display name for this integration.
func (d *Driver) Name() string {
	return DisplayName
}

// Test validates the connection using the provided configuration.
func (d *Driver) Test(configJSON string) drivers.TestResult {
	var config models.TelegramConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid configuration", Details: err.Error()}
	}

	if config.BotToken == "" {
		return drivers.TestResult{Success: false, Message: "Bot token is required"}
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Test by calling Telegram's getMe API
	getMeURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", config.BotToken)
	resp, err := client.Get(getMeURL)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to connect to Telegram", Details: err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid response from Telegram", Details: err.Error()}
	}

	if ok, _ := result["ok"].(bool); !ok {
		desc, _ := result["description"].(string)
		return drivers.TestResult{Success: false, Message: "Telegram authentication failed", Details: desc}
	}

	var botName, username string
	if resultData, ok := result["result"].(map[string]interface{}); ok {
		botName, _ = resultData["first_name"].(string)
		username, _ = resultData["username"].(string)
	}

	// Also check webhook status
	webhookURL := fmt.Sprintf("https://api.telegram.org/bot%s/getWebhookInfo", config.BotToken)
	webhookResp, err := client.Get(webhookURL)
	if err != nil {
		return drivers.TestResult{
			Success: true,
			Message: fmt.Sprintf("Connected to bot: %s (@%s). Warning: Could not check webhook status.", botName, username),
		}
	}
	defer webhookResp.Body.Close()

	webhookBody, _ := io.ReadAll(webhookResp.Body)
	var webhookResult map[string]interface{}
	if err := json.Unmarshal(webhookBody, &webhookResult); err != nil {
		return drivers.TestResult{
			Success: true,
			Message: fmt.Sprintf("Connected to bot: %s (@%s). Warning: Could not parse webhook info.", botName, username),
		}
	}

	// Parse webhook info
	webhookInfo := ""
	if webhookData, ok := webhookResult["result"].(map[string]interface{}); ok {
		configuredURL, _ := webhookData["url"].(string)
		pendingCount, _ := webhookData["pending_update_count"].(float64)
		lastError, _ := webhookData["last_error_message"].(string)

		if configuredURL == "" {
			webhookInfo = "Webhook not configured"
		} else {
			webhookInfo = fmt.Sprintf("Webhook: %s", configuredURL)
			if pendingCount > 0 {
				webhookInfo += fmt.Sprintf(" (%d pending)", int(pendingCount))
			}
			if lastError != "" {
				webhookInfo += fmt.Sprintf(" - Last error: %s", lastError)
			}
		}
	}

	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Connected to bot: %s (@%s). %s", botName, username, webhookInfo),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config models.TelegramConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid Telegram configuration: %w", err)
	}
	if config.BotToken == "" {
		return fmt.Errorf("bot token is required")
	}
	return nil
}

// GetConfigFields returns the required configuration fields.
func (d *Driver) GetConfigFields() []drivers.ConfigField {
	return []drivers.ConfigField{
		{Name: "bot_token", Label: "Bot Token", Type: "password", Required: true, Placeholder: "123456:ABC-DEF..."},
	}
}

// GetMaskedConfig returns the config with sensitive fields masked.
func (d *Driver) GetMaskedConfig(configJSON string) map[string]interface{} {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil
	}

	for key, value := range config {
		if drivers.SensitiveFields[key] {
			if str, ok := value.(string); ok && len(str) > 8 {
				config[key] = str[:4] + "..." + str[len(str)-4:]
			} else if str, ok := value.(string); ok && len(str) > 0 {
				config[key] = "****"
			}
		}
	}

	return config
}

// OnSave is called after the integration is saved.
// It automatically registers the webhook with Telegram.
func (d *Driver) OnSave(configJSON string, status string, webhookBaseURL string) drivers.OnSaveResult {
	if status != "enabled" {
		return drivers.OnSaveResult{Success: true, Message: "Integration disabled, skipping webhook registration"}
	}

	var config models.TelegramConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.OnSaveResult{Success: false, Message: "Invalid configuration", Details: err.Error()}
	}

	if config.BotToken == "" {
		return drivers.OnSaveResult{Success: false, Message: "Bot token is required"}
	}

	// Build webhook URL
	webhookURL := fmt.Sprintf("%s/api/integrations/webhooks/telegram", webhookBaseURL)

	// Register webhook with Telegram
	setWebhookURL := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", config.BotToken)

	payload := map[string]interface{}{
		"url":             webhookURL,
		"allowed_updates": []string{"message", "callback_query"},
	}
	payloadBytes, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(setWebhookURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return drivers.OnSaveResult{Success: false, Message: "Failed to connect to Telegram", Details: err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return drivers.OnSaveResult{Success: false, Message: "Invalid response from Telegram", Details: err.Error()}
	}

	if ok, _ := result["ok"].(bool); !ok {
		desc, _ := result["description"].(string)
		return drivers.OnSaveResult{Success: false, Message: "Failed to register webhook", Details: desc}
	}

	return drivers.OnSaveResult{
		Success: true,
		Message: fmt.Sprintf("Telegram webhook registered: %s", webhookURL),
	}
}
