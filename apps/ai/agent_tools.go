package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
)

// SearchKnowledgeBase is a function variable to avoid circular imports with rag package
// It should be set by the rag package during initialization
var SearchKnowledgeBase func(query string, limit int) (string, error)

// AgentContext holds all context needed for tool execution
type AgentContext struct {
	Conversation *models.Conversation
	Department   *models.Department
	AIAgent      *models.AIAgent
	AgentTools   []models.AIAgentTool
	Client       *models.Client
	Bot          *auth.User
}

// BuiltinToolHandler is the signature for built-in tool handlers
type BuiltinToolHandler func(ctx *AgentContext, args json.RawMessage) (string, bool, error)

// builtinTools maps tool names to their handlers
// Returns: result string, shouldStop bool, error
var builtinTools = map[string]BuiltinToolHandler{
	"handover":            handleHandover,
	"searchKnowledgeBase": handleSearchKnowledgeBase,
	"setUserInfo":         handleSetUserInfo,
	"setPriority":         handleSetPriority,
	"setTag":              handleSetTag,
}

// BuildToolsForAgent creates tool definitions from agent config and AIAgentTool records
func BuildToolsForAgent(agent *models.AIAgent, agentTools []models.AIAgentTool) []ToolDefinition {
	var tools []ToolDefinition

	// Add built-in tools based on agent config flags
	if agent.HandoverEnabled {
		tools = append(tools, buildHandoverTool())
	}
	if agent.UseKnowledgeBase {
		tools = append(tools, buildSearchKnowledgeBaseTool())
	}
	if agent.CollectUserInfo && agent.CollectUserInfoFields != "" {
		tools = append(tools, buildSetUserInfoTool(agent.CollectUserInfoFields))
	}
	if agent.PriorityDetection {
		tools = append(tools, buildSetPriorityTool())
	}
	if agent.AutoTagging {
		tools = append(tools, buildSetTagTool())
	}

	// Add dynamic tools from AIAgentTool records
	for _, agentTool := range agentTools {
		tools = append(tools, buildDynamicTool(agentTool))
	}

	return tools
}

// ExecuteTool routes to built-in handler or HTTP executor
// Returns: result string, shouldStop bool, error
func ExecuteTool(ctx *AgentContext, toolCall ToolCall) (string, bool, error) {
	// Check if it's a built-in tool
	if handler, ok := builtinTools[toolCall.Function.Name]; ok {
		return handler(ctx, []byte(toolCall.Function.Arguments))
	}

	// Otherwise, find the AIAgentTool and execute HTTP call
	return executeCustomTool(ctx, toolCall)
}

// ========== Built-in Tool Definitions ==========

func buildHandoverTool() ToolDefinition {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"reason": {
				"type": "string",
				"description": "The reason for handing over to a human agent"
			}
		},
		"required": ["reason"]
	}`)

	return ToolDefinition{
		Type: "function",
		Function: FunctionDef{
			Name:        "handover",
			Description: "Transfer the conversation to a human agent. Use this when you cannot help the customer or when they explicitly request to speak with a human.",
			Parameters:  params,
		},
	}
}

func buildSearchKnowledgeBaseTool() ToolDefinition {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query to find relevant information in the knowledge base"
			}
		},
		"required": ["query"]
	}`)

	return ToolDefinition{
		Type: "function",
		Function: FunctionDef{
			Name:        "searchKnowledgeBase",
			Description: "Search the knowledge base for relevant information to answer customer questions. Use this when you need factual information about products, policies, or procedures.",
			Parameters:  params,
		},
	}
}

func buildSetUserInfoTool(fields string) ToolDefinition {
	// Parse the fields (comma-separated list)
	fieldList := strings.Split(fields, ",")
	properties := make(map[string]interface{})
	required := []string{}

	for _, field := range fieldList {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		properties[field] = map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("The customer's %s", field),
		}
		required = append(required, field)
	}

	paramsObj := map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}

	params, _ := json.Marshal(paramsObj)

	return ToolDefinition{
		Type: "function",
		Function: FunctionDef{
			Name:        "setUserInfo",
			Description: "Store customer information that has been collected during the conversation. Call this when you have gathered the required information from the customer.",
			Parameters:  json.RawMessage(params),
		},
	}
}

