package ai

import (
	"fmt"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/models"
)

// ProcessIncomingMessage is the main entry point for AI agent processing
// It should be called from the Message.AfterCreate hook
func ProcessIncomingMessage(message *models.Message) error {
	// 1. Validate this is an incoming customer message
	// Skip if it's from an agent (UserID is set) or is a system message
	if message.UserID != nil || message.IsSystemMessage {
		return nil
	}

	// Skip if no client ID
	if message.ClientID == nil {
		return nil
	}

	// 2. Load conversation with department and AI agent
	var conversation models.Conversation
	err := db.Preload("Department").
		Preload("Department.AIAgent").
		Preload("Department.AIAgent.Bot").
		Preload("Client").
		First(&conversation, message.ConversationID).Error
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}

	// 3. Check if department has an AI agent assigned
	if conversation.Department == nil || conversation.Department.AIAgentID == nil {
		log.Debug("No AI agent configured for department, skipping")
		return nil // No AI agent configured
	}

	aiAgent := conversation.Department.AIAgent
	if aiAgent == nil {
		return nil // AI agent not loaded
	}

	// 4. Check if AI agent is active
	if aiAgent.Status != models.AIAgentStatusActive {
		log.Debug("AI agent %d is not active, skipping", aiAgent.ID)
		return nil
	}

	// 4.5. Check if bot handling is enabled for this conversation
	if !conversation.HandleByBot {
		log.Debug("Bot handling disabled for conversation %d, skipping", conversation.ID)
		return nil
	}

	// 5. Check if bot user exists
	if aiAgent.Bot == nil {
		log.Warning("AI agent %d has no bot user configured", aiAgent.ID)
		return nil
	}

	// 6. Check conversation assignments - AI should only respond if:
	//    - No human is assigned to the conversation, OR
	//    - Only the bot is assigned to the conversation
	var assignments []models.ConversationAssignment
	if err := db.Preload("User").
		Where("conversation_id = ?", conversation.ID).
		Find(&assignments).Error; err != nil {
		log.Warning("Failed to load conversation assignments: %v", err)
	}

	// Check if any human agent (non-bot) is assigned
	botUserID := aiAgent.Bot.UserID.String()
	for _, assignment := range assignments {
		if assignment.UserID != nil && assignment.User != nil {
			assignedUserID := assignment.UserID.String()
			// If a human agent (not the bot) is assigned, don't respond
			if assignedUserID != botUserID && assignment.User.Type != "bot" {
				log.Debug("Human agent %s is assigned to conversation %d, AI agent will not respond",
					assignment.User.DisplayName, conversation.ID)
				return nil
			}
		}
	}

	// 7. Load AI agent tools
	var agentTools []models.AIAgentTool
	if err := db.Where("ai_agent_id = ?", aiAgent.ID).Find(&agentTools).Error; err != nil {
		log.Warning("Failed to load AI agent tools: %v", err)
		agentTools = []models.AIAgentTool{}
	}

	// 8. Build context
	ctx := &AgentContext{
		Conversation: &conversation,
		Department:   conversation.Department,
		AIAgent:      aiAgent,
		AgentTools:   agentTools,
		Client:       &conversation.Client,
		Bot:          aiAgent.Bot,
	}

	// 9. Process with agent
	log.Debug("AI agent %d processing message for conversation %d", aiAgent.ID, conversation.ID)
	return processWithAgent(ctx, message)
}

// processWithAgent handles the AI agent processing loop
func processWithAgent(ctx *AgentContext, message *models.Message) error {
	// Get OpenAI client
	client := GetClient()
	if client == nil {
		return fmt.Errorf("AI client not initialized")
	}

	// 1. Generate system prompt
	systemPrompt := GenerateSystemPrompt(ctx.AIAgent, ctx.AgentTools, ctx.Client, ctx.Conversation)

	// 2. Get recent messages based on context window
	contextWindow := ctx.AIAgent.ContextWindow
	if contextWindow <= 0 {
		contextWindow = 10
	}

	var recentMessages []models.Message
	err := db.Where("conversation_id = ?", ctx.Conversation.ID).
		Preload("User").
		Order("created_at DESC").
		Limit(contextWindow).
		Find(&recentMessages).Error
	if err != nil {
		return fmt.Errorf("failed to load recent messages: %w", err)
	}

	// Reverse to get chronological order
	for i, j := 0, len(recentMessages)-1; i < j; i, j = i+1, j-1 {
		recentMessages[i], recentMessages[j] = recentMessages[j], recentMessages[i]
	}

	// 3. Format conversation history
	history := FormatConversationHistory(recentMessages, ctx.Bot.UserID.String())

	// 4. Build messages array with system prompt
	messages := []ToolMessage{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	// 5. Build tool definitions
	tools := BuildToolsForAgent(ctx.AIAgent, ctx.AgentTools)

	// 6. Process with tool call loop
	maxIterations := ctx.AIAgent.MaxToolCalls
	if maxIterations <= 0 {
		maxIterations = 5
	}

	maxTokens := ctx.AIAgent.MaxResponseLength
	if maxTokens <= 0 {
		maxTokens = 2000
	}

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Call OpenAI
		response, err := client.ChatCompletionWithTools(messages, tools, maxTokens, 0.7)
		if err != nil {
			return fmt.Errorf("OpenAI API error: %w", err)
		}

		if len(response.Choices) == 0 {
			return fmt.Errorf("no response from OpenAI")
		}

		choice := response.Choices[0]

		// Check if there are tool calls
		if len(choice.Message.ToolCalls) > 0 {
			// Add assistant message with tool calls to history
			messages = append(messages, CreateAssistantMessageWithToolCalls(choice.Message.Content, choice.Message.ToolCalls))

			// Execute each tool call
			shouldStop := false
			for _, toolCall := range choice.Message.ToolCalls {
				result, stop, err := ExecuteTool(ctx, toolCall)
				if err != nil {
					log.Warning("Tool execution error for %s: %v", toolCall.Function.Name, err)
					result = fmt.Sprintf("Error executing tool: %v", err)
				}

				// Add tool result to messages
				messages = append(messages, CreateToolResultMessage(toolCall.ID, result))

				if stop {
					shouldStop = true
				}
			}

			// If handover was called, stop processing
			if shouldStop {
				log.Info("AI agent processing stopped for conversation %d (handover or stop condition)", ctx.Conversation.ID)
				return nil
			}

			// Continue the loop to get the final response
			continue
		}

		// No tool calls - we have a final response
		if choice.Message.Content != "" {
			return sendBotResponse(ctx, choice.Message.Content)
		}

		// Empty response
		return nil
	}

	// Max iterations reached
	log.Warning("AI agent reached max tool call iterations for conversation %d", ctx.Conversation.ID)
	return nil
}

// sendBotResponse sends a message as the bot user
func sendBotResponse(ctx *AgentContext, content string) error {
	if content == "" {
		return nil
	}

	// Parse bot user ID
	botUserID, err := uuid.Parse(ctx.Bot.UserID.String())
	if err != nil {
		return fmt.Errorf("invalid bot user ID: %w", err)
	}

	// Create message
	msg := models.Message{
		ConversationID: ctx.Conversation.ID,
		UserID:         &botUserID,
		Body:           content,
	}

	// Save message - this will trigger the AfterCreate hook
	// which broadcasts to NATS/WebSocket and sends to external channels
	if err := db.Create(&msg).Error; err != nil {
		return fmt.Errorf("failed to create bot message: %w", err)
	}

	log.Info("AI agent sent response for conversation %d", ctx.Conversation.ID)

	return nil
}
