package ai

import (
	"encoding/json"
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// ConversationSummaryResponse is the response structure for conversation summary
type ConversationSummaryResponse struct {
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	Version      int      `json:"version"`
	MessageCount int      `json:"message_count"`
	NeedsUpdate  bool     `json:"needs_update"`
	Language     string   `json:"language"`
}

// GetConversationSummaryHandler handles GET /api/ai/conversation-summary/:id
// Returns existing summary or indicates that generation is needed
// Supports ?language=xx query param, defaults to user's language, then Accept-Language header, then "en"
func (c Controller) GetConversationSummaryHandler(req *evo.Request) interface{} {
	// Check if user is authenticated
	user := req.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Check if conversation summary feature is enabled
	enabled := models.GetSettingValue("ai.conversation_summary_enabled", "false")
	if enabled != "true" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeForbidden, "Conversation summary feature is disabled", 403, ""))
	}

	conversationID := req.Param("id").Int()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, ""))
	}

	// Determine language: query param > user language > Accept-Language header > default "en"
	language := req.Query("language").String()
	if language == "" {
		language = user.Language
	}
	if language == "" {
		// Try to parse Accept-Language header
		acceptLang := req.Header("Accept-Language")
		if acceptLang != "" && len(acceptLang) >= 2 {
			// Take first 2 chars (e.g., "en-US" -> "en")
			language = strings.ToLower(acceptLang[:2])
		}
	}
	if language == "" {
		language = "en"
	}

	// Get message count for this conversation
	var messageCount int64
	if err := db.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&messageCount).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to count messages", 500, err.Error()))
	}

	// Get existing summary for this language
	var summary models.ConversationSummary
	err := db.Where("conversation_id = ? AND language = ?", conversationID, language).First(&summary).Error

	if err != nil {
		// No summary exists for this language
		return response.OK(ConversationSummaryResponse{
			Summary:      "",
			KeyPoints:    []string{},
			Version:      0,
			MessageCount: int(messageCount),
			NeedsUpdate:  messageCount > 0,
			Language:     language,
		})
	}

	// Parse key points from JSON
	var keyPoints []string
	if summary.KeyPoints != "" {
		json.Unmarshal([]byte(summary.KeyPoints), &keyPoints)
	}

	return response.OK(ConversationSummaryResponse{
		Summary:      summary.Summary,
		KeyPoints:    keyPoints,
		Version:      summary.Version,
		MessageCount: int(messageCount),
		NeedsUpdate:  summary.Version != int(messageCount),
		Language:     language,
	})
}

// GenerateConversationSummaryHandler handles POST /api/ai/conversation-summary/:id/generate
// Generates or regenerates the summary for a conversation
// Supports ?language=xx query param, defaults to user's language, then Accept-Language header, then "en"
func (c Controller) GenerateConversationSummaryHandler(req *evo.Request) interface{} {
	// Check if user is authenticated
	user := req.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	// Check if conversation summary feature is enabled
	enabled := models.GetSettingValue("ai.conversation_summary_enabled", "false")
	if enabled != "true" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeForbidden, "Conversation summary feature is disabled", 403, ""))
	}

	conversationID := req.Param("id").Int()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, ""))
	}

	// Determine language: query param > user language > Accept-Language header > default "en"
	language := req.Query("language").String()
	if language == "" {
		language = user.Language
	}
	if language == "" {
		// Try to parse Accept-Language header
		acceptLang := req.Header("Accept-Language")
		if acceptLang != "" && len(acceptLang) >= 2 {
			// Take first 2 chars (e.g., "en-US" -> "en")
			language = strings.ToLower(acceptLang[:2])
		}
	}
	if language == "" {
		language = "en"
	}

	// Verify conversation exists
	var conversation models.Conversation
	if err := db.First(&conversation, conversationID).Error; err != nil {
		return response.NotFound(req, "Conversation not found")
	}

	// Get all messages for this conversation
	var messages []models.Message
	if err := db.Where("conversation_id = ?", conversationID).
		Preload("User").
		Preload("Client").
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to fetch messages", 500, err.Error()))
	}

	if len(messages) == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "No messages to summarize", 400, ""))
	}

	// Convert to MessageInput format for AI
	var messageInputs []MessageInput
	for _, msg := range messages {
		role := "user"
		author := ""
		if msg.UserID != nil {
			role = "agent"
			if msg.User != nil {
				author = msg.User.Name
			}
		} else if msg.ClientID != nil {
			role = "user"
			if msg.Client != nil {
				author = msg.Client.Name
			}
		}

		messageInputs = append(messageInputs, MessageInput{
			Role:    role,
			Content: msg.Body,
			Author:  author,
		})
	}

	// Generate summary using AI with language
	result, err := Summarize(SummarizeRequest{Messages: messageInputs, Language: language})
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to generate summary", 500, err.Error()))
	}

	// Convert key points to JSON
	keyPointsJSON, _ := json.Marshal(result.KeyPoints)

	// Save or update the summary for this language
	var existingSummary models.ConversationSummary
	err = db.Where("conversation_id = ? AND language = ?", conversationID, language).First(&existingSummary).Error

	if err != nil {
		// Create new summary
		newSummary := models.ConversationSummary{
			ConversationID: uint(conversationID),
			Language:       language,
			Summary:        result.Summary,
			KeyPoints:      string(keyPointsJSON),
			Version:        len(messages),
		}
		if err := db.Create(&newSummary).Error; err != nil {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to save summary", 500, err.Error()))
		}
	} else {
		// Update existing summary
		existingSummary.Summary = result.Summary
		existingSummary.KeyPoints = string(keyPointsJSON)
		existingSummary.Version = len(messages)
		if err := db.Save(&existingSummary).Error; err != nil {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to update summary", 500, err.Error()))
		}
	}

	return response.OK(ConversationSummaryResponse{
		Summary:      result.Summary,
		KeyPoints:    result.KeyPoints,
		Version:      len(messages),
		MessageCount: len(messages),
		NeedsUpdate:  false,
		Language:     language,
	})
}

// Helper to strip HTML from message body if needed
func stripHTML(html string) string {
	// Simple HTML tag removal
	result := html
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}
