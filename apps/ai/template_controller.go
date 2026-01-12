package ai

import (
	"strings"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
)

const (
	SettingKeyBotPromptTemplate = "ai.bot_prompt_template"
)

// GetDefaultTemplateHandler returns the default bot prompt template
func GetDefaultTemplateHandler(r *evo.Request) any {
	return response.OK(map[string]interface{}{
		"template":          GetDefaultBotPromptTemplate(),
		"available_variables": getAvailableVariables(),
	})
}

// GetCurrentTemplateHandler returns the current bot prompt template (custom or default)
func GetCurrentTemplateHandler(r *evo.Request) any {
	template := settings.Get(SettingKeyBotPromptTemplate, "").String()
	if template == "" {
		template = GetDefaultBotPromptTemplate()
	}

	return response.OK(map[string]interface{}{
		"template":            template,
		"is_custom":           settings.Get(SettingKeyBotPromptTemplate, "").String() != "",
		"available_variables": getAvailableVariables(),
	})
}

// PreviewTemplateHandler renders a template with AI agent data for preview
func PreviewTemplateHandler(r *evo.Request) any {
	var req struct {
		AgentID  uint   `json:"agent_id"`
		Template string `json:"template"`
	}

	if err := r.BodyParser(&req); err != nil {
		return response.BadRequest(nil, "Invalid request body")
	}

	// Get the agent
	var agent models.AIAgent
	if err := db.Preload("Bot").First(&agent, req.AgentID).Error; err != nil {
		return response.NotFound(nil, "AI Agent not found")
	}

	// Get project name
	projectName := settings.Get("general.project_name", "").String()
	if projectName == "" {
		projectName = settings.Get("general.company_name", "").String()
	}

	// Build template data
	data := BuildTemplateData(&agent, projectName)

	// Use provided template or current template
	template := req.Template
	if template == "" {
		template = settings.Get(SettingKeyBotPromptTemplate, "").String()
		if template == "" {
			template = GetDefaultBotPromptTemplate()
		}
	}

	// Render the template
	result, err := RenderBotPromptTemplate(template, data)
	if err != nil {
		return response.BadRequest(nil, "Template rendering error: "+err.Error())
	}

	// Load agent tools
	var tools []models.AIAgentTool
	db.Where("ai_agent_id = ?", req.AgentID).Order("id ASC").Find(&tools)

	// Generate tool documentation separately (not part of template)
	toolDocs := GenerateToolDocumentation(&agent, tools)

	return response.OK(map[string]interface{}{
		"prompt":        result,
		"tool_docs":     toolDocs,
		"template_data": data,
	})
}

// ValidateTemplateHandler validates a template syntax
func ValidateTemplateHandler(r *evo.Request) any {
	var req struct {
		Template string `json:"template"`
	}

	if err := r.BodyParser(&req); err != nil {
		return response.BadRequest(nil, "Invalid request body")
	}

	if req.Template == "" {
		return response.BadRequest(nil, "Template is required")
	}

	// Try to render with sample data to validate
	sampleData := TemplateData{
		BotName:               "TestBot",
		ProjectName:           "TestProject",
		AgentName:             "Test Agent",
		GreetingMessage:       "Hello!",
		HandoverEnabled:       true,
		MultiLanguage:         true,
		InternetAccess:        false,
		UseKnowledgeBase:      true,
		UnitConversion:        true,
		CollectUserInfo:       true,
		PriorityDetection:     true,
		AutoTagging:           true,
		UseEmojis:             true,
		Tone:                  "casual",
		ToneDescription:       "friendly, conversational",
		HumorLevel:            50,
		FormalityLevel:        50,
		PersonalityDescription: "moderate humor, balanced formality, use emojis",
		MaxResponseLength:     500,
		MaxResponseWords:      375,
		MaxToolCalls:          5,
		ContextWindow:         10,
		Instructions:          "Test instructions",
		BlockedTopics:         "politics, religion",
		CollectUserInfoFields: "name, email, phone",
	}

	_, err := RenderBotPromptTemplate(req.Template, sampleData)
	if err != nil {
		return response.OK(map[string]interface{}{
			"valid":   false,
			"error":   err.Error(),
		})
	}

	return response.OK(map[string]interface{}{
		"valid": true,
	})
}

