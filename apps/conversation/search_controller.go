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
	"gorm.io/gorm"
)

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

	// Get authenticated user ID from JWT token via User interface
	userIDStr := ""
	if !req.User().Anonymous() {
		user := req.User().Interface().(*auth.User)
		userIDStr = user.UserID.String()
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

	// Apply inbox filter
	if inboxStr := req.Query("inbox_id").String(); inboxStr != "" {
		inboxIDs := strings.Split(inboxStr, ",")
		query = query.Where("conversations.inbox_id IN ?", inboxIDs)
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

	// Apply sorting with whitelist validation to prevent SQL injection
	sortBy := req.Query("sort_by").String()
	// Whitelist of allowed sort columns for conversations
	allowedSortColumns := map[string]bool{
		"id":         true,
		"title":      true,
		"status":     true,
		"priority":   true,
		"created_at": true,
		"updated_at": true,
		"closed_at":  true,
	}
	if !allowedSortColumns[sortBy] {
		sortBy = "updated_at" // Default to safe column
	}

	sortOrder := req.Query("sort_order").String()
	// Validate sort order to prevent SQL injection
	if sortOrder != "asc" && sortOrder != "desc" {
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
		Joins("Client").
		Joins("Department").
		Joins("Channel").
		Preload("Inbox").
		Preload("Client.ExternalIDs").
		Preload("Tags").
		Preload("Assignments", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC").Limit(10)
		}).
		Preload("Assignments.User").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error; err != nil {
		log.Error("Failed to get conversations:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get conversations", 500, err.Error()))
	}

	// Batch load last messages and message counts for all conversations
	conversationIDs := make([]uint, len(conversations))
	for i, conv := range conversations {
		conversationIDs[i] = conv.ID
	}

	// Batch load last message for each conversation
	type lastMessageResult struct {
		ConversationID uint
		ID             uint
		Body           string
		CreatedAt      time.Time
	}
	var lastMessages []lastMessageResult
	if err := db.Raw(`
		SELECT m.conversation_id, m.id, m.body, m.created_at
		FROM messages m
		WHERE m.conversation_id IN (?)
		AND m.id IN (
			SELECT MAX(id) FROM messages
			WHERE conversation_id IN (?)
			GROUP BY conversation_id
		)
	`, conversationIDs, conversationIDs).
		Scan(&lastMessages).Error; err != nil {
		log.Error("Failed to batch load last messages:", err)
		lastMessages = []lastMessageResult{}
	}

	// Batch load message counts for all conversations
	type messageCountResult struct {
		ConversationID uint
		Count          int64
	}
	var messageCounts []messageCountResult
	if err := db.Raw(`
		SELECT conversation_id, COUNT(*) as count
		FROM messages
		WHERE conversation_id IN (?)
		GROUP BY conversation_id
	`, conversationIDs).
		Scan(&messageCounts).Error; err != nil {
		log.Error("Failed to batch load message counts:", err)
		messageCounts = []messageCountResult{}
	}

	// Build maps for O(1) lookup
	lastMessageMap := make(map[uint]lastMessageResult)
	for _, lm := range lastMessages {
		lastMessageMap[lm.ConversationID] = lm
	}

	messageCountMap := make(map[uint]int64)
	for _, mc := range messageCounts {
		messageCountMap[mc.ConversationID] = mc.Count
	}

	// Build response data
	conversationItems := make([]ConversationListItem, 0, len(conversations))
	for _, conv := range conversations {
		// Get last message from map
		var lastMessageAt *string
		var lastMessagePreview *string

		if lastMsg, exists := lastMessageMap[conv.ID]; exists {
			lastMessageAt = new(string)
			*lastMessageAt = lastMsg.CreatedAt.Format("2006-01-02T15:04:05Z07:00")

			preview := lastMsg.Body
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			lastMessagePreview = &preview
		}

		// Get message count from map
		messageCount := messageCountMap[conv.ID]

		// Build conversation number
		conversationNumber := fmt.Sprintf("CONV-%d", conv.ID)

		// Get customer initials
		initials := getInitials(conv.Client.Name)

		// Extract email and phone from external IDs
		var email string
		var phone *string
		externalIDs := make([]ExternalIDInfo, 0, len(conv.Client.ExternalIDs))
		for _, extID := range conv.Client.ExternalIDs {
			externalIDs = append(externalIDs, ExternalIDInfo{
				ID:    extID.ID,
				Type:  extID.Type,
				Value: extID.Value,
			})
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
				Color: "#4ECDC4",
			})
		}

		// Build department info
		var department *DepartmentInfo
		if conv.Department != nil {
			department = &DepartmentInfo{
				ID:        conv.Department.ID,
				Name:      conv.Department.Name,
				AIAgentID: conv.Department.AIAgentID,
			}
		}

		// Build inbox info
		var inbox *InboxInfo
		if conv.Inbox != nil {
			inbox = &InboxInfo{
				ID:   conv.Inbox.ID,
				Name: conv.Inbox.Name,
			}
		}

		conversation := ConversationListItem{
			ID:                  conv.ID,
			ConversationNumber:  conversationNumber,
			Title:               conv.Title,
			Status:              conv.Status,
			Priority:            conv.Priority,
			HandleByBot:         conv.HandleByBot,
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
				AvatarURL:   conv.Client.Avatar,
				Initials:    initials,
				ExternalIDs: externalIDs,
				Language:    conv.Client.Language,
				Timezone:    conv.Client.Timezone,
				Data:        parseJSONToMap(conv.Client.Data),
			},
			AssignedAgents:  assignedAgents,
			Department:      department,
			Inbox:           inbox,
			Tags:            tags,
			MessageCount:    messageCount,
			HasAttachments:  false,
			IP:              conv.IP,
			Browser:         conv.Browser,
			OperatingSystem: conv.OperatingSystem,
			Data:            parseJSONToMap(conv.CustomFields),
		}

		// Set unread count if user is authenticated
		if userIDStr != "" {
			if parsedUserID, err := uuid.Parse(userIDStr); err == nil {
				conversation.UnreadMessagesCount = int(getUnreadCountForConversation(parsedUserID, conv.ID))
			}
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
	if req.Query("include_unread_count").String() == "true" && userIDStr != "" {
		if parsedUserID, err := uuid.Parse(userIDStr); err == nil {
			unreadCount := getTotalUnreadCount(parsedUserID)
			resp.UnreadCount = &unreadCount
		}
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
	if req.User().Anonymous() {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeUnauthorized, "Authentication required", 401, "No authenticated user"))
	}
	user := req.User().Interface().(*auth.User)
	userID := user.UserID

	unreadCount := getTotalUnreadCount(userID)
	return response.OK(map[string]int64{
		"unread_count": unreadCount,
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
	if req.User().Anonymous() {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeUnauthorized, "Authentication required", 401, "No authenticated user"))
	}
	user := req.User().Interface().(*auth.User)
	userID := user.UserID

	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	markedAt := time.Now()
	if err := markConversationAsRead(userID, conversationID); err != nil {
		log.Error("Failed to mark conversation as read:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to mark conversation as read", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"marked_read_at":  markedAt.Format(time.RFC3339),
	})
}
