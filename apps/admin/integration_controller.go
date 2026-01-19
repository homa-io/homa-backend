package admin

import (
	"encoding/json"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/iesreza/homa-backend/apps/ai"
	integrationsDriver "github.com/iesreza/homa-backend/apps/integrations"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ========================================
// Integration Management
// ========================================

// InboxInfo represents basic inbox info for integration response
type InboxInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ListIntegrations returns all integrations with masked configs
func (c Controller) ListIntegrations(request *evo.Request) any {
	var integrations []models.Integration
	err := db.Preload("Inbox").Order("type ASC").Find(&integrations).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Build response with masked configs
	type IntegrationResponse struct {
		ID        uint                   `json:"id"`
		Type      string                 `json:"type"`
		Name      string                 `json:"name"`
		Status    string                 `json:"status"`
		Config    map[string]interface{} `json:"config,omitempty"`
		LastError string                 `json:"last_error,omitempty"`
		InboxID   *uint                  `json:"inbox_id,omitempty"`
		Inbox     *InboxInfo             `json:"inbox,omitempty"`
		TestedAt  *time.Time             `json:"tested_at,omitempty"`
		CreatedAt time.Time              `json:"created_at"`
		UpdatedAt time.Time              `json:"updated_at"`
	}

	result := make([]IntegrationResponse, len(integrations))
	for i, integration := range integrations {
		var inbox *InboxInfo
		if integration.Inbox != nil {
			inbox = &InboxInfo{
				ID:   integration.Inbox.ID,
				Name: integration.Inbox.Name,
			}
		}
		result[i] = IntegrationResponse{
			ID:        integration.ID,
			Type:      integration.Type,
			Name:      integration.Name,
			Status:    integration.Status,
			Config:    integrationsDriver.GetMaskedConfig(integration.Type, integration.Config),
			LastError: integration.LastError,
			InboxID:   integration.InboxID,
			Inbox:     inbox,
			TestedAt:  integration.TestedAt,
			CreatedAt: integration.CreatedAt,
			UpdatedAt: integration.UpdatedAt,
		}
	}

	return response.OK(result)
}

// ListIntegrationTypes returns all available integration types
func (c Controller) ListIntegrationTypes(request *evo.Request) any {
	types := models.GetIntegrationTypes()
	return response.OK(types)
}

// GetIntegration returns a single integration by type
func (c Controller) GetIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	var integration models.Integration
	err := db.Preload("Inbox").Where("type = ?", integrationType).First(&integration).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return empty integration with type info
			return response.OK(map[string]interface{}{
				"type":     integrationType,
				"name":     getIntegrationName(integrationType),
				"status":   models.IntegrationStatusDisabled,
				"config":   nil,
				"inbox_id": nil,
				"inbox":    nil,
			})
		}
		return response.Error(response.ErrInternalError)
	}

	var inbox *InboxInfo
	if integration.Inbox != nil {
		inbox = &InboxInfo{
			ID:   integration.Inbox.ID,
			Name: integration.Inbox.Name,
		}
	}

	return response.OK(map[string]interface{}{
		"id":         integration.ID,
		"type":       integration.Type,
		"name":       integration.Name,
		"status":     integration.Status,
		"config":     integrationsDriver.GetMaskedConfig(integration.Type, integration.Config),
		"last_error": integration.LastError,
		"inbox_id":   integration.InboxID,
		"inbox":      inbox,
		"tested_at":  integration.TestedAt,
		"created_at": integration.CreatedAt,
		"updated_at": integration.UpdatedAt,
	})
}

// GetIntegrationFields returns the configuration fields for an integration type
func (c Controller) GetIntegrationFields(request *evo.Request) any {
	integrationType := request.Param("type").String()

	fields := integrationsDriver.GetConfigFields(integrationType)
	if fields == nil {
		return response.NotFound(request, "Unknown integration type")
	}

	return response.OK(fields)
}

