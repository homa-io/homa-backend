package conversation

import (
	"fmt"
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// ListCustomAttributes returns paginated list of custom attributes with search and filtering
// @Summary List custom attributes
// @Description Get a paginated list of custom attributes with optional filtering
// @Tags Agent - Custom Attributes
// @Accept json
// @Produce json
// @Param search query string false "Search in name, title, description"
// @Param scope query string false "Filter by scope (client or conversation)"
// @Param data_type query string false "Filter by data type (int, float, date, string)"
// @Param visibility query string false "Filter by visibility (everyone, administrator, hidden)"
// @Param order_by query string false "Order by field (name, title, scope, created_at)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response
// @Router /api/agent/attributes [get]
func (ac AgentController) ListCustomAttributes(req *evo.Request) interface{} {
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

	query := db.Model(&models.CustomAttribute{})

	search := req.Query("search").String()
	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("name LIKE ? OR title LIKE ? OR description LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	if scope := req.Query("scope").String(); scope != "" {
		query = query.Where("scope = ?", scope)
	}

	if dataType := req.Query("data_type").String(); dataType != "" {
		query = query.Where("data_type = ?", dataType)
	}

	if visibility := req.Query("visibility").String(); visibility != "" {
		query = query.Where("visibility = ?", visibility)
	}

	orderBy := req.Query("order_by").String()
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

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Error("Failed to count custom attributes:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count attributes", 500, err.Error()))
	}

	var customAttrs []models.CustomAttribute
	if err := query.Limit(limit).Offset(offset).Find(&customAttrs).Error; err != nil {
		log.Error("Failed to fetch custom attributes:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to fetch attributes", 500, err.Error()))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return response.OKWithMeta(customAttrs, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// CreateCustomAttribute creates a new custom attribute
// @Summary Create custom attribute
// @Description Create a new custom attribute
// @Tags Agent - Custom Attributes
// @Accept json
// @Produce json
// @Param body body object true "Custom attribute data"
// @Success 201 {object} response.Response
// @Router /api/agent/attributes [post]
func (ac AgentController) CreateCustomAttribute(req *evo.Request) interface{} {
	type CreateRequest struct {
		Scope       string  `json:"scope"`
		Name        string  `json:"name"`
		DataType    string  `json:"data_type"`
		Validation  *string `json:"validation"`
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Visibility  string  `json:"visibility"`
	}

	var createReq CreateRequest
	if err := req.BodyParser(&createReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if createReq.Scope == "" || createReq.Name == "" || createReq.DataType == "" || createReq.Title == "" || createReq.Visibility == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Missing required fields", 400, "scope, name, data_type, title, and visibility are required"))
	}

	if createReq.Scope != "client" && createReq.Scope != "conversation" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid scope", 400, "Scope must be 'client' or 'conversation'"))
	}

	validDataTypes := map[string]bool{"int": true, "float": true, "date": true, "string": true}
	if !validDataTypes[createReq.DataType] {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid data type", 400, "Data type must be one of: int, float, date, string"))
	}

	validVisibilities := map[string]bool{"everyone": true, "administrator": true, "hidden": true}
	if !validVisibilities[createReq.Visibility] {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid visibility", 400, "Visibility must be one of: everyone, administrator, hidden"))
	}

	name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(createReq.Name), " ", "_"))

	if !isValidAttributeName(name) {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid name format", 400, "Name can only contain lowercase letters and underscores"))
	}

	customAttr := models.CustomAttribute{
		Scope:       createReq.Scope,
		Name:        name,
		DataType:    createReq.DataType,
		Validation:  createReq.Validation,
		Title:       createReq.Title,
		Description: createReq.Description,
		Visibility:  createReq.Visibility,
	}

	if err := db.Create(&customAttr).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "Duplicate") {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeConflict, "Custom attribute with this scope and name already exists", 409, err.Error()))
		}
		log.Error("Failed to create custom attribute:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to create attribute", 500, err.Error()))
	}

	return response.Created(customAttr)
}

// UpdateCustomAttribute updates an existing custom attribute
// @Summary Update custom attribute
// @Description Update an existing custom attribute by scope and name
// @Tags Agent - Custom Attributes
// @Accept json
// @Produce json
// @Param scope path string true "Attribute scope"
// @Param name path string true "Attribute name"
// @Param body body object true "Custom attribute data"
// @Success 200 {object} response.Response
// @Router /api/agent/attributes/{scope}/{name} [put]
func (ac AgentController) UpdateCustomAttribute(req *evo.Request) interface{} {
	scope := req.Param("scope").String()
	name := req.Param("name").String()

	if scope == "" || name == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Missing parameters", 400, "Scope and name are required"))
	}

	type UpdateRequest struct {
		DataType    string  `json:"data_type"`
		Validation  *string `json:"validation"`
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Visibility  string  `json:"visibility"`
	}

	var updateReq UpdateRequest
	if err := req.BodyParser(&updateReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if updateReq.DataType == "" || updateReq.Title == "" || updateReq.Visibility == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Missing required fields", 400, "data_type, title, and visibility are required"))
	}

	validDataTypes := map[string]bool{"int": true, "float": true, "date": true, "string": true}
	if !validDataTypes[updateReq.DataType] {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid data type", 400, "Data type must be one of: int, float, date, string"))
	}

	validVisibilities := map[string]bool{"everyone": true, "administrator": true, "hidden": true}
	if !validVisibilities[updateReq.Visibility] {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid visibility", 400, "Visibility must be one of: everyone, administrator, hidden"))
	}

	var customAttr models.CustomAttribute
	if err := db.Where("scope = ? AND name = ?", scope, name).First(&customAttr).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Custom attribute not found", 404, fmt.Sprintf("No attribute found with scope '%s' and name '%s'", scope, name)))
	}

	if err := db.Model(&customAttr).Updates(models.CustomAttribute{
		DataType:    updateReq.DataType,
		Validation:  updateReq.Validation,
		Title:       updateReq.Title,
		Description: updateReq.Description,
		Visibility:  updateReq.Visibility,
	}).Error; err != nil {
		log.Error("Failed to update custom attribute:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to update attribute", 500, err.Error()))
	}

	db.Where("scope = ? AND name = ?", scope, name).First(&customAttr)

	return response.OK(customAttr)
}

// DeleteCustomAttribute deletes a custom attribute
// @Summary Delete custom attribute
// @Description Delete an existing custom attribute by scope and name
// @Tags Agent - Custom Attributes
// @Accept json
// @Produce json
// @Param scope path string true "Attribute scope"
// @Param name path string true "Attribute name"
// @Success 200 {object} response.Response
// @Router /api/agent/attributes/{scope}/{name} [delete]
func (ac AgentController) DeleteCustomAttribute(req *evo.Request) interface{} {
	scope := req.Param("scope").String()
	name := req.Param("name").String()

	if scope == "" || name == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Missing parameters", 400, "Scope and name are required"))
	}

	var customAttr models.CustomAttribute
	if err := db.Where("scope = ? AND name = ?", scope, name).First(&customAttr).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Custom attribute not found", 404, fmt.Sprintf("No attribute found with scope '%s' and name '%s'", scope, name)))
	}

	if err := db.Delete(&customAttr).Error; err != nil {
		log.Error("Failed to delete custom attribute:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to delete attribute", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"message": "Custom attribute deleted successfully",
		"scope":   scope,
		"name":    name,
	})
}
