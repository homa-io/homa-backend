package admin

import (
	"encoding/json"
	"strings"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/pagination"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/gorm"

	"github.com/getevo/evo/v2"
)

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