// SaveIntegration creates or updates an integration
func (c Controller) SaveIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	var req struct {
		Status  string                 `json:"status"`
		Config  map[string]interface{} `json:"config"`
		InboxID *uint                  `json:"inbox_id"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Get or create integration
	integration, _ := models.GetIntegration(integrationType)
	if integration.ID == 0 {
		integration = &models.Integration{
			Type: integrationType,
			Name: getIntegrationName(integrationType),
		}
	}

	// Merge incoming config with existing to preserve masked sensitive fields
	mergedConfig := integrationsDriver.MergeConfigWithExisting(integration.Config, req.Config)

	// Convert merged config to JSON
	configJSON, err := json.Marshal(mergedConfig)
	if err != nil {
		return response.BadRequest(request, "Invalid configuration")
	}

	// Validate configuration
	if err := integrationsDriver.ValidateConfig(integrationType, string(configJSON)); err != nil {
		return response.BadRequest(request, err.Error())
	}

	integration.Status = req.Status
	integration.Config = string(configJSON)
	integration.LastError = ""
	integration.InboxID = req.InboxID

	if err := models.UpsertIntegration(integration); err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload with inbox
	db.Preload("Inbox").First(integration, integration.ID)

	var inbox *InboxInfo
	if integration.Inbox != nil {
		inbox = &InboxInfo{
			ID:   integration.Inbox.ID,
			Name: integration.Inbox.Name,
		}
	}

	// Call OnSave callback for post-save actions (e.g., webhook registration)
	webhookBaseURL := getWebhookBaseURL(request)
	onSaveResult := integrationsDriver.OnSave(integration.Type, integration.Config, integration.Status, webhookBaseURL)

	return response.OK(map[string]interface{}{
		"id":              integration.ID,
		"type":            integration.Type,
		"name":            integration.Name,
		"status":          integration.Status,
		"config":          integrationsDriver.GetMaskedConfig(integration.Type, integration.Config),
		"last_error":      integration.LastError,
		"inbox_id":        integration.InboxID,
		"inbox":           inbox,
		"tested_at":       integration.TestedAt,
		"updated_at":      integration.UpdatedAt,
		"on_save_success": onSaveResult.Success,
		"on_save_message": onSaveResult.Message,
	})
}

// TestIntegration tests the connection for an integration
func (c Controller) TestIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	var req struct {
		Config map[string]interface{} `json:"config"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Get existing integration config to merge with masked values
	existingIntegration, _ := models.GetIntegration(integrationType)
	if existingIntegration.ID != 0 && existingIntegration.Config != "" {
		// Merge the incoming config with existing config to preserve masked sensitive fields
		req.Config = integrationsDriver.MergeConfigWithExisting(existingIntegration.Config, req.Config)
	}

	// Convert config to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return response.BadRequest(request, "Invalid configuration")
	}

	// Test the integration
	result := integrationsDriver.TestIntegration(integrationType, string(configJSON))

	// Update the integration if it exists
	if result.Success {
		integration, _ := models.GetIntegration(integrationType)
		if integration.ID != 0 {
			now := time.Now()
			integration.TestedAt = &now
			integration.LastError = ""
			db.Save(integration)
		}
	} else {
		integration, _ := models.GetIntegration(integrationType)
		if integration.ID != 0 {
			now := time.Now()
			integration.TestedAt = &now
			integration.LastError = result.Message
			if result.Details != "" {
				integration.LastError += ": " + result.Details
			}
			db.Save(integration)
		}
	}

	return response.OK(result)
}

// DeleteIntegration removes an integration
func (c Controller) DeleteIntegration(request *evo.Request) any {
	integrationType := request.Param("type").String()

	integration, err := models.GetIntegration(integrationType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Integration not found")
		}
		return response.Error(response.ErrInternalError)
	}

	if err := db.Delete(integration).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Integration deleted successfully"})
}

// Helper function to get integration name from type
func getIntegrationName(integrationType string) string {
	types := models.GetIntegrationTypes()
	for _, t := range types {
		if t.Type == integrationType {
			return t.Name
		}
	}
	return integrationType
}

// getWebhookBaseURL returns the API base URL for webhook registration
func getWebhookBaseURL(request *evo.Request) string {
	// First check for configured API base URL
	apiBaseURL := settings.Get("APP.API_BASE_URL").String()
	if apiBaseURL != "" {
		return apiBaseURL
	}

	// Fallback to X-Forwarded headers for reverse proxy setups
	proto := request.Get("X-Forwarded-Proto").String()
	if proto == "" {
		proto = request.Protocol()
		if proto == "HTTP/1.1" || proto == "HTTP/2" {
			proto = "http"
		}
	}

	host := request.Get("X-Forwarded-Host").String()
	if host == "" {
		host = request.Hostname()
	}

	return proto + "://" + host
}

