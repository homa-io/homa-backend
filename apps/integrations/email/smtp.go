package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// SMTPClient handles SMTP connections for sending emails.
type SMTPClient struct {
	config Config
}

// NewSMTPClient creates a new SMTP client with the given configuration.
func NewSMTPClient(config Config) *SMTPClient {
	return &SMTPClient{config: config}
}

// Send sends an email using SMTP and returns the generated Message-ID.
func (c *SMTPClient) Send(email Email) (string, error) {
	return c.SendEmail(email)
}

// SendEmail sends an email using SMTP and returns the generated Message-ID.
func (c *SMTPClient) SendEmail(email Email) (string, error) {
	// Generate message ID if not provided
	messageID := email.MessageID
	if messageID == "" {
		messageID = fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), generateRandomID(8), c.getDomain())
	}
	email.MessageID = messageID

	// Build the email message
	msg, err := c.buildMessage(email)
	if err != nil {
		return "", fmt.Errorf("failed to build message: %w", err)
	}

	// Connect and send
	addr := fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort)

	var conn net.Conn
	var client *smtp.Client

	if c.config.SMTPEncryption == "ssl" || c.config.SMTPPort == 465 {
		// SSL/TLS connection
		tlsConfig := &tls.Config{ServerName: c.config.SMTPHost}
		conn, err = tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return "", fmt.Errorf("failed to connect (SSL): %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, c.config.SMTPHost)
		if err != nil {
			return "", fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// Plain or STARTTLS connection
		client, err = smtp.Dial(addr)
		if err != nil {
			return "", fmt.Errorf("failed to connect: %w", err)
		}

		if c.config.SMTPEncryption == "tls" {
			tlsConfig := &tls.Config{ServerName: c.config.SMTPHost}
			if err = client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return "", fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}
	defer client.Close()

	// Authenticate
	if err := c.authenticate(client); err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	// Set sender
	fromEmail := c.config.FromEmail
	if fromEmail == "" {
		fromEmail = c.config.Email
	}
	if err := client.Mail(fromEmail); err != nil {
		return "", fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	recipients := append(email.To, email.CC...)
	for _, to := range recipients {
		if err := client.Rcpt(to); err != nil {
			return "", fmt.Errorf("failed to add recipient %s: %w", to, err)
		}
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("failed to start data: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return "", fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close data: %w", err)
	}

	if err := client.Quit(); err != nil {
		return "", fmt.Errorf("failed to quit: %w", err)
	}

	return messageID, nil
}

// authenticate handles SMTP authentication.
func (c *SMTPClient) authenticate(client *smtp.Client) error {
	// OAuth2 authentication for Gmail/Outlook
	if c.config.AuthType == AuthTypeOAuth2 && c.config.RefreshToken != "" {
		accessToken, err := c.getOAuth2AccessToken()
		if err != nil {
			return err
		}

		// XOAUTH2 authentication
		auth := NewXOAuth2Auth(c.config.Email, accessToken)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("OAuth2 auth failed: %w", err)
		}
		return nil
	}

	// Basic authentication
	if c.config.SMTPUsername != "" && c.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", c.config.SMTPUsername, c.config.SMTPPassword, c.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("basic auth failed: %w", err)
		}
	}

	return nil
}

// getOAuth2AccessToken retrieves a fresh access token using the refresh token.
func (c *SMTPClient) getOAuth2AccessToken() (string, error) {
	switch c.config.Provider {
	case ProviderGmail:
		return RefreshGmailAccessToken(c.config.ClientID, c.config.ClientSecret, c.config.RefreshToken)
	case ProviderOutlook:
		return RefreshOutlookAccessToken(c.config.ClientID, c.config.ClientSecret, c.config.TenantID, c.config.RefreshToken)
	default:
		return "", fmt.Errorf("OAuth2 not supported for provider: %s", c.config.Provider)
	}
}

