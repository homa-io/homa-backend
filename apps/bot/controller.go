package bot

import (
	"strconv"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

type Controller struct{}

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	Message string `json:"message"`
}

// SendMessage handles POST /api/bot/:bot_id/conversation/:conversation_id
// @Summary Send a message to a conversation as a bot
// @Description Allows a bot to send a message to a specific conversation. Requires security_key in Authorization header.
// @Tags Bot
// @Accept json
// @Produce json
// @Param bot_id path string true "Bot User ID"
// @Param conversation_id path int true "Conversation ID"
// @Param Authorization header string true "Bot security key"
// @Param request body SendMessageRequest true "Message content"
// @Success 201 {object} models.Message "Message created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid input"
// @Failure 401 {object} response.ErrorResponse "Unauthorized - invalid or missing security key"
// @Failure 403 {object} response.ErrorResponse "Forbidden - user is not a bot"
// @Failure 404 {object} response.ErrorResponse "Conversation not found"
// @Router /api/bot/{bot_id}/conversation/{conversation_id} [post]
func (c Controller) SendMessage(request *evo.Request) interface{} {
	// Get bot_id from path
	botIDStr := request.Param("bot_id").String()
	if botIDStr == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Bot ID is required", 400))
	}

	// Parse bot UUID
	botID, err := uuid.Parse(botIDStr)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid bot ID format", 400))
	}

	// Get conversation_id from path
	conversationIDStr := request.Param("conversation_id").String()
	if conversationIDStr == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Conversation ID is required", 400))
	}

	// Parse conversation ID
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid conversation ID format", 400))
	}

	// Get security key from Authorization header
	securityKey := request.Header("Authorization")
	if securityKey == "" {
		return response.Error(response.NewError(response.ErrorCodeUnauthorized, "Authorization header is required", 401))
	}

	// Find the bot user
	var botUser auth.User
	if err := db.Where("id = ?", botID).First(&botUser).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "Bot not found", 404))
	}

	// Verify user is a bot
	if botUser.Type != auth.UserTypeBot {
		return response.Error(response.NewError(response.ErrorCodeForbidden, "User is not a bot", 403))
	}

	// Verify security key
	if botUser.SecurityKey == nil || *botUser.SecurityKey != securityKey {
		return response.Error(response.NewError(response.ErrorCodeUnauthorized, "Invalid security key", 401))
	}

	// Parse request body
	var req SendMessageRequest
	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid request body: "+err.Error(), 400))
	}

	// Validate message
	if req.Message == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Message is required", 400))
	}

	// Check if conversation exists
	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "Conversation not found", 404))
	}

	// Create message
	message := models.Message{
		ConversationID: uint(conversationID),
		UserID:         &botUser.UserID,
		Body:           req.Message,
		CreatedAt:      time.Now(),
	}

	if err := db.Create(&message).Error; err != nil {
		return response.Error(response.NewError(response.ErrorCodeInternalError, "Failed to create message", 500))
	}

	// Load the user relationship for the response
	message.User = &botUser

	return response.Created(message)
}
