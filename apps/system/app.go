package system

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/getevo/restify"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/iesreza/homa-backend/apps/minify"
	"strings"
	"time"
)

// Request size limits
const (
	MaxBodySize       = 1 * 1024 * 1024  // 1MB for regular requests
	MaxUploadSize     = 25 * 1024 * 1024 // 25MB for file uploads
	RateLimitRequests = 100              // requests per minute
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

	var app = evo.GetFiber()

	// Enable request logging if configured
	if settings.Get("APP.LOG_REQUESTS").Bool() {
		app.Use(logger.New())
	}

	// Add rate limiting middleware (100 requests per minute per IP)
	if settings.Get("APP.RATE_LIMIT", true).Bool() {
		app.Use(limiter.New(limiter.Config{
			Max:        RateLimitRequests,
			Expiration: 1 * time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(429).JSON(fiber.Map{
					"error": "Too many requests. Please try again later.",
				})
			},
		}))
		log.Info("Rate limiting enabled: %d requests per minute", RateLimitRequests)
	}

	// NOTE: CORS is handled by nginx reverse proxy, not here.
	// Adding CORS headers in both nginx and Go causes duplicate headers which browsers reject.
	// See /etc/nginx/sites-enabled/api.getevo.dev for CORS configuration.

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

	// Settings APIs (admin only)
	evo.Use("/api/settings", controller.AdminMiddleware)
	evo.Get("/api/settings", controller.GetSettings)
	evo.Put("/api/settings", controller.UpdateSettings)

	// Rate limit settings
	evo.Get("/api/settings/rate-limits", controller.GetRateLimitSettings)
	evo.Get("/api/settings/rate-limits/status", controller.GetRedisStatus)
	evo.Put("/api/settings/rate-limits/:key", controller.UpdateRateLimitSetting)

	evo.Get("/api/settings/:key", controller.GetSetting)
	evo.Put("/api/settings/:key", controller.SetSetting)
	evo.Delete("/api/settings/:key", controller.DeleteSetting)

	// Serve minified widget JS (must be before static handler)
	var minifyController = minify.NewController("./static/widget")
	evo.GetFiber().Get("/widget/*", minifyController.ServeMinifiedJS)

	// Serve static files
	evo.Static("/static", "./static")
	evo.Static("/uploads", "./uploads")
	BasePath = settings.Get("APP.BASE_PATH", "http://localhost:8000").String()

	// Agent dashboard
	evo.Get("/dashboard/login.html", controller.ServeLoginPage)
	evo.Get("/dashboard", controller.ServeDashboard)

	evo.Use("/api/restify", controller.AdminMiddleware)

	// Activity Logs APIs (admin only)
	evo.Use("/api/activity-logs", controller.AdminMiddleware)
	evo.Get("/api/activity-logs", controller.GetActivityLogs)
	evo.Get("/api/activity-logs/:entity_type/:entity_id", controller.GetEntityActivityLogs)

	return nil
}

func (a App) WhenReady() error {
	// Clear minify cache on startup
	minify.DefaultMinifier().ClearCache()
	log.Info("JS minify cache cleared on startup")
	return nil
}

func (a App) Name() string {
	return "system"
}
