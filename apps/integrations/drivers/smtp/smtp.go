// Package smtp provides the SMTP email integration driver.
package smtp

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/smtp"

	"github.com/iesreza/homa-backend/apps/integrations/drivers"
	"github.com/iesreza/homa-backend/apps/models"
)

const (
	// TypeID is the unique identifier for this integration.
	TypeID = "smtp"
	// DisplayName is the human-readable name.
	DisplayName = "SMTP"
)

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
	var config models.SMTPConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return drivers.TestResult{Success: false, Message: "Invalid configuration", Details: err.Error()}
	}

	if config.Host == "" || config.Port == 0 {
		return drivers.TestResult{Success: false, Message: "Host and Port are required"}
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var client *smtp.Client
	var err error

	if config.Encryption == "ssl" || config.Port == 465 {
		// SSL/TLS connection
		tlsConfig := &tls.Config{
			ServerName: config.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return drivers.TestResult{Success: false, Message: "Failed to connect to SMTP server (SSL)", Details: err.Error()}
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, config.Host)
		if err != nil {
			return drivers.TestResult{Success: false, Message: "Failed to create SMTP client", Details: err.Error()}
		}
	} else {
		// Plain or STARTTLS connection
		client, err = smtp.Dial(addr)
		if err != nil {
			return drivers.TestResult{Success: false, Message: "Failed to connect to SMTP server", Details: err.Error()}
		}

		if config.Encryption == "tls" {
			tlsConfig := &tls.Config{
				ServerName: config.Host,
			}
			if err = client.StartTLS(tlsConfig); err != nil {
				return drivers.TestResult{Success: false, Message: "Failed to start TLS", Details: err.Error()}
			}
		}
	}
	defer client.Close()

	// Authenticate if credentials provided
	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := client.Auth(auth); err != nil {
			return drivers.TestResult{Success: false, Message: "SMTP authentication failed", Details: err.Error()}
		}
	}

	return drivers.TestResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to SMTP server: %s:%d", config.Host, config.Port),
	}
}

// Validate checks if the configuration is complete and valid.
func (d *Driver) Validate(configJSON string) error {
	var config models.SMTPConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid SMTP configuration: %w", err)
	}
	if config.Host == "" || config.Port == 0 || config.FromEmail == "" {
		return fmt.Errorf("host, port, and from email are required")
	}
	return nil
}

// GetConfigFields returns the required configuration fields.
func (d *Driver) GetConfigFields() []drivers.ConfigField {
	return []drivers.ConfigField{
		{Name: "host", Label: "SMTP Host", Type: "text", Required: true, Placeholder: "smtp.example.com"},
		{Name: "port", Label: "Port", Type: "number", Required: true, Placeholder: "587"},
		{Name: "username", Label: "Username", Type: "text", Required: false, Placeholder: "user@example.com"},
		{Name: "password", Label: "Password", Type: "password", Required: false, Placeholder: "Password or app-specific password"},
		{Name: "from_email", Label: "From Email", Type: "email", Required: true, Placeholder: "noreply@example.com"},
		{Name: "from_name", Label: "From Name", Type: "text", Required: false, Placeholder: "Support Team"},
		{Name: "encryption", Label: "Encryption", Type: "select", Required: false, Options: []string{"none", "ssl", "tls"}},
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
// SMTP doesn't require any post-save actions.
func (d *Driver) OnSave(configJSON string, status string, webhookBaseURL string) drivers.OnSaveResult {
	return drivers.OnSaveResult{
		Success: true,
		Message: "SMTP integration saved successfully.",
	}
}
