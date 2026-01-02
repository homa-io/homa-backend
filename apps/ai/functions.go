package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// MessageInput represents a message for AI processing
type MessageInput struct {
	Role    string `json:"role"`    // "user" or "agent"
	Content string `json:"content"`
	Author  string `json:"author,omitempty"` // Optional author name
}

// TranslateRequest represents a translation request
type TranslateRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"` // Target language code or name (e.g., "es", "Spanish", "fr", "French")
}

// TranslateResponse represents a translation response
type TranslateResponse struct {
	OriginalText   string `json:"original_text"`
	TranslatedText string `json:"translated_text"`
	TargetLanguage string `json:"target_language"`
}

// ReviseRequest represents a revision request
type ReviseRequest struct {
	Text   string `json:"text"`
	Format string `json:"format"` // e.g., "formal", "casual", "professional", "friendly", "concise", "detailed"
}

// ReviseResponse represents a revision response
type ReviseResponse struct {
	OriginalText string `json:"original_text"`
	RevisedText  string `json:"revised_text"`
	Format       string `json:"format"`
}

// SummarizeRequest represents a summarization request
type SummarizeRequest struct {
	Messages []MessageInput `json:"messages"`
	Language string         `json:"language,omitempty"` // Target language for summary (default: "en")
}

// SummarizeResponse represents a summarization response
type SummarizeResponse struct {
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	MessageCount int      `json:"message_count"`
}

// GenerateArticleSummaryRequest represents a request to generate an article summary
type GenerateArticleSummaryRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// GenerateArticleSummaryResponse represents the generated summary response
type GenerateArticleSummaryResponse struct {
	Summary string `json:"summary"`
}

// SmartReplyRequest represents a smart reply request
type SmartReplyRequest struct {
	AgentMessage    string `json:"agent_message"`
	UserLastMessage string `json:"user_last_message"`
	Tone            string `json:"tone,omitempty"`            // Optional: formal, casual, professional, friendly, empathetic
	TargetLanguage  string `json:"target_language,omitempty"` // Optional: override target language instead of detecting from user message
}

// SmartReplyResponse represents a smart reply response
type SmartReplyResponse struct {
	OriginalText          string   `json:"original_text"`
	ImprovedText          string   `json:"improved_text"`
	DetectedUserLanguage  string   `json:"detected_user_language"`
	DetectedAgentLanguage string   `json:"detected_agent_language"`
	WasTranslated         bool     `json:"was_translated"`
	Improvements          []string `json:"improvements"`
}

// Translate translates text to the specified language
func Translate(req TranslateRequest) (*TranslateResponse, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if req.Language == "" {
		return nil, fmt.Errorf("target language is required")
	}

	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	messages := []ChatMessage{
		{
			Role:    "system",
			Content: `You are a professional translator. Translate the given text to the specified language accurately while preserving the original meaning, tone, and style. Only output the translated text without any explanations or additional comments.`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Translate the following text to %s:\n\n%s", req.Language, req.Text),
		},
	}

	resp, err := client.ChatCompletion(messages, 2000, 0.3)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no translation received")
	}

	return &TranslateResponse{
		OriginalText:   req.Text,
		TranslatedText: strings.TrimSpace(resp.Choices[0].Message.Content),
		TargetLanguage: req.Language,
	}, nil
}

// Revise rewrites text according to the requested format
func Revise(req ReviseRequest) (*ReviseResponse, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if req.Format == "" {
		return nil, fmt.Errorf("format is required")
	}

	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	formatInstructions := getFormatInstructions(req.Format)

	messages := []ChatMessage{
		{
			Role: "system",
			Content: fmt.Sprintf(`You are a professional writing assistant. Rewrite the given text according to the specified style/format. %s

Only output the revised text without any explanations or additional comments.`, formatInstructions),
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Rewrite the following text in a %s style:\n\n%s", req.Format, req.Text),
		},
	}

	resp, err := client.ChatCompletion(messages, 2000, 0.7)
	if err != nil {
		return nil, fmt.Errorf("revision failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no revision received")
	}

	return &ReviseResponse{
		OriginalText: req.Text,
		RevisedText:  strings.TrimSpace(resp.Choices[0].Message.Content),
		Format:       req.Format,
	}, nil
}

