// Package whatsapp provides the WhatsApp Business API integration driver.
package whatsapp

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
	TypeID = "whatsapp"
	// DisplayName is the human-readable name.
	DisplayName = "WhatsApp"
)

// Driver implements the drivers.Driver interface for WhatsApp.
type Driver struct{}

// New creates a new WhatsApp driver instance.
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
	var config models.WhatsAppConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid configuration", Details: err.Error()}
	}

	if config.PhoneNumberID == "" || config.AccessToken == "" {
		return drivers.TestResult{Success: false, Message: "Phone Number ID and Access Token are required"}
	}

	// Test by calling WhatsApp Business API
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s", config.PhoneNumberID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to create request", Details: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to connect to WhatsApp Business API", Details: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return drivers.TestResult{Success: false, Message: "WhatsApp API authentication failed", Details: string(body)}
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid response from WhatsApp", Details: err.Error()}
	}

	displayNumber, _ := result["display_phone_number"].(string)
	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to WhatsApp: %s", displayNumber),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config models.WhatsAppConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid WhatsApp configuration: %w", err)
	}
	if config.PhoneNumberID == "" || config.AccessToken == "" {
		return fmt.Errorf("phone number ID and access token are required")
	}
	return nil
}

// GetConfigFields returns the required configuration fields.
func (d *Driver) GetConfigFields() []drivers.ConfigField {
	return []drivers.ConfigField{
		{Name: "phone_number_id", Label: "Phone Number ID", Type: "text", Required: true, Placeholder: "1234567890"},
		{Name: "business_id", Label: "Business ID", Type: "text", Required: false, Placeholder: "Meta Business ID"},
		{Name: "access_token", Label: "Access Token", Type: "password", Required: true, Placeholder: "Permanent access token"},
		{Name: "webhook_verify_token", Label: "Webhook Verify Token", Type: "text", Required: false, Placeholder: "Custom verification token"},
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
// WhatsApp webhooks are configured in the Meta Developer Console.
func (d *Driver) OnSave(configJSON string, status string, webhookBaseURL string) drivers.OnSaveResult {
	webhookURL := webhookBaseURL + "/api/integrations/webhooks/whatsapp"

	instructions := `WhatsApp integration saved successfully!

To complete setup, configure the webhook in Meta Developer Console:

1. Go to https://developers.facebook.com/
2. Select your App
3. Navigate to "Whatsapp" > "Configuration"
4. Under "Webhook", click "Edit"
5. Set the Callback URL to: ` + webhookURL + `
6. Verify Token: Use the "Webhook Verify Token" from your integration config
7. Click "Verify and Save"
8. Meta will send a verification request to confirm the URL
9. Once verified, enable message delivery notifications

Event Subscriptions to enable:
- messages (receive incoming messages)
- message_status (track delivery status of outgoing messages)

After setup, your WhatsApp business account will send messages to this dashboard.

Note: Ensure your WhatsApp phone number is connected to your Business Account
and has the required permissions for sending/receiving messages.`

	return drivers.OnSaveResult{
		Success: true,
		Message: instructions,
	}
}
