// Package gmail provides the Gmail API integration driver.
package gmail

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/iesreza/homa-backend/apps/integrations/drivers"
	"github.com/iesreza/homa-backend/apps/models"
)

const (
	// TypeID is the unique identifier for this integration.
	TypeID = "gmail"
	// DisplayName is the human-readable name.
	DisplayName = "Gmail"
)

// Driver implements the drivers.Driver interface for Gmail.
type Driver struct{}

// New creates a new Gmail driver instance.
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
	var config models.GmailConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid configuration", Details: err.Error()}
	}

	if config.ClientID == "" || config.ClientSecret == "" || config.RefreshToken == "" {
		return drivers.TestResult{Success: false, Message: "Client ID, Client Secret, and Refresh Token are required"}
	}

	// Exchange refresh token for access token
	tokenURL := "https://oauth2.googleapis.com/token"
	data := fmt.Sprintf("client_id=%s&client_secret=%s&refresh_token=%s&grant_type=refresh_token",
		config.ClientID, config.ClientSecret, config.RefreshToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to connect to Google OAuth", Details: err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tokenResult map[string]interface{}
	if err := json.Unmarshal(body, &tokenResult); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid response from Google", Details: err.Error()}
	}

	if _, ok := tokenResult["error"]; ok {
		errDesc, _ := tokenResult["error_description"].(string)
		return drivers.TestResult{Success: false, Message: "Google authentication failed", Details: errDesc}
	}

	accessToken, _ := tokenResult["access_token"].(string)
	if accessToken == "" {
		return drivers.TestResult{Success: false, Message: "Failed to obtain access token"}
	}

	// Test Gmail API access
	req, _ := http.NewRequest("GET", "https://gmail.googleapis.com/gmail/v1/users/me/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	gmailResp, err := client.Do(req)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to access Gmail API", Details: err.Error()}
	}
	defer gmailResp.Body.Close()

	if gmailResp.StatusCode != 200 {
		body, _ := io.ReadAll(gmailResp.Body)
		return drivers.TestResult{Success: false, Message: "Gmail API access denied", Details: string(body)}
	}

	body, _ = io.ReadAll(gmailResp.Body)
	var profile map[string]interface{}
	json.Unmarshal(body, &profile)
	email, _ := profile["emailAddress"].(string)

	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to Gmail: %s", email),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config models.GmailConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid Gmail configuration: %w", err)
	}
	if config.ClientID == "" || config.ClientSecret == "" || config.RefreshToken == "" {
		return fmt.Errorf("client ID, client secret, and refresh token are required")
	}
	return nil
}

// GetConfigFields returns the required configuration fields.
func (d *Driver) GetConfigFields() []drivers.ConfigField {
	return []drivers.ConfigField{
		{Name: "client_id", Label: "Client ID", Type: "text", Required: true, Placeholder: "xxx.apps.googleusercontent.com"},
		{Name: "client_secret", Label: "Client Secret", Type: "password", Required: true, Placeholder: "Client secret from Google Console"},
		{Name: "refresh_token", Label: "Refresh Token", Type: "password", Required: true, Placeholder: "OAuth refresh token"},
		{Name: "email", Label: "Email Address", Type: "email", Required: false, Placeholder: "user@gmail.com"},
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
// Gmail doesn't require any post-save actions.
func (d *Driver) OnSave(configJSON string, status string, webhookBaseURL string) drivers.OnSaveResult {
	return drivers.OnSaveResult{
		Success: true,
		Message: "Gmail integration saved successfully.",
	}
}
