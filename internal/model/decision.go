package model

// DecisionType is what the agent chose to do with a Task, given its
// thread history — see DESIGN.md §4.
type DecisionType string

const (
	DecisionAskUser DecisionType = "ask_user"
	DecisionExecute DecisionType = "execute"
	DecisionFinish  DecisionType = "finish"
	DecisionRework  DecisionType = "rework"
)

// AgentDecision is the parsed response from an agent Client.Decide call.
//
// Question is set for ask_user. Command/Params are set for execute/rework
// — the concrete action to run and its arguments. Message is set for
// finish (the final result/summary to email) and ask_user (the question
// text is also carried in Message, so callers needing "what to send"
// don't need to switch on DecisionType).
type AgentDecision struct {
	Type     DecisionType
	Question string
	Command  string
	Params   map[string]string
	Message  string
}
