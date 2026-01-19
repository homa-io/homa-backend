package email

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
)

// SendEmailReply sends an email reply for a conversation
// This function is registered with models.SendEmailReply to be called when an agent sends a message
func SendEmailReply(conversationID uint, messageID uint, body string, user *auth.User) error {
	log.Info("[email] Sending email reply for conversation %d, message %d", conversationID, messageID)

	// Get the conversation with client info and external IDs
	var conversation models.Conversation
	if err := db.Preload("Client.ExternalIDs").Preload("Inbox").First(&conversation, conversationID).Error; err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Verify this is an email conversation
	if conversation.ChannelID != "email" {
		return fmt.Errorf("conversation %d is not an email conversation", conversationID)
	}

	// Get customer email from external IDs
	customerEmail := ""
	for _, extID := range conversation.Client.ExternalIDs {
		if extID.Type == models.ExternalIDTypeEmail {
			customerEmail = extID.Value
			break
		}
	}
	if customerEmail == "" {
		return fmt.Errorf("customer has no email address")
	}

	// Get the most recent inbound email for this conversation to get reply-to info
	var lastInboundEmail models.EmailMessage
	err := db.Where("conversation_id = ?", conversationID).
		Where("direction = ?", "inbound").
		Order("received_at DESC").
		First(&lastInboundEmail).Error
	if err != nil {
		log.Warning("[email] No previous inbound email found for conversation %d", conversationID)
	}

	// Get the email integration to use
	emailConfig, integration, err := getEmailIntegrationForConversation(conversation)
	if err != nil {
		return fmt.Errorf("failed to get email integration: %w", err)
	}

	// Build the email subject (Re: original subject)
	subject := GenerateReplySubject(conversation.Title)

	// Build template data
	displayName := "Support"
	avatar := ""
	if user != nil {
		displayName = user.DisplayName
		if user.Avatar != nil {
			avatar = *user.Avatar
		}
	}

	// Get department name
	department := ""
	if conversation.DepartmentID != nil {
		var dept models.Department
		if err := db.First(&dept, *conversation.DepartmentID).Error; err == nil {
			department = dept.Name
		}
	}

	templateData := BuildTemplateData(
		body,
		displayName,
		avatar,
		int(conversationID),
		fmt.Sprintf("CONV-%d", conversationID),
		conversation.Status,
		department,
		conversation.Priority,
	)

	// Render template
	htmlBody, err := RenderTemplate(emailConfig.Template, templateData)
	if err != nil {
		log.Warning("[email] Failed to render template, using plain text: %v", err)
		htmlBody = "<p>" + body + "</p>"
	}

	// Build the email
	email := Email{
		To:       []string{customerEmail},
		From:     emailConfig.FromEmail,
		FromName: emailConfig.FromName,
		Subject:  subject,
		HTMLBody: htmlBody,
		Body:     body,
		Date:     time.Now(),
	}

	// Set reply headers if we have a previous email
	if lastInboundEmail.MessageID != "" {
		email.InReplyTo = lastInboundEmail.MessageID
		// Add all referenced message IDs
		if lastInboundEmail.References != "" {
			email.References = strings.Split(lastInboundEmail.References, " ")
		}
		email.References = append(email.References, lastInboundEmail.MessageID)
	}

	// Send the email using SMTP client
	smtpClient := NewSMTPClient(*emailConfig)
	messageIDHeader, err := smtpClient.Send(email)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Info("[email] Email sent successfully for conversation %d, Message-ID: %s", conversationID, messageIDHeader)

	// Create email tracking record for outbound email
	emailRecord := &models.EmailMessage{
		IntegrationType: integration.Type,
		MessageID:       messageIDHeader,
		ConversationID:  conversationID,
		MessageRecordID: messageID,
		Subject:         subject,
		FromEmail:       emailConfig.FromEmail,
		FromName:        emailConfig.FromName,
		ToEmail:         customerEmail,
		InReplyTo:       email.InReplyTo,
		References:      strings.Join(email.References, " "),
		Direction:       "outbound",
		ReceivedAt:      time.Now(),
	}

	if err := models.CreateEmailMessage(emailRecord); err != nil {
		log.Warning("[email] Failed to create outbound email tracking record: %v", err)
		// Don't fail the operation for tracking error
	}

	return nil
}

