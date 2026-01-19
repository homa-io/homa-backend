package conversation

import (
	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/apps/redis"
)

type App struct{}

func (a App) Register() error {

	return nil
}

func (a App) Router() error {
	var controller = Controller{}
	var agentController = AgentController{}
	var translationController = TranslationController{}

	// Client-facing APIs with rate limiting
	evo.Use("/api/client/conversations", redis.EvoRateLimitMiddleware("client.create_conversation"))
	evo.Put("/api/client/conversations", controller.CreateConversation)
	evo.Post("/api/client/conversations/:conversation_id/:secret/messages", controller.AddClientMessage)
	evo.Get("/api/client/conversations/:conversation_id/:secret", controller.GetConversationWithSecret)
	evo.Delete("/api/client/conversations/:conversation_id/:secret", controller.CloseConversationWithSecret)
	evo.Post("/api/client/upsert", controller.UpsertClient)

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
	evo.Get("/api/agent/me/departments", agentController.GetMyDepartments)
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

	// User Preferences APIs
	evo.Get("/api/agent/me/preferences", agentController.GetUserPreferences)
	evo.Put("/api/agent/me/preferences", agentController.UpdateUserPreferences)
	evo.Get("/api/agent/notification-sounds", agentController.GetNotificationSounds)

	// Translation APIs
	evo.Post("/api/agent/conversations/:id/translations", translationController.GetTranslations)
	evo.Post("/api/agent/conversations/:id/outgoing-translations", translationController.GetOutgoingTranslations)
	evo.Post("/api/agent/conversations/:id/translate-outgoing", translationController.TranslateOutgoing)
	evo.Get("/api/agent/conversations/:id/language-info", translationController.GetLanguageInfo)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "conversation"
}
