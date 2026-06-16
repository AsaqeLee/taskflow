package task

// Status is the lifecycle state of a task aggregate.
type Status string

const (
	StatusOpen       Status = "open"
	StatusAssigned   Status = "assigned"
	StatusInProgress Status = "in_progress"
	StatusSubmitted  Status = "submitted"
	StatusApproved   Status = "approved"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
	StatusDeleted    Status = "deleted"
)

func (s Status) String() string {
	return string(s)
}

func ParseStatus(value string) Status {
	return Status(value)
}
