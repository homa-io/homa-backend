package ai

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/iesreza/homa-backend/apps/models"
)

// DefaultBotPromptTemplate is the default Jet template for AI agent prompts
// Users can customize this via settings
// Note: Tool documentation is generated separately and appended automatically
const DefaultBotPromptTemplate = `# Identity
You are **{{BotName}}**, an AI customer support assistant for **{{ProjectName}}**.
Your role: Help users with questions, troubleshoot issues, and provide accurate information.
Always introduce yourself as "{{BotName}}" when greeting users.
{{if GreetingMessage != ""}}

**Greeting:** When starting a new conversation, greet with:
"{{GreetingMessage}}"
{{end}}

## Rules

### Critical Scope Rules
- **ABSOLUTE RULE**: You can ONLY answer using information returned by your tools. No exceptions.
- When you have NO information from tools, respond with EXACTLY: "I don't have information about that." - then STOP. Say nothing else.
- FORBIDDEN: Do NOT say "however", "but", "you could try", "I recommend", "common solutions", "you might want to" - NEVER add suggestions
- FORBIDDEN: Do NOT give tips, advice, troubleshooting steps, or recommendations unless they came from a tool result
- If a user asks something and the tool returns nothing useful → your ONLY response is: "I don't have information about that."
{{if HandoverEnabled}}
- If user asks for more help after you said you don't have info → offer handover: "Would you like me to connect you with a human agent?"
{{end}}

### Communication Style
- Tone: {{ToneDescription}}
{{if PersonalityDescription != ""}}
- Personality: {{PersonalityDescription}}
{{end}}
{{if MultiLanguage}}
- Your response should match user language exactly
{{end}}
{{if UseEmojis}}
- Use emojis appropriately in responses
{{end}}

### Capabilities
{{if UseKnowledgeBase}}
- Use searchKnowledgeBase tool for answers - no fabrication
{{end}}
{{if InternetAccess}}
- Web search available when needed
{{else}}
- No internet access - use provided context only
{{end}}
{{if UnitConversion}}
- Convert units automatically: bytes→MB/GB, seconds→mins/hrs, timestamps→readable dates
{{end}}
{{if HandoverEnabled}}
- Human handover available when needed (note: may have slower response time)
{{end}}
{{if CollectUserInfo && CollectUserInfoFields != ""}}
- Proactively collect user information: {{CollectUserInfoFields}}. Once collected, call setUserInfo tool.
{{end}}
{{if PriorityDetection}}
- Detect conversation priority based on urgency cues and set accordingly
{{end}}
{{if AutoTagging}}
- Automatically tag conversations based on detected topics
{{end}}

### Restrictions
{{if BlockedTopics != ""}}
- Refuse to discuss: {{BlockedTopics}}
{{end}}
{{if MaxResponseLength > 0}}
- Keep responses under {{MaxResponseLength}} tokens (~{{MaxResponseWords}} words)
{{end}}
{{if MaxToolCalls > 0}}
- Maximum {{MaxToolCalls}} tool calls per message
{{end}}
{{if ContextWindow > 0}}
- Use last {{ContextWindow}} messages for context
{{end}}
{{if Instructions != ""}}

## Custom Instructions
{{Instructions}}
{{end}}

## Context Management
Use conversation history effectively: maintain context across messages, don't re-ask for information already provided, and track multi-step issues to resolution.`

// TemplateData holds all data that can be used in the bot prompt template
type TemplateData struct {
	// Identity
	BotName     string `json:"bot_name"`
	ProjectName string `json:"project_name"`
	AgentName   string `json:"agent_name"`

	// Greeting
	GreetingMessage string `json:"greeting_message"`

	// Pre-generated rules (numbered list)
	Rules string `json:"rules"`

	// Behavior flags (for custom templates)
	HandoverEnabled   bool `json:"handover_enabled"`
	MultiLanguage     bool `json:"multi_language"`
	InternetAccess    bool `json:"internet_access"`
	UseKnowledgeBase  bool `json:"use_knowledge_base"`
	UnitConversion    bool `json:"unit_conversion"`
	CollectUserInfo   bool `json:"collect_user_info"`
	PriorityDetection bool `json:"priority_detection"`
	AutoTagging       bool `json:"auto_tagging"`
	UseEmojis         bool `json:"use_emojis"`

	// Tone and personality
	Tone                   string `json:"tone"`
	ToneDescription        string `json:"tone_description"`
	HumorLevel             int    `json:"humor_level"`
	FormalityLevel         int    `json:"formality_level"`
	PersonalityDescription string `json:"personality_description"`

	// Limits
	MaxResponseLength int `json:"max_response_length"`
	MaxResponseWords  int `json:"max_response_words"`
	MaxToolCalls      int `json:"max_tool_calls"`
	ContextWindow     int `json:"context_window"`

	// Content
	Instructions          string `json:"instructions"`
	BlockedTopics         string `json:"blocked_topics"`
	CollectUserInfoFields string `json:"collect_user_info_fields"`
}

