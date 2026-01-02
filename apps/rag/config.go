package rag

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
)

// Default RAG settings
const (
	DefaultChunkSize       = 500
	DefaultChunkOverlap    = 50
	DefaultMinChunkSize    = 100
	DefaultVectorSize      = 1536 // text-embedding-3-small
	DefaultScoreThreshold  = 0.25 // Lower threshold for multilingual support
	DefaultTopK            = 5    // Number of results to return
)

// RAGConfig holds the RAG configuration
type RAGConfig struct {
	Enabled        bool    `json:"enabled"`
	EmbeddingModel string  `json:"embedding_model"`
	VectorSize     int     `json:"vector_size"`
	ChunkSize      int     `json:"chunk_size"`
	ChunkOverlap   int     `json:"chunk_overlap"`
	MinChunkSize   int     `json:"min_chunk_size"`
	ScoreThreshold float32 `json:"score_threshold"`
	TopK           int     `json:"top_k"`
}

var (
	config     *RAGConfig
	configLock sync.RWMutex
)

// GetConfig returns the current RAG configuration
func GetConfig() *RAGConfig {
	configLock.RLock()
	defer configLock.RUnlock()

	if config == nil {
		return loadConfig()
	}
	return config
}

// getSettingInt gets an integer setting value with a default
func getSettingInt(key string, defaultValue int) int {
	val := models.GetSettingValue(key, "")
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// getSettingFloat32 gets a float32 setting value with a default
func getSettingFloat32(key string, defaultValue float32) float32 {
	val := models.GetSettingValue(key, "")
	if val == "" {
		return defaultValue
	}
	floatVal, err := strconv.ParseFloat(val, 32)
	if err != nil {
		return defaultValue
	}
	return float32(floatVal)
}

// loadConfig loads RAG settings from database
func loadConfig() *RAGConfig {
	cfg := &RAGConfig{
		Enabled:        models.GetSettingValue("rag.enabled", "true") == "true",
		EmbeddingModel: models.GetSettingValue("rag.embedding_model", "text-embedding-3-small"),
		VectorSize:     getSettingInt("rag.vector_size", DefaultVectorSize),
		ChunkSize:      getSettingInt("rag.chunk_size", DefaultChunkSize),
		ChunkOverlap:   getSettingInt("rag.chunk_overlap", DefaultChunkOverlap),
		MinChunkSize:   getSettingInt("rag.min_chunk_size", DefaultMinChunkSize),
		ScoreThreshold: getSettingFloat32("rag.score_threshold", DefaultScoreThreshold),
		TopK:           getSettingInt("rag.top_k", DefaultTopK),
	}
	return cfg
}

// ReloadConfig reloads the RAG configuration from database
func ReloadConfig() {
	configLock.Lock()
	defer configLock.Unlock()
	config = loadConfig()
	log.Info("RAG config reloaded: enabled=%v, model=%s, vectorSize=%d, chunkSize=%d",
		config.Enabled, config.EmbeddingModel, config.VectorSize, config.ChunkSize)
}

// UpdateConfig updates the RAG configuration in database
func UpdateConfig(cfg *RAGConfig) error {
	settings := map[string]string{
		"rag.enabled":         boolToString(cfg.Enabled),
		"rag.embedding_model": cfg.EmbeddingModel,
		"rag.vector_size":     intToString(cfg.VectorSize),
		"rag.chunk_size":      intToString(cfg.ChunkSize),
		"rag.chunk_overlap":   intToString(cfg.ChunkOverlap),
		"rag.min_chunk_size":  intToString(cfg.MinChunkSize),
		"rag.score_threshold": float32ToString(cfg.ScoreThreshold),
		"rag.top_k":           intToString(cfg.TopK),
	}

	for key, value := range settings {
		if err := models.SetSetting(key, value, "string", "rag", ""); err != nil {
			return err
		}
	}

	ReloadConfig()
	return nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func intToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func float32ToString(f float32) string {
	return fmt.Sprintf("%.2f", f)
}

// GetVectorSizeForModel returns the vector size for a given embedding model
func GetVectorSizeForModel(model string) int {
	switch model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536
	}
}
