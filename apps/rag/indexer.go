package rag

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/ai"
	"github.com/iesreza/homa-backend/apps/models"
)

// Reindex job tracking
var (
	reindexMu     sync.Mutex
	reindexStatus = &ReindexStatus{}
)

// ReindexStatus tracks the current reindex job status
type ReindexStatus struct {
	Running       bool   `json:"running"`
	TotalArticles int    `json:"total_articles"`
	Processed     int    `json:"processed"`
	Successful    int    `json:"successful"`
	Failed        int    `json:"failed"`
	Message       string `json:"message"`
}

// GetReindexStatus returns the current reindex status
func GetReindexStatus() ReindexStatus {
	reindexMu.Lock()
	defer reindexMu.Unlock()
	return *reindexStatus
}

// float32ToBytes converts a float32 slice to bytes
func float32ToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := math.Float32bits(f)
		binary.LittleEndian.PutUint32(buf[i*4:], bits)
	}
	return buf
}

// ArticleIndexer implements the KnowledgeBaseIndexer interface
type ArticleIndexer struct{}

// IndexArticle indexes an article in the vector database
func (i *ArticleIndexer) IndexArticle(articleID uuid.UUID) error {
	cfg := GetConfig()
	if !cfg.Enabled {
		log.Debug("RAG is disabled, skipping article indexing")
		return nil
	}

	// Fetch the article
	var article models.KnowledgeBaseArticle
	if err := db.First(&article, "id = ?", articleID).Error; err != nil {
		return fmt.Errorf("failed to fetch article: %w", err)
	}

	// Only index published articles
	if article.Status != "published" {
		log.Debug("Article %s is not published, skipping indexing", articleID)
		return nil
	}

	log.Info("Indexing article: %s (%s)", article.Title, articleID)

	// Delete existing chunks for this article (both DB and Qdrant)
	if err := i.DeleteArticleIndex(articleID); err != nil {
		log.Warning("Failed to delete existing index for article %s: %v", articleID, err)
	}

	// Combine title and content for chunking
	fullContent := article.Title + "\n\n" + article.Content

	// Chunk the content
	chunks := ChunkText(fullContent, cfg.ChunkSize, cfg.ChunkOverlap, cfg.MinChunkSize)
	if len(chunks) == 0 {
		log.Warning("No chunks generated for article %s", articleID)
		return nil
	}

	log.Info("Generated %d chunks for article %s", len(chunks), articleID)

	// Get embeddings for all chunks
	chunkTexts := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkTexts[i] = chunk.Content
	}

	client := ai.GetClient()
	if client == nil {
		return fmt.Errorf("AI client not initialized")
	}

	embeddingResp, err := client.GetEmbedding(chunkTexts)
	if err != nil {
		return fmt.Errorf("failed to get embeddings: %w", err)
	}

	if len(embeddingResp.Data) != len(chunks) {
		return fmt.Errorf("embedding count mismatch: got %d, expected %d", len(embeddingResp.Data), len(chunks))
	}

	// Prepare Qdrant points and DB chunks
	qdrantClient := GetQdrantClient()
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	collectionName := GetCollectionName()
	var qdrantPoints []QdrantPoint
	var dbChunks []models.KnowledgeBaseChunk

	for idx, chunk := range chunks {
		chunkID := uuid.New()
		embedding := embeddingResp.Data[idx].Embedding

		// Prepare Qdrant point
		qdrantPoints = append(qdrantPoints, QdrantPoint{
			ID:     chunkID.String(),
			Vector: embedding,
			Payload: map[string]interface{}{
				"article_id":    articleID.String(),
				"article_title": article.Title,
				"chunk_index":   idx,
				"chunk_content": chunk.Content,
			},
		})

		// Prepare DB chunk with embedding as binary
		dbChunks = append(dbChunks, models.KnowledgeBaseChunk{
			ID:         chunkID,
			ArticleID:  articleID,
			Content:    chunk.Content,
			ChunkIndex: idx,
			TokenCount: chunk.TokenCount,
			Embedding:  float32ToBytes(embedding),
		})
	}

	// Store chunks in database
	if err := db.CreateInBatches(&dbChunks, 100).Error; err != nil {
		return fmt.Errorf("failed to store chunks in database: %w", err)
	}

	// Store vectors in Qdrant
	if err := qdrantClient.UpsertPoints(collectionName, qdrantPoints); err != nil {
		// Rollback DB changes
		db.Where("article_id = ?", articleID).Delete(&models.KnowledgeBaseChunk{})
		return fmt.Errorf("failed to store vectors in Qdrant: %w", err)
	}

	log.Info("Successfully indexed article %s with %d chunks", articleID, len(chunks))
	return nil
}

// DeleteArticleIndex removes an article from the vector database
func (i *ArticleIndexer) DeleteArticleIndex(articleID uuid.UUID) error {
	log.Info("Deleting index for article: %s", articleID)

	// Delete from Qdrant
	qdrantClient := GetQdrantClient()
	if qdrantClient != nil {
		collectionName := GetCollectionName()
		if err := qdrantClient.DeletePoints(collectionName, articleID.String()); err != nil {
			log.Warning("Failed to delete vectors from Qdrant: %v", err)
		}
	}

	// Delete chunks from database
	if err := db.Where("article_id = ?", articleID).Delete(&models.KnowledgeBaseChunk{}).Error; err != nil {
		return fmt.Errorf("failed to delete chunks from database: %w", err)
	}

	return nil
}