// buildMessage constructs the email message with proper MIME encoding.
func (c *SMTPClient) buildMessage(email Email) ([]byte, error) {
	var buf bytes.Buffer

	// Generate message ID if not provided
	messageID := email.MessageID
	if messageID == "" {
		messageID = fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), generateRandomID(8), c.getDomain())
	}

	// From header
	fromName := c.config.FromName
	fromEmail := c.config.FromEmail
	if fromEmail == "" {
		fromEmail = c.config.Email
	}
	if fromName != "" {
		fmt.Fprintf(&buf, "From: %s <%s>\r\n", mime.QEncoding.Encode("utf-8", fromName), fromEmail)
	} else {
		fmt.Fprintf(&buf, "From: %s\r\n", fromEmail)
	}

	// To header
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(email.To, ", "))

	// CC header
	if len(email.CC) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(email.CC, ", "))
	}

	// Subject
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", email.Subject))

	// Date
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))

	// Message-ID
	fmt.Fprintf(&buf, "Message-ID: %s\r\n", messageID)

	// In-Reply-To and References for threading
	if email.InReplyTo != "" {
		fmt.Fprintf(&buf, "In-Reply-To: %s\r\n", email.InReplyTo)
	}
	if len(email.References) > 0 {
		fmt.Fprintf(&buf, "References: %s\r\n", strings.Join(email.References, " "))
	}

	// MIME headers
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")

	// Handle attachments
	if len(email.Attachments) > 0 {
		return c.buildMultipartMessage(&buf, email)
	}

	// Simple message (HTML or plain text)
	if email.HTMLBody != "" {
		fmt.Fprintf(&buf, "Content-Type: text/html; charset=utf-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: quoted-printable\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s", encodeQuotedPrintable(email.HTMLBody))
	} else {
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: quoted-printable\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s", encodeQuotedPrintable(email.Body))
	}

	return buf.Bytes(), nil
}

// buildMultipartMessage creates a multipart message with attachments.
func (c *SMTPClient) buildMultipartMessage(buf *bytes.Buffer, email Email) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary)
	fmt.Fprintf(buf, "\r\n")

	// Text/HTML part
	if email.HTMLBody != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/html; charset=utf-8")
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		part, _ := writer.CreatePart(h)
		part.Write([]byte(encodeQuotedPrintable(email.HTMLBody)))
	} else if email.Body != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/plain; charset=utf-8")
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		part, _ := writer.CreatePart(h)
		part.Write([]byte(encodeQuotedPrintable(email.Body)))
	}

	// Attachments
	for _, att := range email.Attachments {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", att.ContentType)
		h.Set("Content-Transfer-Encoding", "base64")
		h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", att.Filename))
		part, _ := writer.CreatePart(h)

		encoded := base64.StdEncoding.EncodeToString(att.Data)
		// Wrap at 76 characters
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			part.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	writer.Close()
	return buf.Bytes(), nil
}

// getDomain extracts the domain from the email address.
func (c *SMTPClient) getDomain() string {
	email := c.config.FromEmail
	if email == "" {
		email = c.config.Email
	}
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return "localhost"
}

// TestConnection tests the SMTP connection.
func (c *SMTPClient) TestConnection() error {
	addr := fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort)

	var conn net.Conn
	var client *smtp.Client
	var err error

	if c.config.SMTPEncryption == "ssl" || c.config.SMTPPort == 465 {
		tlsConfig := &tls.Config{ServerName: c.config.SMTPHost}
		conn, err = tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect (SSL): %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, c.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		if c.config.SMTPEncryption == "tls" {
			tlsConfig := &tls.Config{ServerName: c.config.SMTPHost}
			if err = client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}
	defer client.Close()

	// Authenticate
	if err := c.authenticate(client); err != nil {
		return err
	}

	return client.Quit()
}

// encodeQuotedPrintable encodes a string in quoted-printable format.
func encodeQuotedPrintable(s string) string {
	var buf strings.Builder
	lineLen := 0

	for _, r := range s {
		if r == '\r' || r == '\n' {
			buf.WriteRune(r)
			lineLen = 0
			continue
		}

		var encoded string
		if r >= 33 && r <= 126 && r != '=' {
			encoded = string(r)
		} else if r == ' ' || r == '\t' {
			encoded = string(r)
		} else {
			encoded = fmt.Sprintf("=%02X", r)
		}

		// Soft line break at 76 characters
		if lineLen+len(encoded) > 75 {
			buf.WriteString("=\r\n")
			lineLen = 0
		}

		buf.WriteString(encoded)
		lineLen += len(encoded)
	}

	return buf.String()
}

// generateRandomID generates a random ID string.
func generateRandomID(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(1) // Ensure different values
	}
	return string(result)
}
