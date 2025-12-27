package ai

import (
	"fmt"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/models"
)

// ArticleIndexer implements the KnowledgeBaseIndexer interface
type ArticleIndexer struct{}

// IndexArticle indexes an article in Qdrant
func (i *ArticleIndexer) IndexArticle(articleID uuid.UUID) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("OpenAI client not initialized")
	}

	qdrant := GetQdrantClient()
	if qdrant == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	// Get the article
	var article models.KnowledgeBaseArticle
	if err := db.Preload("Category").Where("id = ?", articleID).First(&article).Error; err != nil {
		return fmt.Errorf("article not found: %w", err)
	}

	// Only index published articles
	if article.Status != "published" {
		log.Debug("Skipping non-published article: %s", articleID)
		return nil
	}

	// Delete existing vectors for this article
	if err := qdrant.DeleteByArticleID(articleID); err != nil {
		log.Warning("Failed to delete existing vectors: %v", err)
	}

	// Clean and chunk the content
	tokenizer := NewTokenizer()
	cleanContent := tokenizer.CleanText(article.Content)

	// Also include title and excerpt for better search
	fullContent := fmt.Sprintf("%s\n\n%s\n\n%s", article.Title, article.Excerpt, cleanContent)
	chunks := tokenizer.ChunkText(fullContent)

	if len(chunks) == 0 {
		log.Debug("No chunks generated for article: %s", articleID)
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

	// Prepare Qdrant points
	var points []QdrantPoint
	for i, chunk := range chunks {
		if i >= len(embResp.Data) {
			break
		}

		// Build payload with article metadata
		payload := map[string]interface{}{
			"article_id":    articleID.String(),
			"article_title": article.Title,
			"article_slug":  article.Slug,
			"chunk_index":   chunk.Index,
			"chunk_content": chunk.Content,
			"token_count":   chunk.TokenCount,
			"status":        article.Status,
		}

		if article.CategoryID != nil {
			payload["category_id"] = article.CategoryID.String()
		}
		if article.Category != nil {
			payload["category_name"] = article.Category.Name
		}
		if article.Excerpt != "" {
			payload["excerpt"] = article.Excerpt
		}

		pointID := fmt.Sprintf("%s_%d", articleID.String(), chunk.Index)
		points = append(points, QdrantPoint{
			ID:      pointID,
			Vector:  embResp.Data[i].Embedding,
			Payload: payload,
		})
	}

	// Upsert to Qdrant
	if err := qdrant.Upsert(points); err != nil {
		return fmt.Errorf("failed to upsert to Qdrant: %w", err)
	}

	log.Info("Indexed article %s with %d chunks", articleID, len(points))
	return nil
}

// DeleteArticleIndex removes an article's vectors from Qdrant
func (i *ArticleIndexer) DeleteArticleIndex(articleID uuid.UUID) error {
	qdrant := GetQdrantClient()
	if qdrant == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	if err := qdrant.DeleteByArticleID(articleID); err != nil {
		return fmt.Errorf("failed to delete from Qdrant: %w", err)
	}

	log.Info("Deleted index for article: %s", articleID)
	return nil
}

// IndexAllArticles indexes all published articles
func IndexAllArticles() error {
	var articles []models.KnowledgeBaseArticle
	if err := db.Where("status = ?", "published").Find(&articles).Error; err != nil {
		return fmt.Errorf("failed to fetch articles: %w", err)
	}

	indexer := &ArticleIndexer{}
	successCount := 0
	errorCount := 0

	for _, article := range articles {
		if err := indexer.IndexArticle(article.ID); err != nil {
			log.Error("Failed to index article %s: %v", article.ID, err)
			errorCount++
		} else {
			successCount++
		}
	}

	log.Info("Indexed %d articles successfully, %d errors", successCount, errorCount)
	return nil
}

// GetIndexStats returns statistics about the Qdrant index
func GetIndexStats() (map[string]interface{}, error) {
	qdrant := GetQdrantClient()
	if qdrant == nil {
		return nil, fmt.Errorf("Qdrant client not initialized")
	}

	return qdrant.GetCollectionInfo()
}
