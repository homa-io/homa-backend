package livechat

import (
	"github.com/getevo/evo/v2"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers livechat HTTP and WebSocket routes
func RegisterRoutes() error {
	// Serve the livechat UI
	evo.Static("/livechat", "./static/livechat")

	// Get the Fiber app instance
	app := evo.GetFiber()

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket route for conversations
	app.Get("/ws/conversations/:conversation_id/:secret", websocket.New(HandleWebSocket))

	return nil
}
