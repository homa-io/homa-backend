package conversation

import (
	"strconv"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
	"github.com/google/uuid"
)

type Controller struct{}

// CreateConversationRequest represents the request structure for creating a conversation
type CreateConversationRequest struct {
	// Conversation fields
	Title        string         `json:"title" validate:"required,min=1,max=255" example:"Login Issue - Cannot access account"`
	DepartmentID *uint          `json:"department_id" example:"1"`
	ExternalID   *string        `json:"external_id" example:"EXT-123"`
	Status       string         `json:"status" validate:"required,oneof=new wait_for_agent in_progress wait_for_user on_hold resolved closed unresolved spam" example:"new"`
	Priority     string         `json:"priority" validate:"required,oneof=low medium high urgent" example:"medium"`
	Parameters   map[string]any `json:"parameters" swaggertype:"object" example:"issue_type:technical,urgency_level:2"` // Custom attributes for conversation

	// Client fields (for creating new clients)
	ClientName       *string        `json:"client_name" example:"John Doe"`                                                                                                                                 // If provided, creates a new client
	ClientEmail      *string        `json:"client_email" example:"john.doe@example.com"`                                                                                                                    // If provided, used as client email
	ClientID         *string        `json:"client_id" example:"c4ae2903-1127-4229-9e20-3225990af447"`                                                                                                       // If provided, uses existing client
	ClientAttributes map[string]any `json:"client_attributes" swaggertype:"object" example:"age:25,preferred_language:en,subscription_level:premium,annual_revenue:150000.50,registration_date:2024-01-15"` // Custom attributes for client

	// Message
	Message *string `json:"message" example:"I cannot log into my account. Getting error message 'Invalid credentials'."` // Optional initial message
}

// CreateConversationResponse represents the response structure for conversation creation (includes secret)
type CreateConversationResponse struct {
	models.Conversation
	Secret string `json:"secret"` // Include secret in creation response only
}

// AddClientMessageRequest represents the request structure for adding a client message via URL secret
type AddClientMessageRequest struct {
	Message string `json:"message" validate:"required,min=1"`
}

// GetConversationWithSecretResponse represents the response structure for getting conversation with secret
type GetConversationWithSecretResponse struct {
	Conversation  models.Conversation `json:"conversation"`
	Messages      []models.Message    `json:"messages"`
	TotalMessages int64               `json:"total_messages"`
	CurrentOffset int                 `json:"current_offset"`
	MessageLimit  int                 `json:"message_limit"`
}

// GetConversationDetailResponse represents the response structure for conversation detail with relations
type GetConversationDetailResponse struct {
	Conversation  models.Conversation            `json:"conversation"`
	Client        models.Client                  `json:"client"`
	Department    *models.Department             `json:"department"`
	Channel       models.Channel                 `json:"channel"`
	Tags          []models.Tag                   `json:"tags"`
	Assignments   []models.ConversationAssignment `json:"assignments"`
	Messages      []models.Message               `json:"messages"`
	TotalMessages int64                          `json:"total_messages"`
	CurrentPage   int                            `json:"current_page"`
	PageSize      int                            `json:"page_size"`
	TotalPages    int                            `json:"total_pages"`
}

// UpsertClientRequest represents the request structure for upserting a client
type UpsertClientRequest struct {
	Type       string         `json:"type" validate:"required,oneof=email phone whatsapp slack telegram web chat" example:"email"`
	Value      string         `json:"value" validate:"required,min=1" example:"john.doe@example.com"`
	Name       *string        `json:"name" example:"John Doe"`                                                                                                                                 // Optional client name
	Attributes map[string]any `json:"attributes" swaggertype:"object" example:"age:25,preferred_language:en,subscription_level:premium,annual_revenue:150000.50,registration_date:2024-01-15"` // Custom attributes
}

