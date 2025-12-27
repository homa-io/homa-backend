package ai

import (
	"github.com/getevo/evo/v2"
	"github.com/google/uuid"
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

// GenerateResponseHandler handles POST /api/ai/generate-response
// @Summary Generate a response using RAG
// @Description Generates an AI response based on conversation context and knowledge base
// @Tags AI
// @Accept json
// @Produce json
// @Param body body GenerateResponseRequest true "Response generation request"
// @Success 200 {object} GenerateResponseResponse
// @Router /api/ai/generate-response [post]
func (c Controller) GenerateResponseHandler(req *evo.Request) interface{} {
	var genReq GenerateResponseRequest
	if err := req.BodyParser(&genReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if len(genReq.Messages) == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Messages are required", 400, "messages array cannot be empty"))
	}

	result, err := GenerateResponse(genReq)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Response generation failed", 500, err.Error()))
	}

	return response.OK(result)
}

// IndexArticleHandler handles POST /api/ai/index-article/:id
// @Summary Index an article for RAG
// @Description Creates embeddings for a knowledge base article
// @Tags AI - Admin
// @Accept json
// @Produce json
// @Param id path string true "Article UUID"
// @Success 200 {object} map[string]interface{}
// @Router /api/ai/index-article/{id} [post]
func (c Controller) IndexArticleHandler(req *evo.Request) interface{} {
	articleIDStr := req.Param("id").String()
	if articleIDStr == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Article ID is required", 400, "id parameter cannot be empty"))
	}

	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid article ID", 400, err.Error()))
	}

	if err := IndexArticle(articleID); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Indexing failed", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"message":    "Article indexed successfully",
		"article_id": articleID.String(),
	})
}

// ReindexAllHandler handles POST /api/ai/reindex-all
// @Summary Reindex all articles for RAG
// @Description Creates embeddings for all published knowledge base articles
// @Tags AI - Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/ai/reindex-all [post]
func (c Controller) ReindexAllHandler(req *evo.Request) interface{} {
	if err := ReindexAllArticles(); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Reindexing failed", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"message": "All articles reindexed successfully",
	})
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

// GetIndexStatsHandler handles GET /api/ai/index-stats
// @Summary Get Qdrant index statistics
// @Description Returns statistics about the knowledge base vector index
// @Tags AI - Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/ai/index-stats [get]
func (c Controller) GetIndexStatsHandler(req *evo.Request) interface{} {
	stats, err := GetIndexStats()
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to get index stats", 500, err.Error()))
	}

	return response.OK(stats)
}