func buildSetPriorityTool() ToolDefinition {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"priority": {
				"type": "string",
				"enum": ["low", "medium", "high", "urgent"],
				"description": "The priority level for this conversation"
			}
		},
		"required": ["priority"]
	}`)

	return ToolDefinition{
		Type: "function",
		Function: FunctionDef{
			Name:        "setPriority",
			Description: "Set the priority level of the conversation based on urgency. Use 'urgent' for critical issues, 'high' for important matters, 'medium' for standard requests, and 'low' for minor inquiries.",
			Parameters:  params,
		},
	}
}

func buildSetTagTool() ToolDefinition {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"description": "List of tags to add to the conversation for categorization"
			}
		},
		"required": ["tags"]
	}`)

	return ToolDefinition{
		Type: "function",
		Function: FunctionDef{
			Name:        "setTag",
			Description: "Add tags to the conversation to categorize the topic or issue type. Use descriptive tags that help with routing and reporting.",
			Parameters:  params,
		},
	}
}

// ========== Built-in Tool Handlers ==========

// handleHandover transfers conversation to human agent(s)
func handleHandover(ctx *AgentContext, args json.RawMessage) (string, bool, error) {
	var params struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", false, fmt.Errorf("failed to parse handover args: %w", err)
	}

	// Update conversation status to wait_for_agent
	updates := map[string]interface{}{
		"status": models.ConversationStatusWaitForAgent,
	}

	var handoverAgentNames []string
	var anyOnline bool

	// Get handover user IDs - check new field first, fall back to deprecated field
	var userIDs []string

	// Try new HandoverUserIDs field first (JSON array)
	if len(ctx.AIAgent.HandoverUserIDs) > 0 {
		if err := json.Unmarshal(ctx.AIAgent.HandoverUserIDs, &userIDs); err != nil {
			userIDs = []string{}
		}
	}

	// Fall back to deprecated HandoverUserID if new field is empty
	if len(userIDs) == 0 && ctx.AIAgent.HandoverUserID != nil && *ctx.AIAgent.HandoverUserID != "" {
		userIDs = []string{*ctx.AIAgent.HandoverUserID}
	}

	// Assign conversation to all handover users
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	for _, userIDStr := range userIDs {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		// Create assignment for this user
		assignment := models.ConversationAssignment{
			ConversationID: ctx.Conversation.ID,
			UserID:         &userID,
			DepartmentID:   ctx.Conversation.DepartmentID,
		}
		db.Create(&assignment)

		// Get the user's details
		var handoverUser auth.User
		if err := db.Where("user_id = ?", userID).First(&handoverUser).Error; err == nil {
			name := handoverUser.DisplayName
			if name == "" {
				name = handoverUser.Name
			}
			handoverAgentNames = append(handoverAgentNames, name)

			// Check if user is online (last activity within 5 minutes)
			var lastSession models.UserSession
			if err := db.Where("user_id = ? AND last_activity > ?", userID, fiveMinutesAgo).
				Order("last_activity DESC").
				First(&lastSession).Error; err == nil {
				anyOnline = true
			}
		}
	}

	// Update conversation
	if err := db.Model(&models.Conversation{}).Where("id = ?", ctx.Conversation.ID).Updates(updates).Error; err != nil {
		return "", false, fmt.Errorf("failed to update conversation for handover: %w", err)
	}

	// Create a system message about the handover (internal)
	systemMsg := models.Message{
		ConversationID:  ctx.Conversation.ID,
		Body:            fmt.Sprintf("Conversation handed over to human agent(s). Reason: %s", params.Reason),
		IsSystemMessage: true,
	}
	db.Create(&systemMsg)

	// Create a user-facing message from the bot about the handover
	var userMessage string
	if len(handoverAgentNames) > 0 {
		agentNamesStr := strings.Join(handoverAgentNames, ", ")
		if anyOnline {
			userMessage = fmt.Sprintf("I've transferred this conversation to our support team (%s). Someone is currently online and will assist you shortly.", agentNamesStr)
		} else {
			userMessage = fmt.Sprintf("I've transferred this conversation to our support team (%s). They are currently offline, but will get back to you as soon as possible.", agentNamesStr)
		}
	} else {
		userMessage = "I've transferred this conversation to our support team. Please wait and someone will assist you shortly."
	}

	// Translate the message to user's language
	// First try client's language preference, then detect from recent messages
	targetLanguage := ""
	if ctx.Client != nil && ctx.Client.Language != nil && *ctx.Client.Language != "" {
		targetLanguage = *ctx.Client.Language
	}

	// If no language set, detect from recent client messages
	if targetLanguage == "" || targetLanguage == "en" {
		targetLanguage = detectLanguageFromConversation(ctx.Conversation.ID)
	}

	// Translate if not English
	if targetLanguage != "" && targetLanguage != "en" && targetLanguage != "english" {
		translatedMsg := translateMessage(userMessage, targetLanguage)
		if translatedMsg != "" {
			userMessage = translatedMsg
		}
	}

	// Send the handover message as the bot
	if ctx.Bot != nil {
		botMsg := models.Message{
			ConversationID: ctx.Conversation.ID,
			UserID:         &ctx.Bot.UserID,
			Body:           userMessage,
		}
		db.Create(&botMsg)
	}

	log.Info("AI Agent handed over conversation %d to human agent(s) %v (any online: %v). Reason: %s",
		ctx.Conversation.ID, handoverAgentNames, anyOnline, params.Reason)

	// Return result and indicate processing should stop
	return fmt.Sprintf("Handover initiated to %v. User has been notified.", handoverAgentNames), true, nil
}

