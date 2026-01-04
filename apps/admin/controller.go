package admin

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/getevo/pagination"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/ai"
	"github.com/iesreza/homa-backend/apps/auth"
	integrationsDriver "github.com/iesreza/homa-backend/apps/integrations"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/imageutil"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/datatypes"
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

// ========================
// CLIENT MANAGEMENT APIs
// ========================

// ListClients returns paginated list of clients with search and filtering
func (c Controller) ListClients(request *evo.Request) any {
	var clients []models.Client
	query := db.
		Preload("ExternalIDs").
		Preload("Conversations")

	// Search functionality
	search := request.Query("search").String()
	if search != "" {
		query = query.Where(
			"name LIKE ? OR id IN (SELECT client_id FROM client_external_ids WHERE value LIKE ?)",
			"%"+search+"%", "%"+search+"%",
		)
	}

	// Search by custom attributes in data field
	if attrSearch := request.Query("attr_search").String(); attrSearch != "" {
		// Parse as JSON to search within the data field
		query = query.Where("JSON_SEARCH(data, 'one', ?) IS NOT NULL", "%"+attrSearch+"%")
	}

	// Filter by external ID type
	if externalType := request.Query("external_type").String(); externalType != "" {
		query = query.Where("id IN (SELECT client_id FROM client_external_ids WHERE type = ?)", externalType)
	}

	// Order by
	orderBy := request.Query("order_by").String()
	switch orderBy {
	case "name":
		query = query.Order("name ASC")
	case "created_at":
		query = query.Order("created_at DESC")
	case "updated_at":
		query = query.Order("updated_at DESC")
	default:
		query = query.Order("created_at DESC")
	}

	p, err := pagination.New(query, request, &clients, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(clients, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// MergeClients merges multiple clients into one, combining their data and reassigning tickets/messages
func (c Controller) MergeClients(request *evo.Request) any {
	var req struct {
		TargetClientID  uuid.UUID   `json:"target_client_id" validate:"required"`
		SourceClientIDs []uuid.UUID `json:"source_client_ids" validate:"required,min=1"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Verify target client exists
	var targetClient models.Client
	err := db.First(&targetClient, "id = ?", req.TargetClientID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Verify all source clients exist
	var sourceClients []models.Client
	err = db.Where("id IN (?)", req.SourceClientIDs).Find(&sourceClients).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	if len(sourceClients) != len(req.SourceClientIDs) {
		return response.Error(response.ErrInvalidInput)
	}

	// Ensure target client is not in source clients list
	for _, sourceID := range req.SourceClientIDs {
		if sourceID == req.TargetClientID {
			return response.Error(response.ErrInvalidInput)
		}
	}

	// Start transaction
	tx := db.Begin()

	// Move all tickets to target client
	err = tx.Model(&models.Conversation{}).
		Where("client_id IN (?)", req.SourceClientIDs).
		Update("client_id", req.TargetClientID).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Move all messages to target client
	err = tx.Model(&models.Message{}).
		Where("client_id IN (?)", req.SourceClientIDs).
		Update("client_id", req.TargetClientID).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Move all external IDs to target client (avoid duplicates)
	for _, sourceID := range req.SourceClientIDs {
		err = tx.Model(&models.ClientExternalID{}).
			Where("client_id = ?", sourceID).
			Update("client_id", req.TargetClientID).Error
		if err != nil {
			// If there are duplicates, delete the source ones
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				tx.Where("client_id = ?", sourceID).Delete(&models.ClientExternalID{})
			} else {
				tx.Rollback()
				return response.Error(response.ErrInternalError)
			}
		}
	}

	// Merge data from source clients into target client
	var targetData map[string]interface{}
	if len(targetClient.Data) > 0 {
		err = json.Unmarshal(targetClient.Data, &targetData)
		if err != nil {
			targetData = make(map[string]interface{})
		}
	} else {
		targetData = make(map[string]interface{})
	}

	for _, sourceClient := range sourceClients {
		var sourceData map[string]interface{}
		if len(sourceClient.Data) > 0 {
			err = json.Unmarshal(sourceClient.Data, &sourceData)
			if err == nil {
				// Merge source data into target data
				for key, value := range sourceData {
					if _, exists := targetData[key]; !exists {
						targetData[key] = value
					}
				}
			}
		}
	}

	// Update target client with merged data
	mergedData, err := json.Marshal(targetData)
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	err = tx.Model(&targetClient).Update("data", mergedData).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Delete source clients
	err = tx.Where("id IN (?)", req.SourceClientIDs).Delete(&models.Client{}).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Commit transaction
	tx.Commit()

	// Fetch updated target client
	err = db.
		Preload("ExternalIDs").
		Preload("Conversations").
		First(&targetClient, "id = ?", req.TargetClientID).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message":             "Clients merged successfully",
		"target_client":       targetClient,
		"merged_client_ids":   req.SourceClientIDs,
		"merged_client_count": len(req.SourceClientIDs),
	})
}

// GetClient returns a single client by ID with external IDs
func (c Controller) GetClient(request *evo.Request) any {
	clientIDStr := request.Param("id").String()
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var client models.Client
	err = db.
		Preload("ExternalIDs").
		First(&client, "id = ?", clientID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(client)
}

// CreateClient creates a new client with external IDs
func (c Controller) CreateClient(request *evo.Request) any {
	var req struct {
		Name        string                 `json:"name" validate:"required,min=1,max=255"`
		Data        map[string]interface{} `json:"data"`
		Language    *string                `json:"language"`
		Timezone    *string                `json:"timezone"`
		ExternalIDs []struct {
			Type  string `json:"type" validate:"required,oneof=email phone whatsapp slack telegram web chat"`
			Value string `json:"value" validate:"required,min=1,max=255"`
		} `json:"external_ids" validate:"required,min=1"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Validate that at least one external ID is provided
	if len(req.ExternalIDs) == 0 {
		return response.Error(response.ErrInvalidInput)
	}

	// Start transaction
	tx := db.Begin()

	// Create client
	client := models.Client{
		Name:     req.Name,
		Language: req.Language,
		Timezone: req.Timezone,
	}

	// Convert data to JSON if provided
	if req.Data != nil {
		dataJSON, err := json.Marshal(req.Data)
		if err != nil {
			tx.Rollback()
			return response.Error(response.ErrInvalidInput)
		}
		client.Data = dataJSON
	}

	err := tx.Create(&client).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Create external IDs
	for _, extID := range req.ExternalIDs {
		externalID := models.ClientExternalID{
			ClientID: client.ID,
			Type:     extID.Type,
			Value:    extID.Value,
		}
		err = tx.Create(&externalID).Error
		if err != nil {
			tx.Rollback()
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				duplicateErr := response.NewError(response.ErrorCodeConflict, "External ID already exists", 409)
				return response.Error(duplicateErr)
			}
			return response.Error(response.ErrInternalError)
		}
	}

	// Commit transaction
	tx.Commit()

	// Fetch created client with external IDs
	err = db.
		Preload("ExternalIDs").
		First(&client, "id = ?", client.ID).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.Created(client)
}

// UpdateClient updates an existing client and its external IDs
func (c Controller) UpdateClient(request *evo.Request) any {
	clientIDStr := request.Param("id").String()
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var req struct {
		Name        string                 `json:"name"`
		Data        map[string]interface{} `json:"data"`
		Language    *string                `json:"language"`
		Timezone    *string                `json:"timezone"`
		ExternalIDs []struct {
			Type  string `json:"type" validate:"required,oneof=email phone whatsapp slack telegram web chat"`
			Value string `json:"value" validate:"required,min=1,max=255"`
		} `json:"external_ids"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Verify client exists
	var client models.Client
	err = db.First(&client, "id = ?", clientID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Start transaction
	tx := db.Begin()

	// Prepare updates
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Language != nil {
		updates["language"] = req.Language
	}
	if req.Timezone != nil {
		updates["timezone"] = req.Timezone
	}
	if req.Data != nil {
		dataJSON, err := json.Marshal(req.Data)
		if err != nil {
			tx.Rollback()
			return response.Error(response.ErrInvalidInput)
		}
		updates["data"] = dataJSON
	}

	// Update client if there are updates
	if len(updates) > 0 {
		err = tx.Model(&client).Updates(updates).Error
		if err != nil {
			tx.Rollback()
			return response.Error(response.ErrInternalError)
		}
	}

	// Update external IDs if provided (delete old ones and create new ones)
	if req.ExternalIDs != nil {
		// Delete existing external IDs
		err = tx.Where("client_id = ?", clientID).Delete(&models.ClientExternalID{}).Error
		if err != nil {
			tx.Rollback()
			return response.Error(response.ErrInternalError)
		}

		// Create new external IDs
		for _, extID := range req.ExternalIDs {
			externalID := models.ClientExternalID{
				ClientID: clientID,
				Type:     extID.Type,
				Value:    extID.Value,
			}
			err = tx.Create(&externalID).Error
			if err != nil {
				tx.Rollback()
				if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
					duplicateErr := response.NewError(response.ErrorCodeConflict, "External ID already exists", 409)
					return response.Error(duplicateErr)
				}
				return response.Error(response.ErrInternalError)
			}
		}
	}

	// Commit transaction
	tx.Commit()

	// Fetch updated client with external IDs
	err = db.
		Preload("ExternalIDs").
		First(&client, "id = ?", clientID).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(client)
}

// DeleteClient deletes a client and its associated external IDs
func (c Controller) DeleteClient(request *evo.Request) any {
	clientIDStr := request.Param("id").String()
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Verify client exists
	var client models.Client
	err = db.First(&client, "id = ?", clientID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Start transaction
	tx := db.Begin()

	// Delete external IDs
	err = tx.Where("client_id = ?", clientID).Delete(&models.ClientExternalID{}).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Delete client
	err = tx.Delete(&client).Error
	if err != nil {
		tx.Rollback()
		return response.Error(response.ErrInternalError)
	}

	// Commit transaction
	tx.Commit()

	return response.OK(map[string]interface{}{
		"message": "Client deleted successfully",
		"id":      clientID,
		"name":    client.Name,
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

// ========================
// WEBHOOK MANAGEMENT APIs
// ========================

// ListWebhooks returns all webhooks
func (c Controller) ListWebhooks(request *evo.Request) any {
	var webhooks []models.Webhook

	err := db.Order("id DESC").Find(&webhooks).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(webhooks)
}

// GetWebhook returns a single webhook by ID
func (c Controller) GetWebhook(request *evo.Request) any {
	id := request.Param("id").String()
	var webhook models.Webhook

	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(webhook)
}

// CreateWebhook creates a new webhook
func (c Controller) CreateWebhook(request *evo.Request) any {
	var webhook models.Webhook

	if err := request.BodyParser(&webhook); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Validate required fields
	if webhook.Name == "" {
		return response.BadRequest(request, "Name is required")
	}
	if webhook.URL == "" {
		return response.BadRequest(request, "URL is required")
	}

	// Create the webhook
	err := db.Create(&webhook).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(webhook)
}

// UpdateWebhook updates an existing webhook
func (c Controller) UpdateWebhook(request *evo.Request) any {
	id := request.Param("id").String()

	var webhook models.Webhook
	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Parse update data
	var updateData map[string]any
	if err := request.BodyParser(&updateData); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Update the webhook
	err = db.Model(&webhook).Updates(updateData).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload the webhook
	db.First(&webhook, id)

	return response.OK(webhook)
}

// DeleteWebhook deletes a webhook
func (c Controller) DeleteWebhook(request *evo.Request) any {
	id := request.Param("id").String()

	var webhook models.Webhook
	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Delete associated deliveries first
	db.Where("webhook_id = ?", id).Delete(&models.WebhookDelivery{})

	// Delete the webhook
	err = db.Delete(&webhook).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Webhook deleted successfully"})
}

// TestWebhook sends a test webhook
func (c Controller) TestWebhook(request *evo.Request) any {
	id := request.Param("id").String()

	var webhook models.Webhook
	err := db.First(&webhook, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Get the event type from query param, default to webhook.test
	eventType := request.Query("event").String()
	if eventType == "" {
		eventType = models.WebhookEventWebhookTest
	}

	// Build test data based on event type
	testData := buildTestDataForEvent(eventType)

	// Send directly to this specific webhook
	if models.WebhookSender != nil {
		// Force enable for test purposes
		webhook.Enabled = true
		err := models.SendToWebhook(&webhook, eventType, testData)
		if err != nil {
			return response.OK(map[string]any{
				"success": false,
				"message": "Test webhook failed: " + err.Error(),
				"event":   eventType,
			})
		}
	} else {
		return response.OK(map[string]any{
			"success": false,
			"message": "Webhook sender not initialized",
			"event":   eventType,
		})
	}

	return response.OK(map[string]any{
		"success": true,
		"message": "Test webhook sent for event: " + eventType,
		"event":   eventType,
	})
}

// buildTestDataForEvent creates sample test data based on event type
func buildTestDataForEvent(eventType string) map[string]any {
	switch eventType {
	case models.WebhookEventConversationCreated,
		models.WebhookEventConversationUpdated,
		models.WebhookEventConversationClosed:
		return map[string]any{
			"conversation": map[string]any{
				"id":         1,
				"client_id":  "550e8400-e29b-41d4-a716-446655440000",
				"status":     "open",
				"priority":   "normal",
				"subject":    "Test Conversation Subject",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
	case models.WebhookEventConversationStatusChange:
		return map[string]any{
			"conversation": map[string]any{
				"id":         1,
				"client_id":  "550e8400-e29b-41d4-a716-446655440000",
				"status":     "pending",
				"priority":   "normal",
				"subject":    "Test Conversation Subject",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:40:00Z",
			},
			"old_status": "open",
			"new_status": "pending",
		}
	case models.WebhookEventConversationAssigned:
		return map[string]any{
			"conversation": map[string]any{
				"id":          1,
				"client_id":   "550e8400-e29b-41d4-a716-446655440000",
				"status":      "open",
				"priority":    "normal",
				"assigned_to": "550e8400-e29b-41d4-a716-446655440001",
				"subject":     "Test Conversation Subject",
				"created_at":  "2024-01-15T10:30:00Z",
				"updated_at":  "2024-01-15T10:30:00Z",
			},
			"assigned_to": "Agent Name",
			"assigned_by": "Admin User",
		}
	case models.WebhookEventMessageCreated:
		return map[string]any{
			"message": map[string]any{
				"id":              1,
				"conversation_id": 1,
				"client_id":       "550e8400-e29b-41d4-a716-446655440000",
				"body":            "This is a test message body for webhook testing.",
				"content_type":    "text",
				"created_at":      "2024-01-15T10:30:00Z",
			},
			"conversation": map[string]any{
				"id":         1,
				"client_id":  "550e8400-e29b-41d4-a716-446655440000",
				"status":     "open",
				"priority":   "normal",
				"subject":    "Test Conversation Subject",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
	case models.WebhookEventClientCreated,
		models.WebhookEventClientUpdated:
		return map[string]any{
			"client": map[string]any{
				"id":         "550e8400-e29b-41d4-a716-446655440000",
				"name":       "Test Client",
				"data":       map[string]any{"email": "test@example.com", "phone": "+1234567890"},
				"language":   "en",
				"timezone":   "UTC",
				"created_at": "2024-01-15T10:30:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
	case models.WebhookEventUserCreated,
		models.WebhookEventUserUpdated:
		return map[string]any{
			"user": map[string]any{
				"id":           "550e8400-e29b-41d4-a716-446655440001",
				"name":         "Test",
				"last_name":    "Agent",
				"display_name": "Test Agent",
				"email":        "agent@example.com",
				"type":         "agent",
				"status":       "active",
				"created_at":   "2024-01-15T10:30:00Z",
				"updated_at":   "2024-01-15T10:30:00Z",
			},
		}
	default: // webhook.test
		return map[string]any{
			"message":    "This is a test webhook payload",
			"test":       true,
			"webhook_id": 1,
		}
	}
}

// ListWebhookDeliveries returns webhook delivery logs
func (c Controller) ListWebhookDeliveries(request *evo.Request) any {
	var deliveries []models.WebhookDelivery

	query := db.Model(&models.WebhookDelivery{})

	// Filter by webhook_id if provided
	if webhookID := request.Query("webhook_id").String(); webhookID != "" {
		query = query.Where("webhook_id = ?", webhookID)
	}

	// Filter by success if provided
	if success := request.Query("success").String(); success != "" {
		query = query.Where("success = ?", success == "true")
	}

	// Filter by event if provided (supports comma-separated list for multiple events)
	if event := request.Query("event").String(); event != "" {
		events := strings.Split(event, ",")
		if len(events) == 1 {
			query = query.Where("event = ?", event)
		} else {
			query = query.Where("event IN (?)", events)
		}
	}

	// Order by most recent first
	query = query.Order("id DESC")

	p, err := pagination.New(query, request, &deliveries, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(deliveries, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// GetWebhookDelivery returns a single webhook delivery by ID
func (c Controller) GetWebhookDelivery(request *evo.Request) any {
	id := request.Param("id").String()
	var delivery models.WebhookDelivery

	err := db.Preload("Webhook").First(&delivery, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Webhook delivery not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(delivery)
}

// ========================================
// Integration Management
// ========================================

// ListIntegrations returns all integrations with masked configs
func (c Controller) ListIntegrations(request *evo.Request) any {
	integrations, err := models.GetAllIntegrations()
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Build response with masked configs
	type IntegrationResponse struct {
		ID        uint                   `json:"id"`
		Type      string                 `json:"type"`
		Name      string                 `json:"name"`
		Status    string                 `json:"status"`
		Config    map[string]interface{} `json:"config,omitempty"`
		LastError string                 `json:"last_error,omitempty"`
		TestedAt  *time.Time             `json:"tested_at,omitempty"`
		CreatedAt time.Time              `json:"created_at"`
		UpdatedAt time.Time              `json:"updated_at"`
	}

	result := make([]IntegrationResponse, len(integrations))
	for i, integration := range integrations {
		result[i] = IntegrationResponse{
			ID:        integration.ID,
			Type:      integration.Type,
			Name:      integration.Name,
			Status:    integration.Status,
			Config:    integrationsDriver.GetMaskedConfig(integration.Type, integration.Config),
			LastError: integration.LastError,
			TestedAt:  integration.TestedAt,
			CreatedAt: integration.CreatedAt,
			UpdatedAt: integration.UpdatedAt,
		}
	}

	return response.OK(result)
}

// ListIntegrationTypes returns all available integration types
func (c Controller) ListIntegrationTypes(request *evo.Request) any {
	types := models.GetIntegrationTypes()
	return response.OK(types)
}

// GetIntegration returns a single integration by type
func (c Controller) GetIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	integration, err := models.GetIntegration(integrationType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return empty integration with type info
			return response.OK(map[string]interface{}{
				"type":   integrationType,
				"name":   getIntegrationName(integrationType),
				"status": models.IntegrationStatusDisabled,
				"config": nil,
			})
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"id":         integration.ID,
		"type":       integration.Type,
		"name":       integration.Name,
		"status":     integration.Status,
		"config":     integrationsDriver.GetMaskedConfig(integration.Type, integration.Config),
		"last_error": integration.LastError,
		"tested_at":  integration.TestedAt,
		"created_at": integration.CreatedAt,
		"updated_at": integration.UpdatedAt,
	})
}

// GetIntegrationFields returns the configuration fields for an integration type
func (c Controller) GetIntegrationFields(request *evo.Request) any {
	integrationType := request.Param("type").String()

	fields := integrationsDriver.GetConfigFields(integrationType)
	if fields == nil {
		return response.NotFound(request, "Unknown integration type")
	}

	return response.OK(fields)
}

// SaveIntegration creates or updates an integration
func (c Controller) SaveIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	var req struct {
		Status string                 `json:"status"`
		Config map[string]interface{} `json:"config"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Get or create integration
	integration, _ := models.GetIntegration(integrationType)
	if integration.ID == 0 {
		integration = &models.Integration{
			Type: integrationType,
			Name: getIntegrationName(integrationType),
		}
	}

	// Merge incoming config with existing to preserve masked sensitive fields
	mergedConfig := integrationsDriver.MergeConfigWithExisting(integration.Config, req.Config)

	// Convert merged config to JSON
	configJSON, err := json.Marshal(mergedConfig)
	if err != nil {
		return response.BadRequest(request, "Invalid configuration")
	}

	// Validate configuration
	if err := integrationsDriver.ValidateConfig(integrationType, string(configJSON)); err != nil {
		return response.BadRequest(request, err.Error())
	}

	integration.Status = req.Status
	integration.Config = string(configJSON)
	integration.LastError = ""

	if err := models.UpsertIntegration(integration); err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Call OnSave callback for post-save actions (e.g., webhook registration)
	webhookBaseURL := getWebhookBaseURL(request)
	onSaveResult := integrationsDriver.OnSave(integration.Type, integration.Config, integration.Status, webhookBaseURL)

	return response.OK(map[string]interface{}{
		"id":              integration.ID,
		"type":            integration.Type,
		"name":            integration.Name,
		"status":          integration.Status,
		"config":          integrationsDriver.GetMaskedConfig(integration.Type, integration.Config),
		"last_error":      integration.LastError,
		"tested_at":       integration.TestedAt,
		"updated_at":      integration.UpdatedAt,
		"on_save_success": onSaveResult.Success,
		"on_save_message": onSaveResult.Message,
	})
}

// TestIntegration tests the connection for an integration
func (c Controller) TestIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	var req struct {
		Config map[string]interface{} `json:"config"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Get existing integration config to merge with masked values
	existingIntegration, _ := models.GetIntegration(integrationType)
	if existingIntegration.ID != 0 && existingIntegration.Config != "" {
		// Merge the incoming config with existing config to preserve masked sensitive fields
		req.Config = integrationsDriver.MergeConfigWithExisting(existingIntegration.Config, req.Config)
	}

	// Convert config to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return response.BadRequest(request, "Invalid configuration")
	}

	// Test the integration
	result := integrationsDriver.TestIntegration(integrationType, string(configJSON))

	// Update the integration if it exists
	if result.Success {
		integration, _ := models.GetIntegration(integrationType)
		if integration.ID != 0 {
			now := time.Now()
			integration.TestedAt = &now
			integration.LastError = ""
			db.Save(integration)
		}
	} else {
		integration, _ := models.GetIntegration(integrationType)
		if integration.ID != 0 {
			now := time.Now()
			integration.TestedAt = &now
			integration.LastError = result.Message
			if result.Details != "" {
				integration.LastError += ": " + result.Details
			}
			db.Save(integration)
		}
	}

	return response.OK(result)
}

// DeleteIntegration removes an integration
func (c Controller) DeleteIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	integration, err := models.GetIntegration(integrationType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Integration not found")
		}
		return response.Error(response.ErrInternalError)
	}

	if err := db.Delete(integration).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Integration deleted successfully"})
}

// Helper function to get integration name from type
func getIntegrationName(integrationType string) string {
	types := models.GetIntegrationTypes()
	for _, t := range types {
		if t.Type == integrationType {
			return t.Name
		}
	}
	return integrationType
}

// getWebhookBaseURL returns the API base URL for webhook registration
func getWebhookBaseURL(request *evo.Request) string {
	// First check for configured API base URL
	apiBaseURL := settings.Get("APP.API_BASE_URL").String()
	if apiBaseURL != "" {
		return apiBaseURL
	}

	// Fallback to X-Forwarded headers for reverse proxy setups
	proto := request.Get("X-Forwarded-Proto").String()
	if proto == "" {
		proto = request.Protocol()
		if proto == "HTTP/1.1" || proto == "HTTP/2" {
			proto = "http"
		}
	}

	host := request.Get("X-Forwarded-Host").String()
	if host == "" {
		host = request.Hostname()
	}

	return proto + "://" + host
}

// ListAIAgents returns all AI agents
func (c Controller) ListAIAgents(request *evo.Request) any {
	var agents []models.AIAgent

	query := db.Order("id DESC")

	// Filter by status if provided
	if status := request.Query("status").String(); status != "" {
		query = query.Where("status = ?", status)
	}

	// Preload relationships
	query = query.Preload("Bot").Preload("HandoverUser")

	err := query.Find(&agents).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(agents)
}

// GetAIAgent returns a single AI agent by ID
func (c Controller) GetAIAgent(request *evo.Request) any {
	id := request.Param("id").String()
	var agent models.AIAgent

	err := db.Preload("Bot").Preload("HandoverUser").First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(agent)
}

// CreateAIAgent creates a new AI agent
func (c Controller) CreateAIAgent(request *evo.Request) any {
	var agent models.AIAgent

	if err := request.BodyParser(&agent); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Validate required fields
	if agent.Name == "" {
		return response.BadRequest(request, "Name is required")
	}
	if agent.BotID == "" {
		return response.BadRequest(request, "Bot ID is required")
	}

	// Validate tone
	validTones := []string{
		models.AIAgentToneFormal,
		models.AIAgentToneCasual,
		models.AIAgentToneDetailed,
		models.AIAgentTonePrecise,
		models.AIAgentToneEmpathetic,
		models.AIAgentToneTechnical,
	}
	toneValid := false
	for _, t := range validTones {
		if agent.Tone == t {
			toneValid = true
			break
		}
	}
	if !toneValid {
		return response.BadRequest(request, "Invalid tone value")
	}

	// Create the agent
	err := db.Create(&agent).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload with relationships
	db.Preload("Bot").Preload("HandoverUser").First(&agent, agent.ID)

	return response.OK(agent)
}

// UpdateAIAgent updates an existing AI agent
func (c Controller) UpdateAIAgent(request *evo.Request) any {
	id := request.Param("id").String()

	var agent models.AIAgent
	err := db.First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Parse update data
	var updateData map[string]any
	if err := request.BodyParser(&updateData); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Validate tone if it's being updated
	if tone, ok := updateData["tone"].(string); ok {
		validTones := []string{
			models.AIAgentToneFormal,
			models.AIAgentToneCasual,
			models.AIAgentToneDetailed,
			models.AIAgentTonePrecise,
			models.AIAgentToneEmpathetic,
			models.AIAgentToneTechnical,
		}
		toneValid := false
		for _, t := range validTones {
			if tone == t {
				toneValid = true
				break
			}
		}
		if !toneValid {
			return response.BadRequest(request, "Invalid tone value")
		}
	}

	// Convert handover_user_ids array to JSON if present
	if userIds, ok := updateData["handover_user_ids"]; ok {
		jsonBytes, err := json.Marshal(userIds)
		if err != nil {
			return response.BadRequest(request, "Invalid handover_user_ids format")
		}
		updateData["handover_user_ids"] = datatypes.JSON(jsonBytes)
	}

	// Update the agent
	err = db.Model(&agent).Updates(updateData).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload with relationships
	db.Preload("Bot").Preload("HandoverUser").First(&agent, id)

	return response.OK(agent)
}

// DeleteAIAgent deletes an AI agent
func (c Controller) DeleteAIAgent(request *evo.Request) any {
	id := request.Param("id").String()

	var agent models.AIAgent
	err := db.First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Check if agent is being used by any departments
	var count int64
	db.Model(&models.Department{}).Where("ai_agent_id = ?", id).Count(&count)
	if count > 0 {
		return response.BadRequest(request, "Cannot delete AI agent that is assigned to departments")
	}

	// Delete the agent
	err = db.Delete(&agent).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "AI Agent deleted successfully"})
}

// GetAIAgentTemplate generates and returns the system prompt template for an AI agent
func (c Controller) GetAIAgentTemplate(request *evo.Request) any {
	id := request.Param("id").String()

	// Load agent with relationships
	var agent models.AIAgent
	err := db.Preload("Bot").Preload("HandoverUser").First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Load agent tools
	var tools []models.AIAgentTool
	db.Where("ai_agent_id = ?", id).Order("id ASC").Find(&tools)

	// Get project name from settings
	projectName := models.GetSettingValue("general.project_name", "Your Project")

	// Load knowledge base articles (just titles) if knowledge base is enabled
	var kbItems []ai.KnowledgeBaseItem
	if agent.UseKnowledgeBase {
		var articles []models.KnowledgeBaseArticle
		db.Select("id, title").Where("status = ?", "published").Find(&articles)
		for _, a := range articles {
			kbItems = append(kbItems, ai.KnowledgeBaseItem{
				ID:    a.ID.String(),
				Title: a.Title,
			})
		}
	}

	// Generate the template
	template := ai.GenerateAgentTemplate(ai.TemplateContext{
		ProjectName:        projectName,
		Agent:              &agent,
		Tools:              tools,
		KnowledgeBaseItems: kbItems,
	})

	return response.OK(map[string]any{
		"template":    template,
		"token_count": len(template) / 4, // Approximate token count
	})
}

// ==================== AI Agent Tools ====================

// ListAIAgentTools returns all tools for an AI agent
func (c Controller) ListAIAgentTools(request *evo.Request) any {
	agentID := request.Param("agent_id").String()

	// Verify agent exists
	var agent models.AIAgent
	if err := db.First(&agent, agentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	var tools []models.AIAgentTool
	err := db.Where("ai_agent_id = ?", agentID).Order("id ASC").Find(&tools).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tools)
}

// GetAIAgentTool returns a single tool by ID
func (c Controller) GetAIAgentTool(request *evo.Request) any {
	toolID := request.Param("tool_id").String()

	var tool models.AIAgentTool
	err := db.First(&tool, toolID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Tool not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tool)
}

// CreateAIAgentTool creates a new tool for an AI agent
func (c Controller) CreateAIAgentTool(request *evo.Request) any {
	agentID := request.Param("agent_id").String()

	// Verify agent exists
	var agent models.AIAgent
	if err := db.First(&agent, agentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	var tool models.AIAgentTool
	if err := request.BodyParser(&tool); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Set the agent ID
	tool.AIAgentID = agent.ID

	// Validate required fields
	if tool.Name == "" {
		return response.BadRequest(request, "Tool name is required")
	}
	if tool.Endpoint == "" {
		return response.BadRequest(request, "Endpoint is required")
	}

	// Validate method
	validMethods := []string{models.ToolMethodGET, models.ToolMethodPOST, models.ToolMethodPUT, models.ToolMethodPATCH, models.ToolMethodDELETE}
	methodValid := false
	for _, m := range validMethods {
		if tool.Method == m {
			methodValid = true
			break
		}
	}
	if !methodValid {
		tool.Method = models.ToolMethodGET
	}

	// Set default auth type if not provided
	if tool.AuthorizationType == "" {
		tool.AuthorizationType = models.ToolAuthTypeNone
	}

	// Create the tool
	err := db.Create(&tool).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tool)
}

// UpdateAIAgentTool updates an existing tool
func (c Controller) UpdateAIAgentTool(request *evo.Request) any {
	toolID := request.Param("tool_id").String()

	var tool models.AIAgentTool
	err := db.First(&tool, toolID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Tool not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Parse update data into the tool struct directly
	if err := request.BodyParser(&tool); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Save all fields
	err = db.Save(&tool).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tool)
}

// DeleteAIAgentTool deletes a tool
func (c Controller) DeleteAIAgentTool(request *evo.Request) any {
	toolID := request.Param("tool_id").String()

	var tool models.AIAgentTool
	err := db.First(&tool, toolID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Tool not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Delete the tool
	err = db.Delete(&tool).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Tool deleted successfully"})
}
