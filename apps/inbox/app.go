package inbox

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
)

// App represents the Inbox application module
type App struct{}

var _ application.Application = (*App)(nil)

// Register initializes the inbox module
func (App) Register() error {
	log.Info("Registering Inbox app...")

	// Register model for migration
	db.UseModel(models.Inbox{})

	return nil
}

// Router registers HTTP routes for inbox management
func (App) Router() error {
	// Admin endpoints (protected by admin middleware in system app)
	evo.Get("/api/admin/inboxes", ListInboxes)
	evo.Post("/api/admin/inboxes", CreateInbox)
	evo.Get("/api/admin/inboxes/:id", GetInbox)
	evo.Put("/api/admin/inboxes/:id", UpdateInbox)
	evo.Delete("/api/admin/inboxes/:id", DeleteInbox)
	evo.Post("/api/admin/inboxes/:id/regenerate-key", RegenerateAPIKey)

	// Client endpoint for SDK to get inbox config
	evo.Get("/api/client/inbox/:key", GetInboxByAPIKey)

	return nil
}

// WhenReady is called after all apps are ready
func (App) WhenReady() error {
	// Ensure default inbox exists
	inbox, err := models.EnsureDefaultInbox()
	if err != nil {
		log.Warning("[inbox] Failed to ensure default inbox: %v", err)
	} else {
		log.Info("[inbox] Default inbox ready: %s (API Key: %s...)", inbox.Name, inbox.APIKey[:20])
	}

	return nil
}

// Shutdown gracefully stops the inbox module
func (App) Shutdown() error {
	log.Info("Shutting down Inbox app...")
	return nil
}

// Name returns the app name
func (App) Name() string {
	return "inbox"
}
