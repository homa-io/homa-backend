package bot

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
)

// App represents the Bot application
type App struct{}

// Register initializes the Bot app
func (a App) Register() error {
	log.Info("Registering bot app...")
	return nil
}

// Router sets up the Bot API routes
func (a App) Router() error {
	log.Info("Registering bot routes...")
	controller := &Controller{}

	// Bot API routes - protected by security_key in Authorization header
	evo.Post("/api/bot/:bot_id/conversation/:conversation_id", controller.SendMessage)

	return nil
}

// WhenReady is called when the app is ready
func (a App) WhenReady() error {
	log.Info("Bot app ready")
	return nil
}

// Name returns the app name
func (a App) Name() string {
	return "bot"
}
