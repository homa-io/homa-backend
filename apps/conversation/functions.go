package conversation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ConversationInput represents the input structure for creating or updating conversations
type ConversationInput struct {
	Title           string         `json:"title" validate:"min=1,max=255"`
	ClientID        uuid.UUID      `json:"client_id" validate:"required"`
	DepartmentID    *uint          `json:"department_id"`
	ChannelID       string         `json:"channel_id" validate:"required"`
	ExternalID      *string        `json:"external_id"`
	Status          string         `json:"status" validate:"oneof=new wait_for_agent in_progress wait_for_user on_hold resolved closed unresolved spam"`
	Priority        string         `json:"priority" validate:"oneof=low medium high urgent"`
	Parameters      map[string]any `json:"parameters"` // Custom attributes
	Message         *string        `json:"message"`    // Optional initial message
	IP              *string        `json:"ip"`         // Client IP address
	Browser         *string        `json:"browser"`    // Browser name and version
	OperatingSystem *string        `json:"operating_system"` // OS name and version
}

// ClientInput represents the input structure for creating or updating clients
type ClientInput struct {
	Name       string         `json:"name" validate:"required,min=1,max=255"`
	Parameters map[string]any `json:"parameters"` // Custom attributes
}

var validate = validator.New()

// generateSecret generates a random 32-character hexadecimal secret
func generateSecret() string {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CreateConversation creates a new conversation with custom attribute validation and processing
func CreateConversation(input ConversationInput) (*models.Conversation, string, error) {
	// Validate basic conversation fields
	if err := validate.Struct(input); err != nil {
		return nil, "", fmt.Errorf("validation error: %w", err)
	}

	// Process and validate custom attributes
	customFields, err := processCustomAttributes(models.CustomAttributeScopeConversation, input.Parameters)
	if err != nil {
		return nil, "", fmt.Errorf("custom attributes error: %w", err)
	}

	// Ensure customFields is not nil
	if customFields == nil {
		customFields = datatypes.JSON("{}")
	}

	// Generate secret automatically
	secret := generateSecret()

	// Create conversation
	conversation := models.Conversation{
		Title:           input.Title,
		ClientID:        input.ClientID,
		DepartmentID:    input.DepartmentID,
		ChannelID:       input.ChannelID,
		ExternalID:      input.ExternalID,
		Secret:          secret,
		Status:          input.Status,
		Priority:        input.Priority,
		CustomFields:    customFields,
		IP:              input.IP,
		Browser:         input.Browser,
		OperatingSystem: input.OperatingSystem,
	}

	// Save to database
	if err := db.Create(&conversation).Error; err != nil {
		log.Error("Failed to create conversation:", err)
		return nil, "", fmt.Errorf("failed to create conversation: %w", err)
	}

	// Create initial message if provided
	if input.Message != nil && *input.Message != "" {
		message := models.Message{
			ConversationID:        conversation.ID,
			ClientID:        &input.ClientID,
			Body:            *input.Message,
			IsSystemMessage: false,
		}

		if err := db.Create(&message).Error; err != nil {
			log.Error("Failed to create initial message:", err)
			// Don't fail the conversation creation if message creation fails
			// Just log the error
		}
	}

	return &conversation, secret, nil
}

// UpdateConversation updates an existing ticket with custom attribute validation and processing
func UpdateConversation(conversationID uint, input ConversationInput) (*models.Conversation, error) {
	// Find existing conversation
	var conversation models.Conversation
	if err := db.First(&conversation, conversationID).Error; err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Validate basic conversation fields (skip required validation for updates)
	validateStruct := struct {
		Title     string `validate:"omitempty,min=1,max=255"`
		ClientID  string `validate:"omitempty,uuid"`
		ChannelID string `validate:"omitempty,min=1"`
		Status    string `validate:"omitempty,oneof=new wait_for_agent in_progress wait_for_user on_hold resolved closed unresolved spam"`
		Priority  string `validate:"omitempty,oneof=low medium high urgent"`
	}{
		Title:     input.Title,
		ClientID:  input.ClientID.String(),
		ChannelID: input.ChannelID,
		Status:    input.Status,
		Priority:  input.Priority,
	}

	if err := validate.Struct(validateStruct); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Process custom attributes if provided
	var customFields datatypes.JSON
	if input.Parameters != nil {
		var err error
		customFields, err = processCustomAttributes(models.CustomAttributeScopeConversation, input.Parameters)
		if err != nil {
			return nil, fmt.Errorf("custom attributes error: %w", err)
		}

		// Ensure customFields is not nil
		if customFields == nil {
			customFields = datatypes.JSON("{}")
		}
	}

	// Update conversation fields
	updates := make(map[string]interface{})
	if input.Title != "" {
		updates["title"] = input.Title
	}
	if input.ClientID != (uuid.UUID{}) {
		updates["client_id"] = input.ClientID
	}
	if input.DepartmentID != nil {
		updates["department_id"] = input.DepartmentID
	}
	if input.ChannelID != "" {
		updates["channel_id"] = input.ChannelID
	}
	if input.ExternalID != nil {
		updates["external_id"] = input.ExternalID
	}
	if input.Status != "" {
		updates["status"] = input.Status
	}
	if input.Priority != "" {
		updates["priority"] = input.Priority
	}
	if customFields != nil {
		updates["custom_fields"] = customFields
	}

	// Update closed_at timestamp if status is being set to closed
	if input.Status == models.ConversationStatusClosed {
		now := time.Now()
		updates["closed_at"] = &now
	} else if input.Status != "" && input.Status != models.ConversationStatusClosed {
		// Clear closed_at if status is changed from closed to something else
		updates["closed_at"] = nil
	}

	// Save updates to database
	if err := db.Model(&conversation).Updates(updates).Error; err != nil {
		log.Error("Failed to update conversation:", err)
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	// Reload conversation to get updated values
	if err := db.First(&conversation, conversationID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload conversation: %w", err)
	}

	return &conversation, nil
}

// CreateClient creates a new client with custom attribute validation and processing
func CreateClient(input ClientInput) (*models.Client, error) {
	// Validate basic client fields
	if err := validate.Struct(input); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Process and validate custom attributes
	data, err := processCustomAttributes(models.CustomAttributeScopeClient, input.Parameters)
	if err != nil {
		return nil, fmt.Errorf("custom attributes error: %w", err)
	}

	// Ensure data is not nil
	if data == nil {
		data = datatypes.JSON("{}")
	}

	// Create client
	client := models.Client{
		Name: input.Name,
		Data: data,
	}

	// Save to database
	if err := db.Create(&client).Error; err != nil {
		log.Error("Failed to create client:", err)
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &client, nil
}

// UpdateClient updates an existing client with custom attribute validation and processing
func UpdateClient(clientID uuid.UUID, input ClientInput) (*models.Client, error) {
	// Find existing client
	var client models.Client
	if err := db.First(&client, "id = ?", clientID).Error; err != nil {
		return nil, fmt.Errorf("client not found: %w", err)
	}

	// Validate basic client fields
	validateStruct := struct {
		Name string `validate:"omitempty,min=1,max=255"`
	}{
		Name: input.Name,
	}

	if err := validate.Struct(validateStruct); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Process custom attributes if provided
	var data datatypes.JSON
	if input.Parameters != nil {
		var err error
		data, err = processCustomAttributes(models.CustomAttributeScopeClient, input.Parameters)
		if err != nil {
			return nil, fmt.Errorf("custom attributes error: %w", err)
		}
	}

	// Update client fields
	updates := make(map[string]interface{})
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if data != nil {
		updates["data"] = data
	}

	// Save updates to database
	if err := db.Model(&client).Updates(updates).Error; err != nil {
		log.Error("Failed to update client:", err)
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	// Reload client to get updated values
	if err := db.First(&client, "id = ?", clientID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload client: %w", err)
	}

	return &client, nil
}

// processCustomAttributes validates and processes custom attributes according to their definitions
func processCustomAttributes(scope string, parameters map[string]any) (datatypes.JSON, error) {
	if parameters == nil || len(parameters) == 0 {
		return nil, nil
	}

	// Get custom attribute definitions for this scope
	var customAttrs []models.CustomAttribute
	if err := db.Where("scope = ?", scope).Find(&customAttrs).Error; err != nil {
		return nil, fmt.Errorf("failed to load custom attributes: %w", err)
	}

	// Create a map for quick lookup
	attrMap := make(map[string]models.CustomAttribute)
	for _, attr := range customAttrs {
		attrMap[attr.Name] = attr
	}

	// Process each parameter
	result := make(map[string]interface{})
	for key, value := range parameters {
		attr, exists := attrMap[key]
		if !exists {
			// Allow arbitrary custom attributes as strings (for flexible widget integration)
			// Unknown attributes are stored as-is without strict type validation
			result[key] = value
			continue
		}

		// Cast and validate value according to data type for known attributes
		castedValue, err := castAndValidateValue(attr, value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for attribute %s: %w", key, err)
		}

		result[key] = castedValue
	}

	// Convert to JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom attributes: %w", err)
	}

	return datatypes.JSON(jsonData), nil
}

// castAndValidateValue casts a value to the correct type and validates it
func castAndValidateValue(attr models.CustomAttribute, value interface{}) (interface{}, error) {
	// Handle nil values
	if value == nil {
		return nil, nil
	}

	// Cast according to data type
	var castedValue interface{}
	var err error

	switch attr.DataType {
	case models.CustomAttributeDataTypeInt:
		castedValue, err = castToInt(value)
	case models.CustomAttributeDataTypeFloat:
		castedValue, err = castToFloat(value)
	case models.CustomAttributeDataTypeDate:
		castedValue, err = castToDate(value)
	case models.CustomAttributeDataTypeString:
		castedValue = fmt.Sprintf("%v", value)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", attr.DataType)
	}

	if err != nil {
		return nil, err
	}

	// Apply validation if specified
	if attr.Validation != nil && *attr.Validation != "" {
		if err := validateValueWithRules(castedValue, *attr.Validation); err != nil {
			return nil, err
		}
	}

	return castedValue, nil
}

// castToInt converts a value to int
func castToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// castToFloat converts a value to float64
func castToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// castToDate converts a value to time.Time
func castToDate(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		// Try multiple date formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("invalid date format: %s", v)
	case time.Time:
		return v, nil
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", value)
	}
}

// validateValueWithRules validates a value against validation rules
// Supports rules: min, max, pattern, required (comma-separated)
// Example: "min:1,max:100,pattern:^[a-zA-Z]+$"
func validateValueWithRules(value interface{}, rules string) error {
	if rules == "" {
		return nil
	}

	// Parse rules (comma-separated)
	ruleList := strings.Split(rules, ",")

	for _, rule := range ruleList {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		// Handle "required" rule
		if rule == "required" {
			if value == nil {
				return fmt.Errorf("value is required")
			}
			if str, ok := value.(string); ok && str == "" {
				return fmt.Errorf("value is required")
			}
			continue
		}

		// Parse "key:value" format
		parts := strings.SplitN(rule, ":", 2)
		if len(parts) != 2 {
			log.Warning("Invalid validation rule format: %s", rule)
			continue
		}

		ruleName := strings.TrimSpace(parts[0])
		ruleValue := strings.TrimSpace(parts[1])

		switch ruleName {
		case "min":
			minVal, err := strconv.ParseFloat(ruleValue, 64)
			if err != nil {
				log.Warning("Invalid min value in validation rule: %s", ruleValue)
				continue
			}
			if err := validateMin(value, minVal); err != nil {
				return err
			}

		case "max":
			maxVal, err := strconv.ParseFloat(ruleValue, 64)
			if err != nil {
				log.Warning("Invalid max value in validation rule: %s", ruleValue)
				continue
			}
			if err := validateMax(value, maxVal); err != nil {
				return err
			}

		case "pattern":
			if str, ok := value.(string); ok {
				matched, err := regexp.MatchString(ruleValue, str)
				if err != nil {
					log.Warning("Invalid regex pattern in validation rule: %s", ruleValue)
					continue
				}
				if !matched {
					return fmt.Errorf("value does not match required pattern")
				}
			}

		case "minlen":
			minLen, err := strconv.Atoi(ruleValue)
			if err != nil {
				log.Warning("Invalid minlen value in validation rule: %s", ruleValue)
				continue
			}
			if str, ok := value.(string); ok {
				if len(str) < minLen {
					return fmt.Errorf("value must be at least %d characters", minLen)
				}
			}

		case "maxlen":
			maxLen, err := strconv.Atoi(ruleValue)
			if err != nil {
				log.Warning("Invalid maxlen value in validation rule: %s", ruleValue)
				continue
			}
			if str, ok := value.(string); ok {
				if len(str) > maxLen {
					return fmt.Errorf("value must be at most %d characters", maxLen)
				}
			}

		default:
			log.Debug("Unknown validation rule: %s", ruleName)
		}
	}

	return nil
}

// validateMin validates that a numeric value meets the minimum requirement
func validateMin(value interface{}, minVal float64) error {
	switch v := value.(type) {
	case int:
		if float64(v) < minVal {
			return fmt.Errorf("value must be at least %v", minVal)
		}
	case int64:
		if float64(v) < minVal {
			return fmt.Errorf("value must be at least %v", minVal)
		}
	case float64:
		if v < minVal {
			return fmt.Errorf("value must be at least %v", minVal)
		}
	case string:
		// For strings, min refers to length
		if float64(len(v)) < minVal {
			return fmt.Errorf("value must be at least %v characters", minVal)
		}
	}
	return nil
}

// validateMax validates that a numeric value meets the maximum requirement
func validateMax(value interface{}, maxVal float64) error {
	switch v := value.(type) {
	case int:
		if float64(v) > maxVal {
			return fmt.Errorf("value must be at most %v", maxVal)
		}
	case int64:
		if float64(v) > maxVal {
			return fmt.Errorf("value must be at most %v", maxVal)
		}
	case float64:
		if v > maxVal {
			return fmt.Errorf("value must be at most %v", maxVal)
		}
	case string:
		// For strings, max refers to length
		if float64(len(v)) > maxVal {
			return fmt.Errorf("value must be at most %v characters", maxVal)
		}
	}
	return nil
}

// UpsertClient creates a new client if it doesn't exist, or returns existing client if it does
func UpsertClient(clientType, clientValue string) (*models.Client, error) {
	if clientType == "" || clientValue == "" {
		return nil, fmt.Errorf("client type and value are required")
	}

	// First, try to find existing client by external ID
	var externalID models.ClientExternalID
	err := db.Where("type = ? AND value = ?", clientType, clientValue).First(&externalID).Error

	if err == nil {
		// Client exists, return it
		var client models.Client
		if err := db.First(&client, "id = ?", externalID.ClientID).Error; err != nil {
			return nil, fmt.Errorf("failed to load existing client: %w", err)
		}
		return &client, nil
	}

	// Client doesn't exist, create a new one
	// For email type, use the email as the client name initially
	clientName := clientValue
	if clientType == "email" {
		// Extract name part from email if possible, otherwise use full email
		if atIndex := strings.Index(clientValue, "@"); atIndex > 0 {
			clientName = clientValue[:atIndex]
		}
	}

	// Create new client
	client := models.Client{
		Name: clientName,
		Data: datatypes.JSON("{}"),
	}

	if err := db.Create(&client).Error; err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create external ID record
	newExternalID := models.ClientExternalID{
		ClientID: client.ID,
		Type:     clientType,
		Value:    clientValue,
	}

	if err := db.Create(&newExternalID).Error; err != nil {
		log.Warning("Failed to create client external ID:", err)
		// Don't fail the operation, client is already created
	}

	return &client, nil
}

// UpsertClientWithAttributes creates a new client if it doesn't exist, or updates existing client with custom attributes
func UpsertClientWithAttributes(clientType, clientValue string, name *string, attributes map[string]any) (*models.Client, error) {
	if clientType == "" || clientValue == "" {
		return nil, fmt.Errorf("client type and value are required")
	}

	// First, try to find existing client by external ID
	var externalID models.ClientExternalID
	err := db.Where("type = ? AND value = ?", clientType, clientValue).First(&externalID).Error

	if err == nil {
		// Client exists, update it if attributes or name provided
		var client models.Client
		if err := db.First(&client, "id = ?", externalID.ClientID).Error; err != nil {
			return nil, fmt.Errorf("failed to load existing client: %w", err)
		}

		// Process custom attributes if provided
		var data datatypes.JSON
		if attributes != nil {
			data, err = processCustomAttributes(models.CustomAttributeScopeClient, attributes)
			if err != nil {
				return nil, fmt.Errorf("custom attributes error: %w", err)
			}
		}

		// Update client fields if provided
		updates := make(map[string]interface{})
		if name != nil && *name != "" {
			updates["name"] = *name
		}
		if data != nil {
			updates["data"] = data
		}

		// Only update if there are changes
		if len(updates) > 0 {
			if err := db.Model(&client).Updates(updates).Error; err != nil {
				return nil, fmt.Errorf("failed to update client: %w", err)
			}

			// Reload client to get updated values
			if err := db.First(&client, "id = ?", client.ID).Error; err != nil {
				return nil, fmt.Errorf("failed to reload client: %w", err)
			}
		}

		return &client, nil
	}

	// Client doesn't exist, create a new one
	// Determine client name
	clientName := clientValue
	if name != nil && *name != "" {
		clientName = *name
	} else if clientType == "email" {
		// Extract name part from email if possible, otherwise use full email
		if atIndex := strings.Index(clientValue, "@"); atIndex > 0 {
			clientName = clientValue[:atIndex]
		}
	}

	// Process custom attributes if provided
	var data datatypes.JSON
	if attributes != nil {
		data, err = processCustomAttributes(models.CustomAttributeScopeClient, attributes)
		if err != nil {
			return nil, fmt.Errorf("custom attributes error: %w", err)
		}
	}

	// Ensure data is not nil
	if data == nil {
		data = datatypes.JSON("{}")
	}

	// Create new client
	client := models.Client{
		Name: clientName,
		Data: data,
	}

	if err := db.Create(&client).Error; err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create external ID record
	newExternalID := models.ClientExternalID{
		ClientID: client.ID,
		Type:     clientType,
		Value:    clientValue,
	}

	if err := db.Create(&newExternalID).Error; err != nil {
		log.Warning("Failed to create client external ID:", err)
		// Don't fail the operation, client is already created
	}

	return &client, nil
}

// AssignConversationToUser assigns a conversation to a specific user
func AssignConversationToUser(conversationID uint, userID uuid.UUID) (*models.ConversationAssignment, error) {
	// First verify the conversation exists
	var conversation models.Conversation
	if err := db.First(&conversation, conversationID).Error; err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Remove any existing assignments for this conversation
	if err := db.Where("conversation_id = ?", conversationID).Delete(&models.ConversationAssignment{}).Error; err != nil {
		return nil, fmt.Errorf("failed to clear existing assignments: %w", err)
	}

	// Create new assignment to user
	assignment := models.ConversationAssignment{
		ConversationID: conversationID,
		UserID:   &userID,
	}

	if err := db.Create(&assignment).Error; err != nil {
		return nil, fmt.Errorf("failed to assign conversation to user: %w", err)
	}

	return &assignment, nil
}

// AssignConversationToDepartment assigns a conversation to a department
func AssignConversationToDepartment(conversationID uint, departmentID uint) (*models.ConversationAssignment, error) {
	// First verify the conversation exists
	var conversation models.Conversation
	if err := db.First(&conversation, conversationID).Error; err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Verify the department exists
	var department models.Department
	if err := db.First(&department, departmentID).Error; err != nil {
		return nil, fmt.Errorf("department not found: %w", err)
	}

	// Remove any existing assignments for this conversation
	if err := db.Where("conversation_id = ?", conversationID).Delete(&models.ConversationAssignment{}).Error; err != nil {
		return nil, fmt.Errorf("failed to clear existing assignments: %w", err)
	}

	// Create new assignment to department
	assignment := models.ConversationAssignment{
		ConversationID:     conversationID,
		DepartmentID: &departmentID,
	}

	if err := db.Create(&assignment).Error; err != nil {
		return nil, fmt.Errorf("failed to assign conversation to department: %w", err)
	}

	return &assignment, nil
}

// UnassignConversation removes all assignments from a conversation
func UnassignConversation(conversationID uint) error {
	// First verify the conversation exists
	var conversation models.Conversation
	if err := db.First(&conversation, conversationID).Error; err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}

	// Remove all assignments for this conversation
	if err := db.Where("conversation_id = ?", conversationID).Delete(&models.ConversationAssignment{}).Error; err != nil {
		return fmt.Errorf("failed to remove assignments: %w", err)
	}

	return nil
}

// GetConversationAssignments returns all assignments for a conversation
func GetConversationAssignments(conversationID uint) ([]models.ConversationAssignment, error) {
	var assignments []models.ConversationAssignment
	if err := db.Where("conversation_id = ?", conversationID).
		Preload("User").
		Preload("Department").
		Find(&assignments).Error; err != nil {
		return nil, fmt.Errorf("failed to get conversation assignments: %w", err)
	}

	return assignments, nil
}
