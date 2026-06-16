package task

import (
	"github.com/AsaqeLee/taskflow/internal/domain/audit"
	"github.com/AsaqeLee/taskflow/internal/domain/event"
	"github.com/AsaqeLee/taskflow/internal/domain/record"
)

// Transition captures domain events emitted by a successful state change.
type Transition struct {
	Events []event.Event
}

func (t Transition) statusChanged() (StatusChangedEvent, bool) {
	for _, evt := range t.Events {
		if changed, ok := evt.(StatusChangedEvent); ok {
			return changed, true
		}
	}
	return StatusChangedEvent{}, false
}

func (t Transition) recordDraft() (*record.Draft, bool) {
	for _, evt := range t.Events {
		if drafted, ok := evt.(RecordDraftedEvent); ok {
			draft := drafted.Draft
			return &draft, true
		}
	}
	return nil, false
}

// AuditAction returns the audit action associated with the transition.
func (t Transition) AuditAction() audit.Action {
	if changed, ok := t.statusChanged(); ok {
		return changed.AuditAction
	}
	return ""
}

// FromStatus returns the status before the transition.
func (t Transition) FromStatus() Status {
	if changed, ok := t.statusChanged(); ok {
		return changed.FromStatus
	}
	return ""
}

// ToStatus returns the status after the transition.
func (t Transition) ToStatus() Status {
	if changed, ok := t.statusChanged(); ok {
		return changed.ToStatus
	}
	return ""
}

// Record returns an optional collaboration draft produced by the transition.
func (t Transition) Record() *record.Draft {
	draft, ok := t.recordDraft()
	if !ok {
		return nil
	}
	return draft
}

// Bind fills transition metadata that is only known after persistence.
func (t Transition) Bind(taskID, actorID string) Transition {
	events := make([]event.Event, 0, len(t.Events))
	for _, evt := range t.Events {
		switch e := evt.(type) {
		case StatusChangedEvent:
			if e.TaskID == "" {
				e.TaskID = taskID
			}
			if e.ActorID == "" {
				e.ActorID = actorID
			}
			events = append(events, e)
		case RecordDraftedEvent:
			if e.Draft.TaskID == "" {
				e.Draft = e.Draft.AssignTaskID(taskID)
			}
			if e.Draft.AuthorID == "" {
				e.Draft.AuthorID = actorID
			}
			events = append(events, e)
		default:
			events = append(events, evt)
		}
	}
	t.Events = events
	return t
}
