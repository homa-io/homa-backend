package livechat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/apps/nats"
	"github.com/gofiber/contrib/websocket"
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
