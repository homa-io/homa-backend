package conversation

import (
	"fmt"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/imageutil"
	"github.com/iesreza/homa-backend/lib/response"
)

// AgentController handles agent-related API endpoints
type AgentController struct{}

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
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	userIDStr := ""
	if !req.User().Anonymous() {
		user := req.User().Interface().(*auth.User)
		userIDStr = user.UserID.String()
	}

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

	conversationNumber := fmt.Sprintf("CONV-%d", conv.ID)
	initials := getInitials(conv.Client.Name)

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

	tags := make([]TagInfo, 0, len(conv.Tags))
	for _, tag := range conv.Tags {
		tags = append(tags, TagInfo{
			ID:    tag.ID,
			Name:  tag.Name,
			Color: "#4ECDC4",
		})
	}

	var department *DepartmentInfo
	if conv.Department != nil {
		department = &DepartmentInfo{
			ID:        conv.Department.ID,
			Name:      conv.Department.Name,
			AIAgentID: conv.Department.AIAgentID,
		}
	}

	var lastMessage models.Message
	var hasLastMessage bool
	if err := db.Where("conversation_id = ?", conv.ID).
		Order("created_at DESC").
		First(&lastMessage).Error; err == nil {
		hasLastMessage = true
	}

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

	var messageCount int64
	db.Model(&models.Message{}).Where("conversation_id = ?", conv.ID).Count(&messageCount)

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

	conversationItem := ConversationListItem{
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
		Tags:            tags,
		MessageCount:    messageCount,
		HasAttachments:  false,
		IP:              conv.IP,
		Browser:         conv.Browser,
		OperatingSystem: conv.OperatingSystem,
		Data:            parseJSONToMap(conv.CustomFields),
	}

	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}
	limit := req.Query("limit").Int()
	if limit < 1 {
		limit = 1000
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := (page - 1) * limit

	order := req.Query("order").String()
	if order == "" {
		order = "asc"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}

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

	messageItems := make([]MessageItem, 0, len(messages))
	for _, msg := range messages {
		var authorID string
		var authorName string
		var authorType string
		var avatarURL *string

		isAgent := false
		if msg.UserID != nil {
			isAgent = true
			authorType = "agent"
			if msg.User != nil {
				authorID = msg.User.UserID.String()
				authorName = msg.User.DisplayName
				avatarURL = msg.User.Avatar
				if msg.User.Type == auth.UserTypeBot {
					authorType = "bot"
				}
			}
		} else if msg.ClientID != nil {
			authorType = "customer"
			if msg.Client != nil {
				authorID = msg.Client.ID.String()
				authorName = msg.Client.Name
				avatarURL = msg.Client.Avatar
			}
		} else {
			authorType = "system"
			authorID = "system"
			authorName = "System"
			avatarURL = nil
		}

		msgInitials := getInitials(authorName)

		msgType := msg.Type
		if msgType == "" {
			msgType = models.MessageTypeMessage
		}

		messageItem := MessageItem{
			ID:              msg.ID,
			Body:            msg.Body,
			Type:            msgType,
			Language:        msg.Language,
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

	totalPages := int(messageCount) / limit
	if int(messageCount)%limit != 0 {
		totalPages++
	}

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
	clientID := req.Param("client_id").String()
	if clientID == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "client_id is required", 400, "Client ID must be provided"))
	}

	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}

	limit := req.Query("limit").Int()
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	currentConversationID := req.Query("exclude_id").Uint()

	var conversations []models.Conversation
	query := db.Where("client_id = ?", clientID).
		Order("updated_at DESC")

	if currentConversationID > 0 {
		query = query.Where("id != ?", currentConversationID)
	}

	var total int64
	if err := query.Model(&models.Conversation{}).Count(&total).Error; err != nil {
		log.Error("Failed to count client conversations: ", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count conversations", 500, err.Error()))
	}

	if err := query.Limit(limit).Offset(offset).Find(&conversations).Error; err != nil {
		log.Error("Failed to fetch client conversations: ", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to fetch conversations", 500, err.Error()))
	}

	items := make([]PreviousConversationItem, 0, len(conversations))
	for _, conv := range conversations {
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
	HandleByBot  *bool   `json:"handle_by_bot"`
}

// UpdateConversationProperties handles the PATCH /api/agent/conversations/:id endpoint
func (ac AgentController) UpdateConversationProperties(req *evo.Request) interface{} {
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var updateReq UpdateConversationRequest
	if err := req.BodyParser(&updateReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	var conversation models.Conversation
	if err := db.Preload("Department").Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	getStatusDisplayName := func(status string) string {
		statusNames := map[string]string{
			"new":         "New",
			"user_reply":  "User Reply",
			"agent_reply": "Agent Reply",
			"processing":  "Processing",
			"closed":      "Closed",
			"archived":    "Archived",
			"postponed":   "Postponed",
		}
		if name, ok := statusNames[status]; ok {
			return name
		}
		return status
	}

	getPriorityDisplayName := func(priority string) string {
		priorityNames := map[string]string{
			"low":    "Low",
			"medium": "Medium",
			"high":   "High",
			"urgent": "Urgent",
		}
		if name, ok := priorityNames[priority]; ok {
			return name
		}
		return priority
	}

	oldStatus := conversation.Status
	oldPriority := conversation.Priority
	oldDepartmentID := conversation.DepartmentID
	oldDepartmentName := ""
	if conversation.Department != nil {
		oldDepartmentName = conversation.Department.Name
	}
	oldHandleByBot := conversation.HandleByBot

	var actorName string
	var userID *uuid.UUID
	if !req.User().Anonymous() {
		user := req.User().Interface().(*auth.User)
		actorName = user.Name
		if user.LastName != "" {
			actorName = user.Name + " " + user.LastName
		}
		userID = &user.UserID
	}

	updateData := make(map[string]interface{})

	if updateReq.Priority != nil {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[*updateReq.Priority] {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid priority", 400, "Priority must be one of: low, medium, high, urgent"))
		}
		updateData["priority"] = *updateReq.Priority
	}

	if updateReq.Status != nil {
		validStatuses := map[string]bool{
			"new": true, "user_reply": true, "agent_reply": true,
			"processing": true, "closed": true, "archived": true, "postponed": true,
		}
		if !validStatuses[*updateReq.Status] {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid status", 400, "Invalid status value"))
		}
		updateData["status"] = *updateReq.Status

		if *updateReq.Status == "closed" || *updateReq.Status == "archived" {
			now := time.Now()
			updateData["closed_at"] = &now
		}
	}

	if updateReq.DepartmentID != nil {
		updateData["department_id"] = *updateReq.DepartmentID
	}

	if updateReq.HandleByBot != nil {
		updateData["handle_by_bot"] = *updateReq.HandleByBot
	}

	if len(updateData) > 0 {
		if err := db.Model(&conversation).Updates(updateData).Error; err != nil {
			log.Error("Failed to update conversation: ", err)
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to update conversation", 500, err.Error()))
		}

		go func() {
			if updateReq.Status != nil && *updateReq.Status != oldStatus {
				action := fmt.Sprintf(`set conversation status to "%s"`, getStatusDisplayName(*updateReq.Status))
				models.CreateActionMessage(conversationID, userID, actorName, action)
			}

			if updateReq.Priority != nil && *updateReq.Priority != oldPriority {
				action := fmt.Sprintf(`set priority to "%s"`, getPriorityDisplayName(*updateReq.Priority))
				models.CreateActionMessage(conversationID, userID, actorName, action)
			}

			if updateReq.HandleByBot != nil && *updateReq.HandleByBot != oldHandleByBot {
				var action string
				if *updateReq.HandleByBot {
					action = "enabled bot handling"
				} else {
					action = "disabled bot handling"
				}
				models.CreateActionMessage(conversationID, userID, actorName, action)
			}

			if updateReq.DepartmentID != nil && (oldDepartmentID == nil || *updateReq.DepartmentID != *oldDepartmentID) {
				var newDept models.Department
				if err := db.Where("id = ?", *updateReq.DepartmentID).First(&newDept).Error; err == nil {
					var action string
					if oldDepartmentName != "" {
						action = fmt.Sprintf(`switched department from "%s" to "%s"`, oldDepartmentName, newDept.Name)
					} else {
						action = fmt.Sprintf(`set department to "%s"`, newDept.Name)
					}
					models.CreateActionMessage(conversationID, userID, actorName, action)
				}
			}
		}()
	}

	if err := db.Preload("Client").Preload("Department").Preload("Tags").First(&conversation, conversationID).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to reload conversation", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"id":            conversation.ID,
		"priority":      conversation.Priority,
		"status":        conversation.Status,
		"department_id": conversation.DepartmentID,
		"handle_by_bot": conversation.HandleByBot,
		"updated_at":    conversation.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

// GetMyDepartments handles the GET /api/agent/me/departments endpoint
// @Summary Get current user's departments
// @Description Get list of departments the current user belongs to
// @Tags Agent - Departments
// @Accept json
// @Produce json
// @Success 200 {object} []models.Department
// @Router /api/agent/me/departments [get]
func (ac AgentController) GetMyDepartments(req *evo.Request) interface{} {
	user := req.User().(*auth.User)
	if user.Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var userDepartments []models.UserDepartment
	if err := db.Where("user_id = ?", user.UserID).Find(&userDepartments).Error; err != nil {
		log.Error("Failed to get user departments:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get user departments", 500, err.Error()))
	}

	if len(userDepartments) == 0 {
		return response.OK([]models.Department{})
	}

	deptIDs := make([]uint, len(userDepartments))
	for i, ud := range userDepartments {
		deptIDs[i] = ud.DepartmentID
	}

	var departments []models.Department
	if err := db.Where("id IN ?", deptIDs).Find(&departments).Error; err != nil {
		log.Error("Failed to get departments:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get departments", 500, err.Error()))
	}

	return response.OK(departments)
}

// GetUsers handles the GET /api/agent/users endpoint
func (ac AgentController) GetUsers(req *evo.Request) interface{} {
	var users []auth.User
	query := db.Model(&auth.User{}).Select("id, name, last_name, display_name, email, avatar")

	if search := req.Query("search").String(); search != "" {
		query = query.Where(
			"name LIKE ? OR last_name LIKE ? OR display_name LIKE ? OR email LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%",
		)
	}

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

// UploadUserAvatar uploads and processes an avatar for the current user
func (c AgentController) UploadUserAvatar(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)

	var req struct {
		Data string `json:"data" validate:"required"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.Data == "" {
		return response.Error(response.ErrInvalidInput)
	}

	if user.Avatar != nil && *user.Avatar != "" {
		if err := imageutil.DeleteAvatar(*user.Avatar); err != nil {
			log.Warning("Failed to delete old avatar:", err)
		}
	}

	avatarURL, err := imageutil.ProcessAvatarFromBase64(req.Data, "users")
	if err != nil {
		log.Error("Failed to process avatar:", err)
		return response.Error(response.ErrInternalError)
	}

	if err := db.Model(&auth.User{}).Where("id = ?", user.UserID).Update("avatar", avatarURL).Error; err != nil {
		log.Error("Failed to update user avatar:", err)
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"avatar": avatarURL,
	})
}

// DeleteUserAvatar removes the avatar from the current user
func (c AgentController) DeleteUserAvatar(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)

	if user.Avatar != nil && *user.Avatar != "" {
		if err := imageutil.DeleteAvatar(*user.Avatar); err != nil {
			log.Warning("Failed to delete avatar file:", err)
		}
	}

	if err := db.Model(&auth.User{}).Where("id = ?", user.UserID).Update("avatar", nil).Error; err != nil {
		log.Error("Failed to update user:", err)
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message": "Avatar deleted successfully",
	})
}

// GetUserPreferences returns the current user's notification preferences
// @Summary Get user preferences
// @Description Get notification preferences for the current user
// @Tags User Preferences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserPreference
// @Router /api/agent/me/preferences [get]
func (c AgentController) GetUserPreferences(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	prefs, err := models.GetUserPreferences(user.UserID)
	if err != nil {
		log.Error("Failed to get user preferences:", err)
		return response.Error(response.ErrInternalError)
	}

	return response.OK(prefs)
}

// UpdateUserPreferences updates the current user's notification preferences
// @Summary Update user preferences
// @Description Update notification preferences for the current user
// @Tags User Preferences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]interface{} true "Preference updates"
// @Success 200 {object} models.UserPreference
// @Router /api/agent/me/preferences [put]
func (c AgentController) UpdateUserPreferences(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)

	var updates map[string]interface{}
	if err := request.BodyParser(&updates); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Validate notification_sound if provided
	if sound, ok := updates["notification_sound"].(string); ok {
		if !models.IsValidNotificationSound(sound) {
			return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid notification sound", 400))
		}
	}

	// Validate sound_volume if provided
	if volume, ok := updates["sound_volume"].(float64); ok {
		if volume < 0 || volume > 100 {
			return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Sound volume must be between 0 and 100", 400))
		}
	}

	// Filter allowed fields
	allowedFields := map[string]bool{
		"notification_sound":    true,
		"sound_volume":          true,
		"browser_notifications": true,
		"desktop_badge":         true,
	}
	filteredUpdates := make(map[string]interface{})
	for key, value := range updates {
		if allowedFields[key] {
			filteredUpdates[key] = value
		}
	}

	if len(filteredUpdates) == 0 {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "No valid fields to update", 400))
	}

	if err := models.UpdateUserPreferences(user.UserID, filteredUpdates); err != nil {
		log.Error("Failed to update user preferences:", err)
		return response.Error(response.ErrInternalError)
	}

	// Return updated preferences
	prefs, err := models.GetUserPreferences(user.UserID)
	if err != nil {
		log.Error("Failed to get updated preferences:", err)
		return response.Error(response.ErrInternalError)
	}

	return response.OK(prefs)
}

// GetNotificationSounds returns the list of available notification sounds
// @Summary Get available notification sounds
// @Description Get list of available notification sound options
// @Tags User Preferences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} string
// @Router /api/agent/notification-sounds [get]
func (c AgentController) GetNotificationSounds(request *evo.Request) any {
	return response.List(models.NotificationSounds, len(models.NotificationSounds))
}
