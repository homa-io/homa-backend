package livechat

import (
	"os"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers livechat HTTP and WebSocket routes
func RegisterRoutes() error {
	// Initialize JWT secret for agent WebSocket authentication
	initJWTSecret()

	// Serve the livechat UI
	evo.Static("/livechat", "./static/livechat")

	// Serve the chat widget SDK
	evo.Static("/widget", "./static/widget")

	// Get the Fiber app instance
	app := evo.GetFiber()

	// Add CORS headers for widget scripts
	app.Use("/widget", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET")
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Next()
	})

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket route for conversations (client-facing with secret auth)
	app.Get("/ws/conversations/:conversation_id/:secret", websocket.New(HandleWebSocket))

	// WebSocket route for agents (JWT authenticated, receives all conversation events)
	app.Get("/ws/agent", websocket.New(HandleAgentWebSocket))

	return nil
}

// initJWTSecret initializes the JWT secret from config
func initJWTSecret() {
	secret := settings.Get("JWT.SECRET").String()
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		log.Warning("JWT_SECRET not set for livechat WebSocket, using development key")
		secret = "your-secret-key-change-this-in-production"
	}
	SetJWTSecret([]byte(secret))
	log.Debug("Livechat JWT secret initialized")
}
