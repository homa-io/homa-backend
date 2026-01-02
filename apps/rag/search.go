package rag

import (
	"fmt"

	"github.com/iesreza/homa-backend/apps/ai"
)

// SearchResult represents a search result with article info
type ArticleSearchResult struct {
	ArticleID    string  `json:"article_id"`
	ArticleTitle string  `json:"article_title"`
	ChunkContent string  `json:"chunk_content"`
	ChunkIndex   int     `json:"chunk_index"`
	Score        float32 `json:"score"`
}

// Search searches the knowledge base for relevant content
func Search(query string, limit int, scoreThreshold float32) ([]ArticleSearchResult, error) {
	cfg := GetConfig()
	if !cfg.Enabled {
		return nil, fmt.Errorf("RAG is disabled")
	}

	// Get AI client for embeddings
	client := ai.GetClient()
	if client == nil {
		return nil, fmt.Errorf("AI client not initialized")
	}

	// Generate embedding for query
	embeddingResp, err := client.GetEmbedding([]string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to get query embedding: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}

	queryVector := embeddingResp.Data[0].Embedding

	// Search in Qdrant
	qdrantClient := GetQdrantClient()
	if qdrantClient == nil {
		return nil, fmt.Errorf("Qdrant client not initialized")
	}

	collectionName := GetCollectionName()
	results, err := qdrantClient.Search(collectionName, queryVector, limit, scoreThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert to ArticleSearchResult
	var articleResults []ArticleSearchResult
	for _, r := range results {
		articleResults = append(articleResults, ArticleSearchResult{
			ArticleID:    getString(r.Payload, "article_id"),
			ArticleTitle: getString(r.Payload, "article_title"),
			ChunkContent: getString(r.Payload, "chunk_content"),
			ChunkIndex:   getInt(r.Payload, "chunk_index"),
			Score:        r.Score,
		})
	}

	return articleResults, nil
}

// SearchWithContext searches and returns context for AI responses
// Uses configurable threshold to support multilingual queries
func SearchWithContext(query string, limit int) (string, error) {
	cfg := GetConfig()
	results, err := Search(query, limit, cfg.ScoreThreshold)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", nil
	}

	// Build context string from results
	context := "Relevant knowledge base information:\n\n"
	for i, r := range results {
		context += fmt.Sprintf("--- Source %d: %s ---\n%s\n\n", i+1, r.ArticleTitle, r.ChunkContent)
	}

	return context, nil
}

// Helper functions
func getString(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(payload map[string]interface{}, key string) int {
	if val, ok := payload[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}
