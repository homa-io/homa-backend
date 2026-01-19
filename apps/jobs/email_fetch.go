package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/integrations/email"
	"github.com/iesreza/homa-backend/apps/models"
	"gorm.io/datatypes"
)

// Job name constant
const (
	JobFetchEmailMessages = "fetch_email_messages"
)

// EmailFetchResult is the result of the email fetch job
type EmailFetchResult struct {
	IntegrationsProcessed int                    `json:"integrations_processed"`
	EmailsFetched         int                    `json:"emails_fetched"`
	ConversationsCreated  int                    `json:"conversations_created"`
	MessagesAdded         int                    `json:"messages_added"`
	Errors                []string               `json:"errors,omitempty"`
	Details               map[string]interface{} `json:"details,omitempty"`
}

// RegisterEmailFetchJob registers the email fetch background job
func RegisterEmailFetchJob() {
	registry := GetRegistry()

	registry.Register(JobDefinition{
		Name:           JobFetchEmailMessages,
		Description:    "Fetch new emails from IMAP servers and create conversations",
		TimeoutSeconds: 300, // 5 minutes
		Handler:        handleFetchEmailMessages,
	})

	log.Info("[jobs] Registered email fetch job")
}

// handleFetchEmailMessages fetches emails from all enabled email integrations
func handleFetchEmailMessages(ctx context.Context) (interface{}, error) {
	log.Info("[%s] Starting email fetch job", JobFetchEmailMessages)

	result := EmailFetchResult{
		IntegrationsProcessed: 0,
		EmailsFetched:         0,
		ConversationsCreated:  0,
		MessagesAdded:         0,
		Errors:                []string{},
		Details:               make(map[string]interface{}),
	}

	// Get all enabled email integrations
	integrations, err := models.GetEnabledEmailIntegrations()
	if err != nil {
		log.Error("[%s] Failed to get email integrations: %v", JobFetchEmailMessages, err)
		return result, err
	}

	if len(integrations) == 0 {
		log.Info("[%s] No enabled email integrations found", JobFetchEmailMessages)
		return result, nil
	}

	log.Info("[%s] Found %d enabled email integrations", JobFetchEmailMessages, len(integrations))

	// Process each integration
	for _, integration := range integrations {
		select {
		case <-ctx.Done():
			log.Warning("[%s] Job cancelled", JobFetchEmailMessages)
			return result, ctx.Err()
		default:
		}

		intResult, err := processEmailIntegration(ctx, integration)
		if err != nil {
			errMsg := integration.Type + ": " + err.Error()
			result.Errors = append(result.Errors, errMsg)
			log.Error("[%s] Error processing %s: %v", JobFetchEmailMessages, integration.Type, err)

			// Update integration status to error
			db.Model(&integration).Updates(map[string]interface{}{
				"status":     models.IntegrationStatusError,
				"last_error": err.Error(),
			})
			continue
		}

		result.IntegrationsProcessed++
		result.EmailsFetched += intResult.EmailsFetched
		result.ConversationsCreated += intResult.ConversationsCreated
		result.MessagesAdded += intResult.MessagesAdded
		result.Details[integration.Type] = intResult
	}

	log.Info("[%s] Email fetch job completed: %d integrations, %d emails fetched, %d conversations, %d messages",
		JobFetchEmailMessages, result.IntegrationsProcessed, result.EmailsFetched,
		result.ConversationsCreated, result.MessagesAdded)

	return result, nil
}

// integrationResult holds the result for a single integration
type integrationResult struct {
	EmailsFetched        int `json:"emails_fetched"`
	ConversationsCreated int `json:"conversations_created"`
	MessagesAdded        int `json:"messages_added"`
}

