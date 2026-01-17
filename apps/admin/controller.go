package admin

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/pagination"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/imageutil"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/gorm"
)

type Controller struct {
}

// ========================
// TICKET MANAGEMENT APIs
// ========================

// GetUnreadTickets returns unread tickets for the admin
func (c Controller) GetUnreadTickets(request *evo.Request) any {
	var user = request.User().(*auth.User)

	// Get user departments
	var userDepartments []uint
	err := db.Model(&models.UserDepartment{}).
		Where("user_id = ?", user.UserID).
		Pluck("department_id", &userDepartments).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	var tickets []models.Conversation
	query := db.
		Preload("Client").
		Preload("Department").
		Preload("Channel").
		Preload("Tags").
		Preload("Messages").
		Where("status IN (?)", []string{models.ConversationStatusNew, models.ConversationStatusWaitForAgent})

	// Administrators can see all tickets, agents see tickets from their departments or assigned to them
	if user.Type == auth.UserTypeAgent {
		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			userDepartments, user.UserID,
		)
	}

	p, err := pagination.New(query, request, &tickets, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(tickets, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// GetUnreadTicketsCount returns count of unread tickets
func (c Controller) GetUnreadTicketsCount(request *evo.Request) any {
	var user = request.User().(*auth.User)

	// Get user departments
	var userDepartments []uint
	err := db.Model(&models.UserDepartment{}).
		Where("user_id = ?", user.UserID).
		Pluck("department_id", &userDepartments).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	var count int64
	query := db.Model(&models.Conversation{}).
		Where("status IN (?)", []string{models.ConversationStatusNew, models.ConversationStatusWaitForAgent})

	// Administrators can see all tickets, agents see tickets from their departments or assigned to them
	if user.Type == auth.UserTypeAgent {
		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			userDepartments, user.UserID,
		)
	}

	err = query.Count(&count).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]int64{"count": count})
}