// ========================================
// AI Agent Management
// ========================================

// ListAIAgents returns all AI agents
func (c Controller) ListAIAgents(request *evo.Request) any {
	var agents []models.AIAgent

	query := db.Order("id DESC")

	// Filter by status if provided
	if status := request.Query("status").String(); status != "" {
		query = query.Where("status = ?", status)
	}

	// Preload relationships
	query = query.Preload("Bot").Preload("HandoverUser")

	err := query.Find(&agents).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(agents)
}

// GetAIAgent returns a single AI agent by ID
func (c Controller) GetAIAgent(request *evo.Request) any {
	id := request.Param("id").String()
	var agent models.AIAgent

	err := db.Preload("Bot").Preload("HandoverUser").First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(agent)
}

// CreateAIAgent creates a new AI agent
func (c Controller) CreateAIAgent(request *evo.Request) any {
	var agent models.AIAgent

	if err := request.BodyParser(&agent); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Validate required fields
	if agent.Name == "" {
		return response.BadRequest(request, "Name is required")
	}
	if agent.BotID == "" {
		return response.BadRequest(request, "Bot ID is required")
	}

	// Validate tone
	validTones := []string{
		models.AIAgentToneFormal,
		models.AIAgentToneCasual,
		models.AIAgentToneDetailed,
		models.AIAgentTonePrecise,
		models.AIAgentToneEmpathetic,
		models.AIAgentToneTechnical,
	}
	toneValid := false
	for _, t := range validTones {
		if agent.Tone == t {
			toneValid = true
			break
		}
	}
	if !toneValid {
		return response.BadRequest(request, "Invalid tone value")
	}

	// Create the agent
	err := db.Create(&agent).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload with relationships
	db.Preload("Bot").Preload("HandoverUser").First(&agent, agent.ID)

	return response.OK(agent)
}

// UpdateAIAgent updates an existing AI agent
func (c Controller) UpdateAIAgent(request *evo.Request) any {
	id := request.Param("id").String()

	var agent models.AIAgent
	err := db.First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Parse update data
	var updateData map[string]any
	if err := request.BodyParser(&updateData); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Validate tone if it's being updated
	if tone, ok := updateData["tone"].(string); ok {
		validTones := []string{
			models.AIAgentToneFormal,
			models.AIAgentToneCasual,
			models.AIAgentToneDetailed,
			models.AIAgentTonePrecise,
			models.AIAgentToneEmpathetic,
			models.AIAgentToneTechnical,
		}
		toneValid := false
		for _, t := range validTones {
			if tone == t {
				toneValid = true
				break
			}
		}
		if !toneValid {
			return response.BadRequest(request, "Invalid tone value")
		}
	}

	// Convert handover_user_ids array to JSON if present
	if userIds, ok := updateData["handover_user_ids"]; ok {
		jsonBytes, err := json.Marshal(userIds)
		if err != nil {
			return response.BadRequest(request, "Invalid handover_user_ids format")
		}
		updateData["handover_user_ids"] = datatypes.JSON(jsonBytes)
	}

	// Update the agent
	err = db.Model(&agent).Updates(updateData).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Reload with relationships
	db.Preload("Bot").Preload("HandoverUser").First(&agent, id)

	return response.OK(agent)
}

// DeleteAIAgent deletes an AI agent
func (c Controller) DeleteAIAgent(request *evo.Request) any {
	id := request.Param("id").String()

	var agent models.AIAgent
	err := db.First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Check if agent is being used by any departments
	var count int64
	db.Model(&models.Department{}).Where("ai_agent_id = ?", id).Count(&count)
	if count > 0 {
		return response.BadRequest(request, "Cannot delete AI agent that is assigned to departments")
	}

	// Delete the agent
	err = db.Delete(&agent).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "AI Agent deleted successfully"})
}

