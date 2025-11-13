package agent

import (
	"strconv"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
	"github.com/google/uuid"
)

type Controller struct{}

// checkAgentAuth validates user authentication and authorization
func (c *Controller) checkAgentAuth(user *auth.User) *response.AppError {
	if user.Anonymous() {
		err := response.ErrUnauthorized
		return &err
	}

	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		err := response.ErrForbidden
		return &err
	}

	return nil
}

// GetUserDepartmentIDs helper function to get user's department IDs
func (c *Controller) GetUserDepartmentIDs(userID uuid.UUID) ([]uint, error) {
	var userDepts []models.UserDepartment
	err := db.Where("user_id = ?", userID).Find(&userDepts).Error
	if err != nil {
		return nil, err
	}

	departmentIDs := make([]uint, len(userDepts))
	for i, dept := range userDepts {
		departmentIDs[i] = dept.DepartmentID
	}
	return departmentIDs, nil
}

// GetUnreadTickets returns unread tickets for the agent
func (c *Controller) GetUnreadTickets(request *evo.Request) interface{} {
	// TEMPORARY: Auth disabled for testing
	/*
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}
	*/

	// Mock user for testing
	var userID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// Get user's department IDs
	departmentIDs, err := c.GetUserDepartmentIDs(userID)
	if err != nil {
		return response.Error(response.ErrUserDepartments())
	}

	var tickets []models.Conversation
	query := db.Where("status IN (?, ?, ?)", models.ConversationStatusNew, models.ConversationStatusWaitForAgent, models.ConversationStatusInProgress)

	// For agents, show tickets assigned to them or their departments
	query = query.Where(
		"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
		departmentIDs, userID,
	)

	err = query.Preload("Client").Preload("Department").Preload("Channel").Find(&tickets).Error
	if err != nil {
		return response.Error(response.ErrFetchConversations())
	}

	return response.List(tickets, len(tickets))
}

// GetUnreadTicketsCount returns count of unread tickets for the agent
func (c *Controller) GetUnreadTicketsCount(request *evo.Request) interface{} {
	// TEMPORARY: Auth disabled for testing
	/*
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}
	*/

	// Mock user for testing
	var userID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// Get user's department IDs
	departmentIDs, err := c.GetUserDepartmentIDs(userID)
	if err != nil {
		return response.Error(response.ErrUserDepartments())
	}

	var count int64
	query := db.Model(&models.Conversation{}).Where("status IN (?, ?, ?)", models.ConversationStatusNew, models.ConversationStatusWaitForAgent, models.ConversationStatusInProgress)

	// For agents, count tickets assigned to them or their departments
	query = query.Where(
		"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
		departmentIDs, userID,
	)

	err = query.Count(&count).Error
	if err != nil {
		return response.Error(response.ErrCountConversations())
	}

	countData := map[string]interface{}{
		"count": count,
	}
	return response.OK(countData)
}

// GetTicketList returns paginated list of tickets accessible to the agent with advanced filtering and search
func (c *Controller) GetTicketList(request *evo.Request) interface{} {
	// TEMPORARY: Auth disabled for testing
	/*
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}
	*/

	// TEMPORARY: Skip auth and department check for testing
	/*
	// Mock user for testing - Agent type
	var user = &auth.User{
		UserID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Type:   auth.UserTypeAgent,
	}

	// Get user's department IDs
	_, err := c.GetUserDepartmentIDs(user.UserID)
	if err != nil {
		return response.Error(response.ErrUserDepartments())
	}
	*/
	var err error

	// Build base query with proper joins for search
	query := db.Model(&models.Conversation{}).
		Select("conversations.*, clients.name as client_name").
		Joins("LEFT JOIN clients ON conversations.client_id = clients.id").
		Joins("LEFT JOIN client_external_ids ON clients.id = client_external_ids.client_id AND client_external_ids.type = 'email'").
		Joins("LEFT JOIN conversation_tags ON conversations.id = conversation_tags.conversation_id").
		Joins("LEFT JOIN tags ON conversation_tags.tag_id = tags.id")

	// TEMPORARY: Show all conversations for testing
	// if user.Type == auth.UserTypeAgent {
	// 	// For agents, show tickets assigned to them or their departments
	// 	query = query.Where(
	// 		"conversations.department_id IN (?) OR conversations.id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
	// 		departmentIDs, user.UserID,
	// 	)
	// }

	// Apply search filters
	if search := request.Query("search").String(); search != "" {
		query = query.Where(
			"clients.name LIKE ? OR client_external_ids.value LIKE ? OR CAST(conversations.id AS CHAR) LIKE ? OR tags.name LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%",
		)
	}

	// Filter by client name
	if clientName := request.Query("client_name").String(); clientName != "" {
		query = query.Where("clients.name LIKE ?", "%"+clientName+"%")
	}

	// Filter by client email
	if clientEmail := request.Query("client_email").String(); clientEmail != "" {
		query = query.Where("client_external_ids.value LIKE ? AND client_external_ids.type = 'email'", "%"+clientEmail+"%")
	}

	// Filter by ticket ID
	if ticketID := request.Query("ticket_id").String(); ticketID != "" {
		query = query.Where("CAST(conversations.id AS CHAR) LIKE ?", "%"+ticketID+"%")
	}

	// Filter by tag name
	if tagName := request.Query("tag").String(); tagName != "" {
		query = query.Where("tags.name LIKE ?", "%"+tagName+"%")
	}

	// Filter by status
	if status := request.Query("status").String(); status != "" {
		query = query.Where("conversations.status = ?", status)
	}

	// Remove duplicates from joins
	query = query.Group("conversations.id, clients.name")

	// Get total count before applying pagination
	var total int64
	countQuery := query
	countQuery.Count(&total)

	// Apply ordering - unread tickets first, then by date
	unreadStatuses := []string{models.ConversationStatusNew, models.ConversationStatusWaitForAgent, models.ConversationStatusInProgress}
	query = query.Order(
		"CASE WHEN conversations.status IN ('" + unreadStatuses[0] + "','" + unreadStatuses[1] + "','" + unreadStatuses[2] + "') THEN 0 ELSE 1 END, conversations.created_at DESC",
	)

	// Apply pagination
	page := request.Query("page").Int()
	if page <= 0 {
		page = 1
	}
	limit := request.Query("limit").Int()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var tickets []models.Conversation

	// Get paginated results with all necessary preloads
	err = query.Preload("Client").
		Preload("Client.ExternalIDs").
		Preload("Department").
		Preload("Channel").
		Preload("Tags").
		Preload("Assignments").
		Preload("Assignments.User").
		Preload("Assignments.Department").
		Preload("Messages").
		Preload("Messages.Client").
		Preload("Messages.User").
		Offset(offset).
		Limit(limit).
		Find(&tickets).Error

	if err != nil {
		return response.Error(response.ErrFetchConversations())
	}

	return response.Paginated(tickets, page, limit, total)
}

