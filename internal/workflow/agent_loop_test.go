package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/solomonxie/super-email/internal/model"
)

// registerActivities lets tests mock activities by name (OnActivity by
// string requires the name be registered first) without needing a real
// agent.Client or command dispatch table.
func registerActivities(env *testsuite.TestWorkflowEnvironment) {
	a := &Activities{Commands: map[string]ActionFunc{}}
	env.RegisterActivityWithOptions(a.CallAgentActivity, activity.RegisterOptions{Name: CallAgentActivityName})
	env.RegisterActivityWithOptions(a.RunTaskActionActivity, activity.RegisterOptions{Name: RunTaskActionActivityName})
	env.RegisterActivityWithOptions(a.SendEmailActivity, activity.RegisterOptions{Name: SendEmailActivityName})
}

func TestAgentLoopWorkflow_ExecuteThenFinish(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	registerActivities(env)

	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionExecute, Command: "create_note", Params: map[string]string{"content": "x"}}, nil).Once()
	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionFinish, Message: "all done"}, nil).Once()
	env.OnActivity(RunTaskActionActivityName, mock.Anything, mock.Anything, mock.Anything).
		Return("(stub) ran it", nil).Once()
	env.OnActivity(SendEmailActivityName, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.ExecuteWorkflow(AgentLoopWorkflow, Input{TaskID: "t1", InitialDescription: "do a thing", MaxIterations: 10})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result Result
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, model.TaskStatusDone, result.Status)
	require.Equal(t, "all done", result.Message)
	env.AssertExpectations(t)
}

func TestAgentLoopWorkflow_AskUserThenReply(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	registerActivities(env)

	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionAskUser, Question: "which one?", Message: "which one?"}, nil).Once()
	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionFinish, Message: "got it, done"}, nil).Once()
	env.OnActivity(SendEmailActivityName, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Twice()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(EmailReceivedSignal, "the second one")
	}, time.Minute)

	env.ExecuteWorkflow(AgentLoopWorkflow, Input{TaskID: "t2", InitialDescription: "pick one", MaxIterations: 10, AskUserTimeout: time.Hour})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result Result
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, model.TaskStatusDone, result.Status)
	env.AssertExpectations(t)
}

func TestAgentLoopWorkflow_AskUserTimeoutCloses(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	registerActivities(env)

	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionAskUser, Question: "still there?", Message: "still there?"}, nil).Once()
	env.OnActivity(SendEmailActivityName, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.ExecuteWorkflow(AgentLoopWorkflow, Input{TaskID: "t3", InitialDescription: "ping", MaxIterations: 10, AskUserTimeout: time.Hour})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result Result
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, model.TaskStatusClosed, result.Status)
	env.AssertExpectations(t)
}

func TestAgentLoopWorkflow_MaxIterationsForcesStuck(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	registerActivities(env)

	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionExecute, Command: "create_note", Params: map[string]string{}}, nil)
	env.OnActivity(RunTaskActionActivityName, mock.Anything, mock.Anything, mock.Anything).
		Return("(stub) ran it", nil)
	env.OnActivity(SendEmailActivityName, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.ExecuteWorkflow(AgentLoopWorkflow, Input{TaskID: "t4", InitialDescription: "loop forever", MaxIterations: 2})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result Result
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, model.TaskStatusStuck, result.Status)
	env.AssertExpectations(t)
}

func TestAgentLoopWorkflow_ReworkFeedsBackIntoHistory(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	registerActivities(env)

	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionExecute, Command: "fetch", Params: map[string]string{}}, nil).Once()
	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionRework, Command: "fetch", Message: "result was empty, retry"}, nil).Once()
	env.OnActivity(CallAgentActivityName, mock.Anything, mock.Anything).
		Return(model.AgentDecision{Type: model.DecisionFinish, Message: "second try worked"}, nil).Once()
	env.OnActivity(RunTaskActionActivityName, mock.Anything, mock.Anything, mock.Anything).
		Return("(stub) ran it", nil).Twice()
	env.OnActivity(SendEmailActivityName, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.ExecuteWorkflow(AgentLoopWorkflow, Input{TaskID: "t5", InitialDescription: "fetch something", MaxIterations: 10})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result Result
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, model.TaskStatusDone, result.Status)
	env.AssertExpectations(t)
}
