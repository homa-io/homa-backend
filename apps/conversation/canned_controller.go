package conversation

import (
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// ListCannedMessages returns paginated list of canned messages
// @Summary List canned messages
// @Description Get a paginated list of canned messages with optional filtering
// @Tags Agent - Canned Messages
// @Accept json
// @Produce json
// @Param search query string false "Search in title or message content"
// @Param is_active query bool false "Filter by active status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response
// @Router /api/agent/canned-messages [get]
func (ac AgentController) ListCannedMessages(req *evo.Request) interface{} {
	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}

	limit := req.Query("limit").Int()
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	query := db.Model(&models.CannedMessage{})

	search := req.Query("search").String()
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("title LIKE ? OR message LIKE ?", searchTerm, searchTerm)
	}

	if isActiveStr := req.Query("is_active").String(); isActiveStr != "" {
		isActive := isActiveStr == "true" || isActiveStr == "1"
		query = query.Where("is_active = ?", isActive)
	}

	query = query.Order("created_at DESC")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Error("Failed to count canned messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count messages", 500, err.Error()))
	}

	var messages []models.CannedMessage
	if err := query.Limit(limit).Offset(offset).Find(&messages).Error; err != nil {
		log.Error("Failed to fetch canned messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to fetch messages", 500, err.Error()))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return response.OKWithMeta(messages, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetCannedMessage returns a single canned message by ID
// @Summary Get canned message
// @Description Get a single canned message by ID
// @Tags Agent - Canned Messages
// @Accept json
// @Produce json
// @Param id path int true "Message ID"
// @Success 200 {object} response.Response
// @Router /api/agent/canned-messages/{id} [get]
func (ac AgentController) GetCannedMessage(req *evo.Request) interface{} {
	id := req.Param("id").Uint()
	if id == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid message ID", 400, "Message ID must be a positive integer"))
	}

	var message models.CannedMessage
	if err := db.Where("id = ?", id).First(&message).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Canned message not found", 404, err.Error()))
	}

	return response.OK(message)
}

// CreateCannedMessage creates a new canned message
// @Summary Create canned message
// @Description Create a new canned message
// @Tags Agent - Canned Messages
// @Accept json
// @Produce json
// @Param body body object true "Canned message data"
// @Success 201 {object} response.Response
// @Router /api/agent/canned-messages [post]
func (ac AgentController) CreateCannedMessage(req *evo.Request) interface{} {
	type CreateRequest struct {
		Title    string  `json:"title"`
		Message  string  `json:"message"`
		Shortcut *string `json:"shortcut"`
		IsActive *bool   `json:"is_active"`
	}

	var createReq CreateRequest
	if err := req.BodyParser(&createReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if createReq.Title == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Title is required", 400, "title field cannot be empty"))
	}
	if createReq.Message == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Message is required", 400, "message field cannot be empty"))
	}

	isActive := true
	if createReq.IsActive != nil {
		isActive = *createReq.IsActive
	}

	cannedMessage := models.CannedMessage{
		Title:    createReq.Title,
		Message:  createReq.Message,
		Shortcut: createReq.Shortcut,
		IsActive: isActive,
	}

	if err := db.Create(&cannedMessage).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "Duplicate") {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeConflict, "A canned message with this shortcut already exists", 409, err.Error()))
		}
		log.Error("Failed to create canned message:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to create message", 500, err.Error()))
	}

	return response.Created(cannedMessage)
}

// UpdateCannedMessage updates an existing canned message
// @Summary Update canned message
// @Description Update an existing canned message by ID
// @Tags Agent - Canned Messages
// @Accept json
// @Produce json
// @Param id path int true "Message ID"
// @Param body body object true "Canned message data"
// @Success 200 {object} response.Response
// @Router /api/agent/canned-messages/{id} [put]
func (ac AgentController) UpdateCannedMessage(req *evo.Request) interface{} {
	id := req.Param("id").Uint()
	if id == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid message ID", 400, "Message ID must be a positive integer"))
	}

	type UpdateRequest struct {
		Title    string  `json:"title"`
		Message  string  `json:"message"`
		Shortcut *string `json:"shortcut"`
		IsActive *bool   `json:"is_active"`
	}

	var updateReq UpdateRequest
	if err := req.BodyParser(&updateReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if updateReq.Title == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Title is required", 400, "title field cannot be empty"))
	}
	if updateReq.Message == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Message is required", 400, "message field cannot be empty"))
	}

	var cannedMessage models.CannedMessage
	if err := db.Where("id = ?", id).First(&cannedMessage).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Canned message not found", 404, err.Error()))
	}

	updates := map[string]interface{}{
		"title":    updateReq.Title,
		"message":  updateReq.Message,
		"shortcut": updateReq.Shortcut,
	}

	if updateReq.IsActive != nil {
		updates["is_active"] = *updateReq.IsActive
	}

	if err := db.Model(&cannedMessage).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "Duplicate") {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeConflict, "A canned message with this shortcut already exists", 409, err.Error()))
		}
		log.Error("Failed to update canned message:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to update message", 500, err.Error()))
	}

	db.Where("id = ?", id).First(&cannedMessage)

	return response.OK(cannedMessage)
}

// DeleteCannedMessage deletes a canned message
// @Summary Delete canned message
// @Description Delete an existing canned message by ID
// @Tags Agent - Canned Messages
// @Accept json
// @Produce json
// @Param id path int true "Message ID"
// @Success 200 {object} response.Response
// @Router /api/agent/canned-messages/{id} [delete]
func (ac AgentController) DeleteCannedMessage(req *evo.Request) interface{} {
	id := req.Param("id").Uint()
	if id == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid message ID", 400, "Message ID must be a positive integer"))
	}

	var cannedMessage models.CannedMessage
	if err := db.Where("id = ?", id).First(&cannedMessage).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Canned message not found", 404, err.Error()))
	}

	if err := db.Delete(&cannedMessage).Error; err != nil {
		log.Error("Failed to delete canned message:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to delete message", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"message": "Canned message deleted successfully",
		"id":      id,
	})
}
