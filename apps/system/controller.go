package system

import (
	"github.com/iesreza/homa-backend/apps/auth"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/iesreza/homa-backend/apps/models"
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
