package system

import (
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	natsconn "github.com/iesreza/homa-backend/apps/nats"
	"github.com/iesreza/homa-backend/lib/response"
)

type Controller struct {
}

func (c Controller) HealthHandler(request *evo.Request) any {
	return response.OK("ok")
}

func (c Controller) UptimeHandler(request *evo.Request) any {
	uptimeData := map[string]any{
		"uptime": int64(time.Now().Sub(StartupTime).Seconds()),
	}
	return response.OK(uptimeData)
}

// DepartmentListResponse defines the structure for the departments list response
type DepartmentListResponse []models.Department

// GetDepartments returns all available departments
// @Summary Get available departments
// @Description Get a list of all available departments
// @Tags System
// @Accept json
// @Produce json
// @Success 200 {array} models.Department
// @Router /api/system/departments [get]
func (c Controller) GetDepartments(req *evo.Request) interface{} {
	var departments []models.Department
	if err := db.Find(&departments).Error; err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	return response.List(departments, len(departments))
}

// TicketStatus represents a ticket status option
type TicketStatus struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// TicketStatusListResponse defines the structure for the ticket status list response
type TicketStatusListResponse []TicketStatus

// GetTicketStatuses returns all available ticket statuses
// @Summary Get ticket status list
// @Description Get a list of all available ticket statuses with their descriptions
// @Tags System
// @Accept json
// @Produce json
// @Success 200 {array} TicketStatus
// @Router /api/system/ticket-status [get]
func (c Controller) GetTicketStatuses(req *evo.Request) interface{} {
	statuses := []TicketStatus{
		{
			Value:       models.ConversationStatusNew,
			Label:       "New",
			Description: "Newly created ticket awaiting initial review",
		},
		{
			Value:       models.ConversationStatusWaitForAgent,
			Label:       "Wait for Agent",
			Description: "Ticket is waiting for agent response",
		},
		{
			Value:       models.ConversationStatusInProgress,
			Label:       "In Progress",
			Description: "Ticket is actively being worked on",
		},
		{
			Value:       models.ConversationStatusWaitForUser,
			Label:       "Wait for User",
			Description: "Ticket is waiting for user response",
		},
		{
			Value:       models.ConversationStatusOnHold,
			Label:       "On Hold",
			Description: "Ticket is temporarily on hold",
		},
		{
			Value:       models.ConversationStatusResolved,
			Label:       "Resolved",
			Description: "Ticket has been resolved",
		},
		{
			Value:       models.ConversationStatusClosed,
			Label:       "Closed",
			Description: "Ticket is closed and no further action needed",
		},
		{
			Value:       models.ConversationStatusUnresolved,
			Label:       "Unresolved",
			Description: "Ticket could not be resolved",
		},
		{
			Value:       models.ConversationStatusSpam,
			Label:       "Spam",
			Description: "Ticket marked as spam",
		},
	}

	return response.List(statuses, len(statuses))
}

func (c Controller) AdminMiddleware(request *evo.Request) error {
	if request.User().Anonymous() {
		return response.ErrForbidden
	}
	var user = request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAdministrator {
		return response.ErrForbidden
	}
	return request.Next()
}

func (c Controller) ServeDashboard(request *evo.Request) any {
	return request.SendFile("./static/dashboard/index.html")
}

func (c Controller) ServeLoginPage(request *evo.Request) any {
	return request.SendFile("./static/dashboard/login.html")
}

// SettingsResponse represents the response for settings endpoints
type SettingsResponse struct {
	Settings map[string]string `json:"settings"`
}

// SettingUpdateRequest represents a request to update settings
type SettingUpdateRequest struct {
	Settings map[string]string `json:"settings"`
}

// GetSettings returns all settings or settings by category
// @Summary Get settings
// @Description Get all settings or filter by category
// @Tags Settings
// @Accept json
// @Produce json
// @Param category query string false "Category filter (ai, workflow, general)"
// @Success 200 {object} SettingsResponse
// @Router /api/settings [get]
func (c Controller) GetSettings(req *evo.Request) interface{} {
	category := req.Query("category").String()

	var settings []models.Setting
	var err error

	if category != "" {
		settings, err = models.GetSettingsByCategory(category)
	} else {
		settings, err = models.GetAllSettings()
	}

	if err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Convert to map for easier frontend consumption
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	return response.OK(result)
}

// UpdateSettings updates multiple settings at once
// @Summary Update settings
// @Description Update one or more settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param body body SettingUpdateRequest true "Settings to update"
// @Success 200 {object} response.Response
// @Router /api/settings [put]
func (c Controller) UpdateSettings(req *evo.Request) interface{} {
	var request SettingUpdateRequest
	if err := req.BodyParser(&request); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if len(request.Settings) == 0 {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "No settings provided", 400))
	}

	// Update settings
	if err := models.BulkUpdateSettings(request.Settings); err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Publish NATS message if AI settings were updated
	for key := range request.Settings {
		if key == "ai.endpoint" || key == "ai.api_key" || key == "ai.model" {
			if natsconn.IsConnected() {
				if err := natsconn.Publish("settings.ai.reload", []byte("reload")); err != nil {
					log.Warning("Failed to publish AI settings reload: %v", err)
				} else {
					log.Info("Published AI settings reload message via NATS")
				}
			}
			break
		}
	}

	return response.OK("Settings updated successfully")
}

