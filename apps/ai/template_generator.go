package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iesreza/homa-backend/apps/models"
)

// ToneDescriptions maps tone constants to descriptions
var ToneDescriptions = map[string]string{
	models.AIAgentToneFormal:     "professional, business-appropriate",
	models.AIAgentToneCasual:     "friendly, conversational",
	models.AIAgentToneDetailed:   "comprehensive with examples",
	models.AIAgentTonePrecise:    "concise, to-the-point",
	models.AIAgentToneEmpathetic: "warm, understanding",
	models.AIAgentToneTechnical:  "technical terminology preferred",
}

// TemplateContext holds all data needed for template generation
type TemplateContext struct {
	ProjectName        string
	Agent              *models.AIAgent
	Tools              []models.AIAgentTool
	KnowledgeBaseItems []KnowledgeBaseItem
}

// KnowledgeBaseItem represents a knowledge base article for template
type KnowledgeBaseItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GenerateAgentTemplate generates the complete system prompt template for an AI agent
func GenerateAgentTemplate(ctx TemplateContext) string {
	agent := ctx.Agent
	botName := "Assistant"
	if agent.Bot != nil {
		if agent.Bot.DisplayName != "" {
			botName = agent.Bot.DisplayName
		} else if agent.Bot.Name != "" {
			botName = agent.Bot.Name
		}
	}

	project := ctx.ProjectName
	if project == "" {
		project = "the company"
	}

	var sections []string

	// Identity section
	identity := []string{
		"# Identity",
		fmt.Sprintf("You are **%s**, an AI customer support assistant for **%s**.", botName, project),
		"Your role: Help users with questions, troubleshoot issues, and provide accurate information.",
		fmt.Sprintf("Always introduce yourself as \"%s\" when greeting users.", botName),
	}

	if agent.GreetingMessage != "" {
		identity = append(identity, "")
		identity = append(identity, "**Greeting:** When starting a new conversation, greet with:")
		identity = append(identity, fmt.Sprintf("\"%s\"", strings.TrimSpace(agent.GreetingMessage)))
	}

	sections = append(sections, strings.Join(identity, "\n"))

	// Rules section
	rules := []string{}

	// CRITICAL: Strict scope limitation - MUST BE FIRST
	rules = append(rules, "**ABSOLUTE RULE**: You can ONLY answer using information returned by your tools. No exceptions.")
	rules = append(rules, "When you have NO information from tools, respond with EXACTLY: \"I don't have information about that.\" - then STOP. Say nothing else.")
	rules = append(rules, "FORBIDDEN: Do NOT say \"however\", \"but\", \"you could try\", \"I recommend\", \"common solutions\", \"you might want to\" - NEVER add suggestions")
	rules = append(rules, "FORBIDDEN: Do NOT give tips, advice, troubleshooting steps, or recommendations unless they came from a tool result")
	rules = append(rules, "If a user asks something and the tool returns nothing useful → your ONLY response is: \"I don't have information about that.\"")

	if agent.HandoverEnabled {
		rules = append(rules, "If user asks for more help after you said you don't have info → offer handover: \"Would you like me to connect you with a human agent?\"")
	}

	if agent.MultiLanguage {
		rules = append(rules, "Your response should match user language exactly")
	}

	// Tone
	toneDesc := ToneDescriptions[agent.Tone]
	if toneDesc == "" {
		toneDesc = "professional and helpful"
	}
	rules = append(rules, fmt.Sprintf("Tone: %s", toneDesc))

	// Personality traits
	var personalityParts []string
	if agent.HumorLevel > 0 {
		var humorDesc string
		if agent.HumorLevel <= 30 {
			humorDesc = "minimal humor"
		} else if agent.HumorLevel <= 70 {
			humorDesc = "moderate humor"
		} else {
			humorDesc = "playful/witty"
		}
		personalityParts = append(personalityParts, humorDesc)
	}
	if agent.FormalityLevel > 0 {
		var formalDesc string
		if agent.FormalityLevel <= 30 {
			formalDesc = "casual style"
		} else if agent.FormalityLevel <= 70 {
			formalDesc = "balanced formality"
		} else {
			formalDesc = "highly formal"
		}
		personalityParts = append(personalityParts, formalDesc)
	}
	if agent.UseEmojis {
		personalityParts = append(personalityParts, "use emojis")
	}
	if len(personalityParts) > 0 {
		rules = append(rules, fmt.Sprintf("Personality: %s", strings.Join(personalityParts, ", ")))
	}

	if agent.UseKnowledgeBase {
		rules = append(rules, "Use searchKnowledgeBase tool for answers - no fabrication")
	}

	if agent.InternetAccess {
		rules = append(rules, "Web search available")
	} else {
		rules = append(rules, "No internet - use provided context only")
	}

	if agent.UnitConversion {
		rules = append(rules, "Convert units: bytes→MB/GB, seconds→mins/hrs, timestamps→readable")
	}

	if agent.HandoverEnabled {
		rules = append(rules, "Human handover available (warn: slower response)")
	}

	// Blocked topics
	if agent.BlockedTopics != "" {
		rules = append(rules, fmt.Sprintf("Refuse to discuss: %s", strings.TrimSpace(agent.BlockedTopics)))
	}

	// Collect user info
	if agent.CollectUserInfo && agent.CollectUserInfoFields != "" {
		fields := strings.Split(agent.CollectUserInfoFields, ",")
		var cleanFields []string
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f != "" {
				cleanFields = append(cleanFields, f)
			}
		}
		if len(cleanFields) > 0 {
			rules = append(rules, fmt.Sprintf("Proactively ask user for: %s. Once collected, call setUserInfo tool.", strings.Join(cleanFields, ", ")))
		}
	}

	// Max response length
	if agent.MaxResponseLength > 0 {
		words := int(float64(agent.MaxResponseLength) * 0.75)
		rules = append(rules, fmt.Sprintf("Keep responses under %d tokens (~%d words)", agent.MaxResponseLength, words))
	}

	// Max tool calls
	if agent.MaxToolCalls > 0 {
		rules = append(rules, fmt.Sprintf("Max %d tool calls per message", agent.MaxToolCalls))
	}

	// Context window
	if agent.ContextWindow > 0 {
		rules = append(rules, fmt.Sprintf("Use last %d messages for context", agent.ContextWindow))
	}

	// Format rules with numbers
	rulesFormatted := make([]string, len(rules))
	for i, r := range rules {
		rulesFormatted[i] = fmt.Sprintf("%d. %s", i+1, r)
	}
	sections = append(sections, "## Rules\n"+strings.Join(rulesFormatted, "\n"))

	// Custom instructions
	if agent.Instructions != "" {
		sections = append(sections, fmt.Sprintf("## Instructions\n%s", strings.TrimSpace(agent.Instructions)))
	}

	// Tools section
	hasTools := len(ctx.Tools) > 0 || agent.HandoverEnabled || agent.UseKnowledgeBase || agent.CollectUserInfo || agent.PriorityDetection || agent.AutoTagging
	if hasTools {
		toolLines := []string{"## Tools", "Ask user for missing required params before calling.", ""}

		// Knowledge base search tool
		if agent.UseKnowledgeBase {
			toolLines = append(toolLines, "`searchKnowledgeBase(query:string)` - Search the knowledge base for information.")
			toolLines = append(toolLines, "**CRITICAL: ALWAYS search FIRST before answering any question.**")
			toolLines = append(toolLines, "  - ONLY use information from search results - nothing else")
			toolLines = append(toolLines, "  - If no results or not relevant → say ONLY \"I don't have information about that.\" and STOP")
			toolLines = append(toolLines, "  - Do NOT add suggestions, tips, or generic advice after saying you don't have info")
			toolLines = append(toolLines, "**Topics covered:**")
			if len(ctx.KnowledgeBaseItems) > 0 {
				for _, kb := range ctx.KnowledgeBaseItems {
					toolLines = append(toolLines, fmt.Sprintf("  - %s", kb.Title))
				}
			} else {
				toolLines = append(toolLines, "  - Product information, FAQs, documentation, policies, guides")
			}
			toolLines = append(toolLines, "")
		}

		// Collect user info tool
		if agent.CollectUserInfo && agent.CollectUserInfoFields != "" {
			fields := strings.Split(agent.CollectUserInfoFields, ",")
			var cleanFields []string
			for _, f := range fields {
				f = strings.TrimSpace(f)
				if f != "" {
					cleanFields = append(cleanFields, f)
				}
			}
			if len(cleanFields) > 0 {
				toolLines = append(toolLines, "`setUserInfo(data:object)` - Save collected user information.")
				toolLines = append(toolLines, fmt.Sprintf("  Fields to collect: %s", strings.Join(cleanFields, ", ")))
				toolLines = append(toolLines, "  Call with JSON object containing collected fields, e.g.: {\"name\": \"John\", \"email\": \"john@example.com\"}")
				toolLines = append(toolLines, "  Ask naturally during conversation, don't demand all at once")
				toolLines = append(toolLines, "")
			}
		}

		// Priority detection tool
		if agent.PriorityDetection {
			toolLines = append(toolLines, "`setPriority(priority:string)` - Set conversation priority based on urgency.")
			toolLines = append(toolLines, "  Options: \"low\", \"medium\", \"high\", \"urgent\"")
			toolLines = append(toolLines, "  Use when: user expresses urgency, mentions deadlines, or has critical issues")
			toolLines = append(toolLines, "")
		}

		// Auto tagging tool
		if agent.AutoTagging {
			toolLines = append(toolLines, "`setTag(tag:string)` - Tag the conversation based on topic.")
			toolLines = append(toolLines, "  Automatically detect and apply relevant tags from conversation context")
			toolLines = append(toolLines, "  Use early in conversation when topic becomes clear")
			toolLines = append(toolLines, "")
		}

		// Handover tool
		if agent.HandoverEnabled && agent.HandoverUserID != nil {
			toolLines = append(toolLines, "`handover(reason:string)` - Transfer to human agent when: user requests, issue unresolvable, needs authorization")
			toolLines = append(toolLines, "")
		}

		// Custom tools
		for _, tool := range ctx.Tools {
			toolLines = append(toolLines, fmt.Sprintf("`%s` [%s %s]", tool.Name, tool.Method, tool.Endpoint))
			if tool.Description != "" {
				toolLines = append(toolLines, fmt.Sprintf("  Use: %s", tool.Description))
			}

			qp := formatToolParams(tool.QueryParams)
			hp := formatToolParams(tool.HeaderParams)
			bp := formatToolParams(tool.BodyParams)

			if qp != "" {
				toolLines = append(toolLines, fmt.Sprintf("  Query: %s", qp))
			}
			if hp != "" {
				toolLines = append(toolLines, fmt.Sprintf("  Headers: %s", hp))
			}
			if bp != "" {
				toolLines = append(toolLines, fmt.Sprintf("  Body(%s): %s", tool.BodyType, bp))
			}
			if tool.AuthorizationType != models.ToolAuthTypeNone {
				toolLines = append(toolLines, fmt.Sprintf("  Auth: %s", tool.AuthorizationType))
			}
			if tool.ResponseInstructions != "" {
				toolLines = append(toolLines, fmt.Sprintf("  Response: %s", tool.ResponseInstructions))
			}
			toolLines = append(toolLines, "")
		}

		sections = append(sections, strings.Join(toolLines, "\n"))
	}

	// Context usage
	sections = append(sections, "## Context\nUse conversation history: maintain context, don't re-ask known info, track multi-step issues.")

	return strings.Join(sections, "\n\n")
}

// formatToolParams formats tool parameters for display
func formatToolParams(paramsJSON []byte) string {
	if len(paramsJSON) == 0 {
		return ""
	}

	var params []models.ToolParam
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return ""
	}

	if len(params) == 0 {
		return ""
	}

	var parts []string
	for _, p := range params {
		part := fmt.Sprintf("`%s`", p.Key)
		if p.Required {
			part += " *req*"
		}
		if p.ValueType == models.ToolParamValueTypeConstant && p.Value != "" {
			part += fmt.Sprintf(" =\"%s\"", p.Value)
		} else if p.ValueType == models.ToolParamValueTypeByModel {
			part += " (AI fills)"
		}
		if p.DataType != models.ToolParamDataTypeString {
			part += fmt.Sprintf(" [%s]", p.DataType)
		}
		if p.Example != "" {
			part += fmt.Sprintf(" ex:\"%s\"", p.Example)
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, " | ")
}
