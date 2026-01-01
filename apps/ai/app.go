package ai

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	natsconn "github.com/iesreza/homa-backend/apps/nats"
	"github.com/nats-io/nats.go"
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

	// Article summary generation
	evo.Post("/api/ai/generate-summary", controller.GenerateArticleSummaryHandler)

	// Smart reply - analyze, translate if needed, and fix grammar
	evo.Post("/api/ai/smart-reply", controller.SmartReplyHandler)

	return nil
}

// WhenReady is called when the app is ready
func (a App) WhenReady() error {
	// Subscribe to settings reload events
	if natsconn.IsConnected() {
		_, err := natsconn.Subscribe("settings.ai.reload", func(msg *nats.Msg) {
			log.Info("Received AI settings reload message")
			ReloadSettings()
		})
		if err != nil {
			log.Warning("Failed to subscribe to settings.ai.reload: %v", err)
		} else {
			log.Info("Subscribed to settings.ai.reload for realtime settings updates")
		}
	}
	return nil
}

// Name returns the app name
func (a App) Name() string {
	return "ai"
}