// CreateConversation creates a new conversation with custom attributes
// @Summary Create a new conversation
// @Description Create a new conversation with optional custom attributes and auto-generated secret. Channel is automatically set to 'web' for web interface. Supports client creation with custom attributes validation.
// @Tags Client Conversations
// @Accept json
// @Produce json
// @Param body body CreateConversationRequest true "Conversation data with optional client attributes"
// @Success 201 {object} CreateConversationResponse
// @Router /api/client/conversations [put]
func (c Controller) CreateConversation(req *evo.Request) interface{} {
	var input CreateConversationRequest
	if err := req.BodyParser(&input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	// Handle client creation or lookup
	var clientID uuid.UUID
	var err error

	if input.ClientID != nil && *input.ClientID != "" {
		// Use existing client
		clientID, err = uuid.Parse(*input.ClientID)
		if err != nil {
			invalidClientIDErr := response.NewError(response.ErrorCodeInvalidInput, "Invalid client ID format", 400)
			return response.Error(invalidClientIDErr)
		}
	} else if input.ClientName != nil && *input.ClientName != "" {
		// Upsert client based on email if provided, otherwise use name
		var client *models.Client
		if input.ClientEmail != nil && *input.ClientEmail != "" {
			// Use email to upsert client with attributes support
			client, err = UpsertClientWithAttributes("email", *input.ClientEmail, input.ClientName, input.ClientAttributes)
			if err != nil {
				log.Error("Failed to upsert client by email:", err)
				clientUpsertErr := response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to upsert client", 400, err.Error())
				return response.Error(clientUpsertErr)
			}
		} else {
			// Create new client using just the name and attributes
			clientInput := ClientInput{
				Name:       *input.ClientName,
				Parameters: input.ClientAttributes,
			}
			client, err = CreateClient(clientInput)
			if err != nil {
				log.Error("Failed to create client:", err)
				clientCreationErr := response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to create client", 400, err.Error())
				return response.Error(clientCreationErr)
			}
		}
		clientID = client.ID
	} else {
		missingClientErr := response.NewError(response.ErrorCodeMissingRequired, "Either client_id or client_name must be provided", 400)
		return response.Error(missingClientErr)
	}

	// Create conversation input (hardcode channel_id as "web" for web interface)
	conversationInput := ConversationInput{
		Title:        input.Title,
		ClientID:     clientID,
		DepartmentID: input.DepartmentID,
		ChannelID:    "web",
		ExternalID:   input.ExternalID,
		Status:       input.Status,
		Priority:     input.Priority,
		Parameters:   input.Parameters,
		Message:      input.Message,
	}

	// Create the conversation using the business logic function
	conversation, secret, err := CreateConversation(conversationInput)
	if err != nil {
		log.Error("Failed to create conversation:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to create conversation", 500, err.Error()))
	}

	// Load related data for response
	if err := db.Preload("Client").Preload("Department").Preload("Channel").First(conversation, conversation.ID).Error; err != nil {
		log.Warning("Failed to preload conversation relations:", err)
	}

	// Create response with secret included
	conversationResponse := CreateConversationResponse{
		Conversation: *conversation,
		Secret: secret,
	}

	return response.Created(conversationResponse)
}

