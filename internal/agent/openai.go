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

// OpenAIClient talks to the OpenAI Chat Completions API — used when
// response quality matters more than iteration speed/cost (DESIGN.md §4).
type OpenAIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{apiKey: apiKey, model: model, httpClient: http.DefaultClient}
}

type openAIChatRequest struct {
	Model          string            `json:"model"`
	Messages       []openAIMessage   `json:"messages"`
	ResponseFormat openAIResponseFmt `json:"response_format"`
}

type openAIResponseFmt struct {
	Type string `json:"type"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

const openAIChatURL = "https://api.openai.com/v1/chat/completions"

func (c *OpenAIClient) Decide(ctx context.Context, events []model.TaskEvent) (model.AgentDecision, error) {
	reqBody := openAIChatRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildPrompt(events)},
		},
		ResponseFormat: openAIResponseFmt{Type: "json_object"},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("openai: marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatURL, bytes.NewReader(body))
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("openai: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.AgentDecision{}, fmt.Errorf("openai: reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return model.AgentDecision{}, fmt.Errorf("openai: status %d: %s", resp.StatusCode, respBody)
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return model.AgentDecision{}, fmt.Errorf("openai: parsing response envelope: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return model.AgentDecision{}, fmt.Errorf("openai: response had no choices")
	}

	return parseDecision(chatResp.Choices[0].Message.Content)
}
