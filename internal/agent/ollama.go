package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/solomonxie/super-email/internal/model"
)

// OllamaClient talks to a local Ollama instance's HTTP API — no key, no
// cost, the default backend for developing loop mechanics (DESIGN.md §4).
type OllamaClient struct {
	host       string
	model      string
	httpClient *http.Client
}

func NewOllamaClient(host, model string) *OllamaClient {
	return &OllamaClient{host: host, model: model, httpClient: http.DefaultClient}
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

func (c *OllamaClient) Decide(ctx context.Context, events []model.TaskEvent) (model.AgentDecision, error) {
	reqBody := ollamaChatRequest{
		Model: c.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildPrompt(events)},
		},
		Stream: false,
		Format: "json",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("ollama: marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("ollama: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("ollama: reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return model.AgentDecision{}, fmt.Errorf("ollama: status %d: %s", resp.StatusCode, respBody)
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return model.AgentDecision{}, fmt.Errorf("ollama: parsing response envelope: %w", err)
	}

	return parseDecision(chatResp.Message.Content)
}
