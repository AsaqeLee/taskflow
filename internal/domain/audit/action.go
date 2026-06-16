package audit

// Action identifies a system-level task lifecycle event.
type Action string

const (
	ActionCreated   Action = "task_created"
	ActionAssigned  Action = "task_assigned"
	ActionStarted   Action = "task_started"
	ActionSubmitted Action = "task_submitted"
	ActionApproved  Action = "task_approved"
	ActionRejected  Action = "task_rejected"
	ActionClosed    Action = "task_closed"
	ActionCancelled Action = "task_cancelled"
	ActionReopened  Action = "task_reopened"
	ActionDeleted   Action = "task_deleted"
)

func (a Action) String() string {
	return string(a)
}