// GetAIAgentTemplate generates and returns the system prompt template for an AI agent
// Uses the Jet template system with customizable template and separate tool documentation
func (c Controller) GetAIAgentTemplate(request *evo.Request) any {
	id := request.Param("id").String()

	// Load agent with relationships
	var agent models.AIAgent
	err := db.Preload("Bot").Preload("HandoverUser").First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Load agent tools
	var tools []models.AIAgentTool
	db.Where("ai_agent_id = ?", id).Order("id ASC").Find(&tools)

	// Get project name from settings
	projectName := models.GetSettingValue("general.project_name", "Your Project")

	// Build template data for Jet template
	templateData := ai.BuildTemplateData(&agent, projectName)

	// Get the custom template from settings, or use default
	customTemplate := models.GetSettingValue(ai.SettingKeyBotPromptTemplate, "")
	templateContent := customTemplate
	if templateContent == "" {
		templateContent = ai.GetDefaultBotPromptTemplate()
	}

	// Render the Jet template
	prompt, err := ai.RenderBotPromptTemplate(templateContent, templateData)
	if err != nil {
		// Fall back to default template on error
		prompt, _ = ai.RenderBotPromptTemplate(ai.GetDefaultBotPromptTemplate(), templateData)
	}

	// Generate tool documentation separately (not customizable)
	toolDocs := ai.GenerateToolDocumentation(&agent, tools)

	// Combine prompt and tool docs for the full template
	fullTemplate := prompt
	if toolDocs != "" {
		fullTemplate = prompt + "\n\n" + toolDocs
	}

	return response.OK(map[string]any{
		"template":    fullTemplate,
		"prompt":      prompt,
		"tool_docs":   toolDocs,
		"is_custom":   customTemplate != "",
		"token_count": len(fullTemplate) / 4, // Approximate token count
	})
}

// ==================== AI Agent Tools ====================

// ListAIAgentTools returns all tools for an AI agent
func (c Controller) ListAIAgentTools(request *evo.Request) any {
	agentID := request.Param("agent_id").String()

	// Verify agent exists
	var agent models.AIAgent
	if err := db.First(&agent, agentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	var tools []models.AIAgentTool
	err := db.Where("ai_agent_id = ?", agentID).Order("id ASC").Find(&tools).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tools)
}

// GetAIAgentTool returns a single tool by ID
func (c Controller) GetAIAgentTool(request *evo.Request) any {
	toolID := request.Param("tool_id").String()

	var tool models.AIAgentTool
	err := db.First(&tool, toolID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Tool not found")
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tool)
}

// CreateAIAgentTool creates a new tool for an AI agent
func (c Controller) CreateAIAgentTool(request *evo.Request) any {
	agentID := request.Param("agent_id").String()

	// Verify agent exists
	var agent models.AIAgent
	if err := db.First(&agent, agentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "AI Agent not found")
		}
		return response.Error(response.ErrInternalError)
	}

	var tool models.AIAgentTool
	if err := request.BodyParser(&tool); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Set the agent ID
	tool.AIAgentID = agent.ID

	// Validate required fields
	if tool.Name == "" {
		return response.BadRequest(request, "Tool name is required")
	}
	if tool.Endpoint == "" {
		return response.BadRequest(request, "Endpoint is required")
	}

	// Validate method
	validMethods := []string{models.ToolMethodGET, models.ToolMethodPOST, models.ToolMethodPUT, models.ToolMethodPATCH, models.ToolMethodDELETE}
	methodValid := false
	for _, m := range validMethods {
		if tool.Method == m {
			methodValid = true
			break
		}
	}
	if !methodValid {
		tool.Method = models.ToolMethodGET
	}

	// Set default auth type if not provided
	if tool.AuthorizationType == "" {
		tool.AuthorizationType = models.ToolAuthTypeNone
	}

	// Create the tool
	err := db.Create(&tool).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tool)
}

// UpdateAIAgentTool updates an existing tool
func (c Controller) UpdateAIAgentTool(request *evo.Request) any {
	toolID := request.Param("tool_id").String()

	var tool models.AIAgentTool
	err := db.First(&tool, toolID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Tool not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Parse update data into the tool struct directly
	if err := request.BodyParser(&tool); err != nil {
		return response.BadRequest(request, "Invalid request body")
	}

	// Save all fields
	err = db.Save(&tool).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tool)
}

// DeleteAIAgentTool deletes a tool
func (c Controller) DeleteAIAgentTool(request *evo.Request) any {
	toolID := request.Param("tool_id").String()

	var tool models.AIAgentTool
	err := db.First(&tool, toolID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NotFound(request, "Tool not found")
		}
		return response.Error(response.ErrInternalError)
	}

	// Delete the tool
	err = db.Delete(&tool).Error
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Tool deleted successfully"})
}
