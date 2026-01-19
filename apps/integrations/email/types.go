// Package email provides unified email functionality for IMAP/SMTP integrations.
package email

import (
	"time"
)

// AuthType represents the authentication method for email connections.
type AuthType string

const (
	AuthTypeBasic   AuthType = "basic"   // Username/password authentication
	AuthTypeOAuth2  AuthType = "oauth2"  // OAuth2 (Gmail, Outlook)
)

// ProviderType represents the email provider.
type ProviderType string

const (
	ProviderSMTP    ProviderType = "smtp"
	ProviderGmail   ProviderType = "gmail"
	ProviderOutlook ProviderType = "outlook"
)

// Config holds the complete email configuration.
type Config struct {
	Provider ProviderType `json:"provider"`
	AuthType AuthType     `json:"auth_type"`

	// IMAP settings (for receiving)
	IMAPEnabled    bool   `json:"imap_enabled"`
	IMAPHost       string `json:"imap_host"`
	IMAPPort       int    `json:"imap_port"`
	IMAPUsername   string `json:"imap_username"`
	IMAPPassword   string `json:"imap_password"`
	IMAPEncryption string `json:"imap_encryption"` // none, ssl, tls

	// SMTP settings (for sending)
	SMTPHost       string `json:"smtp_host"`
	SMTPPort       int    `json:"smtp_port"`
	SMTPUsername   string `json:"smtp_username"`
	SMTPPassword   string `json:"smtp_password"`
	SMTPEncryption string `json:"smtp_encryption"` // none, ssl, tls

	// Common settings
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	Email     string `json:"email"` // The email address being monitored

	// OAuth2 settings (for Gmail/Outlook)
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	TenantID     string `json:"tenant_id"` // Outlook only

	// HTML template for outgoing emails
	Template string `json:"template"`

	// Inbox assignment
	InboxID *uint `json:"inbox_id,omitempty"`
}

// Email represents a received or sent email.
type Email struct {
	MessageID   string    `json:"message_id"`   // Unique message ID from headers
	InReplyTo   string    `json:"in_reply_to"`  // Message-ID this is replying to
	References  []string  `json:"references"`   // Thread references
	From        string    `json:"from"`         // Sender email
	FromName    string    `json:"from_name"`    // Sender display name
	To          []string  `json:"to"`           // Recipients
	CC          []string  `json:"cc"`           // CC recipients
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`         // Plain text body
	HTMLBody    string    `json:"html_body"`    // HTML body
	Date        time.Time `json:"date"`
	Attachments []Attachment `json:"attachments,omitempty"`
	UID         uint32    `json:"uid"`          // IMAP UID
	SeqNum      uint32    `json:"seq_num"`      // IMAP sequence number
}

// Attachment represents an email attachment.
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Data        []byte `json:"-"` // Not serialized to JSON
}

// TemplateData holds data for rendering email templates.
type TemplateData struct {
	Message                string `json:"message"`
	DisplayName            string `json:"display_name"`
	Avatar                 string `json:"avatar"`
	Date                   string `json:"date"`
	ConversationID         int    `json:"conversation_id"`
	ConversationNumber     string `json:"conversation_number"`
	ConversationStatus     string `json:"conversation_status"`
	ConversationDepartment string `json:"conversation_department"`
	ConversationPriority   string `json:"conversation_priority"`
}

// GmailPresets contains default IMAP/SMTP settings for Gmail.
var GmailPresets = struct {
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int
}{
	IMAPHost: "imap.gmail.com",
	IMAPPort: 993,
	SMTPHost: "smtp.gmail.com",
	SMTPPort: 587,
}

// OutlookPresets contains default IMAP/SMTP settings for Outlook.
var OutlookPresets = struct {
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int
}{
	IMAPHost: "outlook.office365.com",
	IMAPPort: 993,
	SMTPHost: "smtp.office365.com",
	SMTPPort: 587,
}

// DefaultTemplate is the default HTML template for email responses.
const DefaultTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
    .container { max-width: 600px; margin: 0 auto; background: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
    .header { background: #10B981; color: white; padding: 20px; }
    .header h2 { margin: 0; font-size: 18px; }
    .content { padding: 20px; }
    .message { background: #f9fafb; border-left: 4px solid #10B981; padding: 15px; margin: 15px 0; border-radius: 0 8px 8px 0; }
    .agent { display: flex; align-items: center; margin-bottom: 15px; }
    .avatar { width: 40px; height: 40px; border-radius: 50%; margin-right: 12px; background: #10B981; display: flex; align-items: center; justify-content: center; color: white; font-weight: bold; }
    .avatar img { width: 100%; height: 100%; border-radius: 50%; object-fit: cover; }
    .agent-info { flex: 1; }
    .agent-name { font-weight: 600; color: #111827; }
    .date { font-size: 12px; color: #6b7280; }
    .footer { padding: 15px 20px; background: #f9fafb; border-top: 1px solid #e5e7eb; font-size: 12px; color: #6b7280; }
    .meta { font-size: 11px; color: #9ca3af; margin-top: 10px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h2>Support Response</h2>
    </div>
    <div class="content">
      <div class="agent">
        <div class="avatar">
          {{if .Avatar}}<img src="{{.Avatar}}" alt="{{.DisplayName}}">{{else}}{{slice .DisplayName 0 1}}{{end}}
        </div>
        <div class="agent-info">
          <div class="agent-name">{{.DisplayName}}</div>
          <div class="date">{{.Date}}</div>
        </div>
      </div>
      <div class="message">
        {{.Message}}
      </div>
      <div class="meta">
        Ticket: {{.ConversationNumber}} | Status: {{.ConversationStatus}} | Priority: {{.ConversationPriority}}
      </div>
    </div>
    <div class="footer">
      This is an automated response. Please reply to this email to continue the conversation.
    </div>
  </div>
</body>
</html>`
