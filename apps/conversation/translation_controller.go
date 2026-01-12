package conversation

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/ai"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// TranslationController handles translation-related endpoints
type TranslationController struct{}

// GetTranslationsRequest represents the request for getting translations
type GetTranslationsRequest struct {
	MessageIDs []uint `json:"message_ids" validate:"required,min=1"`
}

// GetTranslations fetches or creates translations for messages
// Now uses per-message language detection instead of conversation-level language
// @Summary Get translations for messages
// @Description Get translations for specified messages based on each message's detected language
// @Tags Translation
// @Accept json
// @Produce json
// @Param id path int true "Conversation ID"
// @Param body body GetTranslationsRequest true "Message IDs to translate"
// @Success 200 {object} models.BatchTranslationResponse
// @Router /agent/conversations/{id}/translations [post]
// @Security Bearer
func (c TranslationController) GetTranslations(req *evo.Request) interface{} {
	user := req.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Check if auto_translate_incoming is enabled
	if !user.AutoTranslateIncoming {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeForbidden, "Auto translate incoming is not enabled", 403, "Enable auto_translate_incoming in your profile settings"))
	}

	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var params GetTranslationsRequest
	if err := req.BodyParser(&params); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	if len(params.MessageIDs) == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "No message IDs provided", 400, "message_ids array is required"))
	}

	// Agent language
	agentLang := user.Language
	if agentLang == "" {
		agentLang = "en"
	}

	// Fetch messages with their detected language
	var messages []models.Message
	if err := db.Where("id IN ?", params.MessageIDs).Find(&messages).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to fetch messages", 500, err.Error()))
	}

	// Get the dominant customer language for fallback
	customerLang := getCustomerLanguageFromMessages(conversationID)

	// Build a map for quick lookup and detect language for messages without it
	messageMap := make(map[uint]models.Message)
	for _, msg := range messages {
		// If message doesn't have language detected, detect it now and update the database
		if msg.Language == "" && msg.Body != "" && !msg.IsSystemMessage {
			detectedLang := ai.DetectLanguage(msg.Body)
			if detectedLang != "" {
				msg.Language = detectedLang
				// Update the database
				db.Model(&models.Message{}).Where("id = ?", msg.ID).Update("language", detectedLang)
				log.Info("Detected language '%s' for message %d", detectedLang, msg.ID)
			} else if customerLang != "" && msg.ClientID != nil {
				// For short customer messages where detection fails, use conversation's dominant language
				msg.Language = customerLang
				db.Model(&models.Message{}).Where("id = ?", msg.ID).Update("language", customerLang)
				log.Info("Using fallback language '%s' for short message %d", customerLang, msg.ID)
			}
		}
		messageMap[msg.ID] = msg
	}

	// Filter messages that need translation (message language != agent language)
	var messagesToTranslate []uint
	for _, msgID := range params.MessageIDs {
		msg, exists := messageMap[msgID]
		if !exists {
			continue
		}
		// Skip system messages or messages without detected language
		if msg.IsSystemMessage || msg.Language == "" {
			continue
		}
		// Translate if message language differs from agent language
		if msg.Language != agentLang {
			messagesToTranslate = append(messagesToTranslate, msgID)
		}
	}

	// If no messages need translation, return empty result
	if len(messagesToTranslate) == 0 {
		return response.OK(models.BatchTranslationResponse{
			Translations: []models.TranslationResponse{},
		})
	}

	// Get or create translations for messages that need it
	translations, err := ai.GetOrCreateTranslationsPerMessage(
		conversationID,
		messagesToTranslate,
		messageMap,
		agentLang,
	)
	if err != nil {
		log.Error("Failed to get translations: %v", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to get translations", 500, err.Error()))
	}

	return response.OK(models.BatchTranslationResponse{
		Translations: translations,
	})
}

// GetOutgoingTranslationsRequest represents the request for getting outgoing translations
type GetOutgoingTranslationsRequest struct {
	MessageIDs []uint `json:"message_ids" validate:"required,min=1"`
}