// StartReindexAllArticles starts a background reindex job
func StartReindexAllArticles() error {
	reindexMu.Lock()
	if reindexStatus.Running {
		reindexMu.Unlock()
		return fmt.Errorf("reindex already in progress")
	}

	cfg := GetConfig()
	if !cfg.Enabled {
		reindexMu.Unlock()
		return fmt.Errorf("RAG is disabled")
	}

	// Get all published articles
	var articles []models.KnowledgeBaseArticle
	if err := db.Where("status = ?", "published").Find(&articles).Error; err != nil {
		reindexMu.Unlock()
		return fmt.Errorf("failed to fetch articles: %w", err)
	}

	// Update status
	reindexStatus.Running = true
	reindexStatus.TotalArticles = len(articles)
	reindexStatus.Processed = 0
	reindexStatus.Successful = 0
	reindexStatus.Failed = 0
	reindexStatus.Message = "Starting reindex..."
	reindexMu.Unlock()

	// Run in background goroutine
	go func() {
		log.Info("Starting background reindex of %d published articles", len(articles))

		indexer := &ArticleIndexer{}

		for _, article := range articles {
			if err := indexer.IndexArticle(article.ID); err != nil {
				log.Error("Failed to index article %s: %v", article.ID, err)
				reindexMu.Lock()
				reindexStatus.Failed++
				reindexMu.Unlock()
			} else {
				reindexMu.Lock()
				reindexStatus.Successful++
				reindexMu.Unlock()
			}

			reindexMu.Lock()
			reindexStatus.Processed++
			reindexStatus.Message = fmt.Sprintf("Processing article %d of %d", reindexStatus.Processed, reindexStatus.TotalArticles)
			reindexMu.Unlock()
		}

		reindexMu.Lock()
		reindexStatus.Running = false
		reindexStatus.Message = fmt.Sprintf("Completed: %d successful, %d failed", reindexStatus.Successful, reindexStatus.Failed)
		reindexMu.Unlock()

		log.Info("Background reindex complete: %d/%d articles indexed", reindexStatus.Successful, len(articles))
	}()

	return nil
}

// ReindexAllArticles reindexes all published articles (synchronous, for backwards compatibility)
func ReindexAllArticles() (int, error) {
	cfg := GetConfig()
	if !cfg.Enabled {
		return 0, fmt.Errorf("RAG is disabled")
	}

	// Get all published articles
	var articles []models.KnowledgeBaseArticle
	if err := db.Where("status = ?", "published").Find(&articles).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch articles: %w", err)
	}

	log.Info("Reindexing %d published articles", len(articles))

	indexer := &ArticleIndexer{}
	successCount := 0

	for _, article := range articles {
		if err := indexer.IndexArticle(article.ID); err != nil {
			log.Error("Failed to index article %s: %v", article.ID, err)
		} else {
			successCount++
		}
	}

	log.Info("Reindexing complete: %d/%d articles indexed", successCount, len(articles))
	return successCount, nil
}

// GetIndexedArticleCount returns the count of indexed articles
func GetIndexedArticleCount() (int, error) {
	var count int64
	if err := db.Model(&models.KnowledgeBaseChunk{}).
		Distinct("article_id").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetIndexer returns the article indexer instance
func GetIndexer() *ArticleIndexer {
	return &ArticleIndexer{}
}

// IndexedArticleInfo contains info about an indexed article
type IndexedArticleInfo struct {
	ArticleID   string `json:"article_id"`
	Title       string `json:"title"`
	ChunkCount  int    `json:"chunk_count"`
	TotalTokens int    `json:"total_tokens"`
}

// GetIndexedArticles returns a list of all indexed articles with their chunk info
func GetIndexedArticles() ([]IndexedArticleInfo, error) {
	type ChunkStats struct {
		ArticleID   uuid.UUID `gorm:"column:article_id"`
		ChunkCount  int       `gorm:"column:chunk_count"`
		TotalTokens int       `gorm:"column:total_tokens"`
	}

	var stats []ChunkStats
	if err := db.Model(&models.KnowledgeBaseChunk{}).
		Select("article_id, COUNT(*) as chunk_count, SUM(token_count) as total_tokens").
		Group("article_id").
		Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get chunk stats: %w", err)
	}

	// Get article titles
	articleIDs := make([]uuid.UUID, len(stats))
	for i, s := range stats {
		articleIDs[i] = s.ArticleID
	}

	var articles []models.KnowledgeBaseArticle
	if len(articleIDs) > 0 {
		if err := db.Where("id IN ?", articleIDs).Find(&articles).Error; err != nil {
			return nil, fmt.Errorf("failed to get articles: %w", err)
		}
	}

	// Create title map
	titleMap := make(map[string]string)
	for _, a := range articles {
		titleMap[a.ID.String()] = a.Title
	}

	// Build result
	result := make([]IndexedArticleInfo, len(stats))
	for i, s := range stats {
		result[i] = IndexedArticleInfo{
			ArticleID:   s.ArticleID.String(),
			Title:       titleMap[s.ArticleID.String()],
			ChunkCount:  s.ChunkCount,
			TotalTokens: s.TotalTokens,
		}
	}

	return result, nil
}