// ListTickets returns paginated list of tickets with search and filtering
func (c Controller) ListTickets(request *evo.Request) any {
	var user = request.User().(*auth.User)

	// Get user departments for agents
	var userDepartments []uint
	if user.Type == auth.UserTypeAgent {
		err := db.Model(&models.UserDepartment{}).
			Where("user_id = ?", user.UserID).
			Pluck("department_id", &userDepartments).Error
		if err != nil {
			return response.Error(response.ErrInternalError)
		}
	}

	var tickets []models.Conversation
	query := db.
		Preload("Client").
		Preload("Client.ExternalIDs").
		Preload("Department").
		Preload("Channel").
		Preload("Tags").
		Preload("Assignments").
		Preload("Assignments.User").
		Preload("Assignments.Department")

	// Apply access control
	if user.Type == auth.UserTypeAgent {
		query = query.Where(
			"department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?)",
			userDepartments, user.UserID,
		)
	}

	// Search functionality
	search := request.Query("search").String()
	if search != "" {
		query = query.Where(
			"id = ? OR title LIKE ? OR id IN (SELECT conversation_id FROM messages WHERE body LIKE ?) OR "+
				"client_id IN (SELECT id FROM clients WHERE name LIKE ?) OR "+
				"client_id IN (SELECT client_id FROM client_external_ids WHERE value LIKE ?) OR "+
				"id IN (SELECT conversation_id FROM conversation_tags JOIN tags ON conversation_tags.tag_id = tags.id WHERE tags.name LIKE ?)",
			parseIntOrZero(search), "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%",
		)
	}

	// Filter by status
	if status := request.Query("status").String(); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by priority
	if priority := request.Query("priority").String(); priority != "" {
		query = query.Where("priority = ?", priority)
	}

	// Filter by department
	if departmentID := request.Query("department_id").String(); departmentID != "" {
		if deptID := parseIntOrZero(departmentID); deptID > 0 {
			query = query.Where("department_id = ?", deptID)
		}
	}

	// Filter by tag
	if tagName := request.Query("tag").String(); tagName != "" {
		query = query.Where("id IN (SELECT conversation_id FROM conversation_tags JOIN tags ON conversation_tags.tag_id = tags.id WHERE tags.name = ?)", tagName)
	}

	// Order by creation date with unread on top
	query = query.Order("CASE WHEN status IN ('new', 'wait_for_agent') THEN 0 ELSE 1 END, created_at DESC")

	p, err := pagination.New(query, request, &tickets, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(tickets, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// ChangeTicketStatus changes the status of a ticket
func (c Controller) ChangeTicketStatus(request *evo.Request) any {
	ticketID := parseIntOrZero(request.Param("id").String())
	if ticketID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=new wait_for_agent in_progress wait_for_user on_hold resolved closed unresolved spam"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var user = request.User().(*auth.User)

	// Check access to ticket
	if !c.hasTicketAccess(user, uint(ticketID)) {
		return response.Error(response.ErrForbidden)
	}

	var ticket models.Conversation
	err := db.First(&ticket, ticketID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	err = db.Model(&ticket).Update("status", req.Status).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Create system message for status change
	message := models.Message{
		ConversationID:  uint(ticketID),
		UserID:          &user.UserID,
		Body:            "Status changed to " + req.Status,
		IsSystemMessage: true,
	}
	db.Create(&message)

	return response.OK(map[string]interface{}{
		"message": "Ticket status updated successfully",
		"status":  req.Status,
	})
}

// ReplyToTicket adds a reply message to a ticket
func (c Controller) ReplyToTicket(request *evo.Request) any {
	ticketID := parseIntOrZero(request.Param("id").String())
	if ticketID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Message string `json:"message" validate:"required,min=1"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var user = request.User().(*auth.User)

	// Check access to ticket
	if !c.hasTicketAccess(user, uint(ticketID)) {
		return response.Error(response.ErrForbidden)
	}

	// Verify ticket exists
	var ticket models.Conversation
	err := db.First(&ticket, ticketID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Create the reply message
	message := models.Message{
		ConversationID:  uint(ticketID),
		UserID:          &user.UserID,
		Body:            req.Message,
		IsSystemMessage: false,
	}

	err = db.Create(&message).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Update ticket status to in_progress if it was new
	if ticket.Status == models.ConversationStatusNew || ticket.Status == models.ConversationStatusWaitForAgent {
		db.Model(&ticket).Update("status", models.ConversationStatusWaitForUser)
	}

	return response.Created(message)
}

// AssignTicket assigns a ticket to a user or department
func (c Controller) AssignTicket(request *evo.Request) any {
	ticketID := parseIntOrZero(request.Param("id").String())
	if ticketID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		UserID       *string `json:"user_id"`
		DepartmentID *uint   `json:"department_id"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.UserID == nil && req.DepartmentID == nil {
		return response.Error(response.ErrInvalidInput)
	}

	var user = request.User().(*auth.User)

	// Check access to ticket
	if !c.hasTicketAccess(user, uint(ticketID)) {
		return response.Error(response.ErrForbidden)
	}

	// Verify ticket exists
	var ticket models.Conversation
	err := db.First(&ticket, ticketID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Parse user ID if provided
	var userUUID *uuid.UUID
	if req.UserID != nil && *req.UserID != "" {
		parsedUUID, err := uuid.Parse(*req.UserID)
		if err != nil {
			return response.Error(response.ErrInvalidInput)
		}
		userUUID = &parsedUUID
	}

	// Remove existing assignments
	err = db.Where("conversation_id = ?", ticketID).Delete(&models.ConversationAssignment{}).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Create new assignment
	assignment := models.ConversationAssignment{
		ConversationID: uint(ticketID),
		UserID:         userUUID,
		DepartmentID:   req.DepartmentID,
	}

	err = db.Create(&assignment).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Create system message
	var assignmentMessage string
	if userUUID != nil {
		assignmentMessage = "Ticket assigned to user"
	} else {
		assignmentMessage = "Ticket assigned to department"
	}

	message := models.Message{
		ConversationID:  uint(ticketID),
		UserID:          &user.UserID,
		Body:            assignmentMessage,
		IsSystemMessage: true,
	}
	db.Create(&message)

	return response.OK(map[string]interface{}{
		"message":    "Ticket assigned successfully",
		"assignment": assignment,
	})
}

// ChangeTicketDepartments changes the department of a ticket
func (c Controller) ChangeTicketDepartments(request *evo.Request) any {
	ticketID := parseIntOrZero(request.Param("id").String())
	if ticketID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		DepartmentID *uint `json:"department_id"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var user = request.User().(*auth.User)

	// Check access to ticket
	if !c.hasTicketAccess(user, uint(ticketID)) {
		return response.Error(response.ErrForbidden)
	}

	// Verify ticket exists
	var ticket models.Conversation
	err := db.First(&ticket, ticketID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Update ticket department
	err = db.Model(&ticket).Update("department_id", req.DepartmentID).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Create system message
	message := models.Message{
		ConversationID:  uint(ticketID),
		UserID:          &user.UserID,
		Body:            "Ticket department changed",
		IsSystemMessage: true,
	}
	db.Create(&message)

	return response.OK(map[string]interface{}{
		"message":       "Ticket department updated successfully",
		"department_id": req.DepartmentID,
	})
}

// TagTicket adds or removes tags from a ticket
func (c Controller) TagTicket(request *evo.Request) any {
	ticketID := parseIntOrZero(request.Param("id").String())
	if ticketID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		TagIDs   []uint   `json:"tag_ids"`
		TagNames []string `json:"tag_names"`
		Action   string   `json:"action" validate:"required,oneof=add remove replace"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if len(req.TagIDs) == 0 && len(req.TagNames) == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var user = request.User().(*auth.User)

	// Check access to ticket
	if !c.hasTicketAccess(user, uint(ticketID)) {
		return response.Error(response.ErrForbidden)
	}

	// Verify ticket exists
	var ticket models.Conversation
	err := db.First(&ticket, ticketID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Get tag IDs from names if provided
	var allTagIDs []uint = req.TagIDs
	if len(req.TagNames) > 0 {
		var tags []models.Tag
		err = db.Where("name IN (?)", req.TagNames).Find(&tags).Error
		if err != nil {
			return response.Error(response.ErrInternalError)
		}
		for _, tag := range tags {
			allTagIDs = append(allTagIDs, tag.ID)
		}
	}

	// Handle different actions
	switch req.Action {
	case "replace":
		// Remove all existing tags and add new ones
		err = db.Where("conversation_id = ?", ticketID).Delete(&models.ConversationTag{}).Error
		if err != nil {
			return response.Error(response.ErrInternalError)
		}
		fallthrough
	case "add":
		// Add new tags
		for _, tagID := range allTagIDs {
			conversationTag := models.ConversationTag{
				ConversationID: uint(ticketID),
				TagID:          tagID,
			}
			db.FirstOrCreate(&conversationTag, conversationTag)
		}
	case "remove":
		// Remove specified tags
		if len(allTagIDs) > 0 {
			err = db.Where("conversation_id = ? AND tag_id IN (?)", ticketID, allTagIDs).Delete(&models.ConversationTag{}).Error
			if err != nil {
				return response.Error(response.ErrInternalError)
			}
		}
	}

	return response.OK(map[string]interface{}{
		"message": "Ticket tags updated successfully",
		"action":  req.Action,
		"tag_ids": allTagIDs,
	})
}

// DeleteTicket permanently deletes a ticket and all its related data
func (c Controller) DeleteTicket(request *evo.Request) any {
	ticketID := parseIntOrZero(request.Param("id").String())
	if ticketID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var user = request.User().(*auth.User)

	// Check access to ticket
	if !c.hasTicketAccess(user, uint(ticketID)) {
		return response.Error(response.ErrForbidden)
	}

	// Verify ticket exists
	var ticket models.Conversation
	err := db.First(&ticket, ticketID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Start transaction
	tx := db.Begin()

	// Delete all messages associated with the ticket
	err = tx.Where("conversation_id = ?", ticketID).Delete(&models.Message{}).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Delete all ticket tags
	err = tx.Where("conversation_id = ?", ticketID).Delete(&models.ConversationTag{}).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Delete all ticket assignments
	err = tx.Where("conversation_id = ?", ticketID).Delete(&models.ConversationAssignment{}).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Delete the ticket itself
	err = tx.Delete(&ticket).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Commit transaction
	tx.Commit()

	return response.OK(map[string]interface{}{
		"message":         "Ticket deleted successfully",
		"conversation_id": ticketID,
		"title":           ticket.Title,
	})
}

// DeleteMessage deletes a specific message
func (c Controller) DeleteMessage(request *evo.Request) any {
	messageID := parseIntOrZero(request.Param("id").String())
	if messageID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	// Fetch message with ticket information
	var message models.Message
	err := db.Preload("Ticket").First(&message, messageID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	var user = request.User().(*auth.User)

	// Check access to the ticket that contains this message
	if !c.hasTicketAccess(user, message.ConversationID) {
		return response.Error(response.ErrForbidden)
	}

	// Delete the message
	err = db.Delete(&message).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message":         "Message deleted successfully",
		"message_id":      messageID,
		"conversation_id": message.ConversationID,
	})
}

// ==========================
// DEPARTMENT MANAGEMENT APIs
// ==========================

// CreateDepartment creates a new department
func (c Controller) CreateDepartment(request *evo.Request) any {
	var req struct {
		Name        string   `json:"name" validate:"required,min=1,max=255"`
		Description string   `json:"description"`
		UserIDs     []string `json:"user_ids"`    // UUIDs of users to assign
		AIAgentID   *uint    `json:"ai_agent_id"` // AI Agent to assign (nullable)
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	department := models.Department{
		Name:        req.Name,
		Description: req.Description,
		Status:      models.DepartmentStatusActive,
		AIAgentID:   req.AIAgentID,
	}

	// Use transaction for creating department and assigning users
	tx := db.Begin()

	err := tx.Create(&department).Error
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			duplicateErr := response.NewError(response.ErrorCodeConflict, "Department name already exists", 409)
			return response.Error(duplicateErr)
		}
		return response.Error(response.ErrInternalError)
	}

	// Assign users if provided
	if len(req.UserIDs) > 0 {
		for _, userIDStr := range req.UserIDs {
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				continue // Skip invalid UUIDs
			}
			userDept := models.UserDepartment{
				UserID:       userID,
				DepartmentID: department.ID,
			}
			if err := tx.Create(&userDept).Error; err != nil {
				// Ignore duplicate entries
				if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "unique") {
					tx.Rollback()
					return response.Error(response.ErrInternalError)
				}
			}
		}
	}

	tx.Commit()

	// Reload with users and AI agent
	db.Preload("Users").Preload("AIAgent").First(&department, department.ID)

	return response.Created(department)
}

// EditDepartment updates an existing department
func (c Controller) EditDepartment(request *evo.Request) any {
	departmentID := parseIntOrZero(request.Param("id").String())
	if departmentID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Name        string   `json:"name" validate:"required,min=1,max=255"`
		Description string   `json:"description"`
		Status      string   `json:"status"`      // active, suspended
		UserIDs     []string `json:"user_ids"`    // UUIDs of users to assign (replaces existing)
		AIAgentID   *uint    `json:"ai_agent_id"` // AI Agent to assign (nullable)
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var department models.Department
	err := db.First(&department, departmentID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			notFoundErr := response.NewError(response.ErrorCodeNotFound, "Department not found", 404)
			return response.Error(notFoundErr)
		}
		return response.Error(response.ErrInternalError)
	}

	tx := db.Begin()

	// Update department fields
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"ai_agent_id": req.AIAgentID,
	}
	if req.Status != "" && (req.Status == models.DepartmentStatusActive || req.Status == models.DepartmentStatusSuspended) {
		updates["status"] = req.Status
	}

	err = tx.Model(&department).Updates(updates).Error
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			duplicateErr := response.NewError(response.ErrorCodeConflict, "Department name already exists", 409)
			return response.Error(duplicateErr)
		}
		return response.Error(response.ErrInternalError)
	}

	// Update user assignments if user_ids provided (replace all)
	if req.UserIDs != nil {
		// Remove all existing assignments
		if err := tx.Where("department_id = ?", departmentID).Delete(&models.UserDepartment{}).Error; err != nil {
			tx.Rollback()
			return response.Error(response.ErrInternalError)
		}

		// Add new assignments with priority based on array order
		for priority, userIDStr := range req.UserIDs {
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				continue // Skip invalid UUIDs
			}
			userDept := models.UserDepartment{
				UserID:       userID,
				DepartmentID: uint(departmentID),
				Priority:     priority,
			}
			if err := tx.Create(&userDept).Error; err != nil {
				// Ignore duplicate entries
				if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "unique") {
					tx.Rollback()
					return response.Error(response.ErrInternalError)
				}
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload with users and AI agent
	db.Preload("Users").Preload("AIAgent").First(&department, departmentID)

	return response.OK(department)
}

