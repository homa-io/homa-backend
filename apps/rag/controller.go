package rag

import (
	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/lib/response"
)

// Controller handles RAG-related HTTP requests
type Controller struct{}

// GetSettingsHandler handles GET /api/rag/settings
// @Summary Get RAG settings
// @Description Returns the current RAG configuration settings
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {object} RAGConfig
// @Router /api/rag/settings [get]
func (c Controller) GetSettingsHandler(req *evo.Request) interface{} {
	cfg := GetConfig()
	return response.OK(cfg)
}

// UpdateSettingsRequest represents the update settings request body
type UpdateSettingsRequest struct {
	Enabled        *bool   `json:"enabled"`
	EmbeddingModel *string `json:"embedding_model"`
	VectorSize     *int    `json:"vector_size"`
	ChunkSize      *int    `json:"chunk_size"`
	ChunkOverlap   *int    `json:"chunk_overlap"`
	MinChunkSize   *int    `json:"min_chunk_size"`
}

// UpdateSettingsHandler handles POST /api/rag/settings
// @Summary Update RAG settings
// @Description Updates the RAG configuration settings
// @Tags RAG
// @Accept json
// @Produce json
// @Param body body UpdateSettingsRequest true "Settings update request"
// @Success 200 {object} RAGConfig
// @Router /api/rag/settings [post]
func (c Controller) UpdateSettingsHandler(req *evo.Request) interface{} {
	var updateReq UpdateSettingsRequest
	if err := req.BodyParser(&updateReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	// Get current config
	cfg := GetConfig()

	// Update only provided fields
	if updateReq.Enabled != nil {
		cfg.Enabled = *updateReq.Enabled
	}
	if updateReq.EmbeddingModel != nil {
		cfg.EmbeddingModel = *updateReq.EmbeddingModel
		// Auto-update vector size based on model
		cfg.VectorSize = GetVectorSizeForModel(cfg.EmbeddingModel)
	}
	if updateReq.VectorSize != nil {
		cfg.VectorSize = *updateReq.VectorSize
	}
	if updateReq.ChunkSize != nil {
		cfg.ChunkSize = *updateReq.ChunkSize
	}
	if updateReq.ChunkOverlap != nil {
		cfg.ChunkOverlap = *updateReq.ChunkOverlap
	}
	if updateReq.MinChunkSize != nil {
		cfg.MinChunkSize = *updateReq.MinChunkSize
	}

	// Validate
	if cfg.ChunkSize < 100 || cfg.ChunkSize > 4000 {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid chunk size", 400, "chunk_size must be between 100 and 4000"))
	}
	if cfg.ChunkOverlap < 0 || cfg.ChunkOverlap >= cfg.ChunkSize {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid chunk overlap", 400, "chunk_overlap must be >= 0 and < chunk_size"))
	}
	if cfg.MinChunkSize < 10 || cfg.MinChunkSize > cfg.ChunkSize {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid min chunk size", 400, "min_chunk_size must be between 10 and chunk_size"))
	}

	// Save
	if err := UpdateConfig(cfg); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to update settings", 500, err.Error()))
	}

	return response.OK(cfg)
}

// HealthResponse represents the Qdrant health check response
type HealthResponse struct {
	Healthy    bool            `json:"healthy"`
	Message    string          `json:"message"`
	Collection *CollectionInfo `json:"collection,omitempty"`
}

// GetHealthHandler handles GET /api/rag/health
// @Summary Get RAG/Qdrant health status
// @Description Returns the health status of the RAG system and Qdrant
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /api/rag/health [get]
func (c Controller) GetHealthHandler(req *evo.Request) interface{} {
	cfg := GetConfig()
	if !cfg.Enabled {
		return response.OK(HealthResponse{
			Healthy: false,
			Message: "RAG is disabled",
		})
	}

	client := GetQdrantClient()
	if client == nil {
		return response.OK(HealthResponse{
			Healthy: false,
			Message: "Qdrant client not initialized",
		})
	}

	// Check Qdrant health
	healthy, err := client.Health()
	if err != nil || !healthy {
		return response.OK(HealthResponse{
			Healthy: false,
			Message: "Qdrant is not reachable: " + err.Error(),
		})
	}

	// Get collection info
	collectionName := GetCollectionName()
	collectionInfo, err := client.GetCollectionInfo(collectionName)
	if err != nil {
		return response.OK(HealthResponse{
			Healthy: false,
			Message: "Failed to get collection info: " + err.Error(),
		})
	}

	if !collectionInfo.Exists {
		return response.OK(HealthResponse{
			Healthy:    true,
			Message:    "Qdrant is healthy but collection does not exist",
			Collection: collectionInfo,
		})
	}

	return response.OK(HealthResponse{
		Healthy:    true,
		Message:    "RAG system is healthy",
		Collection: collectionInfo,
	})
}

// ReindexResponse represents the reindex response
type ReindexResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ArticleCount int    `json:"article_count"`
}

// ReindexHandler handles POST /api/rag/reindex
// @Summary Reindex all articles (background)
// @Description Starts a background job to reindex all published articles in the vector database
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {object} ReindexResponse
// @Router /api/rag/reindex [post]
func (c Controller) ReindexHandler(req *evo.Request) interface{} {
	cfg := GetConfig()
	if !cfg.Enabled {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "RAG is disabled", 400, "enable RAG before reindexing"))
	}

	err := StartReindexAllArticles()
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Reindex failed to start", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"success": true,
		"message": "Reindex job started in background",
	})
}

