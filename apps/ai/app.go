package ai

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/iesreza/homa-backend/apps/models"
)

// App represents the AI application
type App struct{}

// Register initializes the AI app
func (a App) Register() error {
	// Check if OpenAI/Assistant is configured (support both config formats)
	apiKey := settings.Get("ASSISTANT.APIKEY").String()
	if apiKey == "" {
		apiKey = settings.Get("OPENAI.API_KEY").String()
	}
	if apiKey == "" {
		log.Warning("ASSISTANT.APIKEY not configured - AI features will be disabled")
		return nil
	}

	// Initialize the OpenAI client
	if err := InitClient(); err != nil {
		log.Error("Failed to initialize OpenAI client: %v", err)
		return err
	}

	// Initialize Qdrant client (required for RAG)
	if err := InitQdrant(); err != nil {
		log.Error("Failed to initialize Qdrant client: %v", err)
		return err
	}

	// Register the indexer with the models package for GORM hooks
	models.SetKnowledgeBaseIndexer(&ArticleIndexer{})

	log.Info("AI app initialized successfully with model: %s", settings.Get("ASSISTANT.MODEL", "gpt-4o").String())
	return nil
}

// Router sets up the AI API routes
func (a App) Router() error {
	controller := Controller{}

	// AI API endpoints (require authentication via middleware applied in main)
	// Translation
	evo.Post("/api/ai/translate", controller.TranslateHandler)

	// Text revision
	evo.Post("/api/ai/revise", controller.ReviseHandler)
	evo.Get("/api/ai/formats", controller.GetFormatsHandler)

	// Conversation summarization
	evo.Post("/api/ai/summarize", controller.SummarizeHandler)

	// RAG-based response generation
	evo.Post("/api/ai/generate-response", controller.GenerateResponseHandler)

	// Admin endpoints for knowledge base indexing
	evo.Post("/api/ai/index-article/:id", controller.IndexArticleHandler)
	evo.Post("/api/ai/reindex-all", controller.ReindexAllHandler)
	evo.Get("/api/ai/index-stats", controller.GetIndexStatsHandler)

	return nil
}

// WhenReady is called when the app is ready
func (a App) WhenReady() error {
	return nil
}

// Name returns the app name
func (a App) Name() string {
	return "ai"
}
