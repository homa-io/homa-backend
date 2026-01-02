package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
)

// QdrantClient is a client for the Qdrant vector database
type QdrantClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// QdrantPoint represents a point in Qdrant
type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float32                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// CollectionInfo represents information about a Qdrant collection
type CollectionInfo struct {
	Name         string `json:"name"`
	VectorSize   int    `json:"vector_size"`
	PointsCount  int    `json:"points_count"`
	Status       string `json:"status"`
	Exists       bool   `json:"exists"`
	Distance     string `json:"distance"`
	OnDiskPayload bool  `json:"on_disk_payload"`
}

var (
	qdrantClient     *QdrantClient
	qdrantClientLock sync.RWMutex
)

// InitQdrant initializes the Qdrant client
func InitQdrant() error {
	qdrantClientLock.Lock()
	defer qdrantClientLock.Unlock()

	url := settings.Get("QDRANT.URL", "http://localhost:6333").String()
	apiKey := settings.Get("QDRANT.API_KEY", "").String()

	qdrantClient = &QdrantClient{
		baseURL: url,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	log.Info("Qdrant client initialized with URL: %s", url)
	return nil
}

// GetQdrantClient returns the Qdrant client instance
func GetQdrantClient() *QdrantClient {
	qdrantClientLock.RLock()
	defer qdrantClientLock.RUnlock()
	return qdrantClient
}

// doRequest performs an HTTP request to Qdrant
func (c *QdrantClient) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Health checks if Qdrant is healthy
func (c *QdrantClient) Health() (bool, error) {
	_, err := c.doRequest("GET", "/", nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetCollectionInfo gets information about a collection
func (c *QdrantClient) GetCollectionInfo(name string) (*CollectionInfo, error) {
	respBody, err := c.doRequest("GET", "/collections/"+name, nil)
	if err != nil {
		// Check if it's a 404 (collection doesn't exist)
		if bytes.Contains([]byte(err.Error()), []byte("404")) {
			return &CollectionInfo{Name: name, Exists: false}, nil
		}
		return nil, err
	}

	var result struct {
		Result struct {
			Status string `json:"status"`
			Config struct {
				Params struct {
					Vectors struct {
						Size     int    `json:"size"`
						Distance string `json:"distance"`
					} `json:"vectors"`
					OnDiskPayload bool `json:"on_disk_payload"`
				} `json:"params"`
			} `json:"config"`
			PointsCount int `json:"points_count"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &CollectionInfo{
		Name:          name,
		Exists:        true,
		VectorSize:    result.Result.Config.Params.Vectors.Size,
		Distance:      result.Result.Config.Params.Vectors.Distance,
		PointsCount:   result.Result.PointsCount,
		Status:        result.Result.Status,
		OnDiskPayload: result.Result.Config.Params.OnDiskPayload,
	}, nil
}

// CreateCollection creates a new collection
func (c *QdrantClient) CreateCollection(name string, vectorSize int) error {
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	_, err := c.doRequest("PUT", "/collections/"+name, body)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Info("Created Qdrant collection: %s with vector size: %d", name, vectorSize)
	return nil
}

// DeleteCollection deletes a collection
func (c *QdrantClient) DeleteCollection(name string) error {
	_, err := c.doRequest("DELETE", "/collections/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	log.Info("Deleted Qdrant collection: %s", name)
	return nil
}

// UpsertPoints inserts or updates points in a collection
func (c *QdrantClient) UpsertPoints(collectionName string, points []QdrantPoint) error {
	body := map[string]interface{}{
		"points": points,
	}

	_, err := c.doRequest("PUT", "/collections/"+collectionName+"/points", body)
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// DeletePoints deletes points from a collection by filter
func (c *QdrantClient) DeletePoints(collectionName string, articleID string) error {
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key":   "article_id",
					"match": map[string]interface{}{"value": articleID},
				},
			},
		},
	}

	_, err := c.doRequest("POST", "/collections/"+collectionName+"/points/delete", body)
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// Search searches for similar vectors in a collection
func (c *QdrantClient) Search(collectionName string, vector []float32, limit int, scoreThreshold float32) ([]SearchResult, error) {
	body := map[string]interface{}{
		"vector":          vector,
		"limit":           limit,
		"with_payload":    true,
		"score_threshold": scoreThreshold,
	}

	respBody, err := c.doRequest("POST", "/collections/"+collectionName+"/points/search", body)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	searchResults := make([]SearchResult, len(result.Result))
	for i, r := range result.Result {
		searchResults[i] = SearchResult{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		}
	}

	return searchResults, nil
}

// CountPoints returns the number of points in a collection
func (c *QdrantClient) CountPoints(collectionName string) (int, error) {
	info, err := c.GetCollectionInfo(collectionName)
	if err != nil {
		return 0, err
	}
	return info.PointsCount, nil
}

// GetCollectionName returns the configured collection name
func GetCollectionName() string {
	return settings.Get("QDRANT.COLLECTION_NAME", "knowledge_base").String()
}