// GetOutgoingTranslations fetches original content for outgoing messages that were translated
// @Summary Get original content for outgoing messages
// @Description Get the original (untranslated) content for agent messages that were auto-translated
// @Tags Translation
// @Accept json
// @Produce json
// @Param id path int true "Conversation ID"
// @Param body body GetOutgoingTranslationsRequest true "Message IDs"
// @Success 200 {object} models.BatchTranslationResponse
// @Router /agent/conversations/{id}/outgoing-translations [post]
// @Security Bearer
func (c TranslationController) GetOutgoingTranslations(req *evo.Request) interface{} {
	user := req.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var params GetOutgoingTranslationsRequest
	if err := req.BodyParser(&params); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	if len(params.MessageIDs) == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "No message IDs provided", 400, "message_ids array is required"))
	}

	// Fetch existing outgoing translations (these store the original content)
	var existingTranslations []models.ConversationMessageTranslation
	db.Where("conversation_id = ? AND message_id IN ? AND type = ?",
		conversationID, params.MessageIDs, models.TranslationTypeOutgoing).
		Find(&existingTranslations)

	// Fetch messages to get the translated content (what customer sees in message.body)
	var messages []models.Message
	db.Where("id IN ?", params.MessageIDs).Find(&messages)
	messageMap := make(map[uint]models.Message)
	for _, msg := range messages {
		messageMap[msg.ID] = msg
	}

	// Build response
	translations := make([]models.TranslationResponse, 0, len(existingTranslations))
	for _, trans := range existingTranslations {
		msg := messageMap[trans.MessageID]
		translations = append(translations, models.TranslationResponse{
			MessageID:         trans.MessageID,
			OriginalContent:   trans.Content, // What agent typed (stored in translation)
			TranslatedContent: msg.Body,      // What customer sees (message body)
			FromLang:          trans.FromLang,
			ToLang:            trans.ToLang,
			Type:              models.TranslationTypeOutgoing,
			IsTranslated:      true,
		})
	}

	return response.OK(models.BatchTranslationResponse{
		Translations: translations,
	})
}

// TranslateOutgoingRequest represents the request for translating outgoing messages
type TranslateOutgoingRequest struct {
	Content string `json:"content" validate:"required"`
}

// TranslateOutgoingResponse represents the response for outgoing translation
type TranslateOutgoingResponse struct {
	OriginalContent   string `json:"original_content"`
	TranslatedContent string `json:"translated_content"`
	FromLang          string `json:"from_lang"`
	ToLang            string `json:"to_lang"`
}

// TranslateOutgoing translates an outgoing message to customer language
// @Summary Translate outgoing message
// @Description Translate a message from agent language to customer language
// @Tags Translation
// @Accept json
// @Produce json
// @Param id path int true "Conversation ID"
// @Param body body TranslateOutgoingRequest true "Content to translate"
// @Success 200 {object} TranslateOutgoingResponse
// @Router /agent/conversations/{id}/translate-outgoing [post]
// @Security Bearer
func (c TranslationController) TranslateOutgoing(req *evo.Request) interface{} {
	user := req.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Check if auto_translate_outgoing is enabled
	if !user.AutoTranslateOutgoing {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeForbidden, "Auto translate outgoing is not enabled", 403, "Enable auto_translate_outgoing in your profile settings"))
	}

	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var params TranslateOutgoingRequest
	if err := req.BodyParser(&params); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	if params.Content == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Content is required", 400, "content field cannot be empty"))
	}

	// Get conversation with client info
	var conversation models.Conversation
	if err := db.Preload("Client").Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, "No conversation exists with the specified ID"))
	}

	// Determine customer language from most recent customer message
	customerLang := getCustomerLanguageFromMessages(conversationID)
	if customerLang == "" {
		// Fallback to client's stored language
		if conversation.Client.Language != nil && *conversation.Client.Language != "" {
			customerLang = *conversation.Client.Language
		} else {
			customerLang = "en"
		}
	}

	// Agent language
	agentLang := user.Language
	if agentLang == "" {
		agentLang = "en"
	}

	// If languages are the same, return original content
	if customerLang == agentLang {
		return response.OK(TranslateOutgoingResponse{
			OriginalContent:   params.Content,
			TranslatedContent: params.Content,
			FromLang:          agentLang,
			ToLang:            customerLang,
		})
	}

	// Translate the message
	translated, err := ai.TranslateText(params.Content, agentLang, customerLang)
	if err != nil {
		log.Error("Failed to translate outgoing message: %v", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to translate message", 500, err.Error()))
	}

	return response.OK(TranslateOutgoingResponse{
		OriginalContent:   params.Content,
		TranslatedContent: translated,
		FromLang:          agentLang,
		ToLang:            customerLang,
	})
}

