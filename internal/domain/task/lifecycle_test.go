package task_test

import (
	"errors"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain"
	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

func TestLifecycle_HappyPathEmitsDomainEvents(t *testing.T) {
	now := time.Now().UTC()
	owner := domainuser.NewActor("u_owner")
	worker := domainuser.NewActor("u_worker")

	task, createTransition, err := domaintask.Create(owner.ID, "Lifecycle", "desc", now)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertStatusEvent(t, createTransition, domainaudit.ActionCreated, "", domaintask.StatusOpen)

	task = task.AssignID("task_lifecycle")
	change, err := task.Assign(owner, worker.ID, now)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	assertStatusEvent(t, change, domainaudit.ActionAssigned, domaintask.StatusOpen, domaintask.StatusAssigned)

	change, err = task.Start(worker, now)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	assertStatusEvent(t, change, domainaudit.ActionStarted, domaintask.StatusAssigned, domaintask.StatusInProgress)

	change, err = task.Submit(worker, "done", now)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	assertStatusEvent(t, change, domainaudit.ActionSubmitted, domaintask.StatusInProgress, domaintask.StatusSubmitted)
	assertRecordEvent(t, change, domainrecord.TypeSubmit)

	change, err = task.Approve(owner, "looks good", now)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	assertStatusEvent(t, change, domainaudit.ActionApproved, domaintask.StatusSubmitted, domaintask.StatusApproved)
	assertRecordEvent(t, change, domainrecord.TypeApprove)

	change, err = task.Close(owner, now)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	assertStatusEvent(t, change, domainaudit.ActionClosed, domaintask.StatusApproved, domaintask.StatusCompleted)
}

func TestLifecycle_RejectReturnsToAssigned(t *testing.T) {
	now := time.Now().UTC()
	owner := domainuser.NewActor("u_owner")
	worker := domainuser.NewActor("u_worker")
	task := domaintask.Restore("task_reject", "Reject path", "", domaintask.StatusSubmitted, owner.ID, worker.ID, now, now, nil, "")

	change, err := task.Reject(owner, "needs work", now)
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if task.Status() != domaintask.StatusAssigned {
		t.Fatalf("expected assigned, got %s", task.Status())
	}
	assertStatusEvent(t, change, domainaudit.ActionRejected, domaintask.StatusSubmitted, domaintask.StatusAssigned)
	assertRecordEvent(t, change, domainrecord.TypeReject)
}

func TestLifecycle_TableDrivenForbiddenCases(t *testing.T) {
	now := time.Now().UTC()
	owner := domainuser.NewActor("u_owner")
	worker := domainuser.NewActor("u_worker")
	other := domainuser.NewActor("u_other")

	cases := []struct {
		name   string
		status domaintask.Status
		run    func(*domaintask.Task) error
		want   error
	}{
		{
			name:   "non creator assign",
			status: domaintask.StatusOpen,
			run: func(task *domaintask.Task) error {
				_, err := task.Assign(other, worker.ID, now)
				return err
			},
			want: domain.ErrForbiddenAssign,
		},
		{
			name:   "non assignee start",
			status: domaintask.StatusAssigned,
			run: func(task *domaintask.Task) error {
				_, err := task.Start(other, now)
				return err
			},
			want: domain.ErrForbiddenStart,
		},
		{
			name:   "non assignee submit",
			status: domaintask.StatusInProgress,
			run: func(task *domaintask.Task) error {
				_, err := task.Submit(other, "done", now)
				return err
			},
			want: domain.ErrForbiddenSubmit,
		},
		{
			name:   "non creator approve",
			status: domaintask.StatusSubmitted,
			run: func(task *domaintask.Task) error {
				_, err := task.Approve(other, "ok", now)
				return err
			},
			want: domain.ErrForbiddenApprove,
		},
		{
			name:   "non creator delete",
			status: domaintask.StatusOpen,
			run: func(task *domaintask.Task) error {
				_, err := task.MarkDeleted(other, now)
				return err
			},
			want: domain.ErrForbiddenDelete,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := domaintask.Restore("task_forbidden", "Forbidden", "", tc.status, owner.ID, worker.ID, now, now, nil, "")
			err := tc.run(&task)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func assertStatusEvent(t *testing.T, transition domaintask.Transition, action domainaudit.Action, from, to domaintask.Status) {
	t.Helper()
	if transition.AuditAction() != action {
		t.Fatalf("expected audit action %s, got %s", action, transition.AuditAction())
	}
	if transition.FromStatus() != from {
		t.Fatalf("expected from status %s, got %s", from, transition.FromStatus())
	}
	if transition.ToStatus() != to {
		t.Fatalf("expected to status %s, got %s", to, transition.ToStatus())
	}
	if len(transition.Events) == 0 {
		t.Fatalf("expected domain events")
	}
	found := false
	for _, evt := range transition.Events {
		if evt.Name() == domaintask.EventStatusChanged {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected status changed event, got %#v", transition.Events)
	}
}

func assertRecordEvent(t *testing.T, transition domaintask.Transition, typ domainrecord.Type) {
	t.Helper()
	draft := transition.Record()
	if draft == nil {
		t.Fatalf("expected record draft")
	}
	if draft.Type != typ {
		t.Fatalf("expected record type %s, got %s", typ, draft.Type)
	}
	found := false
	for _, evt := range transition.Events {
		if evt.Name() == domaintask.EventRecordDrafted {
			if _, ok := evt.(domaintask.RecordDraftedEvent); ok {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected record drafted event")
	}
}
