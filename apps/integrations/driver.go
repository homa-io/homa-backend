// Package integrations provides integration management functionality.
package integrations

import (
	"encoding/json"
	"strings"

	"github.com/iesreza/homa-backend/apps/integrations/drivers"
	"github.com/iesreza/homa-backend/apps/integrations/drivers/gmail"
	"github.com/iesreza/homa-backend/apps/integrations/drivers/outlook"
	"github.com/iesreza/homa-backend/apps/integrations/drivers/slack"
	"github.com/iesreza/homa-backend/apps/integrations/drivers/smtp"
	"github.com/iesreza/homa-backend/apps/integrations/drivers/telegram"
	"github.com/iesreza/homa-backend/apps/integrations/drivers/whatsapp"
	"github.com/iesreza/homa-backend/apps/models"
)

func init() {
	// Register all integration drivers
	drivers.Register(slack.New())
	drivers.Register(telegram.New())
	drivers.Register(whatsapp.New())
	drivers.Register(smtp.New())
	drivers.Register(gmail.New())
	drivers.Register(outlook.New())
}

// TestResult represents the result of an integration test.
type TestResult = drivers.TestResult

// OnSaveResult represents the result of a post-save callback.
type OnSaveResult = drivers.OnSaveResult

// ConfigField represents a configuration field for an integration.
type ConfigField = drivers.ConfigField

// TestIntegration tests the connection for an integration.
func TestIntegration(integrationType string, configJSON string) TestResult {
	driver, ok := drivers.Get(integrationType)
	if !ok {
		return TestResult{
			Success: false,
			Message: "Unknown integration type",
		}
	}
	return driver.Test(configJSON)
}

// ValidateConfig validates the configuration for a specific integration type.
func ValidateConfig(integrationType string, configJSON string) error {
	driver, ok := drivers.Get(integrationType)
	if !ok {
		return nil // Unknown types pass validation (for backwards compatibility)
	}
	return driver.Validate(configJSON)
}

// GetConfigFields returns the required fields for each integration type.
func GetConfigFields(integrationType string) []ConfigField {
	driver, ok := drivers.Get(integrationType)
	if !ok {
		return nil
	}
	return driver.GetConfigFields()
}

// GetMaskedConfig returns the config with sensitive fields masked.
func GetMaskedConfig(integrationType string, configJSON string) map[string]interface{} {
	driver, ok := drivers.Get(integrationType)
	if !ok {
		// Fallback to generic masking if driver not found
		return getMaskedConfigGeneric(configJSON)
	}
	return driver.GetMaskedConfig(configJSON)
}

// OnSave is called after an integration is saved. It performs any necessary
// post-save operations like registering webhooks.
func OnSave(integrationType string, configJSON string, status string, webhookBaseURL string) OnSaveResult {
	driver, ok := drivers.Get(integrationType)
	if !ok {
		return OnSaveResult{Success: true, Message: "No post-save actions required"}
	}
	return driver.OnSave(configJSON, status, webhookBaseURL)
}

// MergeConfigWithExisting merges new config with existing config,
// preserving sensitive fields if the new value appears to be masked (contains "...").
func MergeConfigWithExisting(existingConfigJSON string, newConfig map[string]interface{}) map[string]interface{} {
	// Parse existing config
	var existingConfig map[string]interface{}
	if err := json.Unmarshal([]byte(existingConfigJSON), &existingConfig); err != nil {
		existingConfig = make(map[string]interface{})
	}

	// Start with all new values
	result := make(map[string]interface{})
	for key, value := range newConfig {
		result[key] = value
	}

	// For sensitive fields, check if the value looks masked
	for key := range drivers.SensitiveFields {
		if newVal, ok := newConfig[key]; ok {
			if newStr, isStr := newVal.(string); isStr {
				// If the new value contains "..." it's likely masked - preserve existing
				if strings.Contains(newStr, "...") {
					if existingVal, exists := existingConfig[key]; exists {
						result[key] = existingVal
					}
				}
			}
		}
	}

	return result
}

// EncryptConfig encrypts configuration JSON (placeholder - implement proper encryption).
func EncryptConfig(config string) string {
	// TODO: Implement proper encryption using AES-256-GCM
	return config
}

// DecryptConfig decrypts configuration JSON (placeholder - implement proper decryption).
func DecryptConfig(encryptedConfig string) string {
	// TODO: Implement proper decryption using AES-256-GCM
	return encryptedConfig
}

// GetIntegrationTypes returns all available integration types.
func GetIntegrationTypes() []models.IntegrationTypeInfo {
	return models.GetIntegrationTypes()
}

// getMaskedConfigGeneric masks sensitive fields without knowing the driver type.
func getMaskedConfigGeneric(configJSON string) map[string]interface{} {
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
