package rag

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/ai"
	"github.com/iesreza/homa-backend/apps/models"
	natsconn "github.com/iesreza/homa-backend/apps/nats"
	"github.com/nats-io/nats.go"
)

// App represents the RAG application
type App struct{}

// Register initializes the RAG app
func (a App) Register() error {
	cfg := GetConfig()
	if !cfg.Enabled {
		log.Info("RAG is disabled, skipping initialization")
		return nil
	}

	// Initialize Qdrant client
	if err := InitQdrant(); err != nil {
		log.Error("Failed to initialize Qdrant client: %v", err)
		return err
	}

	// Check Qdrant health
	client := GetQdrantClient()
	if client != nil {
		healthy, err := client.Health()
		if err != nil || !healthy {
			log.Warning("Qdrant is not healthy: %v", err)
		} else {
			log.Info("Qdrant connection established successfully")

			// Ensure collection exists
			collectionName := GetCollectionName()
			info, err := client.GetCollectionInfo(collectionName)
			if err != nil {
				log.Warning("Failed to get collection info: %v", err)
			} else if !info.Exists {
				log.Info("Creating Qdrant collection: %s with vector size: %d", collectionName, cfg.VectorSize)
				if err := client.CreateCollection(collectionName, cfg.VectorSize); err != nil {
					log.Error("Failed to create collection: %v", err)
				}
			} else {
				log.Info("Qdrant collection exists: %s with %d points", collectionName, info.PointsCount)
			}
		}
	}

	// Register the indexer with the models package
	// This connects the GORM hooks to the RAG indexer
	models.SetKnowledgeBaseIndexer(GetIndexer())

	log.Info("RAG app initialized successfully with embedding model: %s", cfg.EmbeddingModel)
	return nil
}

// Router sets up the RAG API routes
func (a App) Router() error {
	controller := Controller{}

	// RAG API endpoints (require authentication via middleware applied in main)
	// Settings
	evo.Get("/api/rag/settings", controller.GetSettingsHandler)
	evo.Post("/api/rag/settings", controller.UpdateSettingsHandler)

	// Health and stats
	evo.Get("/api/rag/health", controller.GetHealthHandler)
	evo.Get("/api/rag/stats", controller.GetStatsHandler)

	// Collection management
	evo.Post("/api/rag/drop-collection", controller.DropCollectionHandler)
	evo.Post("/api/rag/create-collection", controller.CreateCollectionHandler)
	evo.Post("/api/rag/reindex", controller.ReindexHandler)
	evo.Get("/api/rag/reindex-status", controller.GetReindexStatusHandler)

	// Search
	evo.Post("/api/rag/search", controller.SearchHandler)

	// Debug endpoints
	evo.Get("/api/rag/indexed-articles", controller.GetIndexedArticlesHandler)

	return nil
}

// WhenReady is called when the app is ready
func (a App) WhenReady() error {
	// Register knowledge base search function for AI agents
	// This avoids circular import between rag and ai packages
	ai.SearchKnowledgeBase = SearchWithContext
	log.Info("Knowledge base search registered for AI agents")

	// Subscribe to settings reload events
	if natsconn.IsConnected() {
		_, err := natsconn.Subscribe("settings.rag.reload", func(msg *nats.Msg) {
			log.Info("Received RAG settings reload message")
			ReloadConfig()
		})
		if err != nil {
			log.Warning("Failed to subscribe to settings.rag.reload: %v", err)
		} else {
			log.Info("Subscribed to settings.rag.reload for realtime settings updates")
		}
	}
	return nil
}

// Name returns the app name
func (a App) Name() string {
	return "rag"
}