// AddClientMessage adds a message to a conversation using URL-based secret authentication
// @Summary Add client message to ticket with URL secret
// @Description Add a message to a conversation using conversation ID and secret in URL path
// @Tags Client Conversations
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param secret path string true "Conversation secret"
// @Param body body AddClientMessageRequest true "Message data"
// @Success 201 {object} models.Message
// @Router /api/client/conversations/{conversation_id}/{secret}/messages [post]
func (c Controller) AddClientMessage(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	// Get secret from URL
	secret := req.Param("secret").String()
	if secret == "" {
		missingSecretErr := response.NewError(response.ErrorCodeMissingRequired, "Secret is required in URL", 400)
		return response.Error(missingSecretErr)
	}

	var input AddClientMessageRequest
	if err := req.BodyParser(&input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	// Validate input
	if err := validate.Struct(input); err != nil {
		validationErr := response.NewErrorWithDetails(response.ErrorCodeValidationError, "Validation failed", 400, err.Error())
		return response.Error(validationErr)
	}

	// Find conversation and verify secret
	var conversation models.Conversation
	if err := db.First(&conversation, uint(conversationID)).Error; err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Verify secret matches
	if conversation.Secret != secret {
		unauthorizedErr := response.NewError(response.ErrorCodeUnauthorized, "Invalid secret", 401)
		return response.Error(unauthorizedErr)
	}

	// Create message with conversation.ClientID as sender (recognizing client as opener of conversation)
	message := models.Message{
		ConversationID:        conversation.ID,
		ClientID:        &conversation.ClientID, // Message sender is the conversation opener
		Body:            input.Message,
		IsSystemMessage: false,
	}

	if err := db.Create(&message).Error; err != nil {
		log.Error("Failed to create client message:", err)
		return response.Error(response.ErrCreateMessage())
	}

	// Load related data for response
	if err := db.Preload("Conversation").Preload("Client").First(&message, message.ID).Error; err != nil {
		log.Warning("Failed to preload message relations:", err)
	}

	return response.Created(message)
}

// GetConversationWithSecret retrieves ticket messages using secret authentication
// @Summary Get conversation messages with secret
// @Description Retrieve conversation messages using conversation ID and secret for client authentication
// @Tags Client Conversations
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param secret path string true "Conversation secret"
// @Param offset query int false "Message offset for pagination (default: 0)"
// @Param limit query int false "Message limit for pagination (default: 20, max: 100)"
// @Success 200 {object} GetConversationWithSecretResponse
// @Router /api/client/conversations/{conversation_id}/{secret} [get]
func (c Controller) GetConversationWithSecret(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	// Get secret from URL
	secret := req.Param("secret").String()
	if secret == "" {
		return response.Error(response.NewError(response.ErrorCodeMissingRequired, "Secret is required in URL", 400))
	}

	// Parse pagination parameters
	offset := req.Query("offset").Int()
	if offset < 0 {
		offset = 0
	}

	limit := req.Query("limit").Int()
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Find conversation and verify secret
	var conversation models.Conversation
	if err := db.Preload("Client").Preload("Department").Preload("Channel").
		Preload("Tags").Preload("Assignments").
		First(&conversation, uint(conversationID)).Error; err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Verify secret matches
	if conversation.Secret != secret {
		return response.Error(response.NewError(response.ErrorCodeUnauthorized, "Invalid secret", 401))
	}

	// Count total messages for this conversation
	var totalMessages int64
	if err := db.Model(&models.Message{}).Where("conversation_id = ?", conversation.ID).Count(&totalMessages).Error; err != nil {
		log.Error("Failed to count messages:", err)
		return response.Error(response.NewError(response.ErrorCodeDatabaseError, "Failed to count messages", 500))
	}

	// Get messages with pagination, ordered by created_at ASC, with all associations preloaded
	var messages []models.Message
	if err := db.Preload("Conversation").Preload("Client").Preload("User").
		Where("conversation_id = ?", conversation.ID).
		Order("created_at ASC").
		Offset(offset).Limit(limit).
		Find(&messages).Error; err != nil {
		log.Error("Failed to fetch messages:", err)
		return response.Error(response.NewError(response.ErrorCodeDatabaseError, "Failed to fetch messages", 500))
	}

	// Create meta for pagination
	meta := &response.Meta{
		Total:  totalMessages,
		Offset: offset,
		Count:  len(messages),
		Limit:  limit,
	}

	// Create response data
	responseData := map[string]interface{}{
		"conversation":   conversation,
		"messages": messages,
	}

	return response.OKWithMeta(responseData, meta)
}

// CloseConversationWithSecret closes a conversation using secret authentication
// @Summary Close conversation with secret
// @Description Close a conversation (set status to closed) using conversation ID and secret for client authentication
// @Tags Client Conversations
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param secret path string true "Conversation secret"
// @Success 200 {object} models.Conversation
// @Router /api/client/conversations/{conversation_id}/{secret} [delete]
func (c Controller) CloseConversationWithSecret(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	// Get secret from URL
	secret := req.Param("secret").String()
	if secret == "" {
		return response.Error(response.NewError(response.ErrorCodeMissingRequired, "Secret is required in URL", 400))
	}

	// Find conversation and verify secret
	var conversation models.Conversation
	if err := db.First(&conversation, uint(conversationID)).Error; err != nil {
		return response.Error(response.ErrConversationNotFound)
	}

	// Verify secret matches
	if conversation.Secret != secret {
		return response.Error(response.NewError(response.ErrorCodeUnauthorized, "Invalid secret", 401))
	}

	// Update conversation to closed status using the business logic function
	updatedConversation, err := UpdateConversation(uint(conversationID), ConversationInput{
		Status: "closed",
	})
	if err != nil {
		log.Error("Failed to close conversation:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to close conversation", 500, err.Error()))
	}

	// Load related data for response
	if err := db.Preload("Client").Preload("Department").Preload("Channel").First(updatedConversation, updatedConversation.ID).Error; err != nil {
		log.Warning("Failed to preload conversation relations:", err)
	}

	return response.OKWithMessage(updatedConversation, "Conversation closed successfully")
}

