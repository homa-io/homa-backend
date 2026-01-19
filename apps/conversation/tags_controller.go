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

// GetTags handles the GET /api/agent/tags endpoint
// @Summary Get tags list
// @Description Get list of all available tags with usage statistics
// @Tags Agent - Tags
// @Accept json
// @Produce json
// @Success 200 {object} []TagWithUsage
// @Router /api/agent/tags [get]
func (ac AgentController) GetTags(req *evo.Request) interface{} {
	type TagWithUsage struct {
		ID         uint   `json:"id"`
		Name       string `json:"name"`
		Color      string `json:"color"`
		UsageCount int64  `json:"usage_count"`
	}

	var result []TagWithUsage
	if err := db.Raw(`
		SELECT t.id, t.name, '#4ECDC4' as color, COUNT(ct.tag_id) as usage_count
		FROM tags t
		LEFT JOIN conversation_tags ct ON t.id = ct.tag_id
		GROUP BY t.id, t.name
		ORDER BY t.name
	`).Scan(&result).Error; err != nil {
		log.Error("Failed to get tags:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get tags", 500, err.Error()))
	}

	return response.OK(result)
}

// CreateTag handles the POST /api/agent/tags endpoint
func (ac AgentController) CreateTag(req *evo.Request) interface{} {
	type CreateTagRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var createReq CreateTagRequest
	if err := req.BodyParser(&createReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if createReq.Name == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Tag name is required", 400, "Tag name cannot be empty"))
	}

	var existingTag models.Tag
	if err := db.Where("name = ?", createReq.Name).First(&existingTag).Error; err == nil {
		return response.OK(map[string]interface{}{
			"id":    existingTag.ID,
			"name":  existingTag.Name,
			"color": "#4ECDC4",
		})
	}

	tag := models.Tag{
		Name: createReq.Name,
	}

	if err := db.Create(&tag).Error; err != nil {
		log.Error("Failed to create tag:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to create tag", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"id":    tag.ID,
		"name":  tag.Name,
		"color": "#4ECDC4",
	})
}

// UpdateTagsRequest represents the request body for updating tags
type UpdateTagsRequest struct {
	TagIDs []uint `json:"tag_ids"`
}

// UpdateConversationTags handles the PUT /api/agent/conversations/:id/tags endpoint
func (ac AgentController) UpdateConversationTags(req *evo.Request) interface{} {
	conversationID := req.Param("id").Uint()
	if conversationID == 0 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid conversation ID", 400, "Conversation ID must be a positive integer"))
	}

	var tagsReq UpdateTagsRequest
	if err := req.BodyParser(&tagsReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	var conversation models.Conversation
	if err := db.Preload("Tags").Where("id = ?", conversationID).First(&conversation).Error; err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeNotFound, "Conversation not found", 404, fmt.Sprintf("No conversation exists with ID %d", conversationID)))
	}

	// Store old tag names for action message
	oldTagNames := make(map[uint]string)
	for _, tag := range conversation.Tags {
		oldTagNames[tag.ID] = tag.Name
	}

	// Get current user for action messages
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

	// Get tags
	var tags []models.Tag
	if len(tagsReq.TagIDs) > 0 {
		if err := db.Where("id IN ?", tagsReq.TagIDs).Find(&tags).Error; err != nil {
			return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to fetch tags", 500, err.Error()))
		}
	}

	// Replace tags
	if err := db.Model(&conversation).Association("Tags").Replace(tags); err != nil {
		log.Error("Failed to update conversation tags: ", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to update tags", 500, err.Error()))
	}

	// Create action messages for tag changes
	go func() {
		newTagIDs := make(map[uint]bool)
		for _, tag := range tags {
			newTagIDs[tag.ID] = true
		}

		var addedTags []string
		for _, tag := range tags {
			if _, existed := oldTagNames[tag.ID]; !existed {
				addedTags = append(addedTags, tag.Name)
			}
		}

		var removedTags []string
		for id, name := range oldTagNames {
			if !newTagIDs[id] {
				removedTags = append(removedTags, name)
			}
		}

		for _, tagName := range addedTags {
			action := fmt.Sprintf(`added tag "%s"`, tagName)
			models.CreateActionMessage(conversationID, userID, actorName, action)
		}
		for _, tagName := range removedTags {
			action := fmt.Sprintf(`removed tag "%s"`, tagName)
			models.CreateActionMessage(conversationID, userID, actorName, action)
		}
	}()

	// Return updated tags
	var updatedTags []TagInfo
	for _, tag := range tags {
		updatedTags = append(updatedTags, TagInfo{
			ID:   tag.ID,
			Name: tag.Name,
		})
	}

	return response.OK(map[string]interface{}{
		"conversation_id": conversationID,
		"tags":            updatedTags,
	})
}
