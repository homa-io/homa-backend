package livechat

import (
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/log"
)

// App represents the livechat application module
type App struct{}

// Register initializes the livechat app
func (App) Register() error {
	log.Info("Registering livechat app...")
	return nil
}

// Router registers HTTP routes for livechat
func (App) Router() error {
	log.Info("Registering livechat routes...")
	return RegisterRoutes()
}

// WhenReady is called when the app is ready
func (App) WhenReady() error {
	log.Info("Livechat app ready")
	return nil
}

// Name returns the app name
func (App) Name() string {
	return "livechat"
}

// Shutdown gracefully closes the livechat app
func (App) Shutdown() error {
	log.Info("Shutting down livechat app...")
	return nil
}

// GetInstance returns the singleton livechat app instance
func GetInstance() *App {
	return &App{}
}

var _ application.Application = (*App)(nil)
