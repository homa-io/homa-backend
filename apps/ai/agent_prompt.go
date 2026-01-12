package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iesreza/homa-backend/apps/models"
)

// GenerateSystemPrompt creates a system prompt from AIAgent configuration
// Uses the Jet template system with customizable template and separate tool documentation
func GenerateSystemPrompt(agent *models.AIAgent, tools []models.AIAgentTool, client *models.Client, conversation *models.Conversation) string {
	// Get project name from settings
	projectName := models.GetSettingValue("general.project_name", "")
	if projectName == "" {
		projectName = models.GetSettingValue("general.company_name", "the company")
	}

	// Build template data for Jet template
	templateData := BuildTemplateData(agent, projectName)

	// Get the custom template from settings, or use default
	customTemplate := models.GetSettingValue(SettingKeyBotPromptTemplate, "")
	templateContent := customTemplate
	if templateContent == "" {
		templateContent = GetDefaultBotPromptTemplate()
	}

	// Render the Jet template
	prompt, err := RenderBotPromptTemplate(templateContent, templateData)
	if err != nil {
		// Fall back to default template on error
		prompt, _ = RenderBotPromptTemplate(GetDefaultBotPromptTemplate(), templateData)
	}

	// Generate tool documentation separately (not customizable)
	toolDocs := GenerateToolDocumentation(agent, tools)

	// Combine prompt and tool docs
	result := prompt
	if toolDocs != "" {
		result = prompt + "\n\n" + toolDocs
	}

	// Add customer context if available
	customerContext := generateCustomerContext(client, conversation)
	if customerContext != "" {
		result += "\n\n" + customerContext
	}

	return result
}

// generateCustomerContext creates context about the current customer
func generateCustomerContext(client *models.Client, conversation *models.Conversation) string {
	if client == nil {
		return ""
	}

	var parts []string
	parts = append(parts, "## Customer Context")

	if client.Name != "" {
		parts = append(parts, fmt.Sprintf("- Customer name: %s", client.Name))
	}

	if client.Language != nil && *client.Language != "" {
		parts = append(parts, fmt.Sprintf("- Preferred language: %s", *client.Language))
	}

	if client.Timezone != nil && *client.Timezone != "" {
		parts = append(parts, fmt.Sprintf("- Timezone: %s", *client.Timezone))
	}

	// Add any stored user info from previous interactions
	if client.Data != nil {
		var clientData map[string]interface{}
		if err := json.Unmarshal(client.Data, &clientData); err == nil && len(clientData) > 0 {
			parts = append(parts, "- Known information:")
			for k, v := range clientData {
				parts = append(parts, fmt.Sprintf("  - %s: %v", k, v))
			}
		}
	}

	// Conversation context
	if conversation != nil {
		if conversation.Priority != "" && conversation.Priority != models.ConversationPriorityMedium {
			parts = append(parts, fmt.Sprintf("- Current priority: %s", conversation.Priority))
		}
		if conversation.Channel.Name != "" {
			parts = append(parts, fmt.Sprintf("- Channel: %s", conversation.Channel.Name))
		}
	}

	if len(parts) > 1 {
		return strings.Join(parts, "\n")
	}
	return ""
}

// FormatConversationHistory formats recent messages for the AI context
func FormatConversationHistory(messages []models.Message, botID string) []ToolMessage {
	var history []ToolMessage

	for _, msg := range messages {
		var role string
		if msg.IsSystemMessage {
			role = "system"
		} else if msg.UserID != nil && msg.UserID.String() == botID {
			role = "assistant"
		} else if msg.UserID != nil {
			// Message from another agent (not the bot)
			role = "user"
		} else if msg.ClientID != nil {
			role = "user"
		} else {
			role = "user"
		}

		// Add sender context for user messages
		content := msg.Body
		if role == "user" && msg.UserID != nil && msg.UserID.String() != botID {
			// This is from another agent, add context
			if msg.User != nil && msg.User.DisplayName != "" {
				content = fmt.Sprintf("[Agent %s]: %s", msg.User.DisplayName, msg.Body)
			}
		}

		history = append(history, ToolMessage{
			Role:    role,
			Content: content,
		})
	}

	return history
}
