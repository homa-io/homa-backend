package conversation

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// parseJSONToMap converts datatypes.JSON to map[string]interface{}
func parseJSONToMap(data datatypes.JSON) map[string]interface{} {
	if data == nil || len(data) == 0 {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// ConversationListItem represents a conversation in the list view
type ConversationListItem struct {
	ID                  uint                   `json:"id"`
	ConversationNumber  string                 `json:"conversation_number"`
	Title               string                 `json:"title"`
	Status              string                 `json:"status"`
	Priority            string                 `json:"priority"`
	HandleByBot         bool                   `json:"handle_by_bot"`
	Channel             string                 `json:"channel"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
	LastMessageAt       *string                `json:"last_message_at"`
	LastMessagePreview  *string                `json:"last_message_preview"`
	UnreadMessagesCount int                    `json:"unread_messages_count"`
	IsAssignedToMe      bool                   `json:"is_assigned_to_me"`
	Customer            CustomerInfo           `json:"customer"`
	AssignedAgents      []AgentInfo            `json:"assigned_agents"`
	Department          *DepartmentInfo        `json:"department"`
	Inbox               *InboxInfo             `json:"inbox"`
	Tags                []TagInfo              `json:"tags"`
	MessageCount        int64                  `json:"message_count"`
	HasAttachments      bool                   `json:"has_attachments"`
	IP                  *string                `json:"ip"`
	Browser             *string                `json:"browser"`
	OperatingSystem     *string                `json:"operating_system"`
	Data                map[string]interface{} `json:"data,omitempty"`
}

// ExternalIDInfo represents an external identifier for a customer
type ExternalIDInfo struct {
	ID    uint   `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// CustomerInfo represents customer information
type CustomerInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Email       string                 `json:"email"`
	Phone       *string                `json:"phone"`
	AvatarURL   *string                `json:"avatar_url"`
	Initials    string                 `json:"initials"`
	ExternalIDs []ExternalIDInfo       `json:"external_ids"`
	Language    *string                `json:"language"`
	Timezone    *string                `json:"timezone"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// AgentInfo represents agent information
type AgentInfo struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

// DepartmentInfo represents department information
type DepartmentInfo struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	AIAgentID *uint  `json:"ai_agent_id"`
}

// InboxInfo represents inbox information
type InboxInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// TagInfo represents tag information
type TagInfo struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ConversationsSearchResponse represents the paginated response for conversations search
type ConversationsSearchResponse struct {
	Page        int                    `json:"page"`
	Limit       int                    `json:"limit"`
	Total       int64                  `json:"total"`
	TotalPages  int                    `json:"total_pages"`
	UnreadCount *int64                 `json:"unread_count,omitempty"`
	Data        []ConversationListItem `json:"data"`
}

// MessageItem represents a single message in the conversation
type MessageItem struct {
	ID              uint         `json:"id"`
	Body            string       `json:"body"`
	Type            string       `json:"type"` // message or action
	Language        string       `json:"language"`
	IsAgent         bool         `json:"is_agent"`
	IsSystemMessage bool         `json:"is_system_message"`
	CreatedAt       string       `json:"created_at"`
	Author          AuthorInfo   `json:"author"`
	Attachments     []Attachment `json:"attachments"`
}

// AuthorInfo represents the message author information
type AuthorInfo struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"` // customer, agent, or system
	AvatarURL *string `json:"avatar_url"`
	Initials  string  `json:"initials"`
}

// Attachment represents a message attachment
type Attachment struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

// ConversationMessagesResponse represents the response for conversation messages
type ConversationMessagesResponse struct {
	ConversationID uint          `json:"conversation_id"`
	Page           int           `json:"page"`
	Limit          int           `json:"limit"`
	Total          int64         `json:"total"`
	TotalPages     int           `json:"total_pages"`
	Messages       []MessageItem `json:"messages"`
}

// ConversationDetailResponse represents the optimized response with conversation + messages
type ConversationDetailResponse struct {
	Conversation ConversationListItem `json:"conversation"`
	Messages     []MessageItem        `json:"messages"`
	Page         int                  `json:"page"`
	Limit        int                  `json:"limit"`
	Total        int64                `json:"total"`
	TotalPages   int                  `json:"total_pages"`
}
