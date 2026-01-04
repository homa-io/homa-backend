package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ToolDefinition represents a tool that can be called by the model
type ToolDefinition struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef represents a function definition for OpenAI tools
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall represents a tool call from the model
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ToolMessage represents a message containing tool call results
type ToolMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ChatCompletionRequestWithTools extends the request to include tools
type ChatCompletionRequestWithTools struct {
	Model       string           `json:"model"`
	Messages    []ToolMessage    `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

// ChatCompletionResponseWithTools extends the response to include tool calls
type ChatCompletionResponseWithTools struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
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

// ChatCompletionWithTools sends a chat completion request with tools to OpenAI
func (c *OpenAIClient) ChatCompletionWithTools(messages []ToolMessage, tools []ToolDefinition, maxTokens int, temperature float64) (*ChatCompletionResponseWithTools, error) {
	if maxTokens == 0 {
		maxTokens = 2000
	}
	if temperature == 0 {
		temperature = 0.7
	}

	req := ChatCompletionRequestWithTools{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Only include tools if there are any
	if len(tools) > 0 {
		req.Tools = tools
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

	var result ChatCompletionResponseWithTools
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", result.Error.Message)
	}

	return &result, nil
}

// ConvertToToolMessages converts simple ChatMessages to ToolMessages
func ConvertToToolMessages(messages []ChatMessage) []ToolMessage {
	result := make([]ToolMessage, len(messages))
	for i, msg := range messages {
		result[i] = ToolMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// CreateToolResultMessage creates a message containing tool call results
func CreateToolResultMessage(toolCallID string, result string) ToolMessage {
	return ToolMessage{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
	}
}

// CreateAssistantMessageWithToolCalls creates an assistant message with tool calls
func CreateAssistantMessageWithToolCalls(content string, toolCalls []ToolCall) ToolMessage {
	return ToolMessage{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	}
}
