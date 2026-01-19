package conversation

import (
	"fmt"
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/ai"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// GetConversationMessages handles the GET /api/agent/conversations/{conversation_id}/messages endpoint
// @Summary Get conversation messages
// @Description Retrieve all messages for a specific conversation in chronological order
// @Tags Agent - Conversations
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Messages per page (max 100)" default(50)
// @Param order query string false "Sort order (asc or desc)" default(asc)
// @Success 200 {object} ConversationMessagesResponse
// @Router /api/agent/conversations/{conversation_id}/messages [get]
func (ac AgentController) GetConversationMessages(req *evo.Request) interface{} {
	conversationID := req.Param("conversation_id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}

	limit := req.Query("limit").Int()
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	order := req.Query("order").String()
	if order == "" {
		order = "asc"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	var total int64
	if err := db.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&total).Error; err != nil {
		log.Error("Failed to count messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count messages", 500, err.Error()))
	}

	var messages []models.Message
	query := db.Where("conversation_id = ?", conversationID).
		Preload("User").
		Preload("Client").
		Limit(limit).
		Offset(offset)

	if order == "desc" {
		query = query.Order("created_at DESC")
	} else {
		query = query.Order("created_at ASC")
	}

	if err := query.Find(&messages).Error; err != nil {
		log.Error("Failed to get messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to retrieve messages", 500, err.Error()))
	}

	messageItems := make([]MessageItem, 0, len(messages))
	for _, msg := range messages {
		var authorID string
		var authorName string
		var authorType string
		var avatarURL *string

		isAgent := false
		if msg.UserID != nil {
			isAgent = true
			authorType = "agent"
			if msg.User != nil {
				authorID = msg.User.UserID.String()
				authorName = msg.User.DisplayName
				avatarURL = msg.User.Avatar
				if msg.User.Type == auth.UserTypeBot {
					authorType = "bot"
				}
			}
		} else if msg.ClientID != nil {
			authorType = "customer"
			if msg.Client != nil {
				authorID = msg.Client.ID.String()
				authorName = msg.Client.Name
				avatarURL = msg.Client.Avatar
			}
		} else {
			authorType = "system"
			authorID = "system"
			authorName = "System"
			avatarURL = nil
		}

		initials := getInitials(authorName)

		msgType := msg.Type
		if msgType == "" {
			msgType = models.MessageTypeMessage
		}

		messageItem := MessageItem{
			ID:              msg.ID,
			Body:            msg.Body,
			Type:            msgType,
			Language:        msg.Language,
			IsAgent:         isAgent,
			IsSystemMessage: msg.IsSystemMessage,
			CreatedAt:       msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Author: AuthorInfo{
				ID:        authorID,
				Name:      authorName,
				Type:      authorType,
				AvatarURL: avatarURL,
				Initials:  initials,
			},
			Attachments: []Attachment{},
		}

		messageItems = append(messageItems, messageItem)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	resp := ConversationMessagesResponse{
		ConversationID: conversationID,
		Page:           page,
		Limit:          limit,
		Total:          total,
		TotalPages:     totalPages,
		Messages:       messageItems,
	}

	return response.OK(resp)
}

// AddAgentMessageRequest represents the request body for sending agent messages
type AddAgentMessageRequest struct {
	Body string `json:"body"`
}

// AddAgentMessage handles the POST /api/agent/conversations/:id/messages endpoint
// @Summary Send agent message
// @Description Send a message from an agent to a conversation
// @Tags Agent - Conversations
// @Accept json
// @Produce json
// @Param id path int true "Conversation ID"
// @Param body body AddAgentMessageRequest true "Message content"
// @Success 201 {object} response.Response
// @Router /api/agent/conversations/{id}/messages [post]
func (ac AgentController) AddAgentMessage(req *evo.Request) interface{} {
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var user *auth.User
	var userID uuid.UUID
	if !req.User().Anonymous() {
		user = req.User().Interface().(*auth.User)
		userID = user.UserID
	}

	var conversation models.Conversation
	if err := db.Preload("Client").Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	var input AddAgentMessageRequest
	if err := req.BodyParser(&input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	if strings.TrimSpace(input.Body) == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Message body is required", 400, "Message body cannot be empty"))
	}

	originalBody := input.Body
	messageBody := input.Body
	var translationRecord *models.ConversationMessageTranslation

	// Auto-translate outgoing if enabled
	if user != nil && user.AutoTranslateOutgoing {
		customerLang := getCustomerLanguageFromMessages(conversationID)
		if customerLang == "" {
			if conversation.Client.Language != nil && *conversation.Client.Language != "" {
				customerLang = *conversation.Client.Language
			} else {
				customerLang = "en"
			}
		}

		agentLang := user.Language
		if agentLang == "" {
			agentLang = "en"
		}

		if customerLang != agentLang {
			translated, err := ai.TranslateText(input.Body, agentLang, customerLang)
			if err != nil {
				log.Warning("Failed to translate outgoing message: %v", err)
			} else {
				messageBody = translated
				translationRecord = &models.ConversationMessageTranslation{
					ConversationID: conversationID,
					FromLang:       agentLang,
					ToLang:         customerLang,
					Content:        originalBody,
					Type:           models.TranslationTypeOutgoing,
				}
			}
		}
	}

	message := models.Message{
		ConversationID:  conversationID,
		UserID:          &userID,
		Body:            messageBody,
		IsSystemMessage: false,
	}

	if err := db.Create(&message).Error; err != nil {
		log.Error("Failed to create agent message:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to create message", 500, err.Error()))
	}

	if translationRecord != nil {
		translationRecord.MessageID = message.ID
		if err := db.Create(translationRecord).Error; err != nil {
			log.Warning("Failed to save translation record: %v", err)
		}
	}

	if err := db.Preload("Conversation").Preload("User").First(&message, message.ID).Error; err != nil {
		log.Warning("Failed to preload message relations:", err)
	}

	responseData := map[string]interface{}{
		"message": message,
	}
	if translationRecord != nil {
		responseData["translation"] = map[string]interface{}{
			"original_content":   originalBody,
			"translated_content": messageBody,
			"from_lang":          translationRecord.FromLang,
			"to_lang":            translationRecord.ToLang,
		}
	}

	return response.Created(responseData)
}
