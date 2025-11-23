package conversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

type AgentController struct{}

// ConversationListItem represents a conversation in the list view
type ConversationListItem struct {
	ID                   uint                    `json:"id"`
	ConversationNumber   string                  `json:"conversation_number"`
	Title                string                  `json:"title"`
	Status               string                  `json:"status"`
	Priority             string                  `json:"priority"`
	Channel              string                  `json:"channel"`
	CreatedAt            string                  `json:"created_at"`
	UpdatedAt            string                  `json:"updated_at"`
	LastMessageAt        *string                 `json:"last_message_at"`
	LastMessagePreview   *string                 `json:"last_message_preview"`
	UnreadMessagesCount  int                     `json:"unread_messages_count"`
	IsAssignedToMe       bool                    `json:"is_assigned_to_me"`
	Customer             CustomerInfo            `json:"customer"`
	AssignedAgents       []AgentInfo             `json:"assigned_agents"`
	Department           *DepartmentInfo         `json:"department"`
	Tags                 []TagInfo               `json:"tags"`
	MessageCount         int64                   `json:"message_count"`
	HasAttachments       bool                    `json:"has_attachments"`
	IP                   *string                 `json:"ip"`
	Browser              *string                 `json:"browser"`
	OperatingSystem      *string                 `json:"operating_system"`
}