// detectLanguageFromConversation detects the user's language from recent messages
func detectLanguageFromConversation(conversationID uint) string {
	// Get recent client messages
	var recentMessages []models.Message
	err := db.Where("conversation_id = ? AND client_id IS NOT NULL", conversationID).
		Order("created_at DESC").
		Limit(3).
		Find(&recentMessages).Error
	if err != nil || len(recentMessages) == 0 {
		return ""
	}

	// Combine recent messages for language detection
	var messageTexts []string
	for _, msg := range recentMessages {
		if msg.Body != "" {
			messageTexts = append(messageTexts, msg.Body)
		}
	}

	if len(messageTexts) == 0 {
		return ""
	}

	client := GetClient()
	if client == nil {
		return ""
	}

	combinedText := strings.Join(messageTexts, "\n")
	prompt := fmt.Sprintf("What language is this text written in? Reply with ONLY the language code (e.g., 'en', 'fa', 'es', 'fr', 'de', 'ar', 'zh', 'ja', 'ko', 'ru', 'pt', 'it', 'tr', 'nl', 'pl'). If it's English, reply 'en'. Text:\n\n%s", combinedText)

	messages := []ToolMessage{
		{Role: "user", Content: prompt},
	}

	response, err := client.ChatCompletionWithTools(messages, nil, 10, 0.1)
	if err != nil {
		log.Warning("Failed to detect language: %v", err)
		return ""
	}

	if len(response.Choices) > 0 && response.Choices[0].Message.Content != "" {
		lang := strings.TrimSpace(strings.ToLower(response.Choices[0].Message.Content))
		// Clean up response - sometimes it returns "fa" or "Persian" or "fa (Persian)"
		lang = strings.Split(lang, " ")[0]
		lang = strings.Trim(lang, "\"'()")
		return lang
	}

	return ""
}

// translateMessage translates a message to the target language using OpenAI
func translateMessage(message string, targetLanguage string) string {
	client := GetClient()
	if client == nil {
		return ""
	}

	prompt := fmt.Sprintf("Translate the following message to %s. Only output the translation, nothing else:\n\n%s", targetLanguage, message)

	messages := []ToolMessage{
		{Role: "user", Content: prompt},
	}

	response, err := client.ChatCompletionWithTools(messages, nil, 500, 0.3)
	if err != nil {
		log.Warning("Failed to translate handover message: %v", err)
		return ""
	}

	if len(response.Choices) > 0 && response.Choices[0].Message.Content != "" {
		return response.Choices[0].Message.Content
	}

	return ""
}

// handleSearchKnowledgeBase searches the RAG knowledge base
func handleSearchKnowledgeBase(ctx *AgentContext, args json.RawMessage) (string, bool, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", false, fmt.Errorf("failed to parse searchKnowledgeBase args: %w", err)
	}

	if params.Query == "" {
		return "No query provided", false, nil
	}

	// Check if knowledge base search is available
	if SearchKnowledgeBase == nil {
		log.Warning("Knowledge base search not available (SearchKnowledgeBase not registered)")
		return "Knowledge base search is not available.", false, nil
	}

	// Use RAG search via function variable
	context, err := SearchKnowledgeBase(params.Query, 5)
	if err != nil {
		log.Warning("Knowledge base search failed: %v", err)
		return "No relevant information found in the knowledge base.", false, nil
	}

	if context == "" {
		return "No relevant information found in the knowledge base.", false, nil
	}

	return context, false, nil
}

