package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/google/uuid"
)

// QdrantClient handles communication with Qdrant vector database
type QdrantClient struct {
	baseURL    string
	collection string
	httpClient *http.Client
}

var qdrantClient *QdrantClient

// QdrantPoint represents a point in Qdrant
type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantSearchRequest represents a search request
type QdrantSearchRequest struct {
	Vector      []float32              `json:"vector"`
	Limit       int                    `json:"limit"`
	WithPayload bool                   `json:"with_payload"`
	Filter      *QdrantFilter          `json:"filter,omitempty"`
	ScoreThreshold float32             `json:"score_threshold,omitempty"`
}

// QdrantFilter represents a filter for search
type QdrantFilter struct {
	Must   []QdrantCondition `json:"must,omitempty"`
	Should []QdrantCondition `json:"should,omitempty"`
}

// QdrantCondition represents a filter condition
type QdrantCondition struct {
	Key   string        `json:"key"`
	Match QdrantMatch   `json:"match"`
}

// QdrantMatch represents a match condition
type QdrantMatch struct {
	Value interface{} `json:"value"`
}

// QdrantSearchResult represents a search result
type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Version int                    `json:"version"`
	Score   float32                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantSearchResponse represents the response from a search
type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
	Status string               `json:"status"`
	Time   float64              `json:"time"`
}

// QdrantUpsertRequest represents an upsert request
type QdrantUpsertRequest struct {
	Points []QdrantPoint `json:"points"`
}

// QdrantDeleteRequest represents a delete request
type QdrantDeleteRequest struct {
	Points []string `json:"points"`
}

// InitQdrant initializes the Qdrant client
func InitQdrant() error {
	host := settings.Get("QDRANT.HOST", "http://localhost:6333").String()
	collection := settings.Get("QDRANT.COLLECTION", "knowledge_base").String()

	qdrantClient = &QdrantClient{
		baseURL:    host,
		collection: collection,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Ensure collection exists
	if err := qdrantClient.EnsureCollection(); err != nil {
		return fmt.Errorf("failed to ensure Qdrant collection: %w", err)
	}

	log.Info("Qdrant client initialized: %s, collection: %s", host, collection)
	return nil
}

// GetQdrantClient returns the Qdrant client instance
func GetQdrantClient() *QdrantClient {
	return qdrantClient
}

// EnsureCollection creates the collection if it doesn't exist
func (q *QdrantClient) EnsureCollection() error {
	// Check if collection exists
	resp, err := q.httpClient.Get(fmt.Sprintf("%s/collections/%s", q.baseURL, q.collection))
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// Collection exists
		return nil
	}

	// Create collection with OpenAI embedding dimensions (1536 for text-embedding-3-small, 3072 for text-embedding-3-large)
	// Using 1536 as default for text-embedding-3-small
	vectorSize := settings.Get("QDRANT.VECTOR_SIZE", 1536).Int()

	createReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	body, err := json.Marshal(createReq)
	if err != nil {
		return fmt.Errorf("failed to marshal create request: %w", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/collections/%s", q.baseURL, q.collection), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(respBody))
	}

	log.Info("Created Qdrant collection: %s with vector size: %d", q.collection, vectorSize)
	return nil
}

// Upsert adds or updates points in the collection
func (q *QdrantClient) Upsert(points []QdrantPoint) error {
	if len(points) == 0 {
		return nil
	}

	upsertReq := QdrantUpsertRequest{
		Points: points,
	}

	body, err := json.Marshal(upsertReq)
	if err != nil {
		return fmt.Errorf("failed to marshal upsert request: %w", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/collections/%s/points", q.baseURL, q.collection), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert points: %s", string(respBody))
	}

	return nil
}

// Delete removes points from the collection
func (q *QdrantClient) Delete(pointIDs []string) error {
	if len(pointIDs) == 0 {
		return nil
	}

	deleteReq := map[string]interface{}{
		"points": pointIDs,
	}

	body, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/collections/%s/points/delete", q.baseURL, q.collection), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete points: %s", string(respBody))
	}

	return nil
}

// DeleteByArticleID removes all points for a specific article
func (q *QdrantClient) DeleteByArticleID(articleID uuid.UUID) error {
	deleteReq := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "article_id",
					"match": map[string]interface{}{
						"value": articleID.String(),
					},
				},
			},
		},
	}

	body, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/collections/%s/points/delete", q.baseURL, q.collection), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete points: %s", string(respBody))
	}

	return nil
}

// Search performs a vector similarity search
func (q *QdrantClient) Search(vector []float32, limit int, filter *QdrantFilter, scoreThreshold float32) ([]QdrantSearchResult, error) {
	if limit == 0 {
		limit = 5
	}

	searchReq := QdrantSearchRequest{
		Vector:         vector,
		Limit:          limit,
		WithPayload:    true,
		Filter:         filter,
		ScoreThreshold: scoreThreshold,
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, q.collection), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed: %s", string(respBody))
	}

	var searchResp QdrantSearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return searchResp.Result, nil
}

// SearchByText searches using text (generates embedding first)
func (q *QdrantClient) SearchByText(text string, limit int, categoryID *string) ([]QdrantSearchResult, error) {
	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	// Get embedding for the query
	embResp, err := client.GetEmbedding([]string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding received")
	}

	// Build filter if category specified
	var filter *QdrantFilter
	if categoryID != nil && *categoryID != "" {
		filter = &QdrantFilter{
			Must: []QdrantCondition{
				{
					Key:   "category_id",
					Match: QdrantMatch{Value: *categoryID},
				},
				{
					Key:   "status",
					Match: QdrantMatch{Value: "published"},
				},
			},
		}
	} else {
		filter = &QdrantFilter{
			Must: []QdrantCondition{
				{
					Key:   "status",
					Match: QdrantMatch{Value: "published"},
				},
			},
		}
	}

	return q.Search(embResp.Data[0].Embedding, limit, filter, 0.3)
}

// GetCollectionInfo returns information about the collection
func (q *QdrantClient) GetCollectionInfo() (map[string]interface{}, error) {
	resp, err := q.httpClient.Get(fmt.Sprintf("%s/collections/%s", q.baseURL, q.collection))
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
