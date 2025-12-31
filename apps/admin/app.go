package admin

import (
	"github.com/getevo/evo/v2"
)

type App struct {
}

func (a App) Register() error {
	return nil
}

func (a App) Router() error {
	var controller Controller

	// Apply admin authentication middleware to all admin routes
	// Temporarily disabled for testing
	evo.Use("/api/admin", AdminAuthMiddleware)

	// Ticket management APIs
	evo.Get("/api/admin/tickets/unread", controller.GetUnreadTickets)
	evo.Get("/api/admin/tickets/unread/count", controller.GetUnreadTicketsCount)
	evo.Get("/api/admin/tickets", controller.ListTickets)
	evo.Put("/api/admin/tickets/:id/status", controller.ChangeTicketStatus)
	evo.Post("/api/admin/tickets/:id/reply", controller.ReplyToTicket)
	evo.Put("/api/admin/tickets/:id/assign", controller.AssignTicket)
	evo.Put("/api/admin/tickets/:id/departments", controller.ChangeTicketDepartments)
	evo.Put("/api/admin/tickets/:id/tags", controller.TagTicket)
	evo.Delete("/api/admin/tickets/:id", controller.DeleteTicket)

	// Department management APIs
	evo.Post("/api/admin/departments", controller.CreateDepartment)
	evo.Get("/api/admin/departments", controller.ListDepartments)
	evo.Get("/api/admin/departments/:id", controller.GetDepartment)
	evo.Put("/api/admin/departments/:id", controller.EditDepartment)
	evo.Put("/api/admin/departments/:id/suspend", controller.SuspendDepartment)
	evo.Delete("/api/admin/departments/:id", controller.SoftDeleteDepartment)

	// Tag management APIs
	evo.Post("/api/admin/tags", controller.AddTag)
	evo.Delete("/api/admin/tags/:id", controller.DeleteTag)

	// User management APIs
	evo.Post("/api/admin/users", controller.CreateUser)
	evo.Put("/api/admin/users/:id", controller.EditUser)
	evo.Get("/api/admin/users", controller.ListUsers)
	evo.Put("/api/admin/users/:id/departments", controller.AssignUserToDepartment)
	evo.Put("/api/admin/users/:id/block", controller.BlockUser)
	evo.Post("/api/admin/upload/avatar", controller.UploadAvatar)

	// Custom attribute management APIs
	evo.Post("/api/admin/attributes", controller.CreateCustomAttribute)
	evo.Put("/api/admin/attributes/:scope/:name", controller.EditCustomAttribute)
	evo.Delete("/api/admin/attributes/:scope/:name", controller.DeleteCustomAttribute)
	evo.Get("/api/admin/attributes", controller.ListCustomAttributes)

	// Channel management APIs
	evo.Post("/api/admin/channels", controller.CreateChannel)
	evo.Put("/api/admin/channels/:id", controller.EditChannel)
	evo.Delete("/api/admin/channels/:id", controller.DeleteChannel)
	evo.Get("/api/admin/channels", controller.ListChannels)

	// Client management APIs
	evo.Get("/api/admin/clients", controller.ListClients)
	evo.Get("/api/admin/clients/:id", controller.GetClient)
	evo.Post("/api/admin/clients", controller.CreateClient)
	evo.Put("/api/admin/clients/:id", controller.UpdateClient)
	evo.Delete("/api/admin/clients/:id", controller.DeleteClient)
	evo.Post("/api/admin/clients/merge", controller.MergeClients)

	// Message management APIs
	evo.Delete("/api/admin/messages/:id", controller.DeleteMessage)

	// Knowledge Base Article APIs
	evo.Get("/api/admin/kb/articles", controller.ListKBArticles)
	evo.Get("/api/admin/kb/articles/:id", controller.GetKBArticle)
	evo.Post("/api/admin/kb/articles", controller.CreateKBArticle)
	evo.Put("/api/admin/kb/articles/:id", controller.UpdateKBArticle)
	evo.Delete("/api/admin/kb/articles/:id", controller.DeleteKBArticle)

	// Knowledge Base Category APIs
	evo.Get("/api/admin/kb/categories", controller.ListKBCategories)
	evo.Get("/api/admin/kb/categories/:id", controller.GetKBCategory)
	evo.Post("/api/admin/kb/categories", controller.CreateKBCategory)
	evo.Put("/api/admin/kb/categories/:id", controller.UpdateKBCategory)
	evo.Delete("/api/admin/kb/categories/:id", controller.DeleteKBCategory)

	// Knowledge Base Tag APIs
	evo.Get("/api/admin/kb/tags", controller.ListKBTags)
	evo.Get("/api/admin/kb/tags/:id", controller.GetKBTag)
	evo.Post("/api/admin/kb/tags", controller.CreateKBTag)
	evo.Put("/api/admin/kb/tags/:id", controller.UpdateKBTag)
	evo.Delete("/api/admin/kb/tags/:id", controller.DeleteKBTag)

	// Knowledge Base Media Upload
	evo.Post("/api/admin/kb/upload", controller.UploadKBMedia)

	// Webhook management APIs
	evo.Get("/api/admin/webhooks", controller.ListWebhooks)
	evo.Get("/api/admin/webhooks/:id", controller.GetWebhook)
	evo.Post("/api/admin/webhooks", controller.CreateWebhook)
	evo.Put("/api/admin/webhooks/:id", controller.UpdateWebhook)
	evo.Delete("/api/admin/webhooks/:id", controller.DeleteWebhook)
	evo.Post("/api/admin/webhooks/:id/test", controller.TestWebhook)

	// Webhook delivery logs APIs
	evo.Get("/api/admin/webhook_deliveries", controller.ListWebhookDeliveries)
	evo.Get("/api/admin/webhook_deliveries/:id", controller.GetWebhookDelivery)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "admin"
}
