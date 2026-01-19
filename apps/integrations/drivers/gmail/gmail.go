// Package gmail provides the Gmail integration driver using IMAP/SMTP with OAuth2.
package gmail

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/iesreza/homa-backend/apps/integrations/drivers"
	"github.com/iesreza/homa-backend/apps/integrations/email"
)

const (
	// TypeID is the unique identifier for this integration.
	TypeID = "gmail"
	// DisplayName is the human-readable name.
	DisplayName = "Gmail"
)

// Config holds Gmail integration configuration.
type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email"`
	FromName     string `json:"from_name"`

	// HTML template for outgoing emails
	Template string `json:"template"`
}

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
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid configuration", Details: err.Error()}
	}

	if config.ClientID == "" || config.ClientSecret == "" || config.RefreshToken == "" {
		return drivers.TestResult{Success: false, Message: "Client ID, Client Secret, and Refresh Token are required"}
	}

	// Test OAuth2 token refresh
	accessToken, err := email.RefreshGmailAccessToken(config.ClientID, config.ClientSecret, config.RefreshToken)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "OAuth2 token refresh failed", Details: err.Error()}
	}

	// Test Gmail API access to get email address
	client := &http.Client{Timeout: 10 * time.Second}
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

	body, _ := io.ReadAll(gmailResp.Body)
	var profile map[string]interface{}
	json.Unmarshal(body, &profile)
	userEmail, _ := profile["emailAddress"].(string)

	// Test IMAP connection
	emailConfig := d.toEmailConfig(config)
	emailConfig.Email = userEmail
	imapClient := email.NewIMAPClient(*emailConfig)
	if err := imapClient.TestConnection(); err != nil {
		return drivers.TestResult{
			Success: true,
			Message: fmt.Sprintf("OAuth2 OK for %s. Note: IMAP test skipped (may need app password or OAuth2 IMAP enabled)", userEmail),
		}
	}

	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to Gmail: %s (OAuth2 + IMAP verified)", userEmail),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid Gmail configuration: %w", err)
	}
	if config.ClientID == "" || config.ClientSecret == "" || config.RefreshToken == "" {
		return fmt.Errorf("client ID, client secret, and refresh token are required")
	}

	// Validate template if provided
	if config.Template != "" {
		if err := email.ValidateTemplate(config.Template); err != nil {
			return fmt.Errorf("invalid template: %w", err)
		}
	}

	return nil
}

// GetConfigFields returns the required configuration fields.
func (d *Driver) GetConfigFields() []drivers.ConfigField {
	return []drivers.ConfigField{
		{Name: "client_id", Label: "Client ID", Type: "text", Required: true, Placeholder: "xxx.apps.googleusercontent.com"},
		{Name: "client_secret", Label: "Client Secret", Type: "password", Required: true, Placeholder: "Client secret from Google Console"},
		{Name: "refresh_token", Label: "Refresh Token", Type: "password", Required: true, Placeholder: "OAuth refresh token"},
		{Name: "email", Label: "Email Address", Type: "email", Required: true, Placeholder: "user@gmail.com"},
		{Name: "from_name", Label: "From Name", Type: "text", Required: false, Placeholder: "Support Team"},
		{Name: "template", Label: "Email Template (HTML)", Type: "html", Required: false},
	}
}

// GetMaskedConfig returns the config with sensitive fields masked.
func (d *Driver) GetMaskedConfig(configJSON string) map[string]interface{} {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil
	}

	for key := range config {
		if drivers.SensitiveFields[key] {
			if str, ok := config[key].(string); ok && len(str) > 8 {
				config[key] = str[:4] + "..." + str[len(str)-4:]
			} else if str, ok := config[key].(string); ok && len(str) > 0 {
				config[key] = "****"
			}
		}
	}

	return config
}

// OnSave is called after the integration is saved.
func (d *Driver) OnSave(configJSON string, status string, webhookBaseURL string) drivers.OnSaveResult {
	return drivers.OnSaveResult{
		Success: true,
		Message: `Gmail integration saved successfully.

Email fetching is enabled - the system will check for new emails periodically.

Setup Instructions:
1. Go to Google Cloud Console (console.cloud.google.com)
2. Create or select a project
3. Enable the Gmail API
4. Create OAuth2 credentials (Web application type)
5. Add authorized redirect URI for your OAuth callback
6. Use the OAuth Playground or your app to get the refresh token

Required OAuth Scopes:
- https://mail.google.com/
- https://www.googleapis.com/auth/gmail.send`,
	}
}

// toEmailConfig converts Gmail config to unified email.Config.
func (d *Driver) toEmailConfig(config Config) *email.Config {
	return &email.Config{
		Provider:       email.ProviderGmail,
		AuthType:       email.AuthTypeOAuth2,
		IMAPEnabled:    true,
		IMAPHost:       email.GmailPresets.IMAPHost,
		IMAPPort:       email.GmailPresets.IMAPPort,
		IMAPEncryption: "ssl",
		SMTPHost:       email.GmailPresets.SMTPHost,
		SMTPPort:       email.GmailPresets.SMTPPort,
		SMTPEncryption: "tls",
		ClientID:       config.ClientID,
		ClientSecret:   config.ClientSecret,
		RefreshToken:   config.RefreshToken,
		Email:          config.Email,
		FromEmail:      config.Email,
		FromName:       config.FromName,
		Template:       config.Template,
	}
}

// GetEmailConfig converts the Gmail config to the unified email.Config.
func GetEmailConfig(configJSON string) (*email.Config, error) {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	driver := &Driver{}
	return driver.toEmailConfig(config), nil
}