// processEmailIntegration processes a single email integration
func processEmailIntegration(ctx context.Context, integration models.Integration) (*integrationResult, error) {
	result := &integrationResult{}

	// Get email config based on integration type
	emailConfig, err := getEmailConfigForIntegration(integration)
	if err != nil {
		return result, err
	}

	// Skip if IMAP is not enabled (for SMTP-only integrations)
	if !emailConfig.IMAPEnabled {
		log.Info("[%s] IMAP not enabled for %s, skipping", JobFetchEmailMessages, integration.Type)
		return result, nil
	}

	// Get last processed time for this integration
	lastProcessed, err := models.GetLastProcessedTime(integration.Type)
	if err != nil {
		log.Warning("[%s] Failed to get last processed time for %s: %v", JobFetchEmailMessages, integration.Type, err)
		lastProcessed = time.Now().Add(-24 * time.Hour)
	}

	// Create IMAP client and connect
	imapClient := email.NewIMAPClient(*emailConfig)

	// Connect to IMAP server
	if err := imapClient.Connect(); err != nil {
		return result, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer imapClient.Close()

	// Login to IMAP server
	if err := imapClient.Login(); err != nil {
		return result, fmt.Errorf("IMAP login failed: %w", err)
	}

	// Fetch new emails
	emails, err := imapClient.FetchNewEmails(lastProcessed)
	if err != nil {
		return result, err
	}

	result.EmailsFetched = len(emails)
	log.Info("[%s] Fetched %d emails from %s", JobFetchEmailMessages, len(emails), integration.Type)

	// Process each email
	for _, fetchedEmail := range emails {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Check if email already processed
		exists, err := models.EmailMessageExists(fetchedEmail.MessageID)
		if err != nil {
			log.Error("[%s] Failed to check email existence: %v", JobFetchEmailMessages, err)
			continue
		}
		if exists {
			log.Debug("[%s] Email %s already processed, skipping", JobFetchEmailMessages, fetchedEmail.MessageID)
			continue
		}

		// Process the email (create conversation or add to existing)
		isNew, err := processIncomingEmail(ctx, integration, fetchedEmail, emailConfig)
		if err != nil {
			log.Error("[%s] Failed to process email %s: %v", JobFetchEmailMessages, fetchedEmail.MessageID, err)
			continue
		}

		if isNew {
			result.ConversationsCreated++
		} else {
			result.MessagesAdded++
		}
	}

	return result, nil
}

// getEmailConfigForIntegration returns the email config for the given integration
func getEmailConfigForIntegration(integration models.Integration) (*email.Config, error) {
	return email.ParseIntegrationConfig(integration.Type, integration.Config)
}

// processIncomingEmail processes an incoming email and creates/updates conversation
// Returns true if a new conversation was created, false if message was added to existing
func processIncomingEmail(ctx context.Context, integration models.Integration, incomingEmail email.Email, emailConfig *email.Config) (bool, error) {
	// Try to find existing conversation by email threading
	var conversation *models.Conversation
	var err error
	isNewConversation := false

	// 1. First try In-Reply-To header
	if incomingEmail.InReplyTo != "" {
		conversation, err = models.GetConversationByInReplyTo(incomingEmail.InReplyTo)
		if err == nil && conversation != nil {
			log.Debug("[%s] Found conversation %d by In-Reply-To", JobFetchEmailMessages, conversation.ID)
		}
	}

	// 2. Try References header
	if conversation == nil && len(incomingEmail.References) > 0 {
		conversation, err = models.GetConversationByEmailThread(incomingEmail.References)
		if err == nil && conversation != nil {
			log.Debug("[%s] Found conversation %d by References", JobFetchEmailMessages, conversation.ID)
		}
	}

	// 3. Try matching by email + cleaned subject
	if conversation == nil {
		cleanedSubject := email.CleanSubject(incomingEmail.Subject)
		conversation, err = models.GetConversationByEmailAndSubject(incomingEmail.From, cleanedSubject, "email")
		if err == nil && conversation != nil {
			log.Debug("[%s] Found conversation %d by email+subject", JobFetchEmailMessages, conversation.ID)
		}
	}

	// 4. Create new conversation if not found
	if conversation == nil {
		conversation, err = createConversationFromEmail(integration, incomingEmail, emailConfig)
		if err != nil {
			return false, err
		}
		isNewConversation = true
		log.Info("[%s] Created new conversation %d for email from %s", JobFetchEmailMessages, conversation.ID, incomingEmail.From)
	}

	// Add message to conversation
	messageBody := email.ExtractPlainText(incomingEmail)
	if email.IsReplyEmail(incomingEmail.Subject) {
		// Try to extract just the reply portion
		replyText := email.ExtractReplyText(messageBody)
		if replyText != "" {
			messageBody = replyText
		}
	}

	// Create the message (use ClientID to indicate customer message)
	message := &models.Message{
		ConversationID: conversation.ID,
		ClientID:       &conversation.ClientID,
		Body:           messageBody,
	}

	if err := db.Create(message).Error; err != nil {
		return isNewConversation, err
	}

	// Create email tracking record
	emailRecord := &models.EmailMessage{
		IntegrationType: integration.Type,
		MessageID:       incomingEmail.MessageID,
		ConversationID:  conversation.ID,
		MessageRecordID: message.ID,
		Subject:         incomingEmail.Subject,
		FromEmail:       incomingEmail.From,
		FromName:        incomingEmail.FromName,
		ToEmail:         emailConfig.Email,
		InReplyTo:       incomingEmail.InReplyTo,
		References:      strings.Join(incomingEmail.References, " "),
		Direction:       "inbound",
		ReceivedAt:      incomingEmail.Date,
	}

	if err := models.CreateEmailMessage(emailRecord); err != nil {
		log.Error("[%s] Failed to create email tracking record: %v", JobFetchEmailMessages, err)
		// Don't fail the whole operation, just log
	}

	// Update conversation status and updated_at
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if conversation.Status == models.ConversationStatusClosed || conversation.Status == models.ConversationStatusArchived {
		// Reopen if was closed/archived
		updates["status"] = models.ConversationStatusNew
	} else if conversation.Status == models.ConversationStatusWaitForUser {
		// Customer responded, waiting for agent now
		updates["status"] = models.ConversationStatusWaitForAgent
	}
	db.Model(conversation).Updates(updates)

	return isNewConversation, nil
}

// createConversationFromEmail creates a new conversation from an incoming email
func createConversationFromEmail(integration models.Integration, incomingEmail email.Email, emailConfig *email.Config) (*models.Conversation, error) {
	// Find or create customer (client in our model)
	client, err := findOrCreateClientFromEmail(incomingEmail)
	if err != nil {
		return nil, err
	}

	// Clean subject for title
	title := email.CleanSubject(incomingEmail.Subject)
	if title == "" {
		title = "Email conversation"
	}

	// Generate a secret for the conversation
	secret := generateEmailConversationSecret()

	// Create conversation
	conversation := &models.Conversation{
		ClientID:     client.ID,
		ChannelID:    "email",
		Title:        title,
		Secret:       secret,
		Status:       models.ConversationStatusNew,
		Priority:     models.ConversationPriorityMedium,
		InboxID:      integration.InboxID,
		CustomFields: datatypes.JSON("{}"),
	}

	if err := db.Create(conversation).Error; err != nil {
		return nil, err
	}

	return conversation, nil
}

// findOrCreateClientFromEmail finds an existing client by email or creates a new one
func findOrCreateClientFromEmail(incomingEmail email.Email) (*models.Client, error) {
	// Try to find by email in external IDs
	var extID models.ClientExternalID
	err := db.Where("type = ? AND value = ?", models.ExternalIDTypeEmail, incomingEmail.From).First(&extID).Error
	if err == nil {
		// Found existing client
		var client models.Client
		if err := db.First(&client, "id = ?", extID.ClientID).Error; err != nil {
			return nil, err
		}
		return &client, nil
	}

	// Create new client
	name := incomingEmail.FromName
	// If no name, use email prefix
	if name == "" {
		parts := strings.Split(incomingEmail.From, "@")
		if len(parts) > 0 {
			name = parts[0]
		}
	}

	client := models.Client{
		ID:   uuid.New(),
		Name: name,
		Data: datatypes.JSON("{}"),
	}

	if err := db.Create(&client).Error; err != nil {
		return nil, err
	}

	// Create email external ID
	emailExtID := models.ClientExternalID{
		ClientID: client.ID,
		Type:     models.ExternalIDTypeEmail,
		Value:    incomingEmail.From,
	}

	if err := db.Create(&emailExtID).Error; err != nil {
		log.Warning("[%s] Failed to create email external ID: %v", JobFetchEmailMessages, err)
		// Don't fail, client was created
	}

	return &client, nil
}

// generateEmailConversationSecret generates a random 32-character hexadecimal secret
func generateEmailConversationSecret() string {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