// GetReindexStatusHandler handles GET /api/rag/reindex-status
// @Summary Get reindex job status
// @Description Returns the current status of the background reindex job
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {object} ReindexStatus
// @Router /api/rag/reindex-status [get]
func (c Controller) GetReindexStatusHandler(req *evo.Request) interface{} {
	status := GetReindexStatus()
	return response.OK(status)
}

// SearchRequest represents the search request body
type SearchRequest struct {
	Query          string  `json:"query"`
	Limit          int     `json:"limit"`
	ScoreThreshold float32 `json:"score_threshold"`
}

// SearchHandler handles POST /api/rag/search
// @Summary Search the knowledge base
// @Description Performs a vector search on the knowledge base
// @Tags RAG
// @Accept json
// @Produce json
// @Param body body SearchRequest true "Search request"
// @Success 200 {array} ArticleSearchResult
// @Router /api/rag/search [post]
func (c Controller) SearchHandler(req *evo.Request) interface{} {
	var searchReq SearchRequest
	if err := req.BodyParser(&searchReq); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Invalid request body", 400, err.Error()))
	}

	if searchReq.Query == "" {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInvalidInput, "Query is required", 400, "query field cannot be empty"))
	}

	// Set defaults from config
	cfg := GetConfig()
	if searchReq.Limit <= 0 {
		searchReq.Limit = cfg.TopK
	}
	if searchReq.Limit > 20 {
		searchReq.Limit = 20
	}
	if searchReq.ScoreThreshold <= 0 {
		searchReq.ScoreThreshold = cfg.ScoreThreshold
	}

	results, err := Search(searchReq.Query, searchReq.Limit, searchReq.ScoreThreshold)
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Search failed", 500, err.Error()))
	}

	return response.OK(results)
}

// DropCollectionHandler handles POST /api/rag/drop-collection
// @Summary Drop the Qdrant collection
// @Description Drops the Qdrant collection (use with caution)
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/rag/drop-collection [post]
func (c Controller) DropCollectionHandler(req *evo.Request) interface{} {
	client := GetQdrantClient()
	if client == nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Qdrant client not initialized", 500, ""))
	}

	collectionName := GetCollectionName()
	if err := client.DeleteCollection(collectionName); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to drop collection", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"success": true,
		"message": "Collection dropped successfully",
	})
}

// CreateCollectionHandler handles POST /api/rag/create-collection
// @Summary Create the Qdrant collection
// @Description Creates the Qdrant collection with the configured vector size
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/rag/create-collection [post]
func (c Controller) CreateCollectionHandler(req *evo.Request) interface{} {
	cfg := GetConfig()
	client := GetQdrantClient()
	if client == nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Qdrant client not initialized", 500, ""))
	}

	collectionName := GetCollectionName()
	if err := client.CreateCollection(collectionName, cfg.VectorSize); err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to create collection", 500, err.Error()))
	}

	return response.OK(map[string]interface{}{
		"success": true,
		"message": "Collection created successfully",
	})
}

// GetStatsHandler handles GET /api/rag/stats
// @Summary Get RAG statistics
// @Description Returns statistics about the indexed articles
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/rag/stats [get]
func (c Controller) GetStatsHandler(req *evo.Request) interface{} {
	cfg := GetConfig()
	if !cfg.Enabled {
		return response.OK(map[string]interface{}{
			"enabled":               false,
			"indexed_article_count": 0,
			"vector_count":          0,
		})
	}

	// Get indexed article count from DB
	articleCount, err := GetIndexedArticleCount()
	if err != nil {
		articleCount = 0
	}

	// Get vector count from Qdrant
	vectorCount := 0
	client := GetQdrantClient()
	if client != nil {
		collectionName := GetCollectionName()
		if count, err := client.CountPoints(collectionName); err == nil {
			vectorCount = count
		}
	}

	return response.OK(map[string]interface{}{
		"enabled":               cfg.Enabled,
		"indexed_article_count": articleCount,
		"vector_count":          vectorCount,
		"embedding_model":       cfg.EmbeddingModel,
		"vector_size":           cfg.VectorSize,
		"chunk_size":            cfg.ChunkSize,
		"chunk_overlap":         cfg.ChunkOverlap,
	})
}

// IndexedArticle represents an indexed article with its chunk info
type IndexedArticle struct {
	ArticleID   string `json:"article_id"`
	Title       string `json:"title"`
	ChunkCount  int    `json:"chunk_count"`
	TotalTokens int    `json:"total_tokens"`
}

// GetIndexedArticlesHandler handles GET /api/rag/indexed-articles
// @Summary Get list of indexed articles
// @Description Returns a list of all indexed articles with their chunk counts
// @Tags RAG
// @Accept json
// @Produce json
// @Success 200 {array} IndexedArticle
// @Router /api/rag/indexed-articles [get]
func (c Controller) GetIndexedArticlesHandler(req *evo.Request) interface{} {
	articles, err := GetIndexedArticles()
	if err != nil {
		return response.Error(response.NewErrorWithDetails(response.ErrorCodeInternalError, "Failed to get indexed articles", 500, err.Error()))
	}
	return response.OK(articles)
}
