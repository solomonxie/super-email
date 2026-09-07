// Package model holds the types shared between the agent loop workflow,
// the agent client, and storage — see DESIGN.md §4-5.
package model

import "time"

// TaskStatus is the lifecycle state of a Task.
type TaskStatus string

const (
	TaskStatusRunning TaskStatus = "running"
	TaskStatusDone    TaskStatus = "done"
	TaskStatusStuck   TaskStatus = "stuck"  // hit the iteration cap
	TaskStatusClosed  TaskStatus = "closed" // ask_user timed out with no reply
)

// Task is one AgentLoopWorkflow execution: an inbound email thread, or a
// scheduled digest run.
type Task struct {
	ID             string
	ThreadID       string
	Status         TaskStatus
	IterationCount int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// EventRole identifies who produced a TaskEvent.
type EventRole string

const (
	EventRoleUser   EventRole = "user"
	EventRoleAgent  EventRole = "agent"
	EventRoleSystem EventRole = "system"
)

// TaskEvent is one entry in a Task's thread history: the initial
// description, a user's email reply, an agent decision, or a system note
// (action result, timeout, iteration-cap message).
type TaskEvent struct {
	TaskID    string
	Role      EventRole
	Content   string
	CreatedAt time.Time
}
