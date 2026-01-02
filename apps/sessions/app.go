package sessions

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

	// Session management
	evo.Post("/api/sessions/start", controller.StartSession)
	evo.Post("/api/sessions/heartbeat", controller.Heartbeat)
	evo.Post("/api/sessions/end", controller.EndSession)

	// Session listing
	evo.Get("/api/sessions", controller.GetMySessions)
	evo.Get("/api/sessions/active", controller.GetActiveSessions)
	evo.Get("/api/sessions/history", controller.GetSessionHistory)
	evo.Delete("/api/sessions/:id", controller.TerminateSession)
	evo.Post("/api/sessions/terminate-all", controller.TerminateAllOtherSessions)

	// Activity tracking
	evo.Get("/api/activity/daily", controller.GetDailyActivity)
	evo.Get("/api/activity/today", controller.GetTodayActivity)
	evo.Get("/api/activity/summary", controller.GetActivitySummary)
	evo.Get("/api/activity/stats", controller.GetActivityStats)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "sessions"
}
