package conversation

import (
	"fmt"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// AssignRequest represents the request body for assigning conversation
type AssignRequest struct {
	UserIDs      []string `json:"user_ids"`
	DepartmentID *uint    `json:"department_id"`
}

// AssignConversation handles the POST /api/agent/conversations/:id/assign endpoint
func (ac AgentController) AssignConversation(req *evo.Request) interface{} {
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var assignReq AssignRequest
	if err := req.BodyParser(&assignReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	var conversation models.Conversation
	if err := db.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	// Get old assignments with user names for action messages
	var oldAssignments []models.ConversationAssignment
	db.Preload("User").Where("conversation_id = ? AND user_id IS NOT NULL", conversationID).Find(&oldAssignments)
	oldUserNames := make(map[string]string)
	for _, a := range oldAssignments {
		if a.User != nil {
			name := a.User.Name
			if a.User.LastName != "" {
				name = a.User.Name + " " + a.User.LastName
			}
			oldUserNames[a.UserID.String()] = name
		}
	}

	// Get current user for action messages
	var actorName string
	var actorUserID *uuid.UUID
	if !req.User().Anonymous() {
		user := req.User().Interface().(*auth.User)
		actorName = user.Name
		if user.LastName != "" {
			actorName = user.Name + " " + user.LastName
		}
		actorUserID = &user.UserID
	}

	// Clear existing assignments
	if err := db.Where("conversation_id = ?", conversationID).Delete(&models.ConversationAssignment{}).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to clear assignments", 500, err.Error()))
	}

	// Collect new user info for action messages
	newUserNames := make(map[string]string)

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

		// Get user name for action message
		var user auth.User
		if err := db.Where("id = ?", userID).First(&user).Error; err == nil {
			name := user.Name
			if user.LastName != "" {
				name = user.Name + " " + user.LastName
			}
			newUserNames[userIDStr] = name
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

	// Create action messages for assignee changes
	go func() {
		for userIDStr, userName := range newUserNames {
			if _, existed := oldUserNames[userIDStr]; !existed {
				action := fmt.Sprintf(`assigned "%s" to the conversation`, userName)
				models.CreateActionMessage(conversationID, actorUserID, actorName, action)
			}
		}

		for userIDStr, userName := range oldUserNames {
			if _, exists := newUserNames[userIDStr]; !exists {
				action := fmt.Sprintf(`unassigned "%s" from the conversation`, userName)
				models.CreateActionMessage(conversationID, actorUserID, actorName, action)
			}
		}
	}()

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"assigned":        true,
	})
}

// UnassignConversation handles the DELETE /api/agent/conversations/:id/assign endpoint
func (ac AgentController) UnassignConversation(req *evo.Request) interface{} {
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	if err := db.Where("conversation_id = ?", conversationID).Delete(&models.ConversationAssignment{}).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to clear assignments", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"unassigned":      true,
	})
}