// getFormatInstructions returns specific instructions for each format type
func getFormatInstructions(format string) string {
	switch strings.ToLower(format) {
	case "formal":
		return "Use formal language, proper grammar, and professional vocabulary. Avoid contractions and colloquialisms."
	case "casual":
		return "Use a relaxed, conversational tone. Contractions and informal expressions are encouraged."
	case "professional":
		return "Use clear, business-appropriate language. Be direct and concise while maintaining politeness."
	case "friendly":
		return "Use warm, approachable language. Be personable and engaging while remaining helpful."
	case "concise":
		return "Reduce the text to its essential points. Remove redundancy and unnecessary words while preserving meaning."
	case "detailed":
		return "Expand on the content with more explanation and context. Add helpful details and examples where appropriate."
	case "empathetic":
		return "Show understanding and compassion. Acknowledge feelings and provide supportive language."
	case "technical":
		return "Use precise, technical language appropriate for subject matter experts. Include specific terminology."
	default:
		return fmt.Sprintf("Rewrite in a %s style while maintaining the original meaning.", format)
	}
}

// Summarize creates a summary from a list of conversation messages
func Summarize(req SummarizeRequest) (*SummarizeResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}

	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	// Set default language if not provided
	language := req.Language
	if language == "" {
		language = "en"
	}

	// Build language instruction
	languageInstruction := ""
	if language != "en" {
		languageInstruction = fmt.Sprintf("\n\nIMPORTANT: You MUST write the summary in %s language. All bullet points MUST be in %s.", language, language)
	}

	// Format messages for the AI
	var conversationText strings.Builder
	for i, msg := range req.Messages {
		role := "Customer"
		if msg.Role == "agent" {
			role = "Agent"
		}
		if msg.Author != "" {
			role = fmt.Sprintf("%s (%s)", role, msg.Author)
		}
		conversationText.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, role, msg.Content))
	}

	systemPrompt := fmt.Sprintf(`You are a conversation analyst. Extract key points from the given customer support conversation.

Create a bullet point list describing the main events, formatted as:
- User requested X
- Agent responded with Y
- User confirmed/rejected Z

Keep each point concise (one line). Focus on requests, responses, and outcomes.%s

Format your response as:
KEY POINTS:
- [First point]
- [Second point]
- [Third point]
...`, languageInstruction)

	messages := []ChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Extract key points from this conversation:\n\n%s", conversationText.String()),
		},
	}

	resp, err := client.ChatCompletion(messages, 1000, 0.5)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no summary received")
	}

	// Parse the response
	content := resp.Choices[0].Message.Content
	summary, keyPoints := parseSummaryResponse(content)

	return &SummarizeResponse{
		Summary:      summary,
		KeyPoints:    keyPoints,
		MessageCount: len(req.Messages),
	}, nil
}

// parseSummaryResponse parses the structured summary response
// Now only extracts key points and uses them as the summary
func parseSummaryResponse(content string) (string, []string) {
	var keyPoints []string

	lines := strings.Split(content, "\n")
	inKeyPoints := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToUpper(line), "KEY POINTS:") || strings.HasPrefix(strings.ToUpper(line), "KEY_POINTS:") {
			inKeyPoints = true
			continue
		}

		if inKeyPoints && line != "" {
			// Remove numbering and bullets, extract the point text
			point := strings.TrimLeft(line, "0123456789.-•) ")
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		}
	}

	// Join key points with newlines for display as bullet list
	summary := strings.Join(keyPoints, "\n")

	return summary, keyPoints
}

