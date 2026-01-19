// Package smtp provides the SMTP email integration driver.
package smtp

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/smtp"

	"github.com/iesreza/homa-backend/apps/integrations/drivers"
	"github.com/iesreza/homa-backend/apps/integrations/email"
)

const (
	// TypeID is the unique identifier for this integration.
	TypeID = "smtp"
	// DisplayName is the human-readable name.
	DisplayName = "SMTP"
)

// Config holds SMTP integration configuration.
type Config struct {
	// SMTP settings (for sending)
	SMTPHost       string `json:"smtp_host"`
	SMTPPort       int    `json:"smtp_port"`
	SMTPUsername   string `json:"smtp_username"`
	SMTPPassword   string `json:"smtp_password"`
	SMTPEncryption string `json:"smtp_encryption"` // none, ssl, tls

	// IMAP settings (for receiving) - optional
	IMAPEnabled    bool   `json:"imap_enabled"`
	IMAPHost       string `json:"imap_host"`
	IMAPPort       int    `json:"imap_port"`
	IMAPUsername   string `json:"imap_username"`
	IMAPPassword   string `json:"imap_password"`
	IMAPEncryption string `json:"imap_encryption"` // none, ssl, tls

	// Common settings
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`

	// HTML template for outgoing emails
	Template string `json:"template"`
}

// Driver implements the drivers.Driver interface for SMTP.
type Driver struct{}

// New creates a new SMTP driver instance.
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

	// Test SMTP connection
	smtpResult := d.testSMTP(config)
	if !smtpResult.Success {
		return smtpResult
	}

	// Test IMAP if enabled
	if config.IMAPEnabled {
		imapResult := d.testIMAP(config)
		if !imapResult.Success {
			return drivers.TestResult{
				Success: false,
				Message: "SMTP OK, but IMAP failed",
				Details: imapResult.Details,
			}
		}
		return drivers.TestResult{
			Success: true,
			Message: fmt.Sprintf("SMTP and IMAP connections successful. SMTP: %s:%d, IMAP: %s:%d",
				config.SMTPHost, config.SMTPPort, config.IMAPHost, config.IMAPPort),
		}
	}

	return smtpResult
}

// testSMTP tests the SMTP connection.
func (d *Driver) testSMTP(config Config) drivers.TestResult {
	if config.SMTPHost == "" || config.SMTPPort == 0 {
		return drivers.TestResult{Success: false, Message: "SMTP Host and Port are required"}
	}

	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)

	var client *smtp.Client
	var err error

	if config.SMTPEncryption == "ssl" || config.SMTPPort == 465 {
		// SSL/TLS connection
		tlsConfig := &tls.Config{ServerName: config.SMTPHost}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return drivers.TestResult{Success: false, Message: "Failed to connect to SMTP server (SSL)", Details: err.Error()}
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, config.SMTPHost)
		if err != nil {
			return drivers.TestResult{Success: false, Message: "Failed to create SMTP client", Details: err.Error()}
		}
	} else {
		// Plain or STARTTLS connection
		client, err = smtp.Dial(addr)
		if err != nil {
			return drivers.TestResult{Success: false, Message: "Failed to connect to SMTP server", Details: err.Error()}
		}

		if config.SMTPEncryption == "tls" {
			tlsConfig := &tls.Config{ServerName: config.SMTPHost}
			if err = client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return drivers.TestResult{Success: false, Message: "Failed to start TLS", Details: err.Error()}
			}
		}
	}
	defer client.Close()

	// Authenticate if credentials provided
	if config.SMTPUsername != "" && config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return drivers.TestResult{Success: false, Message: "SMTP authentication failed", Details: err.Error()}
		}
	}

	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to SMTP server: %s:%d", config.SMTPHost, config.SMTPPort),
	}
}

// testIMAP tests the IMAP connection.
func (d *Driver) testIMAP(config Config) drivers.TestResult {
	if config.IMAPHost == "" || config.IMAPPort == 0 {
		return drivers.TestResult{Success: false, Message: "IMAP Host and Port are required"}
	}

	emailConfig := email.Config{
		Provider:       email.ProviderSMTP,
		AuthType:       email.AuthTypeBasic,
		IMAPEnabled:    true,
		IMAPHost:       config.IMAPHost,
		IMAPPort:       config.IMAPPort,
		IMAPUsername:   config.IMAPUsername,
		IMAPPassword:   config.IMAPPassword,
		IMAPEncryption: config.IMAPEncryption,
	}

	client := email.NewIMAPClient(emailConfig)
	if err := client.TestConnection(); err != nil {
		return drivers.TestResult{Success: false, Message: "IMAP connection failed", Details: err.Error()}
	}

	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("IMAP connection successful: %s:%d", config.IMAPHost, config.IMAPPort),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid SMTP configuration: %w", err)
	}
	if config.SMTPHost == "" || config.SMTPPort == 0 || config.FromEmail == "" {
		return fmt.Errorf("SMTP host, port, and from email are required")
	}

	// Validate IMAP if enabled
	if config.IMAPEnabled {
		if config.IMAPHost == "" || config.IMAPPort == 0 {
			return fmt.Errorf("IMAP host and port are required when IMAP is enabled")
		}
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
		// SMTP Settings
		{Name: "smtp_host", Label: "SMTP Host", Type: "text", Required: true, Placeholder: "smtp.example.com"},
		{Name: "smtp_port", Label: "SMTP Port", Type: "number", Required: true, Placeholder: "587"},
		{Name: "smtp_username", Label: "SMTP Username", Type: "text", Required: false, Placeholder: "user@example.com"},
		{Name: "smtp_password", Label: "SMTP Password", Type: "password", Required: false, Placeholder: "Password or app-specific password"},
		{Name: "smtp_encryption", Label: "SMTP Encryption", Type: "select", Required: false, Options: []string{"none", "ssl", "tls"}},

		// Common Settings
		{Name: "from_email", Label: "From Email", Type: "email", Required: true, Placeholder: "noreply@example.com"},
		{Name: "from_name", Label: "From Name", Type: "text", Required: false, Placeholder: "Support Team"},

		// IMAP Settings (Optional)
		{Name: "imap_enabled", Label: "Enable IMAP (Receive Emails)", Type: "boolean", Required: false},
		{Name: "imap_host", Label: "IMAP Host", Type: "text", Required: false, Placeholder: "imap.example.com"},
		{Name: "imap_port", Label: "IMAP Port", Type: "number", Required: false, Placeholder: "993"},
		{Name: "imap_username", Label: "IMAP Username", Type: "text", Required: false, Placeholder: "user@example.com"},
		{Name: "imap_password", Label: "IMAP Password", Type: "password", Required: false, Placeholder: "Password"},
		{Name: "imap_encryption", Label: "IMAP Encryption", Type: "select", Required: false, Options: []string{"none", "ssl", "tls"}},

		// Template
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
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.OnSaveResult{Success: false, Message: "Invalid configuration"}
	}

	message := "SMTP integration saved successfully."
	if config.IMAPEnabled {
		message += " Email fetching is enabled - the system will check for new emails periodically."
	}

	return drivers.OnSaveResult{
		Success: true,
		Message: message,
	}
}

// GetEmailConfig converts the SMTP config to the unified email.Config.
func GetEmailConfig(configJSON string) (*email.Config, error) {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	return &email.Config{
		Provider:       email.ProviderSMTP,
		AuthType:       email.AuthTypeBasic,
		IMAPEnabled:    config.IMAPEnabled,
		IMAPHost:       config.IMAPHost,
		IMAPPort:       config.IMAPPort,
		IMAPUsername:   config.IMAPUsername,
		IMAPPassword:   config.IMAPPassword,
		IMAPEncryption: config.IMAPEncryption,
		SMTPHost:       config.SMTPHost,
		SMTPPort:       config.SMTPPort,
		SMTPUsername:   config.SMTPUsername,
		SMTPPassword:   config.SMTPPassword,
		SMTPEncryption: config.SMTPEncryption,
		FromEmail:      config.FromEmail,
		FromName:       config.FromName,
		Email:          config.FromEmail,
		Template:       config.Template,
	}, nil
}