// getEmailIntegrationForConversation gets the email integration config to use for a conversation
func getEmailIntegrationForConversation(conversation models.Conversation) (*Config, *models.Integration, error) {
	// First, try to find an integration that matches the conversation's inbox
	if conversation.InboxID != nil {
		var integration models.Integration
		err := db.Where("inbox_id = ?", *conversation.InboxID).
			Where("status = ?", models.IntegrationStatusEnabled).
			Where("type IN ?", []string{models.IntegrationTypeSMTP, models.IntegrationTypeGmail, models.IntegrationTypeOutlook}).
			First(&integration).Error

		if err == nil {
			config, err := getConfigForIntegration(integration)
			if err == nil {
				return config, &integration, nil
			}
		}
	}

	// Fallback: find any enabled email integration
	var integration models.Integration
	err := db.Where("status = ?", models.IntegrationStatusEnabled).
		Where("type IN ?", []string{models.IntegrationTypeSMTP, models.IntegrationTypeGmail, models.IntegrationTypeOutlook}).
		First(&integration).Error

	if err != nil {
		return nil, nil, fmt.Errorf("no enabled email integration found")
	}

	config, err := getConfigForIntegration(integration)
	if err != nil {
		return nil, nil, err
	}

	return config, &integration, nil
}

// getConfigForIntegration converts integration config JSON to email.Config
func getConfigForIntegration(integration models.Integration) (*Config, error) {
	var baseConfig map[string]interface{}
	if err := json.Unmarshal([]byte(integration.Config), &baseConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	switch integration.Type {
	case models.IntegrationTypeSMTP:
		return parseSMTPConfig(baseConfig)
	case models.IntegrationTypeGmail:
		return parseGmailConfig(baseConfig)
	case models.IntegrationTypeOutlook:
		return parseOutlookConfig(baseConfig)
	default:
		return nil, fmt.Errorf("unsupported integration type: %s", integration.Type)
	}
}

// parseSMTPConfig parses SMTP integration config
func parseSMTPConfig(config map[string]interface{}) (*Config, error) {
	c := &Config{
		Provider:       ProviderSMTP,
		AuthType:       AuthTypeBasic,
		SMTPHost:       getString(config, "smtp_host"),
		SMTPPort:       getInt(config, "smtp_port"),
		SMTPUsername:   getString(config, "smtp_username"),
		SMTPPassword:   getString(config, "smtp_password"),
		SMTPEncryption: getString(config, "smtp_encryption"),
		IMAPEnabled:    getBool(config, "imap_enabled"),
		IMAPHost:       getString(config, "imap_host"),
		IMAPPort:       getInt(config, "imap_port"),
		IMAPUsername:   getString(config, "imap_username"),
		IMAPPassword:   getString(config, "imap_password"),
		IMAPEncryption: getString(config, "imap_encryption"),
		FromEmail:      getString(config, "from_email"),
		FromName:       getString(config, "from_name"),
		Email:          getString(config, "from_email"),
		Template:       getString(config, "template"),
	}

	// Auto-complete IMAP settings from SMTP if not provided
	autoCompleteIMAPSettings(c)

	return c, nil
}

// IMAPPreset holds IMAP server settings for known providers
type IMAPPreset struct {
	Host       string
	Port       int
	Encryption string
}

// smtpToIMAPMapping maps SMTP hosts to their corresponding IMAP settings
var smtpToIMAPMapping = map[string]IMAPPreset{
	// Gmail
	"smtp.gmail.com": {Host: "imap.gmail.com", Port: 993, Encryption: "ssl"},
	// Outlook/Office365
	"smtp.office365.com":    {Host: "outlook.office365.com", Port: 993, Encryption: "ssl"},
	"smtp-mail.outlook.com": {Host: "outlook.office365.com", Port: 993, Encryption: "ssl"},
	// Yahoo
	"smtp.mail.yahoo.com": {Host: "imap.mail.yahoo.com", Port: 993, Encryption: "ssl"},
	// iCloud
	"smtp.mail.me.com": {Host: "imap.mail.me.com", Port: 993, Encryption: "ssl"},
	// AOL
	"smtp.aol.com": {Host: "imap.aol.com", Port: 993, Encryption: "ssl"},
	// Zoho
	"smtp.zoho.com":    {Host: "imap.zoho.com", Port: 993, Encryption: "ssl"},
	"smtppro.zoho.com": {Host: "imappro.zoho.com", Port: 993, Encryption: "ssl"},
	// ProtonMail Bridge
	"127.0.0.1": {Host: "127.0.0.1", Port: 1143, Encryption: "none"},
	// Yandex
	"smtp.yandex.com": {Host: "imap.yandex.com", Port: 993, Encryption: "ssl"},
	// Mail.ru
	"smtp.mail.ru": {Host: "imap.mail.ru", Port: 993, Encryption: "ssl"},
	// GMX
	"mail.gmx.com": {Host: "imap.gmx.com", Port: 993, Encryption: "ssl"},
	// Fastmail
	"smtp.fastmail.com": {Host: "imap.fastmail.com", Port: 993, Encryption: "ssl"},
}

// autoCompleteIMAPSettings fills in missing IMAP settings based on SMTP host
func autoCompleteIMAPSettings(c *Config) {
	// Skip if IMAP host is already set
	if c.IMAPHost != "" {
		// Still fill in missing credentials
		if c.IMAPUsername == "" {
			c.IMAPUsername = c.SMTPUsername
		}
		if c.IMAPPassword == "" {
			c.IMAPPassword = c.SMTPPassword
		}
		if c.IMAPEncryption == "" {
			c.IMAPEncryption = "ssl"
		}
		if c.IMAPPort == 0 {
			c.IMAPPort = 993
		}
		// Auto-enable IMAP if host is configured but enabled flag wasn't set
		if c.IMAPHost != "" && !c.IMAPEnabled {
			c.IMAPEnabled = true
		}
		return
	}

	// Try to find IMAP settings from SMTP host
	smtpHost := strings.ToLower(c.SMTPHost)
	if preset, ok := smtpToIMAPMapping[smtpHost]; ok {
		c.IMAPHost = preset.Host
		c.IMAPPort = preset.Port
		c.IMAPEncryption = preset.Encryption
		c.IMAPEnabled = true

		// Use SMTP credentials for IMAP if not set
		if c.IMAPUsername == "" {
			c.IMAPUsername = c.SMTPUsername
		}
		if c.IMAPPassword == "" {
			c.IMAPPassword = c.SMTPPassword
		}
	}
}

// parseGmailConfig parses Gmail integration config
func parseGmailConfig(config map[string]interface{}) (*Config, error) {
	c := &Config{
		Provider:       ProviderGmail,
		AuthType:       AuthTypeOAuth2,
		IMAPEnabled:    true,
		IMAPHost:       GmailPresets.IMAPHost,
		IMAPPort:       GmailPresets.IMAPPort,
		IMAPEncryption: "ssl",
		SMTPHost:       GmailPresets.SMTPHost,
		SMTPPort:       GmailPresets.SMTPPort,
		SMTPEncryption: "tls",
		ClientID:       getString(config, "client_id"),
		ClientSecret:   getString(config, "client_secret"),
		RefreshToken:   getString(config, "refresh_token"),
		Email:          getString(config, "email"),
		FromEmail:      getString(config, "email"),
		FromName:       getString(config, "from_name"),
		Template:       getString(config, "template"),
	}
	return c, nil
}

// parseOutlookConfig parses Outlook integration config
func parseOutlookConfig(config map[string]interface{}) (*Config, error) {
	c := &Config{
		Provider:       ProviderOutlook,
		AuthType:       AuthTypeOAuth2,
		IMAPEnabled:    true,
		IMAPHost:       OutlookPresets.IMAPHost,
		IMAPPort:       OutlookPresets.IMAPPort,
		IMAPEncryption: "ssl",
		SMTPHost:       OutlookPresets.SMTPHost,
		SMTPPort:       OutlookPresets.SMTPPort,
		SMTPEncryption: "tls",
		ClientID:       getString(config, "client_id"),
		ClientSecret:   getString(config, "client_secret"),
		TenantID:       getString(config, "tenant_id"),
		RefreshToken:   getString(config, "refresh_token"),
		Email:          getString(config, "email"),
		FromEmail:      getString(config, "email"),
		FromName:       getString(config, "from_name"),
		Template:       getString(config, "template"),
	}
	return c, nil
}

// Helper functions for parsing config
func getString(config map[string]interface{}, key string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(config map[string]interface{}, key string) int {
	if v, ok := config[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return 0
}

func getBool(config map[string]interface{}, key string) bool {
	if v, ok := config[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// ParseIntegrationConfig parses integration config JSON to email.Config
// This is used by both the sender and the fetch job to get email configuration
func ParseIntegrationConfig(integrationType, configJSON string) (*Config, error) {
	var baseConfig map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &baseConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	switch integrationType {
	case models.IntegrationTypeSMTP:
		return parseSMTPConfig(baseConfig)
	case models.IntegrationTypeGmail:
		return parseGmailConfig(baseConfig)
	case models.IntegrationTypeOutlook:
		return parseOutlookConfig(baseConfig)
	default:
		return nil, fmt.Errorf("unsupported integration type: %s", integrationType)
	}
}

// RegisterSendEmailReply registers the SendEmailReply function with the models package
// This should be called during application initialization
func RegisterSendEmailReply() {
	models.SendEmailReply = SendEmailReply
	log.Info("[email] Registered SendEmailReply handler")
}

// ConversationEmailInfo holds email-specific metadata for a conversation
type ConversationEmailInfo struct {
	LastMessageID string    `json:"last_message_id"`
	References    []string  `json:"references"`
	ThreadID      string    `json:"thread_id"`
}

// GetConversationEmailInfo retrieves email threading info for a conversation
func GetConversationEmailInfo(conversationID uint) (*ConversationEmailInfo, error) {
	var emailMsg models.EmailMessage
	err := db.Where("conversation_id = ?", conversationID).
		Order("received_at DESC").
		First(&emailMsg).Error

	if err != nil {
		return nil, err
	}

	refs := []string{}
	if emailMsg.References != "" {
		refs = strings.Split(emailMsg.References, " ")
	}

	return &ConversationEmailInfo{
		LastMessageID: emailMsg.MessageID,
		References:    refs,
	}, nil
}

// UpdateIntegrationStatus updates the integration status and last error
func UpdateIntegrationStatus(integrationType, status, lastError string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if lastError != "" {
		updates["last_error"] = lastError
	} else {
		updates["last_error"] = nil
	}

	return db.Model(&models.Integration{}).
		Where("type = ?", integrationType).
		Updates(updates).Error
}

// IntegrationConfigWithDefaults returns the config with default template if none set
func IntegrationConfigWithDefaults(configJSON string) (*Config, error) {
	var baseConfig map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &baseConfig); err != nil {
		return nil, err
	}

	// If no template is set, use default
	if _, ok := baseConfig["template"]; !ok || baseConfig["template"] == "" {
		baseConfig["template"] = DefaultTemplate
	}

	// Re-marshal with default
	updatedJSON, err := json.Marshal(baseConfig)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(updatedJSON, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