// handleSetUserInfo stores collected user information
// Updates client name and merges all data into client.data JSON field
func handleSetUserInfo(ctx *AgentContext, args json.RawMessage) (string, bool, error) {
	// Parse the dynamic fields
	var userInfo map[string]interface{}
	if err := json.Unmarshal(args, &userInfo); err != nil {
		return "", false, fmt.Errorf("failed to parse setUserInfo args: %w", err)
	}

	collectedFields := make([]string, 0, len(userInfo))

	// === Update Client (if available) ===
	if ctx.Client != nil {
		// Update client name if provided
		if name, ok := userInfo["name"].(string); ok && name != "" {
			if err := db.Model(&models.Client{}).Where("id = ?", ctx.Client.ID).Update("name", name).Error; err != nil {
				log.Warning("Failed to update client name: %v", err)
			}
		}

		// Get existing client data and merge all fields
		var clientData map[string]interface{}
		if ctx.Client.Data != nil {
			if err := json.Unmarshal(ctx.Client.Data, &clientData); err != nil {
				clientData = make(map[string]interface{})
			}
		} else {
			clientData = make(map[string]interface{})
		}

		// Merge all fields into client data (overwrite existing keys)
		for k, v := range userInfo {
			clientData[k] = v
			collectedFields = append(collectedFields, k)
		}

		// Save merged client data
		clientDataJSON, err := json.Marshal(clientData)
		if err == nil {
			if err := db.Model(&models.Client{}).Where("id = ?", ctx.Client.ID).Update("data", clientDataJSON).Error; err != nil {
				log.Warning("Failed to update client data: %v", err)
			}
		}
	}

	// === Update Conversation CustomFields ===
	var customFields map[string]interface{}
	if ctx.Conversation.CustomFields != nil {
		if err := json.Unmarshal(ctx.Conversation.CustomFields, &customFields); err != nil {
			customFields = make(map[string]interface{})
		}
	} else {
		customFields = make(map[string]interface{})
	}

	// Add/merge collected user info under "user_info" key
	if existingInfo, ok := customFields["user_info"].(map[string]interface{}); ok {
		// Merge with existing info (overwrite if key exists)
		for k, v := range userInfo {
			existingInfo[k] = v
		}
		customFields["user_info"] = existingInfo
	} else {
		customFields["user_info"] = userInfo
	}

	// Save back to conversation
	customFieldsJSON, err := json.Marshal(customFields)
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal custom fields: %w", err)
	}

	if err := db.Model(&models.Conversation{}).Where("id = ?", ctx.Conversation.ID).
		Update("custom_fields", customFieldsJSON).Error; err != nil {
		return "", false, fmt.Errorf("failed to save user info to conversation: %w", err)
	}

	log.Info("AI Agent collected user info for conversation %d: %v", ctx.Conversation.ID, collectedFields)

	return fmt.Sprintf("User information saved: %s", strings.Join(collectedFields, ", ")), false, nil
}

