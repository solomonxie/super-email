// Package workflow holds AgentLoopWorkflow — the one workflow type that
// backs both inbound email and scheduled digests (DESIGN.md §4) — and its
// activities.
package workflow

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/solomonxie/super-email/internal/model"
)

// EmailReceivedSignal is the name of the signal an ask_user wait blocks
// on — sent by cmd/inbound-webhook (once T2.2 lands) or, for manual
// testing, `temporal workflow signal`.
const EmailReceivedSignal = "EmailReceived"

// EventsQuery returns the task's TaskEvent log so far — lets `temporal
// workflow query` inspect a running loop's decision trail without
// waiting for it to finish (see plan.md's "inspectable" priority).
const EventsQuery = "get_events"

// Input starts an AgentLoopWorkflow. TaskID/ThreadID identify the task
// for logging and (once T3.1 lands) persistence; InitialDescription is
// the first system TaskEvent — "here's what the user's email says", or a
// digest's fixed prompt.
type Input struct {
	TaskID              string
	ThreadID            string
	InitialDescription  string
	MaxIterations       int
	AskUserTimeout      time.Duration
	NotifyEmailFallback string // address to send finish/ask_user/stuck messages to
}

// Result is what the workflow returns on completion.
type Result struct {
	Status  model.TaskStatus
	Message string
}

// AgentLoopWorkflow runs CallAgentActivity in a loop, dispatching each
// AgentDecision per DESIGN.md §4's table, until the agent finishes,
// the iteration cap is hit, or an ask_user wait times out.
func AgentLoopWorkflow(ctx workflow.Context, input Input) (Result, error) {
	logger := workflow.GetLogger(ctx)

	events := []model.TaskEvent{
		{Role: model.EventRoleSystem, Content: input.InitialDescription},
	}

	err := workflow.SetQueryHandler(ctx, EventsQuery, func() ([]model.TaskEvent, error) {
		return events, nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("registering %s query handler: %w", EventsQuery, err)
	}

	signalChan := workflow.GetSignalChannel(ctx, EmailReceivedSignal)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	maxIterations := input.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	for iteration := 1; iteration <= maxIterations; iteration++ {
		var decision model.AgentDecision
		if err := workflow.ExecuteActivity(ctx, CallAgentActivityName, events).Get(ctx, &decision); err != nil {
			return Result{}, fmt.Errorf("CallAgentActivity: %w", err)
		}
		events = append(events, model.TaskEvent{Role: model.EventRoleAgent, Content: decisionSummary(decision)})

		switch decision.Type {
		case model.DecisionAskUser:
			if err := workflow.ExecuteActivity(ctx, SendEmailActivityName, input.NotifyEmailFallback, "Re: your task", decision.Message).Get(ctx, nil); err != nil {
				return Result{}, fmt.Errorf("SendEmailActivity (ask_user): %w", err)
			}

			reply, closed, err := awaitReply(ctx, signalChan, input.AskUserTimeout)
			if err != nil {
				return Result{}, err
			}
			if closed {
				events = append(events, model.TaskEvent{Role: model.EventRoleSystem, Content: "no reply received in time; closing task"})
				return Result{Status: model.TaskStatusClosed, Message: "closed: ask_user timed out with no reply"}, nil
			}
			events = append(events, model.TaskEvent{Role: model.EventRoleUser, Content: reply})

		case model.DecisionExecute, model.DecisionRework:
			var resultMsg string
			if err := workflow.ExecuteActivity(ctx, RunTaskActionActivityName, decision.Command, decision.Params).Get(ctx, &resultMsg); err != nil {
				return Result{}, fmt.Errorf("RunTaskActionActivity: %w", err)
			}
			events = append(events, model.TaskEvent{Role: model.EventRoleSystem, Content: resultMsg})

		case model.DecisionFinish:
			if err := workflow.ExecuteActivity(ctx, SendEmailActivityName, input.NotifyEmailFallback, "Re: your task", decision.Message).Get(ctx, nil); err != nil {
				return Result{}, fmt.Errorf("SendEmailActivity (finish): %w", err)
			}
			return Result{Status: model.TaskStatusDone, Message: decision.Message}, nil

		default:
			return Result{}, fmt.Errorf("unhandled decision type %q", decision.Type)
		}
	}

	logger.Warn("AgentLoopWorkflow hit max iterations, forcing finish", "maxIterations", maxIterations)
	stuckMsg := fmt.Sprintf("I got stuck after %d steps and couldn't finish this task — here's what I have so far.", maxIterations)
	if err := workflow.ExecuteActivity(ctx, SendEmailActivityName, input.NotifyEmailFallback, "Re: your task (stuck)", stuckMsg).Get(ctx, nil); err != nil {
		return Result{}, fmt.Errorf("SendEmailActivity (stuck): %w", err)
	}
	return Result{Status: model.TaskStatusStuck, Message: stuckMsg}, nil
}

// awaitReply blocks on signalChan or timeout, whichever comes first.
// closed=true means the timeout won.
func awaitReply(ctx workflow.Context, signalChan workflow.ReceiveChannel, timeout time.Duration) (reply string, closed bool, err error) {
	if timeout <= 0 {
		timeout = 72 * time.Hour
	}

	selector := workflow.NewSelector(ctx)
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timerFuture := workflow.NewTimer(timerCtx, timeout)

	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &reply)
		cancelTimer()
	})
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		if ferr := f.Get(ctx, nil); ferr == nil {
			closed = true
		}
		// A canceled timer (the signal path won the race) lands here too
		// with a non-nil error; closed stays false in that case.
	})
	selector.Select(ctx)

	return reply, closed, nil
}

// decisionSummary renders an AgentDecision as one TaskEvent line, kept
// in history so the next CallAgentActivity call sees its own past
// reasoning, not just system/user turns.
func decisionSummary(d model.AgentDecision) string {
	switch d.Type {
	case model.DecisionAskUser:
		return fmt.Sprintf("[ask_user] %s", d.Question)
	case model.DecisionExecute:
		return fmt.Sprintf("[execute] %s %v", d.Command, d.Params)
	case model.DecisionRework:
		return fmt.Sprintf("[rework] %s %v — %s", d.Command, d.Params, d.Message)
	case model.DecisionFinish:
		return fmt.Sprintf("[finish] %s", d.Message)
	default:
		return fmt.Sprintf("[%s]", d.Type)
	}
}
