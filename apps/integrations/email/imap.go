package email

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
)

// IMAPClient handles IMAP connections for fetching emails.
type IMAPClient struct {
	config Config
	client *client.Client
}

// NewIMAPClient creates a new IMAP client with the given configuration.
func NewIMAPClient(config Config) *IMAPClient {
	return &IMAPClient{config: config}
}

// Connect establishes a connection to the IMAP server.
func (c *IMAPClient) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.config.IMAPHost, c.config.IMAPPort)

	var conn *client.Client
	var err error

	// Connect based on encryption setting
	if c.config.IMAPEncryption == "ssl" || c.config.IMAPPort == 993 {
		// SSL/TLS connection
		tlsConfig := &tls.Config{ServerName: c.config.IMAPHost}
		conn, err = client.DialTLS(addr, tlsConfig)
	} else {
		// Plain connection (optionally upgrade to STARTTLS)
		conn, err = client.Dial(addr)
		if err == nil && c.config.IMAPEncryption == "tls" {
			tlsConfig := &tls.Config{ServerName: c.config.IMAPHost}
			if err = conn.StartTLS(tlsConfig); err != nil {
				conn.Close()
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	c.client = conn
	return nil
}

// Login authenticates with the IMAP server.
func (c *IMAPClient) Login() error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}

	// Use OAuth2 for Gmail/Outlook if configured
	if c.config.AuthType == AuthTypeOAuth2 && c.config.RefreshToken != "" {
		// Get fresh access token
		accessToken, err := c.getOAuth2AccessToken()
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 access token: %w", err)
		}

		// XOAUTH2 authentication
		saslClient := NewXOAuth2Client(c.config.Email, accessToken)
		if err := c.client.Authenticate(saslClient); err != nil {
			return fmt.Errorf("OAuth2 authentication failed: %w", err)
		}
		return nil
	}

	// Basic authentication
	if err := c.client.Login(c.config.IMAPUsername, c.config.IMAPPassword); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	return nil
}

// getOAuth2AccessToken retrieves a fresh access token using the refresh token.
func (c *IMAPClient) getOAuth2AccessToken() (string, error) {
	switch c.config.Provider {
	case ProviderGmail:
		return RefreshGmailAccessToken(c.config.ClientID, c.config.ClientSecret, c.config.RefreshToken)
	case ProviderOutlook:
		return RefreshOutlookAccessToken(c.config.ClientID, c.config.ClientSecret, c.config.TenantID, c.config.RefreshToken)
	default:
		return "", fmt.Errorf("OAuth2 not supported for provider: %s", c.config.Provider)
	}
}

// Close closes the IMAP connection.
func (c *IMAPClient) Close() error {
	if c.client == nil {
		return nil
	}
	if err := c.client.Logout(); err != nil {
		c.client.Close()
		return err
	}
	return nil
}

// FetchNewEmails fetches emails since the given time from INBOX.
func (c *IMAPClient) FetchNewEmails(since time.Time) ([]Email, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Select INBOX
	mbox, err := c.client.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	if mbox.Messages == 0 {
		return nil, nil
	}

	// Search for messages since the given date
	criteria := imap.NewSearchCriteria()
	criteria.Since = since

	uids, err := c.client.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(uids) == 0 {
		return nil, nil
	}

	// Fetch messages
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	// Request full message with headers and body
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchUid}

	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	go func() {
		done <- c.client.Fetch(seqSet, items, messages)
	}()

	var emails []Email
	for msg := range messages {
		email, err := c.parseMessage(msg, section)
		if err != nil {
			continue // Skip messages that fail to parse
		}
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return emails, fmt.Errorf("fetch failed: %w", err)
	}

	return emails, nil
}

// FetchEmailByUID fetches a specific email by UID.
func (c *IMAPClient) FetchEmailByUID(uid uint32) (*Email, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Select INBOX
	_, err := c.client.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchUid}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.client.UidFetch(seqSet, items, messages)
	}()

	var email *Email
	for msg := range messages {
		e, err := c.parseMessage(msg, section)
		if err == nil {
			email = &e
		}
	}

	if err := <-done; err != nil {
		return email, fmt.Errorf("fetch failed: %w", err)
	}

	return email, nil
}

