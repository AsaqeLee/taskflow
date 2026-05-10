package service

import (
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
)

func TestStartTask_AssigneeCanMoveAssignedTaskToInProgress(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_001",
		Title:       "Start flow",
		Description: "test",
		Status:      TaskStatusAssigned,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.StartTask(model.User{ID: "u_worker_001"}, "task_001")
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}

	if task.Status != TaskStatusInProgress {
		t.Fatalf("expected status %q, got %q", TaskStatusInProgress, task.Status)
	}
	if task.AssigneeID != "u_worker_001" {
		t.Fatalf("expected assignee to remain unchanged, got %q", task.AssigneeID)
	}
	if !task.UpdatedAt.After(now) {
		t.Fatalf("expected updated_at to move forward")
	}
}

func TestStartTask_RejectsNonAssignee(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_002",
		Title:       "Forbidden start",
		Description: "test",
		Status:      TaskStatusAssigned,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.StartTask(model.User{ID: "u_other_001"}, "task_002")
	if err != ErrForbiddenStart {
		t.Fatalf("expected ErrForbiddenStart, got %v", err)
	}
}

func TestStartTask_RejectsNonAssignedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_003",
		Title:       "Wrong status",
		Description: "test",
		Status:      TaskStatusOpen,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.StartTask(model.User{ID: "u_worker_001"}, "task_003")
	if err != ErrInvalidTaskStatusForStart {
		t.Fatalf("expected ErrInvalidTaskStatusForStart, got %v", err)
	}
}

func TestStartTask_RejectsOpenTaskBeforePermissionCheck(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_004",
		Title:       "Open task",
		Description: "test",
		Status:      TaskStatusOpen,
		CreatorID:   "u_owner_001",
		AssigneeID:  "",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.StartTask(model.User{ID: "u_other_001"}, "task_004")
	if err != ErrInvalidTaskStatusForStart {
		t.Fatalf("expected ErrInvalidTaskStatusForStart, got %v", err)
	}
}