// SoftDeleteDepartment soft deletes a department (sets deleted_at)
func (c Controller) SoftDeleteDepartment(request *evo.Request) any {
	departmentID := parseIntOrZero(request.Param("id").String())
	if departmentID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var department models.Department
	err := db.First(&department, departmentID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	err = db.Delete(&department).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message": "Department deleted successfully",
		"id":      departmentID,
	})
}

// ListDepartments returns paginated list of departments with search
func (c Controller) ListDepartments(request *evo.Request) any {
	var departments []models.Department
	query := db.Model(&models.Department{}).Preload("Users").Preload("AIAgent")

	// Search functionality
	search := request.Query("search").String()
	if search != "" {
		query = query.Where("id = ? OR name LIKE ? OR description LIKE ?",
			parseIntOrZero(search), "%"+search+"%", "%"+search+"%")
	}

	// Filter by status
	status := request.Query("status").String()
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Order by
	orderBy := request.Query("order_by").String()
	switch orderBy {
	case "name":
		query = query.Order("name ASC")
	case "id":
		query = query.Order("id ASC")
	default:
		query = query.Order("created_at DESC")
	}

	p, err := pagination.New(query, request, &departments, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(departments, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// GetDepartment returns a single department by ID
func (c Controller) GetDepartment(request *evo.Request) any {
	departmentID := parseIntOrZero(request.Param("id").String())
	if departmentID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var department models.Department
	err := db.Preload("Users").Preload("AIAgent").First(&department, departmentID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			notFoundErr := response.NewError(response.ErrorCodeNotFound, "Department not found", 404)
			return response.Error(notFoundErr)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(department)
}

// SuspendDepartment suspends or activates a department
func (c Controller) SuspendDepartment(request *evo.Request) any {
	departmentID := parseIntOrZero(request.Param("id").String())
	if departmentID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Suspended bool `json:"suspended"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var department models.Department
	err := db.First(&department, departmentID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			notFoundErr := response.NewError(response.ErrorCodeNotFound, "Department not found", 404)
			return response.Error(notFoundErr)
		}
		return response.Error(response.ErrInternalError)
	}

	newStatus := models.DepartmentStatusActive
	if req.Suspended {
		newStatus = models.DepartmentStatusSuspended
	}

	err = db.Model(&department).Update("status", newStatus).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	department.Status = newStatus
	return response.OK(department)
}

// =====================
// TAG MANAGEMENT APIs
// =====================

// AddTag creates a new tag
func (c Controller) AddTag(request *evo.Request) any {
	var req struct {
		Name string `json:"name" validate:"required,min=1,max=100"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Normalize tag name (lowercase, trim spaces)
	tagName := strings.TrimSpace(strings.ToLower(req.Name))
	if tagName == "" {
		return response.Error(response.ErrInvalidInput)
	}

	tag := models.Tag{
		Name: tagName,
	}

	err := db.Create(&tag).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			duplicateErr := response.NewError(response.ErrorCodeConflict, "Tag already exists", 409)
			return response.Error(duplicateErr)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.Created(tag)
}

// DeleteTag deletes a tag and removes it from all tickets
func (c Controller) DeleteTag(request *evo.Request) any {
	tagID := parseIntOrZero(request.Param("id").String())
	if tagID == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	var tag models.Tag
	err := db.First(&tag, tagID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Start transaction to ensure data consistency
	tx := db.Begin()

	// Remove tag from all tickets
	err = tx.Where("tag_id = ?", tagID).Delete(&models.ConversationTag{}).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Delete the tag
	err = tx.Delete(&tag).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	tx.Commit()

	return response.OK(map[string]interface{}{
		"message": "Tag deleted successfully",
		"id":      tagID,
		"name":    tag.Name,
	})
}

// ========================
// USER MANAGEMENT APIs
// ========================

// CreateUser creates a new user (agent or administrator)
func (c Controller) CreateUser(request *evo.Request) any {
	var req struct {
		Name        string `json:"name" validate:"required,min=1,max=255"`
		LastName    string `json:"last_name" validate:"required,min=1,max=255"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email" validate:"required,email,max=255"`
		Password    string `json:"password" validate:"required,min=6"`
		Type        string `json:"type" validate:"required,oneof=agent administrator"`
		Avatar      string `json:"avatar"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Check if user already exists
	var existingUser auth.User
	err := db.Where("email = ?", req.Email).First(&existingUser).Error
	if err == nil {
		duplicateErr := response.NewError(response.ErrorCodeConflict, "User with this email already exists", 409)
		return response.Error(duplicateErr)
	}

	// Set display name if not provided
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name + " " + req.LastName
	}

	// Create new user
	user := auth.User{
		Name:        req.Name,
		LastName:    req.LastName,
		DisplayName: displayName,
		Email:       req.Email,
		Type:        req.Type,
	}

	if req.Avatar != "" {
		user.Avatar = &req.Avatar
	}

	// Set password
	if err := user.SetPassword(req.Password); err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Save user to database
	err = db.Create(&user).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			duplicateErr := response.NewError(response.ErrorCodeConflict, "User with this email already exists", 409)
			return response.Error(duplicateErr)
		}
		return response.Error(response.ErrInternalError)
	}

	// Remove password hash from response
	user.PasswordHash = nil

	return response.Created(user)
}

// EditUser updates an existing user
func (c Controller) EditUser(request *evo.Request) any {
	userIDStr := request.Param("id").String()
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Name        string  `json:"name" validate:"required,min=1,max=255"`
		LastName    string  `json:"last_name" validate:"required,min=1,max=255"`
		DisplayName string  `json:"display_name"`
		Email       string  `json:"email" validate:"required,email,max=255"`
		Password    *string `json:"password"`
		Type        string  `json:"type" validate:"required,oneof=agent administrator"`
		Avatar      *string `json:"avatar"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var user auth.User
	err = db.First(&user, "id = ?", userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Set display name if not provided
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name + " " + req.LastName
	}

	// Update user fields
	updates := map[string]interface{}{
		"name":         req.Name,
		"last_name":    req.LastName,
		"display_name": displayName,
		"email":        req.Email,
		"type":         req.Type,
	}

	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}

	// Update password if provided
	if req.Password != nil && *req.Password != "" {
		if err := user.SetPassword(*req.Password); err != nil {
			return response.Error(response.ErrInternalError)
		}
		updates["password_hash"] = user.PasswordHash
	}

	err = db.Model(&user).Updates(updates).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			duplicateErr := response.NewError(response.ErrorCodeConflict, "User with this email already exists", 409)
			return response.Error(duplicateErr)
		}
		return response.Error(response.ErrInternalError)
	}

	// Fetch updated user
	err = db.First(&user, "id = ?", userID).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Remove password hash from response
	user.PasswordHash = nil

	return response.OK(user)
}

// UploadAvatar handles avatar file upload with automatic resizing
func (c Controller) UploadAvatar(request *evo.Request) any {
	var req struct {
		Data string `json:"data" validate:"required"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.Data == "" {
		return response.Error(response.ErrInvalidInput)
	}

	// Process and save avatar using imageutil (automatically resizes to 64x64)
	avatarURL, err := imageutil.ProcessAvatarFromBase64(req.Data, "users")
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Return the URL path
	return response.OK(map[string]string{
		"url": avatarURL,
	})
}

// ListUsers returns paginated list of users with search and filtering
func (c Controller) ListUsers(request *evo.Request) any {
	var users []auth.User
	query := db.Model(&auth.User{}).Select("id, name, last_name, display_name, email, type, status, avatar, created_at, updated_at")

	// Search functionality
	search := request.Query("search").String()
	if search != "" {
		query = query.Where(
			"name LIKE ? OR last_name LIKE ? OR display_name LIKE ? OR email LIKE ? OR type LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%",
		)
	}

	// Filter by type
	if userType := request.Query("type").String(); userType != "" {
		query = query.Where("type = ?", userType)
	}

	// Filter by status
	if status := request.Query("status").String(); status != "" {
		query = query.Where("status = ?", status)
	}

	// Order by
	orderBy := request.Query("order_by").String()
	switch orderBy {
	case "name":
		query = query.Order("name ASC")
	case "last_name":
		query = query.Order("last_name ASC")
	case "display_name":
		query = query.Order("display_name ASC")
	case "email":
		query = query.Order("email ASC")
	default:
		query = query.Order("created_at DESC")
	}

	p, err := pagination.New(query, request, &users, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Return in format expected by frontend: { users: [...], total: N, page: N, ... }
	return response.OK(map[string]interface{}{
		"users":       users,
		"total":       p.Records,
		"page":        p.CurrentPage,
		"page_size":   p.Size,
		"total_pages": p.Pages,
	})
}

// AssignUserToDepartment assigns a user to one or more departments
func (c Controller) AssignUserToDepartment(request *evo.Request) any {
	userIDStr := request.Param("id").String()
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		DepartmentIDs []uint `json:"department_ids" validate:"required,min=1"`
		Action        string `json:"action" validate:"required,oneof=add remove replace"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Verify user exists
	var user auth.User
	err = db.First(&user, "id = ?", userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Verify departments exist
	var existingDepartments []models.Department
	err = db.Where("id IN (?)", req.DepartmentIDs).Find(&existingDepartments).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	if len(existingDepartments) != len(req.DepartmentIDs) {
		return response.Error(response.ErrInvalidInput)
	}

	// Handle different actions
	switch req.Action {
	case "replace":
		// Remove all existing department assignments
		err = db.Where("user_id = ?", userID).Delete(&models.UserDepartment{}).Error
		if err != nil {
			return response.Error(response.ErrInternalError)
		}
		fallthrough
	case "add":
		// Add new department assignments
		for _, deptID := range req.DepartmentIDs {
			userDept := models.UserDepartment{
				UserID:       userID,
				DepartmentID: deptID,
			}
			db.FirstOrCreate(&userDept, userDept)
		}
	case "remove":
		// Remove specified department assignments
		err = db.Where("user_id = ? AND department_id IN (?)", userID, req.DepartmentIDs).Delete(&models.UserDepartment{}).Error
		if err != nil {
			return response.Error(response.ErrInternalError)
		}
	}

	return response.OK(map[string]interface{}{
		"message":        "User department assignments updated successfully",
		"action":         req.Action,
		"department_ids": req.DepartmentIDs,
	})
}

// BlockUser blocks or unblocks a user's access to the platform
func (c Controller) BlockUser(request *evo.Request) any {
	userIDStr := request.Param("id").String()
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Blocked bool `json:"blocked"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Verify user exists
	var user auth.User
	err = db.First(&user, "id = ?", userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrConversationNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// For now, we'll implement blocking by invalidating the API key
	// In a full implementation, you might add a 'blocked' field to the User model
	if req.Blocked {
		// Clear API key to block access
		err = db.Model(&user).Update("api_key", nil).Error
	} else {
		// Generate new API key to unblock (if they had one)
		if user.APIKey != nil {
			apiKey, err := user.GenerateAPIKey()
			if err != nil {
				return response.Error(response.ErrInternalError)
			}
			err = db.Model(&user).Update("api_key", apiKey).Error
		}
	}

	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	action := "unblocked"
	if req.Blocked {
		action = "blocked"
	}

	return response.OK(map[string]interface{}{
		"message": "User " + action + " successfully",
		"user_id": userID,
		"blocked": req.Blocked,
	})
}

// ===============================
// CUSTOM ATTRIBUTE MANAGEMENT APIs
// ===============================

// CreateCustomAttribute creates a new custom attribute
func (c Controller) CreateCustomAttribute(request *evo.Request) any {
	var req struct {
		Scope       string  `json:"scope" validate:"required,oneof=client conversation"`
		Name        string  `json:"name" validate:"required,min=1,max=100"`
		DataType    string  `json:"data_type" validate:"required,oneof=int float date string"`
		Validation  *string `json:"validation"`
		Title       string  `json:"title" validate:"required,min=1,max=255"`
		Description *string `json:"description"`
		Visibility  string  `json:"visibility" validate:"required,oneof=everyone administrator hidden"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Normalize name (lowercase, replace spaces with underscores)
	name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Name), " ", "_"))

	// Validate name format (only lowercase letters and underscores)
	if !isValidAttributeName(name) {
		return response.Error(response.ErrInvalidInput)
	}

	customAttr := models.CustomAttribute{
		Scope:       req.Scope,
		Name:        name,
		DataType:    req.DataType,
		Validation:  req.Validation,
		Title:       req.Title,
		Description: req.Description,
		Visibility:  req.Visibility,
	}

	err := db.Create(&customAttr).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			duplicateErr := response.NewError(response.ErrorCodeConflict, "Custom attribute with this scope and name already exists", 409)
			return response.Error(duplicateErr)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.Created(customAttr)
}

// EditCustomAttribute updates an existing custom attribute
func (c Controller) EditCustomAttribute(request *evo.Request) any {
	scope := request.Param("scope").String()
	name := request.Param("name").String()

	if scope == "" || name == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		DataType    string  `json:"data_type" validate:"required,oneof=int float date string"`
		Validation  *string `json:"validation"`
		Title       string  `json:"title" validate:"required,min=1,max=255"`
		Description *string `json:"description"`
		Visibility  string  `json:"visibility" validate:"required,oneof=everyone administrator hidden"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var customAttr models.CustomAttribute
	err := db.Where("scope = ? AND name = ?", scope, name).First(&customAttr).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	err = db.Model(&customAttr).Updates(models.CustomAttribute{
		DataType:    req.DataType,
		Validation:  req.Validation,
		Title:       req.Title,
		Description: req.Description,
		Visibility:  req.Visibility,
	}).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(customAttr)
}

// DeleteCustomAttribute deletes a custom attribute
func (c Controller) DeleteCustomAttribute(request *evo.Request) any {
	scope := request.Param("scope").String()
	name := request.Param("name").String()

	if scope == "" || name == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var customAttr models.CustomAttribute
	err := db.Where("scope = ? AND name = ?", scope, name).First(&customAttr).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	err = db.Delete(&customAttr).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message": "Custom attribute deleted successfully",
		"scope":   scope,
		"name":    name,
	})
}

// ListCustomAttributes returns paginated list of custom attributes with search and filtering
func (c Controller) ListCustomAttributes(request *evo.Request) any {
	var customAttrs []models.CustomAttribute
	query := db.Model(&models.CustomAttribute{})

	// Search functionality
	search := request.Query("search").String()
	if search != "" {
		query = query.Where("name LIKE ? OR title LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Filter by scope
	if scope := request.Query("scope").String(); scope != "" {
		query = query.Where("scope = ?", scope)
	}

	// Filter by data type
	if dataType := request.Query("data_type").String(); dataType != "" {
		query = query.Where("data_type = ?", dataType)
	}

	// Filter by visibility
	if visibility := request.Query("visibility").String(); visibility != "" {
		query = query.Where("visibility = ?", visibility)
	}

	// Order by
	orderBy := request.Query("order_by").String()
	switch orderBy {
	case "name":
		query = query.Order("name ASC")
	case "title":
		query = query.Order("title ASC")
	case "scope":
		query = query.Order("scope ASC, name ASC")
	default:
		query = query.Order("created_at DESC")
	}

	p, err := pagination.New(query, request, &customAttrs, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(customAttrs, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// ========================
// CHANNEL MANAGEMENT APIs
// ========================

// CreateChannel creates a new channel
func (c Controller) CreateChannel(request *evo.Request) any {
	var req struct {
		ID            string                 `json:"id" validate:"required,min=1,max=50"`
		Name          string                 `json:"name" validate:"required,min=1,max=255"`
		Logo          *string                `json:"logo"`
		Configuration map[string]interface{} `json:"configuration"`
		Enabled       *bool                  `json:"enabled"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Normalize ID (lowercase, no spaces)
	channelID := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.ID), " ", "_"))

	// Convert configuration to JSON
	var configJSON []byte
	if req.Configuration != nil {
		var err error
		configJSON, err = json.Marshal(req.Configuration)
		if err != nil {
			return response.Error(response.ErrInvalidInput)
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	channel := models.Channel{
		ID:      channelID,
		Name:    req.Name,
		Logo:    req.Logo,
		Enabled: enabled,
	}

	if configJSON != nil {
		channel.Configuration = configJSON
	}

	err := db.Create(&channel).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			duplicateErr := response.NewError(response.ErrorCodeConflict, "Channel with this ID already exists", 409)
			return response.Error(duplicateErr)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.Created(channel)
}

// EditChannel updates an existing channel
func (c Controller) EditChannel(request *evo.Request) any {
	channelID := request.Param("id").String()
	if channelID == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Name          string                 `json:"name" validate:"required,min=1,max=255"`
		Logo          *string                `json:"logo"`
		Configuration map[string]interface{} `json:"configuration"`
		Enabled       *bool                  `json:"enabled"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var channel models.Channel
	err := db.First(&channel, "id = ?", channelID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Prepare updates
	updates := map[string]interface{}{
		"name": req.Name,
	}

	if req.Logo != nil {
		updates["logo"] = req.Logo
	}

	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if req.Configuration != nil {
		configJSON, err := json.Marshal(req.Configuration)
		if err != nil {
			return response.Error(response.ErrInvalidInput)
		}
		updates["configuration"] = configJSON
	}

	err = db.Model(&channel).Updates(updates).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Fetch updated channel
	err = db.First(&channel, "id = ?", channelID).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(channel)
}

// DeleteChannel deletes a channel
func (c Controller) DeleteChannel(request *evo.Request) any {
	channelID := request.Param("id").String()
	if channelID == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var channel models.Channel
	err := db.First(&channel, "id = ?", channelID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Check if channel is being used by tickets
	var ticketCount int64
	err = db.Model(&models.Conversation{}).Where("channel_id = ?", channelID).Count(&ticketCount).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	if ticketCount > 0 {
		conflictErr := response.NewError(response.ErrorCodeConflict, "Cannot delete channel that is being used by tickets", 409)
		return response.Error(conflictErr)
	}

	err = db.Delete(&channel).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message": "Channel deleted successfully",
		"id":      channelID,
	})
}

// ListChannels returns paginated list of channels with search and filtering
func (c Controller) ListChannels(request *evo.Request) any {
	var channels []models.Channel
	query := db.Model(&models.Channel{})

	// Search functionality
	search := request.Query("search").String()
	if search != "" {
		query = query.Where("id LIKE ? OR name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Filter by enabled status
	if enabled := request.Query("enabled").String(); enabled != "" {
		if enabled == "true" {
			query = query.Where("enabled = ?", true)
		} else if enabled == "false" {
			query = query.Where("enabled = ?", false)
		}
	}

	// Order by
	orderBy := request.Query("order_by").String()
	switch orderBy {
	case "name":
		query = query.Order("name ASC")
	case "id":
		query = query.Order("id ASC")
	case "enabled":
		query = query.Order("enabled DESC, name ASC")
	default:
		query = query.Order("created_at DESC")
	}

	p, err := pagination.New(query, request, &channels, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(channels, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// Helper functions

func parseIntOrZero(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

func (c Controller) hasTicketAccess(user *auth.User, ticketID uint) bool {
	if user.Type == auth.UserTypeAdministrator {
		return true
	}

	// For agents, check if ticket is in their departments or assigned to them
	var userDepartments []uint
	err := db.Model(&models.UserDepartment{}).
		Where("user_id = ?", user.UserID).
		Pluck("department_id", &userDepartments).Error
	if err != nil {
		return false
	}

	var count int64
	err = db.Model(&models.Conversation{}).
		Where("id = ? AND (department_id IN (?) OR id IN (SELECT conversation_id FROM conversation_assignments WHERE user_id = ?))",
			ticketID, userDepartments, user.UserID).
		Count(&count).Error
	if err != nil {
		return false
	}

	return count > 0
}

// Helper function to validate attribute name
func isValidAttributeName(name string) bool {
	if name == "" {
		return false
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || char == '_') {
			return false
		}
	}
	return true
}
