package agent

import (
	"testing"

	"github.com/solomonxie/super-email/internal/config"
	"github.com/solomonxie/super-email/internal/model"
)

func TestParseDecisionExecute(t *testing.T) {
	raw := `{"type": "execute", "command": "create_note", "params": {"content": "buy milk"}}`
	d, err := parseDecision(raw)
	if err != nil {
		t.Fatalf("parseDecision() error = %v", err)
	}
	if d.Type != model.DecisionExecute {
		t.Errorf("Type = %q, want execute", d.Type)
	}
	if d.Command != "create_note" {
		t.Errorf("Command = %q, want create_note", d.Command)
	}
	if d.Params["content"] != "buy milk" {
		t.Errorf("Params[content] = %q, want %q", d.Params["content"], "buy milk")
	}
}

func TestParseDecisionToleratesSurroundingProse(t *testing.T) {
	raw := "Sure, here's my decision:\n{\"type\": \"finish\", \"message\": \"done\"}\nHope that helps!"
	d, err := parseDecision(raw)
	if err != nil {
		t.Fatalf("parseDecision() error = %v", err)
	}
	if d.Type != model.DecisionFinish || d.Message != "done" {
		t.Errorf("got %+v", d)
	}
}

func TestParseDecisionRejectsUnknownType(t *testing.T) {
	if _, err := parseDecision(`{"type": "bogus"}`); err == nil {
		t.Fatal("want error for unknown decision type, got nil")
	}
}

func TestParseDecisionRejectsExecuteWithoutCommand(t *testing.T) {
	if _, err := parseDecision(`{"type": "execute"}`); err == nil {
		t.Fatal("want error for execute without command, got nil")
	}
}

func TestParseDecisionRejectsNonJSON(t *testing.T) {
	if _, err := parseDecision("not json at all"); err == nil {
		t.Fatal("want error for non-JSON response, got nil")
	}
}

func TestNewClientUnknownProvider(t *testing.T) {
	_, err := NewClient(config.Config{LLMProvider: "bogus"})
	if err == nil {
		t.Fatal("want error for unknown provider, got nil")
	}
}
