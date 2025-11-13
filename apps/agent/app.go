package agent

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

	// Unread tickets
	evo.Get("/api/agent/tickets/unread", controller.GetUnreadTickets)
	evo.Get("/api/agent/tickets/unread/count", controller.GetUnreadTicketsCount)

	// Ticket management
	evo.Get("/api/agent/tickets", controller.GetTicketList)
	evo.Put("/api/agent/tickets/:id/status", controller.ChangeTicketStatus)
	evo.Post("/api/agent/tickets/:id/reply", controller.ReplyToTicket)
	evo.Put("/api/agent/tickets/:id/assign", controller.AssignTicket)
	evo.Put("/api/agent/tickets/:id/department", controller.ChangeTicketDepartment)
	evo.Put("/api/agent/tickets/:id/tags", controller.TagTicket)

	// Tag management
	evo.Post("/api/agent/tags", controller.AddTag)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "agent"
}
