package workflow

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"

	"github.com/solomonxie/super-email/internal/agent"
	"github.com/solomonxie/super-email/internal/model"
)

// Activity type names — used both to register (cmd/worker) and to
// ExecuteActivity by name (agent_loop.go), so the two stay in sync.
const (
	CallAgentActivityName     = "CallAgentActivity"
	RunTaskActionActivityName = "RunTaskActionActivity"
	SendEmailActivityName     = "SendEmailActivity"
)

// ActionFunc runs one execute/rework command and returns the result text
// that goes back into the task's history. The Phase 1 dispatch table
// (see NewActivities) is empty, so every command falls through to a stub
// that logs what it was asked to do — enough to drive and inspect the
// full loop before any real action exists (DESIGN.md §4, plan.md Phase 1).
type ActionFunc func(ctx context.Context, params map[string]string) (string, error)

// Activities bundles the dependencies CallAgentActivity/RunTaskActionActivity/
// SendEmailActivity need, so cmd/worker can construct one and register its
// methods on the worker.
type Activities struct {
	Agent    agent.Client
	Commands map[string]ActionFunc
}

func NewActivities(agentClient agent.Client) *Activities {
	return &Activities{
		Agent:    agentClient,
		Commands: map[string]ActionFunc{}, // Phase 1: empty — every command hits the stub
	}
}

// CallAgentActivity asks the configured LLM backend for the next
// AgentDecision given the task's history so far.
func (a *Activities) CallAgentActivity(ctx context.Context, events []model.TaskEvent) (model.AgentDecision, error) {
	return a.Agent.Decide(ctx, events)
}

// RunTaskActionActivity dispatches to the registered command, or a stub
// that logs the command + params and returns a canned result if nothing's
// registered for it yet.
func (a *Activities) RunTaskActionActivity(ctx context.Context, command string, params map[string]string) (string, error) {
	if fn, ok := a.Commands[command]; ok {
		return fn(ctx, params)
	}

	logger := activity.GetLogger(ctx)
	logger.Info("RunTaskActionActivity: stub — no handler registered for command", "command", command, "params", params)
	return fmt.Sprintf("(stub) ran %q with %v — no real handler wired up yet", command, params), nil
}

// SendEmailActivity is a stub until internal/mail's send client lands
// (T2.3) — logs what would have been sent instead of calling SES.
func (a *Activities) SendEmailActivity(ctx context.Context, to, subject, body string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("SendEmailActivity: stub — logging instead of sending", "to", to, "subject", subject, "body", body)
	return nil
}
