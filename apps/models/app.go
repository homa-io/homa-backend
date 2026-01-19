package models

import (
	"github.com/getevo/evo/v2/lib/args"
	"github.com/getevo/evo/v2/lib/db"
)

type App struct{}

func (a App) Register() error {
	// Register all models with GORM (auth models are now registered in auth app)
	db.UseModel(Client{})
	db.UseModel(ClientExternalID{})
	db.UseModel(Department{})
	db.UseModel(AIAgent{})
	db.UseModel(AIAgentTool{})
	db.UseModel(Channel{})
	db.UseModel(Conversation{})
	db.UseModel(Message{})
	db.UseModel(Tag{})
	db.UseModel(ConversationAssignment{})
	db.UseModel(ConversationTag{})
	db.UseModel(UserDepartment{})
	db.UseModel(ConversationReadStatus{})
	db.UseModel(ActivityLog{})
	db.UseModel(CustomAttribute{})
	db.UseModel(Webhook{})
	db.UseModel(WebhookDelivery{})
	db.UseModel(CannedMessage{})

	// Knowledge Base models for RAG
	db.UseModel(KnowledgeBaseArticle{})
	db.UseModel(KnowledgeBaseCategory{})
	db.UseModel(KnowledgeBaseTag{})
	db.UseModel(KnowledgeBaseArticleTag{})
	db.UseModel(KnowledgeBaseChunk{})
	db.UseModel(KnowledgeBaseMedia{})

	// Settings model
	db.UseModel(Setting{})

	// Integrations model
	db.UseModel(Integration{})

	// User session tracking models
	db.UseModel(UserSession{})
	db.UseModel(UserDailyActivity{})

	// Conversation summary model
	db.UseModel(ConversationSummary{})

	// User preferences model
	db.UseModel(UserPreference{})

	// Email tracking model
	db.UseModel(EmailMessage{})

	return nil
}

func (a App) Router() error {
	return nil
}

func (a App) WhenReady() error {
	if args.Exists("--migration-do") {
		err := db.DoMigration()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a App) Name() string {
	return "models"
}