// handleSetPriority sets the conversation priority
func handleSetPriority(ctx *AgentContext, args json.RawMessage) (string, bool, error) {
	var params struct {
		Priority string `json:"priority"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", false, fmt.Errorf("failed to parse setPriority args: %w", err)
	}

	// Validate priority
	validPriorities := map[string]bool{
		models.ConversationPriorityLow:    true,
		models.ConversationPriorityMedium: true,
		models.ConversationPriorityHigh:   true,
		models.ConversationPriorityUrgent: true,
	}

	if !validPriorities[params.Priority] {
		return fmt.Sprintf("Invalid priority: %s", params.Priority), false, nil
	}

	// Update conversation priority
	if err := db.Model(&models.Conversation{}).Where("id = ?", ctx.Conversation.ID).
		Update("priority", params.Priority).Error; err != nil {
		return "", false, fmt.Errorf("failed to set priority: %w", err)
	}

	log.Info("AI Agent set priority for conversation %d to: %s", ctx.Conversation.ID, params.Priority)

	return fmt.Sprintf("Priority set to: %s", params.Priority), false, nil
}

// handleSetTag adds tags to the conversation
func handleSetTag(ctx *AgentContext, args json.RawMessage) (string, bool, error) {
	var params struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", false, fmt.Errorf("failed to parse setTag args: %w", err)
	}

	if len(params.Tags) == 0 {
		return "No tags provided", false, nil
	}

	addedTags := []string{}

	for _, tagName := range params.Tags {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		// Find or create the tag
		var tag models.Tag
		if err := db.Where("name = ?", tagName).First(&tag).Error; err != nil {
			// Tag doesn't exist, create it
			tag = models.Tag{Name: tagName}
			if err := db.Create(&tag).Error; err != nil {
				log.Warning("Failed to create tag %s: %v", tagName, err)
				continue
			}
		}

		// Add tag to conversation (using many2many relationship)
		if err := db.Exec("INSERT IGNORE INTO conversation_tags (conversation_id, tag_id) VALUES (?, ?)",
			ctx.Conversation.ID, tag.ID).Error; err != nil {
			log.Warning("Failed to add tag %s to conversation: %v", tagName, err)
			continue
		}

		addedTags = append(addedTags, tagName)
	}

	if len(addedTags) > 0 {
		log.Info("AI Agent added tags to conversation %d: %v", ctx.Conversation.ID, addedTags)
		return fmt.Sprintf("Tags added: %s", strings.Join(addedTags, ", ")), false, nil
	}

	return "No tags were added", false, nil
}

// ========== Dynamic Tool Functions ==========

// buildDynamicTool converts AIAgentTool to OpenAI function definition
func buildDynamicTool(tool models.AIAgentTool) ToolDefinition {
	// Build parameters schema from both QueryParams and BodyParams
	// (for GET requests, params are typically in query; for POST, in body)
	params := buildParamsSchemaFromAll(tool.QueryParams, tool.BodyParams)

	return ToolDefinition{
		Type: "function",
		Function: FunctionDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  params,
		},
	}
}

// buildParamsSchemaFromAll creates a JSON schema from both query and body params
// This ensures GET requests with query params are properly exposed to the model
func buildParamsSchemaFromAll(queryParams []byte, bodyParams []byte) json.RawMessage {
	properties := make(map[string]interface{})
	required := []string{}

	// Process query params
	if len(queryParams) > 0 {
		var params []models.ToolParam
		if err := json.Unmarshal(queryParams, &params); err == nil {
			for _, p := range params {
				// Only include params that should be filled by the model
				if p.ValueType != models.ToolParamValueTypeByModel {
					continue
				}

				prop := map[string]interface{}{
					"type": p.DataType,
				}
				if p.Example != "" {
					prop["description"] = fmt.Sprintf("Example: %s", p.Example)
				}

				properties[p.Key] = prop

				if p.Required {
					required = append(required, p.Key)
				}
			}
		}
	}

	// Process body params
	if len(bodyParams) > 0 {
		var params []models.ToolParam
		if err := json.Unmarshal(bodyParams, &params); err == nil {
			for _, p := range params {
				// Only include params that should be filled by the model
				if p.ValueType != models.ToolParamValueTypeByModel {
					continue
				}

				prop := map[string]interface{}{
					"type": p.DataType,
				}
				if p.Example != "" {
					prop["description"] = fmt.Sprintf("Example: %s", p.Example)
				}

				properties[p.Key] = prop

				if p.Required {
					required = append(required, p.Key)
				}
			}
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	result, _ := json.Marshal(schema)
	return json.RawMessage(result)
}

// buildParamsSchema creates a JSON schema from ToolParam array (kept for compatibility)
func buildParamsSchema(bodyParams []byte) json.RawMessage {
	return buildParamsSchemaFromAll(nil, bodyParams)
}

// executeCustomTool makes HTTP request to the configured endpoint
func executeCustomTool(ctx *AgentContext, toolCall ToolCall) (string, bool, error) {
	// Find the AIAgentTool by name
	var tool *models.AIAgentTool
	for i := range ctx.AgentTools {
		if ctx.AgentTools[i].Name == toolCall.Function.Name {
			tool = &ctx.AgentTools[i]
			break
		}
	}

	if tool == nil {
		return "", false, fmt.Errorf("tool not found: %s", toolCall.Function.Name)
	}

	// Parse the arguments provided by the model
	var modelArgs map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &modelArgs); err != nil {
		modelArgs = make(map[string]interface{})
	}

	// Build the HTTP request
	endpoint := tool.Endpoint

	// Process query params
	if len(tool.QueryParams) > 0 {
		var queryParams []models.ToolParam
		if err := json.Unmarshal(tool.QueryParams, &queryParams); err == nil {
			queryValues := url.Values{}
			for _, p := range queryParams {
				value := resolveParamValue(p, modelArgs, ctx)
				if value != "" {
					queryValues.Set(p.Key, value)
				}
			}
			if len(queryValues) > 0 {
				if strings.Contains(endpoint, "?") {
					endpoint += "&" + queryValues.Encode()
				} else {
					endpoint += "?" + queryValues.Encode()
				}
			}
		}
	}

	// Build request body
	var requestBody io.Reader
	if tool.Method != models.ToolMethodGET && len(tool.BodyParams) > 0 {
		var bodyParams []models.ToolParam
		if err := json.Unmarshal(tool.BodyParams, &bodyParams); err == nil {
			bodyMap := make(map[string]interface{})
			for _, p := range bodyParams {
				value := resolveParamValue(p, modelArgs, ctx)
				if value != "" {
					bodyMap[p.Key] = convertToType(value, p.DataType)
				}
			}

			if tool.BodyType == models.ToolBodyTypeJSON {
				bodyJSON, _ := json.Marshal(bodyMap)
				requestBody = bytes.NewReader(bodyJSON)
			} else if tool.BodyType == models.ToolBodyTypeFormValue {
				formData := url.Values{}
				for k, v := range bodyMap {
					formData.Set(k, fmt.Sprintf("%v", v))
				}
				requestBody = strings.NewReader(formData.Encode())
			}
		}
	}

	// Create HTTP request
	req, err := http.NewRequest(tool.Method, endpoint, requestBody)
	if err != nil {
		return "", false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if tool.BodyType == models.ToolBodyTypeJSON {
		req.Header.Set("Content-Type", "application/json")
	} else if tool.BodyType == models.ToolBodyTypeFormValue {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Process header params
	if len(tool.HeaderParams) > 0 {
		var headerParams []models.ToolParam
		if err := json.Unmarshal(tool.HeaderParams, &headerParams); err == nil {
			for _, p := range headerParams {
				value := resolveParamValue(p, modelArgs, ctx)
				if value != "" {
					req.Header.Set(p.Key, value)
				}
			}
		}
	}

	// Apply authorization
	switch tool.AuthorizationType {
	case models.ToolAuthTypeBearer:
		req.Header.Set("Authorization", "Bearer "+tool.AuthorizationValue)
	case models.ToolAuthTypeBasic:
		auth := base64.StdEncoding.EncodeToString([]byte(tool.AuthorizationValue))
		req.Header.Set("Authorization", "Basic "+auth)
	case models.ToolAuthTypeAPIKey:
		if tool.AuthorizationHeader != "" {
			req.Header.Set(tool.AuthorizationHeader, tool.AuthorizationValue)
		}
	}

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read response: %w", err)
	}

	// Format response
	result := string(respBody)

	// If there are response instructions, include them
	if tool.ResponseInstructions != "" {
		result = fmt.Sprintf("API Response:\n%s\n\nInstructions: %s", result, tool.ResponseInstructions)
	}

	log.Info("AI Agent executed custom tool %s for conversation %d", tool.Name, ctx.Conversation.ID)

	return result, false, nil
}

// resolveParamValue resolves the value of a parameter based on its type
func resolveParamValue(param models.ToolParam, modelArgs map[string]interface{}, ctx *AgentContext) string {
	switch param.ValueType {
	case models.ToolParamValueTypeConstant:
		return param.Value
	case models.ToolParamValueTypeByModel:
		if val, ok := modelArgs[param.Key]; ok {
			return fmt.Sprintf("%v", val)
		}
		return ""
	case models.ToolParamValueTypeVariable:
		return resolveVariable(param.Value, ctx)
	default:
		return param.Value
	}
}

// resolveVariable resolves a variable reference to its actual value
func resolveVariable(variable string, ctx *AgentContext) string {
	switch variable {
	case "conversation_id":
		return fmt.Sprintf("%d", ctx.Conversation.ID)
	case "client_id":
		return ctx.Conversation.ClientID.String()
	case "client_name":
		if ctx.Client != nil {
			return ctx.Client.Name
		}
		return ""
	case "department_id":
		if ctx.Conversation.DepartmentID != nil {
			return fmt.Sprintf("%d", *ctx.Conversation.DepartmentID)
		}
		return ""
	case "channel_id":
		return ctx.Conversation.ChannelID
	default:
		return variable
	}
}

// convertToType converts a string value to the specified data type
func convertToType(value string, dataType string) interface{} {
	switch dataType {
	case models.ToolParamDataTypeInt:
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	case models.ToolParamDataTypeFloat:
		var f float64
		fmt.Sscanf(value, "%f", &f)
		return f
	case models.ToolParamDataTypeBool:
		return value == "true" || value == "1"
	default:
		return value
	}
}