// ChangeTicketStatus changes the status of a ticket
func (c *Controller) ChangeTicketStatus(request *evo.Request) interface{} {
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}

	ticketID, err := strconv.ParseUint(request.Param("id").String(), 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	var requestData struct {
		Status string `json:"status" validate:"required,oneof=new open pending resolved closed"`
	}

	if err := request.BodyParser(&requestData); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Check if user has access to this ticket
	var ticket models.Conversation
	query := db.Where("id = ?", ticketID)

	if user.Type == auth.UserTypeAgent {
		departmentIDs, err := c.GetUserDepartmentIDs(user.UserID)
		if err != nil {
			return response.Error(response.ErrUserDepartments())
		}

		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			departmentIDs, user.UserID,
		)
	}

	err = query.First(&ticket).Error
	if err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Update ticket status
	err = db.Model(&ticket).Update("status", requestData.Status).Error
	if err != nil {
		return response.Error(response.ErrUpdateConversationStatus())
	}

	return response.OKWithMessage(ticket, "Ticket status updated successfully")
}

// ReplyToTicket adds a message reply to a ticket
func (c *Controller) ReplyToTicket(request *evo.Request) interface{} {
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}

	ticketID, err := strconv.ParseUint(request.Param("id").String(), 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	var requestData struct {
		Message string `json:"message" validate:"required,min=1"`
	}

	if err := request.BodyParser(&requestData); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Check if user has access to this ticket
	var ticket models.Conversation
	query := db.Where("id = ?", ticketID)

	if user.Type == auth.UserTypeAgent {
		departmentIDs, err := c.GetUserDepartmentIDs(user.UserID)
		if err != nil {
			return response.Error(response.ErrUserDepartments())
		}

		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			departmentIDs, user.UserID,
		)
	}

	err = query.First(&ticket).Error
	if err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Create message
	message := models.Message{
		ConversationID:  uint(ticketID),
		UserID:    &user.UserID,
		Body:      requestData.Message,
		CreatedAt: time.Now(),
	}

	err = db.Create(&message).Error
	if err != nil {
		return response.Error(response.ErrCreateMessage())
	}

	return response.Created(message)
}

// AssignTicket assigns a ticket to another user or department
func (c *Controller) AssignTicket(request *evo.Request) interface{} {
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}

	ticketID, err := strconv.ParseUint(request.Param("id").String(), 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	var requestData struct {
		UserID       *string `json:"user_id"`
		DepartmentID *uint   `json:"department_id"`
	}

	if err := request.BodyParser(&requestData); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if requestData.UserID == nil && requestData.DepartmentID == nil {
		return response.Error(response.ErrMissingUserOrDepartment())
	}

	// Check if user has access to this ticket
	var ticket models.Conversation
	query := db.Where("id = ?", ticketID)

	if user.Type == auth.UserTypeAgent {
		departmentIDs, err := c.GetUserDepartmentIDs(user.UserID)
		if err != nil {
			return response.Error(response.ErrUserDepartments())
		}

		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			departmentIDs, user.UserID,
		)
	}

	err = query.First(&ticket).Error
	if err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Remove existing assignments
	db.Where("conversation_id = ?", ticketID).Delete(&models.ConversationAssignment{})

	// Create new assignment
	assignment := models.ConversationAssignment{
		ConversationID: uint(ticketID),
		DepartmentID:   requestData.DepartmentID,
	}

	if requestData.UserID != nil {
		userUUID, err := uuid.Parse(*requestData.UserID)
		if err != nil {
			return response.Error(response.ErrInvalidUserID)
		}
		assignment.UserID = &userUUID
	}

	err = db.Create(&assignment).Error
	if err != nil {
		return response.Error(response.ErrAssignConversation())
	}

	return response.Created(assignment)
}