// UpsertClient creates or returns an existing client based on type and value
// @Summary Upsert a client
// @Description Create a new client if it doesn't exist, or return existing client if it does. Supports custom attributes validation and storage.
// @Tags Clients
// @Accept json
// @Produce json
// @Param body body UpsertClientRequest true "Client type, value, optional name and custom attributes"
// @Success 200 {object} models.Client
// @Router /api/client/upsert [put]
func (c Controller) UpsertClient(req *evo.Request) interface{} {
	var input UpsertClientRequest
	if err := req.BodyParser(&input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	// Validate input
	if err := validate.Struct(input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeValidationError, "Validation failed", 400, err.Error()))
	}

	// Upsert the client with attributes support
	client, err := UpsertClientWithAttributes(input.Type, input.Value, input.Name, input.Attributes)
	if err != nil {
		log.Error("Failed to upsert client:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to upsert client", 400, err.Error()))
	}

	// Load related data for response
	if err := db.Preload("ExternalIDs").First(client, client.ID).Error; err != nil {
		log.Warning("Failed to preload client relations:", err)
	}

	return response.OK(client)
}

// AssignToUserRequest represents the request structure for assigning a conversation to a user
type AssignToUserRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// AssignToDepartmentRequest represents the request structure for assigning a conversation to a department
type AssignToDepartmentRequest struct {
	DepartmentID uint `json:"department_id" validate:"required,min=1"`
}

// AssignConversationToUser assigns a conversation to a specific user
// @Summary Assign conversation to user
// @Description Assign a conversation to a specific user for handling
// @Tags Admin - Conversation Assignments
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param body body AssignToUserRequest true "User assignment data"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/conversations/{conversation_id}/assign/user [post]
func (c Controller) AssignConversationToUser(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	var input AssignToUserRequest
	if err := req.BodyParser(&input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	// Validate input
	if err := validate.Struct(input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeValidationError, "Validation failed", 400, err.Error()))
	}

	// Parse user ID
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return response.Error(response.ErrInvalidUserID)
	}

	// Assign the conversation
	assignment, err := AssignConversationToUser(uint(conversationID), userID)
	if err != nil {
		log.Error("Failed to assign conversation to user:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to assign conversation to user", 400, err.Error()))
	}

	return response.Created(assignment)
}

// AssignConversationToDepartment assigns a conversation to a department
// @Summary Assign conversation to department
// @Description Assign a conversation to a department for handling
// @Tags Admin - Conversation Assignments
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param body body AssignToDepartmentRequest true "Department assignment data"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/conversations/{conversation_id}/assign/department [post]
func (c Controller) AssignConversationToDepartment(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	var input AssignToDepartmentRequest
	if err := req.BodyParser(&input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request format", 400, err.Error()))
	}

	// Validate input
	if err := validate.Struct(input); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeValidationError, "Validation failed", 400, err.Error()))
	}

	// Assign the conversation
	assignment, err := AssignConversationToDepartment(uint(conversationID), input.DepartmentID)
	if err != nil {
		log.Error("Failed to assign conversation to department:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to assign conversation to department", 400, err.Error()))
	}

	return response.Created(assignment)
}

