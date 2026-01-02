// Package drivers provides the interface and common types for integration drivers.
package drivers

// TestResult represents the result of a connection test.
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// OnSaveResult represents the result of a post-save callback.
type OnSaveResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ConfigField represents a configuration field for an integration.
type ConfigField struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // text, password, email, number, select
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"` // For select type
}

// Driver defines the interface that all integration drivers must implement.
type Driver interface {
	// Type returns the unique identifier for this integration type.
	Type() string

	// Name returns the display name for this integration.
	Name() string

	// Test validates the connection using the provided configuration.
	Test(configJSON string) TestResult

	// Validate checks if the configuration is complete and valid.
	Validate(configJSON string) error

	// GetConfigFields returns the required configuration fields.
	GetConfigFields() []ConfigField

	// GetMaskedConfig returns the config with sensitive fields masked.
	GetMaskedConfig(configJSON string) map[string]interface{}

	// OnSave is called after the integration is saved.
	// webhookBaseURL is provided for integrations that need to register webhooks.
	OnSave(configJSON string, status string, webhookBaseURL string) OnSaveResult
}

// SensitiveFields contains field names that should be masked in responses.
var SensitiveFields = map[string]bool{
	"bot_token":       true,
	"signing_secret":  true,
	"app_level_token": true,
	"access_token":    true,
	"password":        true,
	"client_secret":   true,
	"refresh_token":   true,
}
