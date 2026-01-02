// Package slack provides the Slack integration driver.
package slack

import (
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
	TypeID = "slack"
	// DisplayName is the human-readable name.
	DisplayName = "Slack"
)

// Driver implements the drivers.Driver interface for Slack.
type Driver struct{}

// New creates a new Slack driver instance.
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
	var config models.SlackConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid configuration", Details: err.Error()}
	}

	if config.BotToken == "" {
		return drivers.TestResult{Success: false, Message: "Bot token is required"}
	}

	// Test by calling Slack's auth.test API
	req, err := http.NewRequest("GET", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to create request", Details: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+config.BotToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to connect to Slack", Details: err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid response from Slack", Details: err.Error()}
	}

	if ok, _ := result["ok"].(bool); !ok {
		errMsg, _ := result["error"].(string)
		return drivers.TestResult{Success: false, Message: "Slack authentication failed", Details: errMsg}
	}

	teamName, _ := result["team"].(string)
	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to Slack workspace: %s", teamName),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config models.SlackConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid Slack configuration: %w", err)
	}
	if config.BotToken == "" {
		return fmt.Errorf("bot token is required")
	}
	return nil
}

// GetConfigFields returns the required configuration fields.
func (d *Driver) GetConfigFields() []drivers.ConfigField {
	return []drivers.ConfigField{
		{Name: "bot_token", Label: "Bot Token", Type: "password", Required: true, Placeholder: "xoxb-..."},
		{Name: "signing_secret", Label: "Signing Secret", Type: "password", Required: false, Placeholder: "Signing secret for webhook verification"},
		{Name: "app_level_token", Label: "App Level Token", Type: "password", Required: false, Placeholder: "xapp-... (for Socket Mode)"},
		{Name: "default_channel_id", Label: "Default Channel ID", Type: "text", Required: false, Placeholder: "C01234567"},
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
// Slack uses event subscriptions configured in the Slack app settings, so no auto-registration is needed.
func (d *Driver) OnSave(configJSON string, status string, webhookBaseURL string) drivers.OnSaveResult {
	return drivers.OnSaveResult{
		Success: true,
		Message: "Slack integration saved. Configure event subscriptions in your Slack App settings.",
	}
}
