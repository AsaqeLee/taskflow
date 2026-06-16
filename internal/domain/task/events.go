package task

import (
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain/audit"
	"github.com/AsaqeLee/taskflow/internal/domain/event"
	"github.com/AsaqeLee/taskflow/internal/domain/record"
)

const (
	EventStatusChanged = "task.status_changed"
	EventRecordDrafted = "task.record_drafted"
)

// StatusChangedEvent records a task lifecycle transition.
type StatusChangedEvent struct {
	TaskID      string
	ActorID     string
	AuditAction audit.Action
	FromStatus  Status
	ToStatus    Status
	occurredAt  time.Time
}

func NewStatusChangedEvent(taskID, actorID string, action audit.Action, from, to Status, at time.Time) StatusChangedEvent {
	return StatusChangedEvent{
		TaskID:      taskID,
		ActorID:     actorID,
		AuditAction: action,
		FromStatus:  from,
		ToStatus:    to,
		occurredAt:  at,
	}
}

func (e StatusChangedEvent) Name() string          { return EventStatusChanged }
func (e StatusChangedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e StatusChangedEvent) At() time.Time         { return e.occurredAt }

// RecordDraftedEvent records collaboration text produced by a transition.
type RecordDraftedEvent struct {
	Draft      record.Draft
	occurredAt time.Time
}

func NewRecordDraftedEvent(draft record.Draft, at time.Time) RecordDraftedEvent {
	return RecordDraftedEvent{Draft: draft, occurredAt: at}
}

func (e RecordDraftedEvent) Name() string          { return EventRecordDrafted }
func (e RecordDraftedEvent) OccurredAt() time.Time { return e.occurredAt }

func buildTransition(taskID, actorID string, action audit.Action, from, to Status, at time.Time, draft *record.Draft) Transition {
	events := []event.Event{
		NewStatusChangedEvent(taskID, actorID, action, from, to, at),
	}
	if draft != nil {
		events = append(events, NewRecordDraftedEvent(draft.AssignTaskID(taskID), at))
	}
	return Transition{Events: events}
}

func buildCreateTransition(actorID string, at time.Time) Transition {
	return Transition{
		Events: []event.Event{
			NewStatusChangedEvent("", actorID, audit.ActionCreated, "", StatusOpen, at),
		},
	}
}