// parseMessage parses an IMAP message into our Email struct.
func (c *IMAPClient) parseMessage(msg *imap.Message, section *imap.BodySectionName) (Email, error) {
	email := Email{
		UID:    msg.Uid,
		SeqNum: msg.SeqNum,
	}

	// Parse envelope
	if msg.Envelope != nil {
		email.Subject = msg.Envelope.Subject
		email.Date = msg.Envelope.Date
		email.MessageID = msg.Envelope.MessageId
		email.InReplyTo = msg.Envelope.InReplyTo

		if len(msg.Envelope.From) > 0 {
			from := msg.Envelope.From[0]
			email.From = from.Address()
			email.FromName = from.PersonalName
		}

		for _, addr := range msg.Envelope.To {
			email.To = append(email.To, addr.Address())
		}
		for _, addr := range msg.Envelope.Cc {
			email.CC = append(email.CC, addr.Address())
		}
	}

	// Parse body
	body := msg.GetBody(section)
	if body == nil {
		return email, nil
	}

	// Parse the message
	entity, err := message.Read(body)
	if err != nil {
		// Try parsing as simple mail
		bodyBytes, _ := io.ReadAll(body)
		email.Body = string(bodyBytes)
		return email, nil
	}

	// Extract References header
	if refs := entity.Header.Get("References"); refs != "" {
		email.References = strings.Fields(refs)
	}

	// Parse multipart or simple body
	if mr := entity.MultipartReader(); mr != nil {
		// Multipart message
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}

			contentType, _, _ := part.Header.ContentType()
			switch contentType {
			case "text/plain":
				bodyBytes, _ := io.ReadAll(part.Body)
				email.Body = string(bodyBytes)
			case "text/html":
				bodyBytes, _ := io.ReadAll(part.Body)
				email.HTMLBody = string(bodyBytes)
			default:
				// Handle attachment - get filename from Content-Disposition or Content-Type params
				disposition, dispParams, _ := part.Header.ContentDisposition()
				filename := ""
				if disposition == "attachment" || disposition == "inline" {
					filename = dispParams["filename"]
				}
				if filename == "" {
					// Try to get from Content-Type params
					_, ctParams, _ := part.Header.ContentType()
					filename = ctParams["name"]
				}
				if filename != "" {
					data, _ := io.ReadAll(part.Body)
					email.Attachments = append(email.Attachments, Attachment{
						Filename:    filename,
						ContentType: contentType,
						Size:        int64(len(data)),
						Data:        data,
					})
				}
			}
		}
	} else {
		// Simple message
		contentType, _, _ := entity.Header.ContentType()
		bodyBytes, _ := io.ReadAll(entity.Body)
		if contentType == "text/html" {
			email.HTMLBody = string(bodyBytes)
		} else {
			email.Body = string(bodyBytes)
		}
	}

	return email, nil
}

// MarkAsRead marks an email as read (seen).
func (c *IMAPClient) MarkAsRead(uid uint32) error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.SeenFlag}

	return c.client.UidStore(seqSet, item, flags, nil)
}

// TestConnection tests the IMAP connection.
func (c *IMAPClient) TestConnection() error {
	if err := c.Connect(); err != nil {
		return err
	}
	defer c.Close()

	if err := c.Login(); err != nil {
		return err
	}

	// Try to list mailboxes as a simple test
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.client.List("", "*", mailboxes)
	}()

	// Drain mailboxes channel
	for range mailboxes {
	}

	return <-done
}

// ParseEmailAddress extracts email address from a string like "Name <email@example.com>".
func ParseEmailAddress(s string) (name, email string) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		// If parsing fails, assume it's just an email
		return "", s
	}
	return addr.Name, addr.Address
}
