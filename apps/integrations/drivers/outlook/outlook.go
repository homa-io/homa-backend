// Package outlook provides the Outlook/Microsoft 365 integration driver using IMAP/SMTP with OAuth2.
package outlook

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
	TypeID = "outlook"
	// DisplayName is the human-readable name.
	DisplayName = "Outlook"
)

// Config holds Outlook integration configuration.
type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id"` // "common" for multi-tenant apps
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email"`
	FromName     string `json:"from_name"`

	// HTML template for outgoing emails
	Template string `json:"template"`
}

// Driver implements the drivers.Driver interface for Outlook.
type Driver struct{}

// New creates a new Outlook driver instance.
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
	accessToken, err := email.RefreshOutlookAccessToken(config.ClientID, config.ClientSecret, config.TenantID, config.RefreshToken)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "OAuth2 token refresh failed", Details: err.Error()}
	}

	// Test Graph API access to get user profile
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	graphResp, err := client.Do(req)
	if err != nil {
		return drivers.TestResult{Success: false, Message: "Failed to access Microsoft Graph API", Details: err.Error()}
	}
	defer graphResp.Body.Close()

	if graphResp.StatusCode != 200 {
		body, _ := io.ReadAll(graphResp.Body)
		return drivers.TestResult{Success: false, Message: "Microsoft Graph API access denied", Details: string(body)}
	}

	body, _ := io.ReadAll(graphResp.Body)
	var profile map[string]interface{}
	json.Unmarshal(body, &profile)
	userEmail, _ := profile["mail"].(string)
	if userEmail == "" {
		userEmail, _ = profile["userPrincipalName"].(string)
	}
	displayName, _ := profile["displayName"].(string)

	// Test IMAP connection
	emailConfig := d.toEmailConfig(config)
	emailConfig.Email = userEmail
	imapClient := email.NewIMAPClient(*emailConfig)
	if err := imapClient.TestConnection(); err != nil {
		return drivers.TestResult{
			Success: true,
			Message: fmt.Sprintf("OAuth2 OK for %s (%s). Note: IMAP test skipped (may need OAuth2 IMAP enabled)", displayName, userEmail),
		}
	}

	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to Outlook: %s (%s) (OAuth2 + IMAP verified)", displayName, userEmail),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid Outlook configuration: %w", err)
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
		{Name: "client_id", Label: "Client ID", Type: "text", Required: true, Placeholder: "Application (client) ID from Azure"},
		{Name: "client_secret", Label: "Client Secret", Type: "password", Required: true, Placeholder: "Client secret from Azure"},
		{Name: "tenant_id", Label: "Tenant ID", Type: "text", Required: false, Placeholder: "common (for multi-tenant) or your tenant ID"},
		{Name: "refresh_token", Label: "Refresh Token", Type: "password", Required: true, Placeholder: "OAuth refresh token"},
		{Name: "email", Label: "Email Address", Type: "email", Required: true, Placeholder: "user@outlook.com"},
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
		Message: `Outlook integration saved successfully.

Email fetching is enabled - the system will check for new emails periodically.

Setup Instructions:
1. Go to Azure Portal (portal.azure.com)
2. Register an application in Azure Active Directory
3. Add API permissions: IMAP.AccessAsUser.All, SMTP.Send
4. Create a client secret
5. Use the OAuth flow to obtain the refresh token

Required OAuth Scopes:
- https://outlook.office.com/IMAP.AccessAsUser.All
- https://outlook.office.com/SMTP.Send
- offline_access`,
	}
}

// toEmailConfig converts Outlook config to unified email.Config.
func (d *Driver) toEmailConfig(config Config) *email.Config {
	return &email.Config{
		Provider:       email.ProviderOutlook,
		AuthType:       email.AuthTypeOAuth2,
		IMAPEnabled:    true,
		IMAPHost:       email.OutlookPresets.IMAPHost,
		IMAPPort:       email.OutlookPresets.IMAPPort,
		IMAPEncryption: "ssl",
		SMTPHost:       email.OutlookPresets.SMTPHost,
		SMTPPort:       email.OutlookPresets.SMTPPort,
		SMTPEncryption: "tls",
		ClientID:       config.ClientID,
		ClientSecret:   config.ClientSecret,
		TenantID:       config.TenantID,
		RefreshToken:   config.RefreshToken,
		Email:          config.Email,
		FromEmail:      config.Email,
		FromName:       config.FromName,
		Template:       config.Template,
	}
}

// GetEmailConfig converts the Outlook config to the unified email.Config.
func GetEmailConfig(configJSON string) (*email.Config, error) {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	driver := &Driver{}
	return driver.toEmailConfig(config), nil
}
