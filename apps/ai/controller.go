package ai

import (
	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/lib/response"
)

// Controller handles AI-related HTTP requests
type Controller struct{}

// TranslateHandler handles POST /api/ai/translate
// @Summary Translate text to a target language
// @Description Translates the given text to the specified language using AI
// @Tags AI
// @Accept json
// @Produce json
// @Param body body TranslateRequest true "Translation request"
// @Success 200 {object} TranslateResponse
// @Router /api/ai/translate [post]
func (c Controller) TranslateHandler(req *evo.Request) interface{} {
	var translateReq TranslateRequest
	if err := req.BodyParser(&translateReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if translateReq.Text == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Text is required", 400, "text field cannot be empty"))
	}

	if translateReq.Language == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Language is required", 400, "language field cannot be empty"))
	}

	result, err := Translate(translateReq)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Translation failed", 500, err.Error()))
	}

	return response.OK(result)
}

// ReviseHandler handles POST /api/ai/revise
// @Summary Revise text according to a format
// @Description Rewrites the given text according to the specified format/style using AI
// @Tags AI
// @Accept json
// @Produce json
// @Param body body ReviseRequest true "Revision request"
// @Success 200 {object} ReviseResponse
// @Router /api/ai/revise [post]
func (c Controller) ReviseHandler(req *evo.Request) interface{} {
	var reviseReq ReviseRequest
	if err := req.BodyParser(&reviseReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if reviseReq.Text == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Text is required", 400, "text field cannot be empty"))
	}

	if reviseReq.Format == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Format is required", 400, "format field cannot be empty"))
	}

	result, err := Revise(reviseReq)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Revision failed", 500, err.Error()))
	}

	return response.OK(result)
}

// SummarizeHandler handles POST /api/ai/summarize
// @Summary Summarize a conversation
// @Description Creates a summary of the given conversation messages
// @Tags AI
// @Accept json
// @Produce json
// @Param body body SummarizeRequest true "Summarization request"
// @Success 200 {object} SummarizeResponse
// @Router /api/ai/summarize [post]
func (c Controller) SummarizeHandler(req *evo.Request) interface{} {
	var summarizeReq SummarizeRequest
	if err := req.BodyParser(&summarizeReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if len(summarizeReq.Messages) == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Messages are required", 400, "messages array cannot be empty"))
	}

	result, err := Summarize(summarizeReq)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Summarization failed", 500, err.Error()))
	}

	return response.OK(result)
}

// GetFormatsHandler handles GET /api/ai/formats
// @Summary Get available revision formats
// @Description Returns a list of available text revision formats
// @Tags AI
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/ai/formats [get]
func (c Controller) GetFormatsHandler(req *evo.Request) interface{} {
	formats := []map[string]string{
		{"id": "formal", "name": "Formal", "description": "Professional and formal language"},
		{"id": "casual", "name": "Casual", "description": "Relaxed, conversational tone"},
		{"id": "professional", "name": "Professional", "description": "Business-appropriate language"},
		{"id": "friendly", "name": "Friendly", "description": "Warm and approachable"},
		{"id": "concise", "name": "Concise", "description": "Brief and to the point"},
		{"id": "detailed", "name": "Detailed", "description": "Expanded with more context"},
		{"id": "empathetic", "name": "Empathetic", "description": "Understanding and supportive"},
		{"id": "technical", "name": "Technical", "description": "Precise technical language"},
	}

	return response.OK(formats)
}

// GenerateArticleSummaryHandler handles POST /api/ai/generate-summary
// @Summary Generate a summary for an article
// @Description Generates a concise summary for a knowledge base article
// @Tags AI
// @Accept json
// @Produce json
// @Param body body GenerateArticleSummaryRequest true "Article summary request"
// @Success 200 {object} GenerateArticleSummaryResponse
// @Router /api/ai/generate-summary [post]
func (c Controller) GenerateArticleSummaryHandler(req *evo.Request) interface{} {
	var genReq GenerateArticleSummaryRequest
	if err := req.BodyParser(&genReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if genReq.Content == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Content is required", 400, "content field cannot be empty"))
	}

	result, err := GenerateArticleSummary(genReq)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Summary generation failed", 500, err.Error()))
	}

	return response.OK(result)
}

// SmartReplyHandler handles POST /api/ai/smart-reply
// @Summary Smart reply - analyze, translate if needed, and fix grammar
// @Description Analyzes agent message, compares with user language, translates if needed, and fixes grammar
// @Tags AI
// @Accept json
// @Produce json
// @Param body body SmartReplyRequest true "Smart reply request"
// @Success 200 {object} SmartReplyResponse
// @Router /api/ai/smart-reply [post]
func (c Controller) SmartReplyHandler(req *evo.Request) interface{} {
	var smartReq SmartReplyRequest
	if err := req.BodyParser(&smartReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if smartReq.AgentMessage == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Agent message is required", 400, "agent_message field cannot be empty"))
	}

	if smartReq.UserLastMessage == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "User last message is required", 400, "user_last_message field cannot be empty"))
	}

	result, err := SmartReply(smartReq)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Smart reply failed", 500, err.Error()))
	}

	return response.OK(result)
}
