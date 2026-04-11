package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaClient communicates with the Ollama API.
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// OllamaRequest is the request body for the Ollama chat API.
type OllamaRequest struct {
	Model    string            `json:"model"`
	Messages []OllamaMessage   `json:"messages"`
	Stream   bool              `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// OllamaMessage represents a single message in the Ollama chat.
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaResponse is the response body from the Ollama chat API.
type OllamaResponse struct {
	Message OllamaMessage `json:"message"`
}

// NewOllamaClient creates a new OllamaClient with a 5-minute timeout for LLM generation.
func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Model returns the configured model name.
func (c *OllamaClient) Model() string {
	return c.model
}

// Generate sends a system+user prompt to Ollama and returns the generated text.
func (c *OllamaClient) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	ollamaReq := OllamaRequest{
		Model: c.model,
		Messages: []OllamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: false,
		Options: map[string]interface{}{
			"num_ctx":     2048,
			"num_predict": 512,
			"temperature": 0.7,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Use a dedicated context with 5-minute timeout independent of the caller's context,
	// because HTTP gateway/proxy may cancel the upstream context sooner.
	ollamaCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	reqURL := fmt.Sprintf("%s/api/chat", c.baseURL)
	req, err := http.NewRequestWithContext(ollamaCtx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}
