package livechat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/gofiber/contrib/websocket"
	"github.com/golang-jwt/jwt/v5"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/apps/nats"
	natsclient "github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type WebSocketConn struct {
	conn  *websocket.Conn
	mutex sync.Mutex
}

var (
	// Store active WebSocket connections
	wsConnections = make(map[uint]*sync.Map) // conversation_id -> map[connection_id]*WebSocketConn
	wsLock        sync.RWMutex
)

// HandleWebSocket handles WebSocket connections for conversations
func HandleWebSocket(c *websocket.Conn) {
	conversationIDStr := c.Params("conversation_id")
	secret := c.Params("secret")

	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		log.Error("Invalid conversation ID: %v", err)
		c.Close()
		return
	}

	// Verify conversation exists and secret matches
	var conversation models.Conversation
	if err := db.Where("id = ? AND secret = ?", conversationID, secret).First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Warning("Conversation not found or invalid secret: %d", conversationID)
		} else {
			log.Error("Error verifying conversation: %v", err)
		}
		c.Close()
		return
	}

	log.Info("WebSocket connected for conversation %d", conversationID)

	// Register this WebSocket connection first
	connectionID := fmt.Sprintf("%p", c)
	wsConn := &WebSocketConn{conn: c}
	wsLock.Lock()
	if _, exists := wsConnections[uint(conversationID)]; !exists {
		wsConnections[uint(conversationID)] = &sync.Map{}
	}
	wsConnections[uint(conversationID)].Store(connectionID, wsConn)
	wsLock.Unlock()

	// Subscribe to NATS channel for this conversation
	// Each WebSocket connection subscribes independently and receives all messages
	subject := fmt.Sprintf("conversation.%d", conversationID)
	sub, err := nats.Subscribe(subject, func(msg *natsclient.Msg) {
		// Filter out action messages from client view (they are internal activity logs)
		var msgData map[string]interface{}
		if err := json.Unmarshal(msg.Data, &msgData); err == nil {
			// Only filter message.created events, let other events through
			if event, ok := msgData["event"].(string); ok && event == "message.created" {
				// Check if this message has type "action"
				if message, ok := msgData["message"].(map[string]interface{}); ok {
					if msgType, ok := message["type"].(string); ok && msgType == "action" {
						// Skip action messages for clients
						return
					}
				}
			}
		}

		// Send NATS message to THIS WebSocket connection only
		wsConn.mutex.Lock()
		err := wsConn.conn.WriteMessage(websocket.TextMessage, msg.Data)
		wsConn.mutex.Unlock()
		if err != nil {
			log.Error("Error sending message to WebSocket: %v", err)
		}
	})

	if err != nil {
		log.Error("Failed to subscribe to NATS: %v", err)
		c.Close()
		return
	}
	defer sub.Unsubscribe()

	// Clean up on disconnect
	defer func() {
		wsLock.Lock()
		if connections, exists := wsConnections[uint(conversationID)]; exists {
			connections.Delete(connectionID)
			// Remove empty map
			count := 0
			connections.Range(func(_, _ interface{}) bool {
				count++
				return true
			})
			if count == 0 {
				delete(wsConnections, uint(conversationID))
			}
		}
		wsLock.Unlock()
		log.Info("WebSocket disconnected for conversation %d", conversationID)
	}()

	// Read messages from WebSocket (we don't expect client to send messages via WS)
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket error: %v", err)
			}
			break
		}
	}
}

// BroadcastToConversation sends a message to all WebSocket connections for a conversation
func BroadcastToConversation(conversationID uint, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Publish to NATS - this will trigger the subscription handlers
	subject := fmt.Sprintf("conversation.%d", conversationID)
	return nats.Publish(subject, jsonData)
}

// Agent WebSocket connections - for dashboard agents
var (
	agentConnections = &sync.Map{} // connectionID -> *WebSocketConn
)

// HandleAgentWebSocket handles WebSocket connections for agents (JWT authenticated)
// This allows agents to receive all conversation events in real-time
func HandleAgentWebSocket(c *websocket.Conn) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		log.Warning("Agent WebSocket: No token provided")
		c.WriteJSON(map[string]string{"error": "authentication required"})
		c.Close()
		return
	}

	// Validate JWT token
	userID, err := validateJWTToken(token)
	if err != nil {
		log.Warning("Agent WebSocket: Invalid token: %v", err)
		c.WriteJSON(map[string]string{"error": "invalid token"})
		c.Close()
		return
	}

	log.Info("Agent WebSocket connected for user %s", userID)

	// Register this connection
	connectionID := fmt.Sprintf("%p", c)
	wsConn := &WebSocketConn{conn: c}
	agentConnections.Store(connectionID, wsConn)

	// Subscribe to all conversation events using wildcard
	// This receives: conversation.created, conversation.updated, message.created, etc.
	sub, err := nats.Subscribe("conversations.>", func(msg *natsclient.Msg) {
		wsConn.mutex.Lock()
		err := wsConn.conn.WriteMessage(websocket.TextMessage, msg.Data)
		wsConn.mutex.Unlock()
		if err != nil {
			log.Error("Error sending message to agent WebSocket: %v", err)
		}
	})

	if err != nil {
		log.Error("Agent WebSocket: Failed to subscribe to NATS: %v", err)
		c.WriteJSON(map[string]string{"error": "subscription failed"})
		c.Close()
		return
	}
	defer sub.Unsubscribe()

	// Also subscribe to individual conversation updates (conversation.{id} pattern)
	subConv, err := nats.Subscribe("conversation.>", func(msg *natsclient.Msg) {
		wsConn.mutex.Lock()
		err := wsConn.conn.WriteMessage(websocket.TextMessage, msg.Data)
		wsConn.mutex.Unlock()
		if err != nil {
			log.Error("Error sending message to agent WebSocket: %v", err)
		}
	})

	if err != nil {
		log.Error("Agent WebSocket: Failed to subscribe to conversation NATS: %v", err)
	} else {
		defer subConv.Unsubscribe()
	}

	// Send confirmation
	c.WriteJSON(map[string]string{"status": "connected", "user_id": userID})

	// Clean up on disconnect
	defer func() {
		agentConnections.Delete(connectionID)
		log.Info("Agent WebSocket disconnected for user %s", userID)
	}()

	// Keep connection alive and handle ping/pong
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("Agent WebSocket error: %v", err)
			}
			break
		}
	}
}

// validateJWTToken validates a JWT token and returns the user ID
func validateJWTToken(tokenString string) (string, error) {
	// Import auth package's JWT validation
	token, err := parseJWT(tokenString)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	return claims.UserID, nil
}

// jwtClaims mirrors auth.Claims for JWT parsing
type jwtClaims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Departments []string `json:"departments"`
	jwt.RegisteredClaims
}

// parseJWT parses and validates a JWT token
func parseJWT(tokenString string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
}

// jwtSecret will be set during initialization
var jwtSecret []byte

// SetJWTSecret sets the JWT secret for token validation
func SetJWTSecret(secret []byte) {
	jwtSecret = secret
}
