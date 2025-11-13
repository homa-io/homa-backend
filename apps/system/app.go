package system

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/getevo/restify"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"strings"
	"time"
)

var StartupTime = time.Now()
var BasePath = ""

type App struct {
}

func (a App) Register() error {
	var logLevel = settings.Get("APP.LOG_LEVEL", "info").String()
	switch strings.ToLower(logLevel) {
	case "debug", "dev", "development":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn", "warning":
		log.SetLevel(log.WarningLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "critical", "crit":
		log.SetLevel(log.CriticalLevel)
	default:
		log.SetLevel(log.WarningLevel)
	}

	if settings.Get("APP.LOG_REQUESTS").Bool() {
		var app = evo.GetFiber()
		// Enable request logging
		app.Use(logger.New())
	}

	restify.SetPrefix("/api/restify")

	return nil
}

func (a App) Router() error {
	var controller Controller
	evo.Get("/health", controller.HealthHandler)
	evo.Get("/uptime", controller.UptimeHandler)

	// Public APIs
	evo.Get("/api/system/departments", controller.GetDepartments)
	evo.Get("/api/system/ticket-status", controller.GetTicketStatuses)

	// Serve static files
	evo.Static("/static", "./static")
	BasePath = settings.Get("APP.BASE_PATH", "http://localhost:8000").String()

	// Agent dashboard
	evo.Get("/dashboard/login.html", controller.ServeLoginPage)
	evo.Get("/dashboard", controller.ServeDashboard)

	evo.Use("/api/restify", controller.AdminMiddleware)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "system"
}