// BuildTemplateData creates template data from an AI agent
func BuildTemplateData(agent *models.AIAgent, projectName string) TemplateData {
	botName := "Assistant"
	if agent.Bot != nil {
		if agent.Bot.DisplayName != "" {
			botName = agent.Bot.DisplayName
		} else if agent.Bot.Name != "" {
			botName = agent.Bot.Name
		}
	}

	if projectName == "" {
		projectName = "the company"
	}

	// Tone description
	toneDesc := ToneDescriptions[agent.Tone]
	if toneDesc == "" {
		toneDesc = "professional and helpful"
	}

	// Personality description
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

	// Calculate max response words
	maxWords := 0
	if agent.MaxResponseLength > 0 {
		maxWords = int(float64(agent.MaxResponseLength) * 0.75)
	}

	// Generate rules
	rules := generateRules(agent, toneDesc, strings.Join(personalityParts, ", "), maxWords)

	return TemplateData{
		BotName:                botName,
		ProjectName:            projectName,
		AgentName:              agent.Name,
		GreetingMessage:        strings.TrimSpace(agent.GreetingMessage),
		Rules:                  rules,
		HandoverEnabled:        agent.HandoverEnabled,
		MultiLanguage:          agent.MultiLanguage,
		InternetAccess:         agent.InternetAccess,
		UseKnowledgeBase:       agent.UseKnowledgeBase,
		UnitConversion:         agent.UnitConversion,
		CollectUserInfo:        agent.CollectUserInfo,
		PriorityDetection:      agent.PriorityDetection,
		AutoTagging:            agent.AutoTagging,
		UseEmojis:              agent.UseEmojis,
		Tone:                   agent.Tone,
		ToneDescription:        toneDesc,
		HumorLevel:             agent.HumorLevel,
		FormalityLevel:         agent.FormalityLevel,
		PersonalityDescription: strings.Join(personalityParts, ", "),
		MaxResponseLength:      agent.MaxResponseLength,
		MaxResponseWords:       maxWords,
		MaxToolCalls:           agent.MaxToolCalls,
		ContextWindow:          agent.ContextWindow,
		Instructions:           strings.TrimSpace(agent.Instructions),
		BlockedTopics:          strings.TrimSpace(agent.BlockedTopics),
		CollectUserInfoFields:  strings.TrimSpace(agent.CollectUserInfoFields),
	}
}

// generateRules creates the numbered rules list based on agent configuration
func generateRules(agent *models.AIAgent, toneDesc, personalityDesc string, maxWords int) string {
	var rules []string

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
	rules = append(rules, fmt.Sprintf("Tone: %s", toneDesc))

	// Personality traits
	if personalityDesc != "" {
		rules = append(rules, fmt.Sprintf("Personality: %s", personalityDesc))
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
		rules = append(rules, fmt.Sprintf("Keep responses under %d tokens (~%d words)", agent.MaxResponseLength, maxWords))
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
	var numberedRules []string
	for i, r := range rules {
		numberedRules = append(numberedRules, fmt.Sprintf("%d. %s", i+1, r))
	}

	return strings.Join(numberedRules, "\n")
}

// RenderBotPromptTemplate renders a Jet template with the given data
func RenderBotPromptTemplate(templateContent string, data TemplateData) (string, error) {
	// Create a new Jet template set with loader from memory
	loader := jet.NewInMemLoader()
	loader.Set("prompt", templateContent)

	set := jet.NewSet(loader)

	// Get the template
	tmpl, err := set.GetTemplate("prompt")
	if err != nil {
		return "", err
	}

	// Create vars map from data
	vars := make(jet.VarMap)
	vars.Set("BotName", data.BotName)
	vars.Set("ProjectName", data.ProjectName)
	vars.Set("AgentName", data.AgentName)
	vars.Set("GreetingMessage", data.GreetingMessage)
	vars.Set("Rules", data.Rules)
	vars.Set("HandoverEnabled", data.HandoverEnabled)
	vars.Set("MultiLanguage", data.MultiLanguage)
	vars.Set("InternetAccess", data.InternetAccess)
	vars.Set("UseKnowledgeBase", data.UseKnowledgeBase)
	vars.Set("UnitConversion", data.UnitConversion)
	vars.Set("CollectUserInfo", data.CollectUserInfo)
	vars.Set("PriorityDetection", data.PriorityDetection)
	vars.Set("AutoTagging", data.AutoTagging)
	vars.Set("UseEmojis", data.UseEmojis)
	vars.Set("Tone", data.Tone)
	vars.Set("ToneDescription", data.ToneDescription)
	vars.Set("HumorLevel", data.HumorLevel)
	vars.Set("FormalityLevel", data.FormalityLevel)
	vars.Set("PersonalityDescription", data.PersonalityDescription)
	vars.Set("MaxResponseLength", data.MaxResponseLength)
	vars.Set("MaxResponseWords", data.MaxResponseWords)
	vars.Set("MaxToolCalls", data.MaxToolCalls)
	vars.Set("ContextWindow", data.ContextWindow)
	vars.Set("Instructions", data.Instructions)
	vars.Set("BlockedTopics", data.BlockedTopics)
	vars.Set("CollectUserInfoFields", data.CollectUserInfoFields)

	// Render the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars, nil); err != nil {
		return "", err
	}

	// Clean up excess blank lines
	result := buf.String()
	result = cleanupTemplateOutput(result)

	return result, nil
}

// cleanupTemplateOutput removes excess blank lines from template output
func cleanupTemplateOutput(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	prevEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isEmpty := trimmed == ""

		// Skip consecutive empty lines
		if isEmpty && prevEmpty {
			continue
		}

		result = append(result, line)
		prevEmpty = isEmpty
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

// GetDefaultBotPromptTemplate returns the default template
func GetDefaultBotPromptTemplate() string {
	return DefaultBotPromptTemplate
}
