package task_test

import (
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

func TestStart_AssigneeCanMoveAssignedTaskToInProgress(t *testing.T) {
	now := time.Now().UTC()
	task := domaintask.Restore(
		"task_001",
		"Start flow",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_worker_001",
		now,
		now.Add(-time.Second),
		nil,
		"",
	)

	change, err := task.Start(domainuser.NewActor("u_worker_001"), now)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if task.Status() != domaintask.StatusInProgress {
		t.Fatalf("expected status %q, got %q", domaintask.StatusInProgress, task.Status())
	}
	if change.FromStatus() != domaintask.StatusAssigned {
		t.Fatalf("expected from status %q, got %q", domaintask.StatusAssigned, change.FromStatus())
	}
}

func TestStart_RejectsNonAssignee(t *testing.T) {
	now := time.Now().UTC()
	task := domaintask.Restore("task_002", "Forbidden start", "test", domaintask.StatusAssigned, "u_owner_001", "u_worker_001", now, now, nil, "")

	_, err := task.Start(domainuser.NewActor("u_other_001"), now)
	if err != domain.ErrForbiddenStart {
		t.Fatalf("expected ErrForbiddenStart, got %v", err)
	}
}

func TestStart_RejectsNonAssignedStatus(t *testing.T) {
	now := time.Now().UTC()
	task := domaintask.Restore("task_003", "Wrong status", "test", domaintask.StatusOpen, "u_owner_001", "u_worker_001", now, now, nil, "")

	_, err := task.Start(domainuser.NewActor("u_worker_001"), now)
	if err != domain.ErrInvalidTaskStatusForStart {
		t.Fatalf("expected ErrInvalidTaskStatusForStart, got %v", err)
	}
}

func TestCancel_RecordsPreviousStatusInTransition(t *testing.T) {
	now := time.Now().UTC()
	task := domaintask.Restore("task_004", "Cancel me", "test", domaintask.StatusInProgress, "u_owner_001", "u_worker_001", now, now, nil, "")

	change, err := task.Cancel(domainuser.NewActor("u_owner_001"), "no longer needed", now)
	if err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if change.FromStatus() != domaintask.StatusInProgress {
		t.Fatalf("expected from status %q, got %q", domaintask.StatusInProgress, change.FromStatus())
	}
	if change.ToStatus() != domaintask.StatusCancelled {
		t.Fatalf("expected to status %q, got %q", domaintask.StatusCancelled, change.ToStatus())
	}
}

func TestCreate_RejectsShortTitle(t *testing.T) {
	_, _, err := domaintask.Create("u_owner_001", "ab", "desc", time.Now().UTC())
	if err != domain.ErrTooShortTaskTitle {
		t.Fatalf("expected ErrTooShortTaskTitle, got %v", err)
	}
}
