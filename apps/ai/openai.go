package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/getevo/evo/v2/lib/settings"
)

// OpenAI API client
type OpenAIClient struct {
	apiKey         string
	baseURL        string
	threadEndpoint string
	httpClient     *http.Client
	model          string
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// ChatCompletionRequest represents the request to OpenAI Chat API
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatCompletionResponse represents the response from OpenAI Chat API
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// EmbeddingRequest represents the request to OpenAI Embeddings API
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents the response from OpenAI Embeddings API
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

var client *OpenAIClient

// InitClient initializes the OpenAI client
func InitClient() error {
	// Support both old and new config formats
	apiKey := settings.Get("ASSISTANT.APIKEY").String()
	if apiKey == "" {
		apiKey = settings.Get("OPENAI.API_KEY").String()
	}
	if apiKey == "" {
		return fmt.Errorf("ASSISTANT.APIKEY is not configured")
	}

	// Get endpoint - strip /assistants suffix to get base URL for chat completions
	endpoint := settings.Get("ASSISTANT.ENDPOINT", "https://api.openai.com/v1/assistants").String()
	baseURL := endpoint
	if len(endpoint) > 11 && endpoint[len(endpoint)-11:] == "/assistants" {
		baseURL = endpoint[:len(endpoint)-11]
	}

	threadEndpoint := settings.Get("ASSISTANT.THEAD_ENDPOINT", "https://api.openai.com/v1/threads").String()
	model := settings.Get("ASSISTANT.MODEL", "gpt-4o").String()

	client = &OpenAIClient{
		apiKey:         apiKey,
		baseURL:        baseURL,
		threadEndpoint: threadEndpoint,
		model:          model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	return nil
}

// GetClient returns the OpenAI client instance
func GetClient() *OpenAIClient {
	return client
}

// ChatCompletion sends a chat completion request to OpenAI
func (c *OpenAIClient) ChatCompletion(messages []ChatMessage, maxTokens int, temperature float64) (*ChatCompletionResponse, error) {
	if maxTokens == 0 {
		maxTokens = 2000
	}
	if temperature == 0 {
		temperature = 0.7
	}

	req := ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result ChatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", result.Error.Message)
	}

	return &result, nil
}

// GetEmbedding generates embeddings for the given texts
func (c *OpenAIClient) GetEmbedding(texts []string) (*EmbeddingResponse, error) {
	embeddingModel := settings.Get("OPENAI.EMBEDDING_MODEL", "text-embedding-3-small").String()

	req := EmbeddingRequest{
		Model: embeddingModel,
		Input: texts,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result EmbeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", result.Error.Message)
	}

	return &result, nil
}
