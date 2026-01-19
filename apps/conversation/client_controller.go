package conversation

import (
	"encoding/json"
	"fmt"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/imageutil"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/datatypes"
)

// ListClients returns paginated list of clients with search and filtering
func (c AgentController) ListClients(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrForbidden)
	}

	var clients []models.Client
	query := db.Preload("ExternalIDs")

	search := request.Query("search").String()
	if search != "" {
		query = query.Where(
			"name LIKE ? OR id IN (SELECT client_id FROM client_external_ids WHERE value LIKE ?)",
			"%"+search+"%", "%"+search+"%",
		)
	}

	sortBy := request.Query("sort_by").String()
	sortOrder := request.Query("sort_order").String()
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	switch sortBy {
	case "name":
		query = query.Order(fmt.Sprintf("name %s", sortOrder))
	case "updated_at":
		query = query.Order(fmt.Sprintf("updated_at %s", sortOrder))
	default:
		query = query.Order(fmt.Sprintf("created_at %s", sortOrder))
	}

	page := request.Query("page").Int()
	if page < 1 {
		page = 1
	}
	limit := request.Query("limit").Int()
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	db.Model(&models.Client{}).Count(&total)

	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit)

	if err := query.Find(&clients).Error; err != nil {
		log.Error("Failed to list clients:", err)
		return response.Error(response.ErrInternalError)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return response.OKWithMeta(clients, &response.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetClient returns a single client by ID
func (c AgentController) GetClient(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrForbidden)
	}

	clientID := request.Param("id").String()
	if clientID == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var client models.Client
	if err := db.Preload("ExternalIDs").Where("id = ?", clientID).First(&client).Error; err != nil {
		return response.Error(response.ErrNotFound)
	}

	return response.OK(client)
}

// CreateClient creates a new client
func (c AgentController) CreateClient(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrForbidden)
	}

	var req struct {
		Name        string   `json:"name" validate:"required"`
		Language    *string  `json:"language"`
		Timezone    *string  `json:"timezone"`
		ExternalIDs []struct {
			Type  string `json:"type" validate:"required,oneof=email phone whatsapp slack telegram web chat"`
			Value string `json:"value" validate:"required"`
		} `json:"external_ids"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	client := models.Client{
		Name:     req.Name,
		Language: req.Language,
		Timezone: req.Timezone,
	}

	if err := db.Create(&client).Error; err != nil {
		log.Error("Failed to create client:", err)
		return response.Error(response.ErrInternalError)
	}

	for _, extID := range req.ExternalIDs {
		externalID := models.ClientExternalID{
			ClientID: client.ID,
			Type:     extID.Type,
			Value:    extID.Value,
		}
		if err := db.Create(&externalID).Error; err != nil {
			log.Error("Failed to create external ID:", err)
		}
	}

	db.Preload("ExternalIDs").First(&client, "id = ?", client.ID)

	return response.OK(client)
}

// UpdateClient updates an existing client
func (c AgentController) UpdateClient(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrForbidden)
	}

	clientID := request.Param("id").String()
	if clientID == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var client models.Client
	if err := db.Where("id = ?", clientID).First(&client).Error; err != nil {
		return response.Error(response.ErrNotFound)
	}

	var req struct {
		Name        *string                `json:"name"`
		Language    *string                `json:"language"`
		Timezone    *string                `json:"timezone"`
		Data        map[string]interface{} `json:"data"`
		ExternalIDs []struct {
			Type  string `json:"type" validate:"oneof=email phone whatsapp slack telegram web chat"`
			Value string `json:"value"`
		} `json:"external_ids"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.Name != nil {
		client.Name = *req.Name
	}
	if req.Language != nil {
		client.Language = req.Language
	}
	if req.Timezone != nil {
		client.Timezone = req.Timezone
	}

	if req.Data != nil {
		dataBytes, err := json.Marshal(req.Data)
		if err != nil {
			log.Error("Failed to marshal data:", err)
			return response.Error(response.ErrInvalidInput)
		}
		client.Data = datatypes.JSON(dataBytes)
	}

	if err := db.Save(&client).Error; err != nil {
		log.Error("Failed to update client:", err)
		return response.Error(response.ErrInternalError)
	}

	if req.ExternalIDs != nil {
		db.Where("client_id = ?", client.ID).Delete(&models.ClientExternalID{})

		for _, extID := range req.ExternalIDs {
			externalID := models.ClientExternalID{
				ClientID: client.ID,
				Type:     extID.Type,
				Value:    extID.Value,
			}
			if err := db.Create(&externalID).Error; err != nil {
				log.Error("Failed to create external ID:", err)
			}
		}
	}

	db.Preload("ExternalIDs").First(&client, "id = ?", client.ID)

	return response.OK(client)
}

// DeleteClient deletes a client
func (c AgentController) DeleteClient(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrForbidden)
	}

	clientID := request.Param("id").String()
	if clientID == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var client models.Client
	if err := db.Where("id = ?", clientID).First(&client).Error; err != nil {
		return response.Error(response.ErrNotFound)
	}

	db.Where("client_id = ?", client.ID).Delete(&models.ClientExternalID{})

	if err := db.Delete(&client).Error; err != nil {
		log.Error("Failed to delete client:", err)
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message": "Client deleted successfully",
		"id":      clientID,
	})
}

// UploadClientAvatar uploads and processes an avatar for a client
func (c AgentController) UploadClientAvatar(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrForbidden)
	}

	clientID := request.Param("id").String()
	if clientID == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var client models.Client
	if err := db.Where("id = ?", clientID).First(&client).Error; err != nil {
		return response.Error(response.ErrNotFound)
	}

	var req struct {
		Data string `json:"data" validate:"required"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.Data == "" {
		return response.Error(response.ErrInvalidInput)
	}

	if client.Avatar != nil && *client.Avatar != "" {
		if err := imageutil.DeleteAvatar(*client.Avatar); err != nil {
			log.Warning("Failed to delete old avatar:", err)
		}
	}

	avatarURL, err := imageutil.ProcessAvatarFromBase64(req.Data, "clients")
	if err != nil {
		log.Error("Failed to process avatar:", err)
		return response.Error(response.ErrInternalError)
	}

	client.Avatar = &avatarURL
	if err := db.Save(&client).Error; err != nil {
		log.Error("Failed to update client avatar:", err)
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"avatar": avatarURL,
	})
}

// DeleteClientAvatar removes the avatar from a client
func (c AgentController) DeleteClientAvatar(request *evo.Request) any {
	if request.User().Anonymous() {
		return response.Error(response.ErrUnauthorized)
	}

	user := request.User().Interface().(*auth.User)
	if user.Type != auth.UserTypeAgent && user.Type != auth.UserTypeAdministrator {
		return response.Error(response.ErrForbidden)
	}

	clientID := request.Param("id").String()
	if clientID == "" {
		return response.Error(response.ErrInvalidInput)
	}

	var client models.Client
	if err := db.Where("id = ?", clientID).First(&client).Error; err != nil {
		return response.Error(response.ErrNotFound)
	}

	if client.Avatar != nil && *client.Avatar != "" {
		if err := imageutil.DeleteAvatar(*client.Avatar); err != nil {
			log.Warning("Failed to delete avatar file:", err)
		}
	}

	client.Avatar = nil
	if err := db.Save(&client).Error; err != nil {
		log.Error("Failed to update client:", err)
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]interface{}{
		"message": "Avatar deleted successfully",
	})
}