// getAvailableVariables returns the list of variables available in the template
func getAvailableVariables() []map[string]string {
	return []map[string]string{
		{"name": "BotName", "type": "string", "description": "Display name of the bot user"},
		{"name": "ProjectName", "type": "string", "description": "Project/company name from settings"},
		{"name": "AgentName", "type": "string", "description": "Name of the AI agent"},
		{"name": "GreetingMessage", "type": "string", "description": "Custom greeting message"},
		{"name": "Rules", "type": "string", "description": "Auto-generated numbered rules list (scope limits, tone, language, knowledge base usage, handover, response limits, etc.)"},
		{"name": "Instructions", "type": "string", "description": "Custom instructions for the agent"},
		{"name": "HandoverEnabled", "type": "bool", "description": "Whether handover to human is enabled"},
		{"name": "MultiLanguage", "type": "bool", "description": "Whether to respond in user's language"},
		{"name": "InternetAccess", "type": "bool", "description": "Whether web search is available"},
		{"name": "UseKnowledgeBase", "type": "bool", "description": "Whether to use knowledge base search"},
		{"name": "UnitConversion", "type": "bool", "description": "Whether to auto-convert units"},
		{"name": "CollectUserInfo", "type": "bool", "description": "Whether to collect user information"},
		{"name": "PriorityDetection", "type": "bool", "description": "Whether to detect conversation priority"},
		{"name": "AutoTagging", "type": "bool", "description": "Whether to auto-tag conversations"},
		{"name": "UseEmojis", "type": "bool", "description": "Whether to use emojis in responses"},
		{"name": "Tone", "type": "string", "description": "Tone setting (formal, casual, etc.)"},
		{"name": "ToneDescription", "type": "string", "description": "Human-readable tone description"},
		{"name": "HumorLevel", "type": "int", "description": "Humor level (0-100)"},
		{"name": "FormalityLevel", "type": "int", "description": "Formality level (0-100)"},
		{"name": "PersonalityDescription", "type": "string", "description": "Combined personality traits"},
		{"name": "MaxResponseLength", "type": "int", "description": "Max response length in tokens"},
		{"name": "MaxResponseWords", "type": "int", "description": "Approximate max words"},
		{"name": "MaxToolCalls", "type": "int", "description": "Maximum tool calls per message"},
		{"name": "ContextWindow", "type": "int", "description": "Number of messages for context"},
		{"name": "BlockedTopics", "type": "string", "description": "Topics the agent should refuse to discuss"},
		{"name": "CollectUserInfoFields", "type": "string", "description": "Fields to collect from user"},
	}
}

// GenerateToolDocumentation generates the tool documentation section
// This is generated separately and not part of the customizable template
func GenerateToolDocumentation(agent *models.AIAgent, tools []models.AIAgentTool) string {
	// Use the existing tools section generation from template_generator.go
	ctx := TemplateContext{
		Agent: agent,
		Tools: tools,
	}

	// Get just the tools section
	return generateToolsSection(ctx)
}

// generateToolsSection generates just the tools documentation
func generateToolsSection(ctx TemplateContext) string {
	agent := ctx.Agent
	hasTools := len(ctx.Tools) > 0 || agent.HandoverEnabled || agent.UseKnowledgeBase || agent.CollectUserInfo || agent.PriorityDetection || agent.AutoTagging

	if !hasTools {
		return ""
	}

	var lines []string
	lines = append(lines, "## Tools")
	lines = append(lines, "Ask user for missing required params before calling.")
	lines = append(lines, "")

	// Knowledge base search tool
	if agent.UseKnowledgeBase {
		lines = append(lines, "`searchKnowledgeBase(query:string)` - Search the knowledge base for information.")
		lines = append(lines, "**CRITICAL: ALWAYS search FIRST before answering any question.**")
		lines = append(lines, "  - ONLY use information from search results - nothing else")
		lines = append(lines, "  - If no results or not relevant → say ONLY \"I don't have information about that.\" and STOP")
		lines = append(lines, "  - Do NOT add suggestions, tips, or generic advice after saying you don't have info")
		lines = append(lines, "")
	}

	// Collect user info tool
	if agent.CollectUserInfo && agent.CollectUserInfoFields != "" {
		lines = append(lines, "`setUserInfo(data:object)` - Save collected user information.")
		lines = append(lines, "  Fields to collect: "+agent.CollectUserInfoFields)
		lines = append(lines, "  Call with JSON object containing collected fields")
		lines = append(lines, "  Ask naturally during conversation, don't demand all at once")
		lines = append(lines, "")
	}

	// Priority detection tool
	if agent.PriorityDetection {
		lines = append(lines, "`setPriority(priority:string)` - Set conversation priority based on urgency.")
		lines = append(lines, "  Options: \"low\", \"medium\", \"high\", \"urgent\"")
		lines = append(lines, "  Use when: user expresses urgency, mentions deadlines, or has critical issues")
		lines = append(lines, "")
	}

	// Auto tagging tool
	if agent.AutoTagging {
		lines = append(lines, "`setTag(tag:string)` - Tag the conversation based on topic.")
		lines = append(lines, "  Automatically detect and apply relevant tags from conversation context")
		lines = append(lines, "  Use early in conversation when topic becomes clear")
		lines = append(lines, "")
	}

	// Handover tool
	if agent.HandoverEnabled && agent.HandoverUserID != nil {
		lines = append(lines, "`handover(reason:string)` - Transfer to human agent when: user requests, issue unresolvable, needs authorization")
		lines = append(lines, "")
	}

	// Custom tools
	for _, tool := range ctx.Tools {
		lines = append(lines, "`"+tool.Name+"` ["+tool.Method+" "+tool.Endpoint+"]")
		if tool.Description != "" {
			lines = append(lines, "  Use: "+tool.Description)
		}

		qp := formatToolParams(tool.QueryParams)
		hp := formatToolParams(tool.HeaderParams)
		bp := formatToolParams(tool.BodyParams)

		if qp != "" {
			lines = append(lines, "  Query: "+qp)
		}
		if hp != "" {
			lines = append(lines, "  Headers: "+hp)
		}
		if bp != "" {
			lines = append(lines, "  Body("+tool.BodyType+"): "+bp)
		}
		if tool.AuthorizationType != models.ToolAuthTypeNone {
			lines = append(lines, "  Auth: "+tool.AuthorizationType)
		}
		if tool.ResponseInstructions != "" {
			lines = append(lines, "  Response: "+tool.ResponseInstructions)
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
