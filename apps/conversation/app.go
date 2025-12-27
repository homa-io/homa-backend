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
	evo.Get("/api/agent/conversations/:id", agentController.GetConversationDetail)
	evo.Get("/api/agent/conversations/:conversation_id/messages", agentController.GetConversationMessages)
	evo.Post("/api/agent/conversations/:id/messages", agentController.AddAgentMessage)
	evo.Get("/api/agent/conversations/unread-count", agentController.GetUnreadCount)
	evo.Patch("/api/agent/conversations/:id/read", agentController.MarkConversationRead)
	evo.Get("/api/agent/departments", agentController.GetDepartments)
	evo.Get("/api/agent/users", agentController.GetUsers)
	evo.Get("/api/agent/tags", agentController.GetTags)
	evo.Post("/api/agent/tags", agentController.CreateTag)
	evo.Get("/api/agent/clients/:client_id/conversations", agentController.GetClientPreviousConversations)
	evo.Get("/api/agent/clients", agentController.ListClients)
	evo.Get("/api/agent/clients/:id", agentController.GetClient)
	evo.Post("/api/agent/clients", agentController.CreateClient)
	evo.Put("/api/agent/clients/:id", agentController.UpdateClient)
	evo.Delete("/api/agent/clients/:id", agentController.DeleteClient)
	evo.Post("/api/agent/clients/:id/avatar", agentController.UploadClientAvatar)
	evo.Delete("/api/agent/clients/:id/avatar", agentController.DeleteClientAvatar)
	evo.Patch("/api/agent/conversations/:id", agentController.UpdateConversationProperties)
	evo.Put("/api/agent/conversations/:id/tags", agentController.UpdateConversationTags)
	evo.Post("/api/agent/conversations/:id/assign", agentController.AssignConversation)
	evo.Delete("/api/agent/conversations/:id/assign", agentController.UnassignConversation)

	// Agent Custom Attributes APIs
	evo.Get("/api/agent/attributes", agentController.ListCustomAttributes)
	evo.Post("/api/agent/attributes", agentController.CreateCustomAttribute)
	evo.Put("/api/agent/attributes/:scope/:name", agentController.UpdateCustomAttribute)
	evo.Delete("/api/agent/attributes/:scope/:name", agentController.DeleteCustomAttribute)

	// Agent Canned Messages APIs
	evo.Get("/api/agent/canned-messages", agentController.ListCannedMessages)
	evo.Get("/api/agent/canned-messages/:id", agentController.GetCannedMessage)
	evo.Post("/api/agent/canned-messages", agentController.CreateCannedMessage)
	evo.Put("/api/agent/canned-messages/:id", agentController.UpdateCannedMessage)
	evo.Delete("/api/agent/canned-messages/:id", agentController.DeleteCannedMessage)

	// User Avatar APIs
	evo.Post("/api/agent/me/avatar", agentController.UploadUserAvatar)
	evo.Delete("/api/agent/me/avatar", agentController.DeleteUserAvatar)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "conversation"
}