// ChangeTicketDepartment changes the department of a ticket
func (c *Controller) ChangeTicketDepartment(request *evo.Request) interface{} {
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}

	ticketID, err := strconv.ParseUint(request.Param("id").String(), 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	var requestData struct {
		DepartmentID uint `json:"department_id" validate:"required"`
	}

	if err := request.BodyParser(&requestData); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Check if user has access to this ticket
	var ticket models.Conversation
	query := db.Where("id = ?", ticketID)

	if user.Type == auth.UserTypeAgent {
		departmentIDs, err := c.GetUserDepartmentIDs(user.UserID)
		if err != nil {
			return response.Error(response.ErrUserDepartments())
		}

		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			departmentIDs, user.UserID,
		)
	}

	err = query.First(&ticket).Error
	if err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Update ticket department
	err = db.Model(&ticket).Update("department_id", requestData.DepartmentID).Error
	if err != nil {
		return response.Error(response.ErrUpdateConversationDepartment())
	}

	return response.OKWithMessage(ticket, "Ticket department updated successfully")
}

// TagTicket adds/removes tags from a ticket
func (c *Controller) TagTicket(request *evo.Request) interface{} {
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}

	ticketID, err := strconv.ParseUint(request.Param("id").String(), 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	var requestData struct {
		TagIDs   []uint   `json:"tag_ids"`
		TagNames []string `json:"tag_names"`
	}

	if err := request.BodyParser(&requestData); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Validate that at least one of tag_ids or tag_names is provided
	if len(requestData.TagIDs) == 0 && len(requestData.TagNames) == 0 {
		return response.Error(response.ErrMissingTagsOrIDs())
	}

	// Check if user has access to this ticket
	var ticket models.Conversation
	query := db.Where("id = ?", ticketID)

	if user.Type == auth.UserTypeAgent {
		departmentIDs, err := c.GetUserDepartmentIDs(user.UserID)
		if err != nil {
			return response.Error(response.ErrUserDepartments())
		}

		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			departmentIDs, user.UserID,
		)
	}

	err = query.First(&ticket).Error
	if err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Remove existing tags
	db.Where("conversation_id = ?", ticketID).Delete(&models.ConversationTag{})

	// Collect all tag IDs to be assigned
	var allTagIDs []uint
	var createdTags []models.Tag

	// Add existing tag IDs
	allTagIDs = append(allTagIDs, requestData.TagIDs...)

	// Process tag names - create tags if they don't exist
	for _, tagName := range requestData.TagNames {
		var tag models.Tag

		// Try to find existing tag by name
		err := db.Where("name = ?", tagName).First(&tag).Error
		if err != nil {
			// Tag doesn't exist, create it
			tag = models.Tag{
				Name: tagName,
			}
			err = db.Create(&tag).Error
			if err != nil {
				return response.Error(response.ErrCreateTagWithName(tagName))
			}
			createdTags = append(createdTags, tag)
		}

		allTagIDs = append(allTagIDs, tag.ID)
	}

	// Create ticket-tag associations
	for _, tagID := range allTagIDs {
		conversationTag := models.ConversationTag{
			ConversationID: uint(ticketID),
			TagID:          tagID,
		}
		db.Create(&conversationTag)
	}

	responseData := map[string]interface{}{
		"total_tags": len(allTagIDs),
	}

	// Include created tags in response if any were created
	if len(createdTags) > 0 {
		responseData["created_tags"] = createdTags
	}

	return response.OKWithMessage(responseData, "Ticket tags updated successfully")
}

// AddTag creates a new tag
func (c *Controller) AddTag(request *evo.Request) interface{} {
	// Check if user is logged in first
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	var user = request.User().Interface().(*auth.User)

	// Check authentication and authorization
	if authErr := c.checkAgentAuth(user); authErr != nil {
		return response.Error(*authErr)
	}

	var requestData struct {
		Name string `json:"name" validate:"required,min=1,max=100"`
	}

	if err := request.BodyParser(&requestData); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Create tag
	tag := models.Tag{
		Name: requestData.Name,
	}

	err := db.Create(&tag).Error
	if err != nil {
		return response.Error(response.ErrCreateTag())
	}

	return response.Created(tag)
}