// UnassignConversation removes all assignments from a conversation
// @Summary Unassign conversation
// @Description Remove all assignments from a conversation
// @Tags Admin - Conversation Assignments
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/conversations/{conversation_id}/unassign [delete]
func (c Controller) UnassignConversation(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	// Unassign the conversation
	if err := UnassignConversation(uint(conversationID)); err != nil {
		log.Error("Failed to unassign conversation:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to unassign conversation", 400, err.Error()))
	}

	return response.Message("Conversation unassigned successfully")
}

// GetConversationAssignments returns all assignments for a conversation
// @Summary Get conversation assignments
// @Description Get all assignments for a specific conversation
// @Tags Admin - Conversation Assignments
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Success 200 {object} []models.ConversationAssignment
// @Router /api/admin/conversations/{conversation_id}/assignments [get]
func (c Controller) GetConversationAssignments(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	// Get assignments
	assignments, err := GetConversationAssignments(uint(conversationID))
	if err != nil {
		log.Error("Failed to get conversation assignments:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get conversation assignments", 500, err.Error()))
	}

	return response.OK(assignments)
}

// GetConversationDetail returns a conversation with all relations and paginated messages
// @Summary Get conversation detail with relations
// @Description Get a single conversation with all related data (client, department, channel, tags, assignments) and paginated messages. Requires authentication.
// @Tags Admin - Conversations
// @Accept json
// @Produce json
// @Param conversation_id path int true "Conversation ID"
// @Param page query int false "Page number for messages pagination (default: 1)"
// @Param page_size query int false "Page size for messages pagination (default: 20, max: 100)"
// @Success 200 {object} GetConversationDetailResponse
// @Router /api/admin/conversations/{conversation_id} [get]
func (c Controller) GetConversationDetail(req *evo.Request) interface{} {
	// Parse conversation ID
	conversationIDStr := req.Param("conversation_id").String()
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		return response.Error(response.ErrInvalidConversationID)
	}

	// Get pagination parameters with defaults
	// Default to showing all messages (high limit) for conversation detail view
	page := req.Query("page").Int()
	if page < 1 {
		page = 1
	}

	pageSize := req.Query("page_size").Int()
	if pageSize < 1 {
		pageSize = 1000 // Default to showing all messages
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get conversation with all relations
	var conversation models.Conversation
	if err := db.Preload("Client").
		Preload("Department").
		Preload("Channel").
		Preload("Tags").
		Preload("Assignments").
		Preload("Assignments.User").
		First(&conversation, uint(conversationID)).Error; err != nil {
		log.Error("Failed to get conversation:", err)
		return response.Error(response.ErrConversationNotFound)
	}

	// Get total message count
	var totalMessages int64
	if err := db.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&totalMessages).Error; err != nil {
		log.Error("Failed to count messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to count messages", 500, err.Error()))
	}

	// Get paginated messages
	var messages []models.Message
	if err := db.Where("conversation_id = ?", conversationID).
		Preload("User").
		Preload("Client").
		Order("created_at ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&messages).Error; err != nil {
		log.Error("Failed to get messages:", err)
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeDatabaseError, "Failed to get messages", 500, err.Error()))
	}

	// Calculate total pages
	totalPages := int(totalMessages) / pageSize
	if int(totalMessages)%pageSize != 0 {
		totalPages++
	}

	// Build response
	resp := GetConversationDetailResponse{
		Conversation:  conversation,
		Client:        conversation.Client,
		Department:    conversation.Department,
		Channel:       conversation.Channel,
		Tags:          conversation.Tags,
		Assignments:   conversation.Assignments,
		Messages:      messages,
		TotalMessages: totalMessages,
		CurrentPage:   page,
		PageSize:      pageSize,
		TotalPages:    totalPages,
	}

	return response.OK(resp)
}
