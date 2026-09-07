// Package agent talks to the LLM backend that drives AgentLoopWorkflow's
// decisions — see DESIGN.md §4. Client is one interface, two
// implementations (ollama.go, openai.go); internal/config picks which one
// a given deployment/task uses.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/solomonxie/super-email/internal/config"
	"github.com/solomonxie/super-email/internal/model"
)

// Client decides the next AgentDecision for a task, given its full thread
// history so far (oldest first).
type Client interface {
	Decide(ctx context.Context, events []model.TaskEvent) (model.AgentDecision, error)
}

// NewClient builds the Client selected by cfg.LLMProvider.
func NewClient(cfg config.Config) (Client, error) {
	switch cfg.LLMProvider {
	case config.LLMProviderOllama:
		return NewOllamaClient(cfg.OllamaHost, cfg.OllamaModel), nil
	case config.LLMProviderOpenAI:
		return NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel), nil
	default:
		return nil, fmt.Errorf("agent: unknown LLM provider %q", cfg.LLMProvider)
	}
}

// systemPrompt explains the loop contract shared by every backend: the
// four decision types, and that execute/rework must name a concrete
// command + params for RunTaskActionActivity's dispatch table to run.
const systemPrompt = `You are the decision-making step of a personal task-agent loop. You are given the full history of one task's thread (user messages, your own prior decisions, and system results from actions you've run) and must decide the single next step.

Respond with ONLY a JSON object, no other text, matching this shape:
{"type": "ask_user"|"execute"|"finish"|"rework", "question": "", "command": "", "params": {}, "message": ""}

Rules for each type:
- "ask_user": the task is unclear or missing information. Set "question" to what you need to know, and "message" to the same text (it's what gets emailed).
- "execute": you know exactly what to do. Set "command" to a short snake_case action name (e.g. "create_note") and "params" to a flat string map of its arguments. Leave "question" and "message" empty.
- "rework": the previous action's result (the most recent system message) wasn't good enough. Same as "execute" — set "command"/"params" for the retry — but set "message" to explain, briefly, why the prior result was rejected.
- "finish": the task is done. Set "message" to the final result/summary to email back. Leave "command"/"params" empty.

If the task's intent is unclear and nothing else fits, prefer "execute" with command "create_note" over asking — capturing what was sent beats silently failing.`

// buildPrompt renders the thread history into the single user-turn text
// sent alongside systemPrompt.
func buildPrompt(events []model.TaskEvent) string {
	var b strings.Builder
	b.WriteString("Thread history (oldest first):\n")
	for _, e := range events {
		fmt.Fprintf(&b, "[%s] %s\n", e.Role, e.Content)
	}
	b.WriteString("\nDecide the next step now.")
	return b.String()
}

// decisionWire is the on-the-wire JSON shape both backends parse their
// response into.
type decisionWire struct {
	Type     string            `json:"type"`
	Question string            `json:"question"`
	Command  string            `json:"command"`
	Params   map[string]string `json:"params"`
	Message  string            `json:"message"`
}

// parseDecision extracts the JSON object from an LLM response (tolerating
// leading/trailing prose some models add despite instructions) and
// validates it into a model.AgentDecision.
func parseDecision(raw string) (model.AgentDecision, error) {
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start < 0 || end < start {
		return model.AgentDecision{}, fmt.Errorf("agent: no JSON object in response: %q", raw)
	}

	var w decisionWire
	if err := json.Unmarshal([]byte(raw[start:end+1]), &w); err != nil {
		return model.AgentDecision{}, fmt.Errorf("agent: parsing decision JSON: %w", err)
	}

	d := model.AgentDecision{
		Type:     model.DecisionType(w.Type),
		Question: w.Question,
		Command:  w.Command,
		Params:   w.Params,
		Message:  w.Message,
	}

	switch d.Type {
	case model.DecisionAskUser, model.DecisionExecute, model.DecisionFinish, model.DecisionRework:
	default:
		return model.AgentDecision{}, fmt.Errorf("agent: unknown decision type %q", w.Type)
	}
	if (d.Type == model.DecisionExecute || d.Type == model.DecisionRework) && d.Command == "" {
		return model.AgentDecision{}, fmt.Errorf("agent: decision type %q requires a command", d.Type)
	}

	return d, nil
}
