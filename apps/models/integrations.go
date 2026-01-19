package models

import (
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/restify"
)

// Integration types
const (
	IntegrationTypeSlack    = "slack"
	IntegrationTypeTelegram = "telegram"
	IntegrationTypeWhatsApp = "whatsapp"
	IntegrationTypeSMTP     = "smtp"
	IntegrationTypeGmail    = "gmail"
	IntegrationTypeOutlook  = "outlook"
)

// Integration statuses
const (
	IntegrationStatusDisabled = "disabled"
	IntegrationStatusEnabled  = "enabled"
	IntegrationStatusError    = "error"
)

// Integration represents a messaging channel integration configuration
type Integration struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"size:50;not null;uniqueIndex" json:"type"` // slack, telegram, whatsapp, smtp, gmail, outlook
	Name      string    `gorm:"size:255;not null" json:"name"`
	Status    string    `gorm:"size:50;not null;default:'disabled'" json:"status"` // disabled, enabled, error
	Config    string    `gorm:"type:text" json:"-"`                                // Encrypted JSON config (hidden from API)
	LastError string    `gorm:"type:text" json:"last_error,omitempty"`
	InboxID   *uint     `gorm:"index" json:"inbox_id,omitempty"`                   // Default inbox for conversations from this integration
	Inbox     *Inbox    `gorm:"foreignKey:InboxID" json:"inbox,omitempty"`
	TestedAt  *time.Time `json:"tested_at,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	restify.API `json:"-"`
}

// TableName returns the table name for the Integration model
func (Integration) TableName() string {
	return "integrations"
}

// SlackConfig holds Slack integration configuration
type SlackConfig struct {
	BotToken          string `json:"bot_token"`
	SigningSecret     string `json:"signing_secret"`
	AppLevelToken     string `json:"app_level_token,omitempty"`
	DefaultChannelID  string `json:"default_channel_id,omitempty"`
}

// TelegramConfig holds Telegram integration configuration
type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	WebhookURL string `json:"webhook_url,omitempty"`
}

// WhatsAppConfig holds WhatsApp Business integration configuration
type WhatsAppConfig struct {
	PhoneNumberID   string `json:"phone_number_id"`
	BusinessID      string `json:"business_id"`
	AccessToken     string `json:"access_token"`
	WebhookVerifyToken string `json:"webhook_verify_token,omitempty"`
}

// SMTPConfig holds SMTP integration configuration
type SMTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"from_email"`
	FromName   string `json:"from_name"`
	Encryption string `json:"encryption"` // none, ssl, tls
}

// GmailConfig holds Gmail integration configuration
type GmailConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email"`
}

// OutlookConfig holds Outlook integration configuration
type OutlookConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id"`
	RefreshToken string `json:"refresh_token"`
	Email        string `json:"email"`
}

// GetIntegration retrieves an integration by type
func GetIntegration(integrationType string) (*Integration, error) {
	var integration Integration
	err := db.Where("type = ?", integrationType).First(&integration).Error
	return &integration, err
}

// GetAllIntegrations retrieves all integrations
func GetAllIntegrations() ([]Integration, error) {
	var integrations []Integration
	err := db.Order("type ASC").Find(&integrations).Error
	return integrations, err
}

// GetEnabledIntegrations retrieves all enabled integrations
func GetEnabledIntegrations() ([]Integration, error) {
	var integrations []Integration
	err := db.Where("status = ?", IntegrationStatusEnabled).Find(&integrations).Error
	return integrations, err
}

// GetEnabledEmailIntegrations retrieves all enabled email integrations (SMTP, Gmail, Outlook)
func GetEnabledEmailIntegrations() ([]Integration, error) {
	var integrations []Integration
	emailTypes := []string{IntegrationTypeSMTP, IntegrationTypeGmail, IntegrationTypeOutlook}
	err := db.Where("status = ?", IntegrationStatusEnabled).
		Where("type IN ?", emailTypes).
		Find(&integrations).Error
	return integrations, err
}

// UpsertIntegration creates or updates an integration by type
func UpsertIntegration(integration *Integration) error {
	var existing Integration
	err := db.Where("type = ?", integration.Type).First(&existing).Error
	if err != nil {
		// Create new
		return db.Create(integration).Error
	}
	// Update existing
	integration.ID = existing.ID
	return db.Save(integration).Error
}

// IntegrationTypeInfo provides display info for integration types
type IntegrationTypeInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// GetIntegrationTypes returns all available integration types with info
func GetIntegrationTypes() []IntegrationTypeInfo {
	return []IntegrationTypeInfo{
		{
			Type:        IntegrationTypeSlack,
			Name:        "Slack",
			Description: "Send and receive messages via Slack channels",
			Icon:        "slack",
		},
		{
			Type:        IntegrationTypeTelegram,
			Name:        "Telegram",
			Description: "Connect with customers through Telegram bot",
			Icon:        "telegram",
		},
		{
			Type:        IntegrationTypeWhatsApp,
			Name:        "WhatsApp Business",
			Description: "Handle customer support via WhatsApp Business API",
			Icon:        "whatsapp",
		},
		{
			Type:        IntegrationTypeSMTP,
			Name:        "SMTP",
			Description: "Send emails through custom SMTP server",
			Icon:        "mail",
		},
		{
			Type:        IntegrationTypeGmail,
			Name:        "Gmail",
			Description: "Manage email conversations via Gmail",
			Icon:        "gmail",
		},
		{
			Type:        IntegrationTypeOutlook,
			Name:        "Outlook",
			Description: "Integrate with Microsoft Outlook for email",
			Icon:        "outlook",
		},
	}
}