type ExternalIDInfo struct {
	ID    uint   `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type CustomerInfo struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Email       string           `json:"email"`
	Phone       *string          `json:"phone"`
	AvatarURL   *string          `json:"avatar_url"`
	Initials    string           `json:"initials"`
	ExternalIDs []ExternalIDInfo `json:"external_ids"`
	Language    *string          `json:"language"`
	Timezone    *string          `json:"timezone"`
}

type AgentInfo struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

type DepartmentInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

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

// SearchConversations handles the GET /api/agent/conversations/search endpoint
// @Summary Search and filter conversations
// @Description Get a comprehensive, filterable, searchable list of conversations for the agent dashboard
// @Tags Agent - Conversations
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page (max 100)" default(50)
// @Param search query string false "Full-text search across title, messages, customer name, email"
// @Param status query string false "Comma-separated status values (new,open,in_progress,etc)"
// @Param priority query string false "Comma-separated priority values (low,medium,high,urgent)"
// @Param channel query string false "Comma-separated channel IDs"
// @Param department_id query string false "Comma-separated department IDs"
// @Param tags query string false "Comma-separated tag names or IDs"
// @Param assigned_to_me query boolean false "Filter conversations assigned to authenticated agent"
// @Param unassigned query boolean false "Filter unassigned conversations only"
// @Param has_unread query boolean false "Filter conversations with unread messages"
// @Param sort_by query string false "Sort field (created_at,updated_at,priority,status)" default(updated_at)
// @Param sort_order query string false "Sort order (asc,desc)" default(desc)
// @Param include_unread_count query boolean false "Include total unread count in response"
// @Success 200 {object} ConversationsSearchResponse
// @Router /api/agent/conversations/search [get]
func (ac AgentController) SearchConversations(req *evo.Request) interface{} {
	// Get pagination parameters
	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}

	limit := req.Query("limit").Int()
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Get authenticated user ID (from JWT token)
	userID := req.Get("user_id")
	userIDStr := ""
	if userID.String() != "" {
		userIDStr = userID.String()
	}

	// Build query
	query := db.Model(&models.Conversation{})

	// Apply search filter
	search := req.Query("search").String()
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where(
			db.Where("conversations.title LIKE ?", searchTerm).
				Or("conversations.id IN (?)",
					db.Model(&models.Message{}).
						Select("conversation_id").
						Where("body LIKE ?", searchTerm),
				).
				Or("conversations.client_id IN (?)",
					db.Model(&models.Client{}).
						Select("id").
						Where("name LIKE ? OR data LIKE ?", searchTerm, searchTerm),
				),
		)
	}

	// Apply status filter
	if statusStr := req.Query("status").String(); statusStr != "" {
		statuses := strings.Split(statusStr, ",")
		query = query.Where("conversations.status IN ?", statuses)
	}

	// Apply priority filter
	if priorityStr := req.Query("priority").String(); priorityStr != "" {
		priorities := strings.Split(priorityStr, ",")
		query = query.Where("conversations.priority IN ?", priorities)
	}

	// Apply channel filter
	if channelStr := req.Query("channel").String(); channelStr != "" {
		channels := strings.Split(channelStr, ",")
		query = query.Where("conversations.channel_id IN ?", channels)
	}

	// Apply department filter
	if deptStr := req.Query("department_id").String(); deptStr != "" {
		deptIDs := strings.Split(deptStr, ",")
		query = query.Where("conversations.department_id IN ?", deptIDs)
	}

	// Apply tags filter
	if tagsStr := req.Query("tags").String(); tagsStr != "" {
		tagNames := strings.Split(tagsStr, ",")
		query = query.Where("conversations.id IN (?)",
			db.Model(&models.ConversationTag{}).
				Select("conversation_id").
				Joins("JOIN tags ON tags.id = conversation_tags.tag_id").
				Where("tags.name IN ?", tagNames),
		)
	}

	// Apply assigned_to_me filter
	if req.Query("assigned_to_me").String() == "true" && userIDStr != "" {
		query = query.Where("conversations.id IN (?)",
			db.Model(&models.ConversationAssignment{}).
				Select("conversation_id").
				Where("user_id = ?", userIDStr),
		)
	}

	// Apply unassigned filter
	if req.Query("unassigned").String() == "true" {
		query = query.Where("conversations.id NOT IN (?)",
			db.Model(&models.ConversationAssignment{}).
				Select("DISTINCT conversation_id"),
		)
	}

	// Apply sorting
	sortBy := req.Query("sort_by").String()
	if sortBy == "" {
		sortBy = "updated_at"
	}
	sortOrder := req.Query("sort_order").String()
	if sortOrder == "" {
		sortOrder = "desc"
	}
	query = query.Order(fmt.Sprintf("conversations.%s %s", sortBy, sortOrder))

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Error("Failed to count conversations:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count conversations", 500, err.Error()))
	}

	// Get conversations with relations
	var conversations []models.Conversation
	if err := query.
		Preload("Client").
		Preload("Client.ExternalIDs").
		Preload("Department").
		Preload("Channel").
		Preload("Tags").
		Preload("Assignments").
		Preload("Assignments.User").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error; err != nil {
		log.Error("Failed to get conversations:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get conversations", 500, err.Error()))
	}

	// Build response data
	conversationItems := make([]ConversationListItem, 0, len(conversations))
	for _, conv := range conversations {
		// Get last message
		var lastMessage models.Message
		var hasLastMessage bool
		if err := db.Where("conversation_id = ?", conv.ID).
			Order("created_at DESC").
			First(&lastMessage).Error; err == nil {
			hasLastMessage = true
		}

		// Get message count
		var messageCount int64
		db.Model(&models.Message{}).Where("conversation_id = ?", conv.ID).Count(&messageCount)

		// Build conversation number (format: CONV-{ID})
		conversationNumber := fmt.Sprintf("CONV-%d", conv.ID)

		// Get customer initials
		initials := ""
		if len(conv.Client.Name) > 0 {
			parts := strings.Fields(conv.Client.Name)
			if len(parts) >= 2 {
				initials = string(parts[0][0]) + string(parts[1][0])
			} else if len(parts) == 1 && len(parts[0]) > 0 {
				initials = string(parts[0][0])
			}
			initials = strings.ToUpper(initials)
		}

		// Extract email and phone from external IDs and build external IDs list
		var email string
		var phone *string
		externalIDs := make([]ExternalIDInfo, 0, len(conv.Client.ExternalIDs))
		for _, extID := range conv.Client.ExternalIDs {
			externalIDs = append(externalIDs, ExternalIDInfo{
				ID:    extID.ID,
				Type:  extID.Type,
				Value: extID.Value,
			})
			// Extract primary email and phone
			if extID.Type == "email" && email == "" {
				email = extID.Value
			} else if extID.Type == "phone" && phone == nil {
				phoneValue := extID.Value
				phone = &phoneValue
			}
		}

		// Build assigned agents list
		assignedAgents := make([]AgentInfo, 0, len(conv.Assignments))
		isAssignedToMe := false
		for _, assignment := range conv.Assignments {
			if assignment.User != nil {
				userIDFromAssignment := assignment.User.UserID.String()
				assignedAgents = append(assignedAgents, AgentInfo{
					ID:        userIDFromAssignment,
					Name:      assignment.User.DisplayName,
					AvatarURL: assignment.User.Avatar,
				})
				if userIDFromAssignment == userIDStr {
					isAssignedToMe = true
				}
			}
		}

		// Build tags list
		tags := make([]TagInfo, 0, len(conv.Tags))
		for _, tag := range conv.Tags {
			tags = append(tags, TagInfo{
				ID:    tag.ID,
				Name:  tag.Name,
				Color: "#4ECDC4", // Default color since tags table doesn't have color field
			})
		}

		// Build department info
		var department *DepartmentInfo
		if conv.Department != nil {
			department = &DepartmentInfo{
				ID:   conv.Department.ID,
				Name: conv.Department.Name,
			}
		}

		// Build last message preview
		var lastMessageAt *string
		var lastMessagePreview *string
		if hasLastMessage {
			lastMessageAt = new(string)
			*lastMessageAt = lastMessage.CreatedAt.Format("2006-01-02T15:04:05Z07:00")

			preview := lastMessage.Body
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			lastMessagePreview = &preview
		}

		conversation := ConversationListItem{
			ID:                  conv.ID,
			ConversationNumber:  conversationNumber,
			Title:               conv.Title,
			Status:              conv.Status,
			Priority:            conv.Priority,
			Channel:             conv.ChannelID,
			CreatedAt:           conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:           conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastMessageAt:       lastMessageAt,
			LastMessagePreview:  lastMessagePreview,
			UnreadMessagesCount: 0, // TODO: Implement unread messages tracking
			IsAssignedToMe:      isAssignedToMe,
			Customer: CustomerInfo{
				ID:          conv.Client.ID.String(),
				Name:        conv.Client.Name,
				Email:       email,
				Phone:       phone,
				AvatarURL:   nil,
				Initials:    initials,
				ExternalIDs: externalIDs,
				Language:    conv.Client.Language,
				Timezone:    conv.Client.Timezone,
			},
			AssignedAgents:  assignedAgents,
			Department:      department,
			Tags:            tags,
			MessageCount:    messageCount,
			HasAttachments:  false, // TODO: Check for attachments in messages
			IP:              conv.IP,
			Browser:         conv.Browser,
			OperatingSystem: conv.OperatingSystem,
		}

		conversationItems = append(conversationItems, conversation)
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	// Build response
	resp := ConversationsSearchResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		Data:       conversationItems,
	}

	// Include unread count if requested
	if req.Query("include_unread_count").String() == "true" {
		var unreadCount int64 = 0 // TODO: Implement unread count logic
		resp.UnreadCount = &unreadCount
	}

	return response.OK(resp)
}

// GetUnreadCount handles the GET /api/agent/conversations/unread-count endpoint
// @Summary Get unread conversations count
// @Description Get the total count of unread conversations for the authenticated agent
// @Tags Agent - Conversations
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int64
// @Router /api/agent/conversations/unread-count [get]
func (ac AgentController) GetUnreadCount(req *evo.Request) interface{} {
	// TODO: Implement unread count logic
	// For now, return 0
	return response.OK(map[string]int64{
		"unread_count": 0,
	})
}

// MarkConversationRead handles the PATCH /api/agent/conversations/{id}/read endpoint
// @Summary Mark conversation as read
// @Description Mark all messages in a conversation as read
// @Tags Agent - Conversations
// @Accept json
// @Produce json
// @Param id path int true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/agent/conversations/{id}/read [patch]
func (ac AgentController) MarkConversationRead(req *evo.Request) interface{} {
	// TODO: Implement mark as read logic
	// For now, return success
	conversationID := req.Param("id").String()

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"marked_read_at":  "2025-01-21T15:00:00Z",
	})
}

// GetDepartments handles the GET /api/agent/departments endpoint
// @Summary Get departments list
// @Description Get list of all departments for filtering
// @Tags Agent - Departments
// @Accept json
// @Produce json
// @Success 200 {object} []DepartmentWithCount
// @Router /api/agent/departments [get]
func (ac AgentController) GetDepartments(req *evo.Request) interface{} {
	var departments []models.Department
	if err := db.Find(&departments).Error; err != nil {
		log.Error("Failed to get departments:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get departments", 500, err.Error()))
	}

	type DepartmentWithCount struct {
		ID         uint   `json:"id"`
		Name       string `json:"name"`
		AgentCount int64  `json:"agent_count"`
	}

	result := make([]DepartmentWithCount, 0, len(departments))
	for _, dept := range departments {
		var agentCount int64
		db.Model(&models.UserDepartment{}).Where("department_id = ?", dept.ID).Count(&agentCount)

		result = append(result, DepartmentWithCount{
			ID:         dept.ID,
			Name:       dept.Name,
			AgentCount: agentCount,
		})
	}

	return response.OK(result)
}

// GetUsers handles the GET /api/agent/users endpoint
func (ac AgentController) GetUsers(req *evo.Request) interface{} {
	var users []auth.User
	query := db.Model(&auth.User{}).Select("id, name, last_name, display_name, email, avatar")

	// Optional search parameter
	if search := req.Query("search").String(); search != "" {
		query = query.Where(
			"name LIKE ? OR last_name LIKE ? OR display_name LIKE ? OR email LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%",
		)
	}

	// Get users
	if err := query.Find(&users).Error; err != nil {
		log.Error("Failed to get users:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get users", 500, err.Error()))
	}

	type UserResponse struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		LastName    string    `json:"last_name"`
		DisplayName string    `json:"display_name"`
		Email       string    `json:"email"`
		Avatar      *string   `json:"avatar"`
	}

	result := make([]UserResponse, 0, len(users))
	for _, user := range users {
		result = append(result, UserResponse{
			ID:          user.UserID,
			Name:        user.Name,
			LastName:    user.LastName,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Avatar:      user.Avatar,
		})
	}

	return response.OK(result)
}

// GetTags handles the GET /api/agent/tags endpoint
// @Summary Get tags list
// @Description Get list of all available tags with usage statistics
// @Tags Agent - Tags
// @Accept json
// @Produce json
// @Success 200 {object} []TagWithUsage
// @Router /api/agent/tags [get]
func (ac AgentController) GetTags(req *evo.Request) interface{} {
	var tags []models.Tag
	if err := db.Find(&tags).Error; err != nil {
		log.Error("Failed to get tags:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get tags", 500, err.Error()))
	}

	type TagWithUsage struct {
		ID         uint   `json:"id"`
		Name       string `json:"name"`
		Color      string `json:"color"`
		UsageCount int64  `json:"usage_count"`
	}

	result := make([]TagWithUsage, 0, len(tags))
	for _, tag := range tags {
		var usageCount int64
		db.Model(&models.ConversationTag{}).Where("tag_id = ?", tag.ID).Count(&usageCount)

		result = append(result, TagWithUsage{
			ID:         tag.ID,
			Name:       tag.Name,
			Color:      "#4ECDC4", // Default color since tags table doesn't have color field
			UsageCount: usageCount,
		})
	}

	return response.OK(result)
}

// CreateTag handles the POST /api/agent/tags endpoint
func (ac AgentController) CreateTag(req *evo.Request) interface{} {
	type CreateTagRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var createReq CreateTagRequest
	if err := req.BodyParser(&createReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	// Validate tag name
	if createReq.Name == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Tag name is required", 400, "Tag name cannot be empty"))
	}

	// Check if tag already exists
	var existingTag models.Tag
	if err := db.Where("name = ?", createReq.Name).First(&existingTag).Error; err == nil {
		// Tag already exists, return it
		return response.OK(map[string]interface{}{
			"id":    existingTag.ID,
			"name":  existingTag.Name,
			"color": "#4ECDC4",
		})
	}

	// Create new tag
	tag := models.Tag{
		Name: createReq.Name,
	}

	if err := db.Create(&tag).Error; err != nil {
		log.Error("Failed to create tag:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to create tag", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"id":    tag.ID,
		"name":  tag.Name,
		"color": "#4ECDC4",
	})
}

// MessageItem represents a single message in the conversation
type MessageItem struct {
	ID              uint         `json:"id"`
	Body            string       `json:"body"`
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

// Attachment represents a message attachment (future implementation)
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

// GetConversationMessages handles the GET /api/agent/conversations/{conversation_id}/messages endpoint
// @Summary Get conversation messages
// @Description Retrieve all messages for a specific conversation in chronological order
// @Tags Agent - Conversations
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Messages per page (max 100)" default(50)
// @Param order query string false "Sort order (asc or desc)" default(asc)
// @Success 200 {object} ConversationMessagesResponse
// @Router /api/agent/conversations/{conversation_id}/messages [get]
func (ac AgentController) GetConversationMessages(req *evo.Request) interface{} {
	// Get conversation ID from path parameter
	conversationID := req.Param("conversation_id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	// Check if conversation exists
	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	// Get pagination parameters
	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}

	limit := req.Query("limit").Int()
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Get sort order
	order := req.Query("order").String()
	if order == "" {
		order = "asc"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	// Get total count of messages
	var total int64
	if err := db.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&total).Error; err != nil {
		log.Error("Failed to count messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count messages", 500, err.Error()))
	}

	// Get messages with user and client preloaded
	var messages []models.Message
	query := db.Where("conversation_id = ?", conversationID).
		Preload("User").
		Preload("Client").
		Limit(limit).
		Offset(offset)

	if order == "desc" {
		query = query.Order("created_at DESC")
	} else {
		query = query.Order("created_at ASC")
	}

	if err := query.Find(&messages).Error; err != nil {
		log.Error("Failed to get messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to retrieve messages", 500, err.Error()))
	}

	// Build response
	messageItems := make([]MessageItem, 0, len(messages))
	for _, msg := range messages {
		var authorID string
		var authorName string
		var authorType string
		var avatarURL *string
		var initials string

		// Determine author based on user_id and client_id
		isAgent := false
		if msg.UserID != nil {
			// Message from agent
			isAgent = true
			authorType = "agent"
			if msg.User != nil {
				authorID = msg.User.UserID.String()
				authorName = msg.User.DisplayName
				avatarURL = msg.User.Avatar
			}
		} else if msg.ClientID != nil {
			// Message from customer
			authorType = "customer"
			if msg.Client != nil {
				authorID = msg.Client.ID.String()
				authorName = msg.Client.Name
				avatarURL = nil // Client doesn't have avatar in current schema
			}
		} else {
			// System message
			authorType = "system"
			authorID = "system"
			authorName = "System"
			avatarURL = nil
		}

		// Generate initials
		if len(authorName) > 0 {
			parts := strings.Fields(authorName)
			if len(parts) >= 2 {
				initials = string(parts[0][0]) + string(parts[1][0])
			} else if len(parts) == 1 && len(parts[0]) > 0 {
				initials = string(parts[0][0])
			}
			initials = strings.ToUpper(initials)
		}

		messageItem := MessageItem{
			ID:              msg.ID,
			Body:            msg.Body,
			IsAgent:         isAgent,
			IsSystemMessage: msg.IsSystemMessage,
			CreatedAt:       msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Author: AuthorInfo{
				ID:        authorID,
				Name:      authorName,
				Type:      authorType,
				AvatarURL: avatarURL,
				Initials:  initials,
			},
			Attachments: []Attachment{}, // Empty array for now
		}

		messageItems = append(messageItems, messageItem)
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	resp := ConversationMessagesResponse{
		ConversationID: conversationID,
		Page:           page,
		Limit:          limit,
		Total:          total,
		TotalPages:     totalPages,
		Messages:       messageItems,
	}

	return response.OK(resp)
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

// GetConversationDetail handles the GET /api/agent/conversations/{id} endpoint
// @Summary Get conversation detail with messages (optimized single-call endpoint)
// @Description Get complete conversation details along with messages in a single API call
// @Tags Agent - Conversations
// @Accept json
// @Produce json
// @Param id path int true "Conversation ID"
// @Param page query int false "Page number for messages" default(1)
// @Param limit query int false "Messages per page (max 100)" default(50)
// @Param order query string false "Sort order for messages (asc or desc)" default(asc)
// @Success 200 {object} ConversationDetailResponse
// @Router /api/agent/conversations/{id} [get]
func (ac AgentController) GetConversationDetail(req *evo.Request) interface{} {
	// Get conversation ID from path parameter
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	// Get authenticated user ID (from JWT token)
	userID := req.Get("user_id")
	userIDStr := ""
	if userID.String() != "" {
		userIDStr = userID.String()
	}

	// Get conversation with all relations
	var conv models.Conversation
	if err := db.Where("id = ?", conversationID).
		Preload("Client").
		Preload("Client.ExternalIDs").
		Preload("Department").
		Preload("Channel").
		Preload("Tags").
		Preload("Assignments").
		Preload("Assignments.User").
		First(&conv).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	// Build conversation number
	conversationNumber := fmt.Sprintf("CONV-%d", conv.ID)

	// Get customer initials
	initials := ""
	if len(conv.Client.Name) > 0 {
		parts := strings.Fields(conv.Client.Name)
		if len(parts) >= 2 {
			initials = string(parts[0][0]) + string(parts[1][0])
		} else if len(parts) == 1 && len(parts[0]) > 0 {
			initials = string(parts[0][0])
		}
		initials = strings.ToUpper(initials)
	}

	// Build assigned agents list
	assignedAgents := make([]AgentInfo, 0, len(conv.Assignments))
	isAssignedToMe := false
	for _, assignment := range conv.Assignments {
		if assignment.User != nil {
			userIDFromAssignment := assignment.User.UserID.String()
			assignedAgents = append(assignedAgents, AgentInfo{
				ID:        userIDFromAssignment,
				Name:      assignment.User.DisplayName,
				AvatarURL: assignment.User.Avatar,
			})
			if userIDFromAssignment == userIDStr {
				isAssignedToMe = true
			}
		}
	}

	// Build tags list
	tags := make([]TagInfo, 0, len(conv.Tags))
	for _, tag := range conv.Tags {
		tags = append(tags, TagInfo{
			ID:    tag.ID,
			Name:  tag.Name,
			Color: "#4ECDC4",
		})
	}

	// Build department info
	var department *DepartmentInfo
	if conv.Department != nil {
		department = &DepartmentInfo{
			ID:   conv.Department.ID,
			Name: conv.Department.Name,
		}
	}

	// Get last message
	var lastMessage models.Message
	var hasLastMessage bool
	if err := db.Where("conversation_id = ?", conv.ID).
		Order("created_at DESC").
		First(&lastMessage).Error; err == nil {
		hasLastMessage = true
	}

	// Build last message preview
	var lastMessageAt *string
	var lastMessagePreview *string
	if hasLastMessage {
		lastMessageAt = new(string)
		*lastMessageAt = lastMessage.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		preview := lastMessage.Body
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		lastMessagePreview = &preview
	}

	// Get message count
	var messageCount int64
	db.Model(&models.Message{}).Where("conversation_id = ?", conv.ID).Count(&messageCount)

	// Extract email and phone from external IDs and build external IDs list
	var email string
	var phone *string
	externalIDs := make([]ExternalIDInfo, 0, len(conv.Client.ExternalIDs))
	for _, extID := range conv.Client.ExternalIDs {
		externalIDs = append(externalIDs, ExternalIDInfo{
			ID:    extID.ID,
			Type:  extID.Type,
			Value: extID.Value,
		})
		// Extract primary email and phone
		if extID.Type == "email" && email == "" {
			email = extID.Value
		} else if extID.Type == "phone" && phone == nil {
			phoneValue := extID.Value
			phone = &phoneValue
		}
	}

	// Build conversation item
	conversationItem := ConversationListItem{
		ID:                  conv.ID,
		ConversationNumber:  conversationNumber,
		Title:               conv.Title,
		Status:              conv.Status,
		Priority:            conv.Priority,
		Channel:             conv.ChannelID,
		CreatedAt:           conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastMessageAt:       lastMessageAt,
		LastMessagePreview:  lastMessagePreview,
		UnreadMessagesCount: 0,
		IsAssignedToMe:      isAssignedToMe,
		Customer: CustomerInfo{
			ID:          conv.Client.ID.String(),
			Name:        conv.Client.Name,
			Email:       email,
			Phone:       phone,
			AvatarURL:   nil,
			Initials:    initials,
			ExternalIDs: externalIDs,
			Language:    conv.Client.Language,
			Timezone:    conv.Client.Timezone,
		},
		AssignedAgents:  assignedAgents,
		Department:      department,
		Tags:            tags,
		MessageCount:    messageCount,
		HasAttachments:  false,
		IP:              conv.IP,
		Browser:         conv.Browser,
		OperatingSystem: conv.OperatingSystem,
	}

	// Get pagination parameters for messages
	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}
	limit := req.Query("limit").Int()
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Get sort order
	order := req.Query("order").String()
	if order == "" {
		order = "asc"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	// Get messages
	var messages []models.Message
	query := db.Where("conversation_id = ?", conversationID).
		Preload("User").
		Preload("Client").
		Limit(limit).
		Offset(offset)

	if order == "desc" {
		query = query.Order("created_at DESC")
	} else {
		query = query.Order("created_at ASC")
	}

	if err := query.Find(&messages).Error; err != nil {
		log.Error("Failed to get messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to retrieve messages", 500, err.Error()))
	}

	// Build message items
	messageItems := make([]MessageItem, 0, len(messages))
	for _, msg := range messages {
		var authorID string
		var authorName string
		var authorType string
		var avatarURL *string
		var msgInitials string

		isAgent := false
		if msg.UserID != nil {
			isAgent = true
			authorType = "agent"
			if msg.User != nil {
				authorID = msg.User.UserID.String()
				authorName = msg.User.DisplayName
				avatarURL = msg.User.Avatar
			}
		} else if msg.ClientID != nil {
			authorType = "customer"
			if msg.Client != nil {
				authorID = msg.Client.ID.String()
				authorName = msg.Client.Name
				avatarURL = nil
			}
		} else {
			authorType = "system"
			authorID = "system"
			authorName = "System"
			avatarURL = nil
		}

		// Generate initials
		if len(authorName) > 0 {
			parts := strings.Fields(authorName)
			if len(parts) >= 2 {
				msgInitials = string(parts[0][0]) + string(parts[1][0])
			} else if len(parts) == 1 && len(parts[0]) > 0 {
				msgInitials = string(parts[0][0])
			}
			msgInitials = strings.ToUpper(msgInitials)
		}

		messageItem := MessageItem{
			ID:              msg.ID,
			Body:            msg.Body,
			IsAgent:         isAgent,
			IsSystemMessage: msg.IsSystemMessage,
			CreatedAt:       msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Author: AuthorInfo{
				ID:        authorID,
				Name:      authorName,
				Type:      authorType,
				AvatarURL: avatarURL,
				Initials:  msgInitials,
			},
			Attachments: []Attachment{},
		}

		messageItems = append(messageItems, messageItem)
	}

	// Calculate total pages
	totalPages := int(messageCount) / limit
	if int(messageCount)%limit != 0 {
		totalPages++
	}

	// Build response
	resp := ConversationDetailResponse{
		Conversation: conversationItem,
		Messages:     messageItems,
		Page:         page,
		Limit:        limit,
		Total:        messageCount,
		TotalPages:   totalPages,
	}

	return response.OK(resp)
}

// PreviousConversationItem represents a simplified conversation item for history
type PreviousConversationItem struct {
	ID                 uint   `json:"id"`
	ConversationNumber string `json:"conversation_number"`
	Title              string `json:"title"`
	Status             string `json:"status"`
	Priority           string `json:"priority"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// PreviousConversationsResponse represents the response structure for previous conversations
type PreviousConversationsResponse struct {
	Page       int                        `json:"page"`
	Limit      int                        `json:"limit"`
	Total      int64                      `json:"total"`
	TotalPages int                        `json:"total_pages"`
	Data       []PreviousConversationItem `json:"data"`
}

// GetClientPreviousConversations handles the GET /api/agent/clients/:client_id/conversations endpoint
func (ac AgentController) GetClientPreviousConversations(req *evo.Request) interface{} {
	// Get client ID from path parameter
	clientID := req.Param("client_id").String()
	if clientID == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "client_id is required", 400, "Client ID must be provided"))
	}

	// Get pagination parameters
	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}

	limit := req.Query("limit").Int()
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Get current conversation ID to exclude (optional)
	currentConversationID := req.Query("exclude_id").Uint()

	// Query conversations for this client
	var conversations []models.Conversation
	query := db.Where("client_id = ?", clientID).
		Order("updated_at DESC")

	// Exclude current conversation if specified
	if currentConversationID > 0 {
		query = query.Where("id != ?", currentConversationID)
	}

	// Count total
	var total int64
	if err := query.Model(&models.Conversation{}).Count(&total).Error; err != nil {
		log.Error("Failed to count client conversations: ", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count conversations", 500, err.Error()))
	}

	// Get paginated results
	if err := query.Limit(limit).Offset(offset).Find(&conversations).Error; err != nil {
		log.Error("Failed to fetch client conversations: ", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to fetch conversations", 500, err.Error()))
	}

	// Map to response items
	items := make([]PreviousConversationItem, 0, len(conversations))
	for _, conv := range conversations {
		// Build conversation number (format: CONV-{ID})
		conversationNumber := fmt.Sprintf("CONV-%d", conv.ID)

		items = append(items, PreviousConversationItem{
			ID:                 conv.ID,
			ConversationNumber: conversationNumber,
			Title:              conv.Title,
			Status:             conv.Status,
			Priority:           conv.Priority,
			CreatedAt:          conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:          conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	resp := PreviousConversationsResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		Data:       items,
	}

	return response.OK(resp)
}

// UpdateConversationRequest represents the request body for updating a conversation
type UpdateConversationRequest struct {
	Priority     *string `json:"priority"`
	Status       *string `json:"status"`
	DepartmentID *uint   `json:"department_id"`
}

// UpdateConversationProperties handles the PATCH /api/agent/conversations/:id endpoint
func (ac AgentController) UpdateConversationProperties(req *evo.Request) interface{} {
	// Get conversation ID
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	// Parse request body
	var updateReq UpdateConversationRequest
	if err := req.BodyParser(&updateReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	// Find conversation
	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	// Update fields if provided
	updateData := make(map[string]interface{})

	if updateReq.Priority != nil {
		// Validate priority
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[*updateReq.Priority] {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid priority", 400, "Priority must be one of: low, medium, high, urgent"))
		}
		updateData["priority"] = *updateReq.Priority
	}

	if updateReq.Status != nil {
		// Validate status
		validStatuses := map[string]bool{
			"new": true, "user_reply": true, "agent_reply": true,
			"processing": true, "closed": true, "archived": true, "postponed": true,
		}
		if !validStatuses[*updateReq.Status] {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid status", 400, "Invalid status value"))
		}
		updateData["status"] = *updateReq.Status

		// If closing or archiving, set closed_at
		if *updateReq.Status == "closed" || *updateReq.Status == "archived" {
			now := time.Now()
			updateData["closed_at"] = &now
		}
	}

	if updateReq.DepartmentID != nil {
		updateData["department_id"] = *updateReq.DepartmentID
	}

	// Perform update
	if len(updateData) > 0 {
		if err := db.Model(&conversation).Updates(updateData).Error; err != nil {
			log.Error("Failed to update conversation: ", err)
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to update conversation", 500, err.Error()))
		}
	}

	// Reload conversation with relations
	if err := db.Preload("Client").Preload("Department").Preload("Tags").First(&conversation, conversationID).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to reload conversation", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"id":            conversation.ID,
		"priority":      conversation.Priority,
		"status":        conversation.Status,
		"department_id": conversation.DepartmentID,
		"updated_at":    conversation.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdateTagsRequest represents the request body for updating tags
type UpdateTagsRequest struct {
	TagIDs []uint `json:"tag_ids"`
}

// UpdateConversationTags handles the PUT /api/agent/conversations/:id/tags endpoint
func (ac AgentController) UpdateConversationTags(req *evo.Request) interface{} {
	// Get conversation ID
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	// Parse request body
	var tagsReq UpdateTagsRequest
	if err := req.BodyParser(&tagsReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	// Find conversation
	var conversation models.Conversation
	if err := db.Preload("Tags").Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	// Get tags
	var tags []models.Tag
	if len(tagsReq.TagIDs) > 0 {
		if err := db.Where("id IN ?", tagsReq.TagIDs).Find(&tags).Error; err != nil {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to fetch tags", 500, err.Error()))
		}
	}

	// Replace tags
	if err := db.Model(&conversation).Association("Tags").Replace(tags); err != nil {
		log.Error("Failed to update conversation tags: ", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to update tags", 500, err.Error()))
	}

	// Return updated tags
	var updatedTags []TagInfo
	for _, tag := range tags {
		updatedTags = append(updatedTags, TagInfo{
			ID:   tag.ID,
			Name: tag.Name,
		})
	}

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"tags":            updatedTags,
	})
}

// AssignRequest represents the request body for assigning conversation
type AssignRequest struct {
	UserIDs      []string `json:"user_ids"`
	DepartmentID *uint    `json:"department_id"`
}

// AssignConversation handles the POST /api/agent/conversations/:id/assign endpoint
func (ac AgentController) AssignConversation(req *evo.Request) interface{} {
	// Get conversation ID
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	// Parse request body
	var assignReq AssignRequest
	if err := req.BodyParser(&assignReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	// Find conversation
	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	// Clear existing assignments
	if err := db.Where("conversation_id = ?", conversationID).Delete(&models.ConversationAssignment{}).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to clear assignments", 500, err.Error()))
	}

	// Assign to users
	for _, userIDStr := range assignReq.UserIDs {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid user ID", 400, err.Error()))
		}

		assignment := models.ConversationAssignment{
			ConversationID: conversationID,
			UserID:         &userID,
		}
		if err := db.Create(&assignment).Error; err != nil {
			log.Error("Failed to assign user: ", err)
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to assign user", 500, err.Error()))
		}
	}

	// Assign to department
	if assignReq.DepartmentID != nil {
		assignment := models.ConversationAssignment{
			ConversationID: conversationID,
			DepartmentID:   assignReq.DepartmentID,
		}
		if err := db.Create(&assignment).Error; err != nil {
			log.Error("Failed to assign department: ", err)
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to assign department", 500, err.Error()))
		}
	}

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"assigned":        true,
	})
}

// UnassignConversation handles the DELETE /api/agent/conversations/:id/assign endpoint
func (ac AgentController) UnassignConversation(req *evo.Request) interface{} {
	// Get conversation ID
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	// Clear all assignments
	if err := db.Where("conversation_id = ?", conversationID).Delete(&models.ConversationAssignment{}).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to clear assignments", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"unassigned":      true,
	})
}
