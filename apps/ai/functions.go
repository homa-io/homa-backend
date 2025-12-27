package ai

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/models"
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
}

// SummarizeResponse represents a summarization response
type SummarizeResponse struct {
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	MessageCount int      `json:"message_count"`
}

// GenerateResponseRequest represents a response generation request
type GenerateResponseRequest struct {
	Messages     []MessageInput `json:"messages"`
	Context      string         `json:"context,omitempty"`       // Optional additional context
	MaxResults   int            `json:"max_results,omitempty"`   // Max knowledge base results to use
	CategoryID   *string        `json:"category_id,omitempty"`   // Optional category filter for KB search
}

// GenerateResponseResponse represents a response generation response
type GenerateResponseResponse struct {
	Response       string              `json:"response"`
	Sources        []KnowledgeSource   `json:"sources,omitempty"`
	Confidence     float64             `json:"confidence"`
}

// KnowledgeSource represents a knowledge base source used for response generation
type KnowledgeSource struct {
	ArticleID    string  `json:"article_id"`
	ArticleTitle string  `json:"article_title"`
	Excerpt      string  `json:"excerpt"`
	Relevance    float64 `json:"relevance"`
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
			Role: "system",
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

	messages := []ChatMessage{
		{
			Role: "system",
			Content: `You are a conversation analyst. Summarize the given customer support conversation. Provide:
1. A concise summary paragraph (2-3 sentences)
2. Key points as a numbered list

Format your response as:
SUMMARY:
[Your summary here]

KEY POINTS:
1. [Point 1]
2. [Point 2]
...`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Summarize this conversation:\n\n%s", conversationText.String()),
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
func parseSummaryResponse(content string) (string, []string) {
	var summary string
	var keyPoints []string

	lines := strings.Split(content, "\n")
	inSummary := false
	inKeyPoints := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToUpper(line), "SUMMARY:") {
			inSummary = true
			inKeyPoints = false
			// Check if summary is on the same line
			rest := strings.TrimPrefix(line, "SUMMARY:")
			rest = strings.TrimPrefix(rest, "Summary:")
			if rest = strings.TrimSpace(rest); rest != "" {
				summary = rest
			}
			continue
		}

		if strings.HasPrefix(strings.ToUpper(line), "KEY POINTS:") || strings.HasPrefix(strings.ToUpper(line), "KEY_POINTS:") {
			inSummary = false
			inKeyPoints = true
			continue
		}

		if inSummary && line != "" {
			if summary != "" {
				summary += " " + line
			} else {
				summary = line
			}
		}

		if inKeyPoints && line != "" {
			// Remove numbering and bullets
			point := strings.TrimLeft(line, "0123456789.-) ")
			if point != "" {
				keyPoints = append(keyPoints, point)
			}
		}
	}

	return summary, keyPoints
}

// GenerateResponse generates a response using RAG (Retrieval-Augmented Generation)
func GenerateResponse(req GenerateResponseRequest) (*GenerateResponseResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}

	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	if req.MaxResults == 0 {
		req.MaxResults = 5
	}

	// Extract the last user message as the query
	var query string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			query = req.Messages[i].Content
			break
		}
	}

	if query == "" {
		// Use the last message if no user message found
		query = req.Messages[len(req.Messages)-1].Content
	}

	// Search knowledge base for relevant content
	sources, relevantContent, err := searchKnowledgeBase(query, req.MaxResults, req.CategoryID)
	if err != nil {
		log.Warning("Knowledge base search failed: %v", err)
		// Continue without KB content
	}

	// Build conversation context
	var conversationContext strings.Builder
	for _, msg := range req.Messages {
		role := "Customer"
		if msg.Role == "agent" {
			role = "Agent"
		}
		conversationContext.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	// Build system prompt with knowledge base context
	systemPrompt := `You are a helpful customer support agent. Your goal is to assist customers with their questions and issues.

Guidelines:
- Be professional, friendly, and helpful
- Provide accurate information based on the knowledge base when available
- If you don't know the answer, acknowledge it and offer to escalate or find more information
- Keep responses concise but complete
- Address the customer's specific question or concern`

	if relevantContent != "" {
		systemPrompt += fmt.Sprintf(`

KNOWLEDGE BASE INFORMATION (use this to answer the customer's question):
%s`, relevantContent)
	}

	if req.Context != "" {
		systemPrompt += fmt.Sprintf(`

ADDITIONAL CONTEXT:
%s`, req.Context)
	}

	messages := []ChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Based on the conversation below, generate an appropriate response to the customer's latest message.\n\nConversation:\n%s\n\nGenerate a helpful response:", conversationContext.String()),
		},
	}

	resp, err := client.ChatCompletion(messages, 1000, 0.7)
	if err != nil {
		return nil, fmt.Errorf("response generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	// Calculate confidence based on whether we found relevant KB content
	confidence := 0.5
	if len(sources) > 0 {
		// Average the relevance scores
		var totalRelevance float64
		for _, s := range sources {
			totalRelevance += s.Relevance
		}
		confidence = totalRelevance / float64(len(sources))
	}

	return &GenerateResponseResponse{
		Response:   strings.TrimSpace(resp.Choices[0].Message.Content),
		Sources:    sources,
		Confidence: confidence,
	}, nil
}

// searchKnowledgeBase searches for relevant content in the knowledge base using Qdrant
func searchKnowledgeBase(query string, maxResults int, categoryID *string) ([]KnowledgeSource, string, error) {
	qdrant := GetQdrantClient()
	if qdrant == nil {
		return nil, "", fmt.Errorf("Qdrant client not initialized")
	}

	// Search using Qdrant
	results, err := qdrant.SearchByText(query, maxResults, categoryID)
	if err != nil {
		return nil, "", fmt.Errorf("Qdrant search failed: %w", err)
	}

	if len(results) == 0 {
		return nil, "", nil
	}

	// Build sources and content from Qdrant results
	var sources []KnowledgeSource
	var contentBuilder strings.Builder
	seenArticles := make(map[string]bool)

	for _, result := range results {
		articleID, _ := result.Payload["article_id"].(string)
		articleTitle, _ := result.Payload["article_title"].(string)
		chunkContent, _ := result.Payload["chunk_content"].(string)
		excerpt, _ := result.Payload["excerpt"].(string)

		if articleTitle == "" {
			articleTitle = "Unknown Article"
		}

		if !seenArticles[articleID] {
			seenArticles[articleID] = true

			sourceExcerpt := excerpt
			if sourceExcerpt == "" {
				sourceExcerpt = truncateText(chunkContent, 200)
			}

			sources = append(sources, KnowledgeSource{
				ArticleID:    articleID,
				ArticleTitle: articleTitle,
				Excerpt:      sourceExcerpt,
				Relevance:    float64(result.Score),
			})
		}

		contentBuilder.WriteString(fmt.Sprintf("--- From: %s ---\n%s\n\n",
			articleTitle,
			chunkContent))
	}

	return sources, contentBuilder.String(), nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// bytesToFloat32 converts a byte slice to a float32 slice
func bytesToFloat32(data []byte) []float32 {
	if len(data) == 0 || len(data)%4 != 0 {
		return nil
	}

	result := make([]float32, len(data)/4)
	for i := 0; i < len(result); i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		result[i] = math.Float32frombits(bits)
	}
	return result
}

// float32ToBytes converts a float32 slice to a byte slice
func Float32ToBytes(data []float32) []byte {
	result := make([]byte, len(data)*4)
	for i, v := range data {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(result[i*4:], bits)
	}
	return result
}

// truncateText truncates text to a maximum length
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// IndexArticle creates embeddings for an article and stores them as chunks
func IndexArticle(articleID uuid.UUID) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("OpenAI client not initialized")
	}

	// Get the article
	var article models.KnowledgeBaseArticle
	if err := db.Where("id = ?", articleID).First(&article).Error; err != nil {
		return fmt.Errorf("article not found: %w", err)
	}

	// Delete existing chunks
	if err := db.Where("article_id = ?", articleID).Delete(&models.KnowledgeBaseChunk{}).Error; err != nil {
		return fmt.Errorf("failed to delete existing chunks: %w", err)
	}

	// Clean and chunk the content
	tokenizer := NewTokenizer()
	cleanContent := tokenizer.CleanText(article.Content)
	chunks := tokenizer.ChunkText(cleanContent)

	if len(chunks) == 0 {
		return nil
	}

	// Get embeddings for all chunks
	var texts []string
	for _, chunk := range chunks {
		texts = append(texts, chunk.Content)
	}

	embResp, err := client.GetEmbedding(texts)
	if err != nil {
		return fmt.Errorf("failed to get embeddings: %w", err)
	}

	// Create chunk records
	for i, chunk := range chunks {
		var embedding []byte
		if i < len(embResp.Data) {
			embedding = Float32ToBytes(embResp.Data[i].Embedding)
		}

		dbChunk := models.KnowledgeBaseChunk{
			ID:         uuid.New(),
			ArticleID:  articleID,
			Content:    chunk.Content,
			ChunkIndex: chunk.Index,
			TokenCount: chunk.TokenCount,
			Embedding:  embedding,
		}

		if err := db.Create(&dbChunk).Error; err != nil {
			log.Error("Failed to create chunk: %v", err)
		}
	}

	return nil
}

// ReindexAllArticles reindexes all published articles
func ReindexAllArticles() error {
	var articles []models.KnowledgeBaseArticle
	if err := db.Where("status = ?", "published").Find(&articles).Error; err != nil {
		return fmt.Errorf("failed to fetch articles: %w", err)
	}

	for _, article := range articles {
		if err := IndexArticle(article.ID); err != nil {
			log.Error("Failed to index article %s: %v", article.ID, err)
		}
	}

	return nil
}
