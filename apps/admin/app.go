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
	evo.Put("/api/admin/departments/:id", controller.EditDepartment)
	evo.Delete("/api/admin/departments/:id", controller.SoftDeleteDepartment)
	evo.Get("/api/admin/departments", controller.ListDepartments)

	// Tag management APIs
	evo.Post("/api/admin/tags", controller.AddTag)
	evo.Delete("/api/admin/tags/:id", controller.DeleteTag)

	// User management APIs
	evo.Post("/api/admin/users", controller.CreateUser)
	evo.Put("/api/admin/users/:id", controller.EditUser)
	evo.Get("/api/admin/users", controller.ListUsers)
	evo.Put("/api/admin/users/:id/departments", controller.AssignUserToDepartment)
	evo.Put("/api/admin/users/:id/block", controller.BlockUser)

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

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "admin"
}