// GenerateArticleSummary generates a summary for an article based on its title and content
func GenerateArticleSummary(req GenerateArticleSummaryRequest) (*GenerateArticleSummaryResponse, error) {
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	// Strip HTML tags from content for cleaner AI processing
	cleanContent := stripHTMLTags(req.Content)

	messages := []ChatMessage{
		{
			Role: "system",
			Content: `You are a professional content writer for a knowledge base. Generate a concise and engaging summary for the given article.

IMPORTANT: You MUST read and analyze the FULL article content provided below, not just the title.

The summary should:
- Be 1-2 sentences (maximum 200 characters)
- Accurately describe what the article content is about
- Capture the main topic, key points, and value proposition from the content
- Be written in a clear, informative style that reflects the article body

Only output the summary text without any explanations, quotes, or additional comments.`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Generate a summary for this knowledge base article.\n\nArticle Title: %s\n\nArticle Content:\n%s", req.Title, cleanContent),
		},
	}

	resp, err := client.ChatCompletion(messages, 300, 0.7)
	if err != nil {
		return nil, fmt.Errorf("summary generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no summary received")
	}

	return &GenerateArticleSummaryResponse{
		Summary: strings.TrimSpace(resp.Choices[0].Message.Content),
	}, nil
}

// SmartReply analyzes agent message, detects languages, translates if needed, and fixes grammar
func SmartReply(req SmartReplyRequest) (*SmartReplyResponse, error) {
	if req.AgentMessage == "" {
		return nil, fmt.Errorf("agent message is required")
	}
	if req.UserLastMessage == "" {
		return nil, fmt.Errorf("user last message is required")
	}

	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	// Build tone instruction if provided
	toneInstruction := ""
	if req.Tone != "" {
		toneInstruction = fmt.Sprintf("\n6. IMPORTANT: Apply a %s tone to the message - this is mandatory", req.Tone)
	}

	// Build language instruction
	languageInstruction := "3. If the agent's reply is in a different language than the user's message, translate it to the user's language"
	targetLangNote := ""
	if req.TargetLanguage != "" {
		languageInstruction = fmt.Sprintf("3. MANDATORY: You MUST translate the agent's reply to %s - the output MUST be in %s language regardless of the original language", req.TargetLanguage, req.TargetLanguage)
		targetLangNote = fmt.Sprintf("\n\nCRITICAL: The improved_text MUST be written in %s language. This is a hard requirement.", req.TargetLanguage)
	}

	systemPrompt := fmt.Sprintf(`You are a smart reply assistant for customer support agents. Your task is to:
1. Detect the language of the user's last message (this is the language the user prefers)
2. Detect the language of the agent's reply
%s
4. Fix any grammatical errors and improve the text quality while preserving the original meaning
5. Make the message sound professional and friendly%s%s

You MUST respond in this exact JSON format:
{
  "detected_user_language": "the language name of user's message (e.g., English, Spanish, Persian, etc.)",
  "detected_agent_language": "the language name of agent's message",
  "was_translated": true/false,
  "improved_text": "the final improved message in the target language",
  "improvements": ["list of improvements made, e.g., 'Fixed grammar', 'Translated from English to Persian', 'Applied formal tone']
}

Only output valid JSON, nothing else.`, languageInstruction, toneInstruction, targetLangNote)

	messages := []ChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role: "user",
			Content: fmt.Sprintf("User's last message:\n\"%s\"\n\nAgent's reply to improve:\n\"%s\"", req.UserLastMessage, req.AgentMessage),
		},
	}

	resp, err := client.ChatCompletion(messages, 2000, 0.3)
	if err != nil {
		return nil, fmt.Errorf("smart reply failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response received")
	}

	// Parse the JSON response
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	// Remove markdown code blocks if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var aiResponse struct {
		DetectedUserLanguage  string   `json:"detected_user_language"`
		DetectedAgentLanguage string   `json:"detected_agent_language"`
		WasTranslated         bool     `json:"was_translated"`
		ImprovedText          string   `json:"improved_text"`
		Improvements          []string `json:"improvements"`
	}

	if err := json.Unmarshal([]byte(content), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w (content: %s)", err, content)
	}

	return &SmartReplyResponse{
		OriginalText:          req.AgentMessage,
		ImprovedText:          aiResponse.ImprovedText,
		DetectedUserLanguage:  aiResponse.DetectedUserLanguage,
		DetectedAgentLanguage: aiResponse.DetectedAgentLanguage,
		WasTranslated:         aiResponse.WasTranslated,
		Improvements:          aiResponse.Improvements,
	}, nil
}

// stripHTMLTags removes HTML tags from a string and returns clean text
func stripHTMLTags(html string) string {
	// Remove HTML tags using regex
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Clean up multiple spaces and newlines
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
