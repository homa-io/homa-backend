package inbox

import (
	"encoding/json"

	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

// ListInboxes returns all inboxes
// GET /api/admin/inboxes
func ListInboxes(r *evo.Request) any {
	inboxes, err := models.GetAllInboxes()
	if err != nil {
		return response.InternalError(nil, "Failed to get inboxes")
	}
	return response.OK(inboxes)
}

// GetInbox returns a single inbox by ID
// GET /api/admin/inboxes/:id
func GetInbox(r *evo.Request) any {
	id := r.Param("id").Uint()
	if id == 0 {
		return response.BadRequest(nil, "Invalid inbox ID")
	}

	inbox, err := models.GetInboxByID(id)
	if err != nil {
		return response.NotFound(nil, "Inbox not found")
	}

	return response.OK(inbox)
}

// CreateInboxRequest represents the request to create an inbox
type CreateInboxRequest struct {
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	SDKConfig           map[string]any `json:"sdk_config"`
	ConversationTimeout int            `json:"conversation_timeout"`
	Enabled             bool           `json:"enabled"`
}

// CreateInbox creates a new inbox
// POST /api/admin/inboxes
func CreateInbox(r *evo.Request) any {
	var req CreateInboxRequest
	if err := r.BodyParser(&req); err != nil {
		return response.BadRequest(nil, "Invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(nil, "Name is required")
	}

	// Convert sdk_config to JSON (default to empty object if not provided)
	var sdkConfig []byte
	if req.SDKConfig != nil {
		var err error
		sdkConfig, err = json.Marshal(req.SDKConfig)
		if err != nil {
			return response.BadRequest(nil, "Invalid SDK config")
		}
	} else {
		sdkConfig = []byte("{}")
	}

	inbox := &models.Inbox{
		Name:                req.Name,
		Description:         req.Description,
		SDKConfig:           sdkConfig,
		ConversationTimeout: req.ConversationTimeout,
		Enabled:             req.Enabled,
	}

	if err := models.CreateInbox(inbox); err != nil {
		return response.InternalError(nil, "Failed to create inbox")
	}

	return response.Created(inbox)
}

// UpdateInboxRequest represents the request to update an inbox
type UpdateInboxRequest struct {
	Name                *string        `json:"name"`
	Description         *string        `json:"description"`
	SDKConfig           map[string]any `json:"sdk_config"`
	ConversationTimeout *int           `json:"conversation_timeout"`
	Enabled             *bool          `json:"enabled"`
}

// UpdateInbox updates an existing inbox
// PUT /api/admin/inboxes/:id
func UpdateInbox(r *evo.Request) any {
	id := r.Param("id").Uint()
	if id == 0 {
		return response.BadRequest(nil, "Invalid inbox ID")
	}

	inbox, err := models.GetInboxByID(id)
	if err != nil {
		return response.NotFound(nil, "Inbox not found")
	}

	var req UpdateInboxRequest
	if err := r.BodyParser(&req); err != nil {
		return response.BadRequest(nil, "Invalid request body")
	}

	if req.Name != nil {
		inbox.Name = *req.Name
	}
	if req.Description != nil {
		inbox.Description = *req.Description
	}
	if req.SDKConfig != nil {
		sdkConfig, err := json.Marshal(req.SDKConfig)
		if err != nil {
			return response.BadRequest(nil, "Invalid SDK config")
		}
		inbox.SDKConfig = sdkConfig
	}
	if req.ConversationTimeout != nil {
		inbox.ConversationTimeout = *req.ConversationTimeout
	}
	if req.Enabled != nil {
		inbox.Enabled = *req.Enabled
	}

	if err := models.UpdateInbox(inbox); err != nil {
		return response.InternalError(nil, "Failed to update inbox")
	}

	return response.OK(inbox)
}

// DeleteInbox deletes an inbox
// DELETE /api/admin/inboxes/:id
func DeleteInbox(r *evo.Request) any {
	id := r.Param("id").Uint()
	if id == 0 {
		return response.BadRequest(nil, "Invalid inbox ID")
	}

	// Check if inbox exists
	_, err := models.GetInboxByID(id)
	if err != nil {
		return response.NotFound(nil, "Inbox not found")
	}

	// TODO: Check if inbox has conversations and handle accordingly
	// For now, we'll allow deletion

	if err := models.DeleteInbox(id); err != nil {
		return response.InternalError(nil, "Failed to delete inbox")
	}

	return response.OK(map[string]string{"message": "Inbox deleted successfully"})
}

// RegenerateAPIKey generates a new API key for an inbox
// POST /api/admin/inboxes/:id/regenerate-key
func RegenerateAPIKey(r *evo.Request) any {
	id := r.Param("id").Uint()
	if id == 0 {
		return response.BadRequest(nil, "Invalid inbox ID")
	}

	inbox, err := models.GetInboxByID(id)
	if err != nil {
		return response.NotFound(nil, "Inbox not found")
	}

	inbox.APIKey = models.GenerateInboxAPIKey()

	if err := models.UpdateInbox(inbox); err != nil {
		return response.InternalError(nil, "Failed to regenerate API key")
	}

	return response.OK(inbox)
}

// GetInboxByAPIKey returns an inbox by its API key (for SDK validation)
// GET /api/client/inbox/:key
func GetInboxByAPIKey(r *evo.Request) any {
	apiKey := r.Param("key").String()
	if apiKey == "" {
		return response.BadRequest(nil, "API key is required")
	}

	inbox, err := models.GetInboxByAPIKey(apiKey)
	if err != nil {
		return response.NotFound(nil, "Inbox not found")
	}

	if !inbox.Enabled {
		return response.Forbidden(nil, "Inbox is disabled")
	}

	// Return only public info for client SDK
	return response.OK(map[string]any{
		"id":         inbox.ID,
		"name":       inbox.Name,
		"sdk_config": inbox.SDKConfig,
	})
}