// GetSetting returns a single setting by key
// @Summary Get single setting
// @Description Get a single setting by key
// @Tags Settings
// @Accept json
// @Produce json
// @Param key path string true "Setting key"
// @Success 200 {object} models.Setting
// @Router /api/settings/{key} [get]
func (c Controller) GetSetting(req *evo.Request) interface{} {
	key := req.Param("key").String()
	if key == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Key is required", 400))
	}

	setting, err := models.GetSetting(key)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeNotFound, "Setting not found", 404))
	}

	return response.OK(setting)
}

// SetSetting creates or updates a single setting
// @Summary Set single setting
// @Description Create or update a single setting
// @Tags Settings
// @Accept json
// @Produce json
// @Param key path string true "Setting key"
// @Param body body models.Setting true "Setting data"
// @Success 200 {object} response.Response
// @Router /api/settings/{key} [put]
func (c Controller) SetSetting(req *evo.Request) interface{} {
	key := req.Param("key").String()
	if key == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Key is required", 400))
	}

	var setting models.Setting
	if err := req.BodyParser(&setting); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if err := models.SetSetting(key, setting.Value, setting.Type, setting.Category, setting.Label); err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	// Publish NATS message if AI settings were updated
	if key == "ai.endpoint" || key == "ai.api_key" || key == "ai.model" {
		if natsconn.IsConnected() {
			if err := natsconn.Publish("settings.ai.reload", []byte("reload")); err != nil {
				log.Warning("Failed to publish AI settings reload: %v", err)
			} else {
				log.Info("Published AI settings reload message via NATS")
			}
		}
	}

	return response.OK("Setting updated successfully")
}

// DeleteSetting deletes a setting by key
// @Summary Delete setting
// @Description Delete a setting by key
// @Tags Settings
// @Accept json
// @Produce json
// @Param key path string true "Setting key"
// @Success 200 {object} response.Response
// @Router /api/settings/{key} [delete]
func (c Controller) DeleteSetting(req *evo.Request) interface{} {
	key := req.Param("key").String()
	if key == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Key is required", 400))
	}

	if err := models.DeleteSetting(key); err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	return response.OK("Setting deleted successfully")
}

// GetActivityLogs returns activity logs with filtering and pagination
// @Summary Get activity logs
// @Description Get activity logs with optional filtering by entity type, entity ID, action, and user
// @Tags Activity Logs
// @Accept json
// @Produce json
// @Param entity_type query string false "Entity type filter (conversation, client, user, etc.)"
// @Param entity_id query string false "Entity ID filter"
// @Param action query string false "Action filter (create, update, delete, etc.)"
// @Param user_id query string false "User ID filter (UUID)"
// @Param limit query int false "Limit (default 50, max 100)"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} response.Response
// @Router /api/activity-logs [get]
func (c Controller) GetActivityLogs(req *evo.Request) interface{} {
	entityType := req.Query("entity_type").String()
	entityID := req.Query("entity_id").String()
	action := req.Query("action").String()
	userIDStr := req.Query("user_id").String()
	limit := req.Query("limit").Int()
	offset := req.Query("offset").Int()

	var userID *uuid.UUID
	if userIDStr != "" {
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Invalid user ID format", 400))
		}
		userID = &parsedID
	}

	logs, total, err := models.GetActivityLogs(entityType, entityID, action, userID, limit, offset)
	if err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	return response.ListWithTotal(logs, int(total))
}

// GetEntityActivityLogs returns activity logs for a specific entity
// @Summary Get entity activity logs
// @Description Get activity logs for a specific entity by type and ID
// @Tags Activity Logs
// @Accept json
// @Produce json
// @Param entity_type path string true "Entity type (conversation, client, user, etc.)"
// @Param entity_id path string true "Entity ID"
// @Param limit query int false "Limit (default 50)"
// @Success 200 {object} response.Response
// @Router /api/activity-logs/{entity_type}/{entity_id} [get]
func (c Controller) GetEntityActivityLogs(req *evo.Request) interface{} {
	entityType := req.Param("entity_type").String()
	entityID := req.Param("entity_id").String()
	limit := req.Query("limit").Int()

	if entityType == "" || entityID == "" {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Entity type and ID are required", 400))
	}

	logs, err := models.GetEntityActivityLogs(entityType, entityID, limit)
	if err != nil {
		return response.Error(response.ErrDatabaseError)
	}

	return response.List(logs, len(logs))
}
