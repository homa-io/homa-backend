package conversation

import (
	"github.com/getevo/evo/v2"
)

type App struct{}

func (a App) Register() error {

	return nil
}

func (a App) Router() error {
	var controller = Controller{}
	var agentController = AgentController{}

	// Client-facing APIs
	evo.Put("/api/client/conversations", controller.CreateConversation)
	evo.Post("/api/client/conversations/:conversation_id/:secret/messages", controller.AddClientMessage)
	evo.Get("/api/client/conversations/:conversation_id/:secret", controller.GetConversationWithSecret)
	evo.Delete("/api/client/conversations/:conversation_id/:secret", controller.CloseConversationWithSecret)
	evo.Post("/api/client/upsert", controller.UpsertClient) // Changed to POST to match UI

	// Admin conversation APIs
	evo.Get("/api/admin/conversations/:conversation_id", controller.GetConversationDetail)

	// Admin assignment APIs
	evo.Post("/api/admin/conversations/:conversation_id/assign/user", controller.AssignConversationToUser)
	evo.Post("/api/admin/conversations/:conversation_id/assign/department", controller.AssignConversationToDepartment)
	evo.Delete("/api/admin/conversations/:conversation_id/unassign", controller.UnassignConversation)
	evo.Get("/api/admin/conversations/:conversation_id/assignments", controller.GetConversationAssignments)

	// Agent APIs for conversations
	evo.Get("/api/agent/conversations/search", agentController.SearchConversations)
	evo.Get("/api/agent/conversations/:conversation_id/messages", agentController.GetConversationMessages)
	evo.Get("/api/agent/conversations/unread-count", agentController.GetUnreadCount)
	evo.Patch("/api/agent/conversations/:id/read", agentController.MarkConversationRead)
	evo.Get("/api/agent/departments", agentController.GetDepartments)
	evo.Get("/api/agent/tags", agentController.GetTags)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "conversation"
}
