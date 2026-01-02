package main

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/iesreza/homa-backend/apps/admin"
	"github.com/iesreza/homa-backend/apps/agent"
	"github.com/iesreza/homa-backend/apps/ai"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/bot"
	"github.com/iesreza/homa-backend/apps/conversation"
	"github.com/iesreza/homa-backend/apps/integrations"
	"github.com/iesreza/homa-backend/apps/livechat"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/apps/nats"
	"github.com/iesreza/homa-backend/apps/rag"
	"github.com/iesreza/homa-backend/apps/sessions"
	"github.com/iesreza/homa-backend/apps/storage"
	"github.com/iesreza/homa-backend/apps/swagger"
	"github.com/iesreza/homa-backend/apps/system"
	"github.com/iesreza/homa-backend/apps/webhook"
)

func main() {
	evo.Setup()

	var apps = application.GetInstance()
	apps.Register(system.App{}, auth.App{}, models.App{}, nats.App{}, storage.App{}, conversation.App{}, agent.App{}, admin.App{}, webhook.App{}, livechat.App{}, swagger.App{}, ai.App{}, bot.App{}, sessions.App{}, integrations.App{}, rag.App{})

	evo.Run()
}