// getCustomerLanguageFromMessages gets the most common language from customer messages in a conversation
// Also detects language for messages that don't have it set
func getCustomerLanguageFromMessages(conversationID uint) string {
	// First, get messages that don't have language detected and detect them
	var messagesWithoutLang []models.Message
	db.Where("conversation_id = ? AND client_id IS NOT NULL AND (language IS NULL OR language = '') AND is_system_message = false", conversationID).
		Order("created_at DESC").
		Limit(20).
		Find(&messagesWithoutLang)

	// Detect and update language for these messages
	for _, msg := range messagesWithoutLang {
		if msg.Body != "" {
			detectedLang := ai.DetectLanguage(msg.Body)
			if detectedLang != "" {
				db.Model(&models.Message{}).Where("id = ?", msg.ID).Update("language", detectedLang)
				log.Info("Backfilled language '%s' for message %d", detectedLang, msg.ID)
			}
		}
	}

	// Now get messages with language
	var messages []models.Message
	db.Where("conversation_id = ? AND client_id IS NOT NULL AND language IS NOT NULL AND language != ''", conversationID).
		Order("created_at DESC").
		Limit(10).
		Find(&messages)

	if len(messages) == 0 {
		return ""
	}

	// Count language occurrences
	langCount := make(map[string]int)
	for _, msg := range messages {
		langCount[msg.Language]++
	}

	// Find most common language
	var mostCommonLang string
	var maxCount int
	for lang, count := range langCount {
		if count > maxCount {
			maxCount = count
			mostCommonLang = lang
		}
	}

	return mostCommonLang
}

// GetLanguageInfo returns language information for a conversation
// Now includes per-message language detection status
// @Summary Get language info for conversation
// @Description Get agent language info and auto-translation settings for a conversation
// @Tags Translation
// @Produce json
// @Param id path int true "Conversation ID"
// @Success 200 {object} object
// @Router /agent/conversations/{id}/language-info [get]
// @Security Bearer
func (c TranslationController) GetLanguageInfo(req *evo.Request) interface{} {
	user := req.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	// Get conversation
	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, "No conversation exists with the specified ID"))
	}

	// Agent language
	agentLang := user.Language
	if agentLang == "" {
		agentLang = "en"
	}

	// Get the dominant customer language from messages
	customerLang := getCustomerLanguageFromMessages(conversationID)

	// Count messages that would need translation
	var messageCountNeedingTranslation int64
	db.Model(&models.Message{}).
		Where("conversation_id = ? AND language IS NOT NULL AND language != '' AND language != ? AND is_system_message = false", conversationID, agentLang).
		Count(&messageCountNeedingTranslation)

	return response.OK(map[string]interface{}{
		"agent_language":                  agentLang,
		"detected_customer_language":      customerLang,
		"messages_needing_translation":    messageCountNeedingTranslation,
		"needs_translation":               messageCountNeedingTranslation > 0 || (customerLang != "" && customerLang != agentLang),
		"auto_translate_incoming":         user.AutoTranslateIncoming,
		"auto_translate_outgoing":         user.AutoTranslateOutgoing,
		"per_message_language_detection":  true,
	})
}
