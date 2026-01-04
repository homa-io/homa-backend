package aiagents

import (
	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/apps/admin"
)

type App struct {
}

func (a App) Register() error {
	return nil
}

func (a App) Router() error {
	// Register restify routes for AI agents at /api/admin/ai-agents
	// Note: Restify API is embedded in AIAgent model, routes are auto-generated

	// Apply admin authentication to AI agent routes
	evo.Use("/api/admin/ai-agents", admin.AdminAuthMiddleware)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "aiagents"
}
