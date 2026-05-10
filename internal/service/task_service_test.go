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

func TestSubmitTask_AssigneeCanMoveInProgressTaskToSubmitted(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(model.Task{
		ID:          "task_010",
		Title:       "Submit flow",
		Description: "test",
		Status:      TaskStatusInProgress,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   beforeUpdate,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.SubmitTask(model.User{ID: "u_worker_001"}, "task_010")
	if err != nil {
		t.Fatalf("SubmitTask returned error: %v", err)
	}

	if task.Status != TaskStatusSubmitted {
		t.Fatalf("expected status %q, got %q", TaskStatusSubmitted, task.Status)
	}
	if task.AssigneeID != "u_worker_001" {
		t.Fatalf("expected assignee to remain unchanged, got %q", task.AssigneeID)
	}
	if !task.UpdatedAt.After(beforeUpdate) {
		t.Fatalf("expected updated_at to move forward")
	}
}

func TestSubmitTask_RejectsNonAssignee(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_011",
		Title:       "Forbidden submit",
		Description: "test",
		Status:      TaskStatusInProgress,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.SubmitTask(model.User{ID: "u_other_001"}, "task_011")
	if err != ErrForbiddenSubmit {
		t.Fatalf("expected ErrForbiddenSubmit, got %v", err)
	}
}

func TestSubmitTask_RejectsNonInProgressStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_012",
		Title:       "Wrong status",
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

	_, err = svc.SubmitTask(model.User{ID: "u_worker_001"}, "task_012")
	if err != ErrInvalidTaskStatusForSubmit {
		t.Fatalf("expected ErrInvalidTaskStatusForSubmit, got %v", err)
	}
}

func TestRejectTask_OwnerCanMoveSubmittedTaskToAssigned(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(model.Task{
		ID:          "task_020",
		Title:       "Reject flow",
		Description: "test",
		Status:      TaskStatusSubmitted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   beforeUpdate,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.RejectTask(model.User{ID: "u_owner_001"}, "task_020")
	if err != nil {
		t.Fatalf("RejectTask returned error: %v", err)
	}

	if task.Status != TaskStatusAssigned {
		t.Fatalf("expected status %q, got %q", TaskStatusAssigned, task.Status)
	}
	if task.AssigneeID != "u_worker_001" {
		t.Fatalf("expected assignee to remain unchanged, got %q", task.AssigneeID)
	}
	if !task.UpdatedAt.After(beforeUpdate) {
		t.Fatalf("expected updated_at to move forward")
	}
}

func TestRejectTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_021",
		Title:       "Forbidden reject",
		Description: "test",
		Status:      TaskStatusSubmitted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.RejectTask(model.User{ID: "u_other_001"}, "task_021")
	if err != ErrForbiddenReject {
		t.Fatalf("expected ErrForbiddenReject, got %v", err)
	}
}

func TestRejectTask_RejectsNonSubmittedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_022",
		Title:       "Wrong reject status",
		Description: "test",
		Status:      TaskStatusInProgress,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.RejectTask(model.User{ID: "u_owner_001"}, "task_022")
	if err != ErrInvalidTaskStatusForReject {
		t.Fatalf("expected ErrInvalidTaskStatusForReject, got %v", err)
	}
}

func TestApproveTask_OwnerCanMoveSubmittedTaskToApproved(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(model.Task{
		ID:          "task_030",
		Title:       "Approve flow",
		Description: "test",
		Status:      TaskStatusSubmitted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   beforeUpdate,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.ApproveTask(model.User{ID: "u_owner_001"}, "task_030")
	if err != nil {
		t.Fatalf("ApproveTask returned error: %v", err)
	}

	if task.Status != TaskStatusApproved {
		t.Fatalf("expected status %q, got %q", TaskStatusApproved, task.Status)
	}
	if !task.UpdatedAt.After(beforeUpdate) {
		t.Fatalf("expected updated_at to move forward")
	}
}

func TestApproveTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_031",
		Title:       "Forbidden approve",
		Description: "test",
		Status:      TaskStatusSubmitted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.ApproveTask(model.User{ID: "u_other_001"}, "task_031")
	if err != ErrForbiddenApprove {
		t.Fatalf("expected ErrForbiddenApprove, got %v", err)
	}
}

func TestApproveTask_RejectsNonSubmittedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_032",
		Title:       "Wrong approve status",
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

	_, err = svc.ApproveTask(model.User{ID: "u_owner_001"}, "task_032")
	if err != ErrInvalidTaskStatusForApprove {
		t.Fatalf("expected ErrInvalidTaskStatusForApprove, got %v", err)
	}
}

func TestCloseTask_OwnerCanMoveApprovedTaskToCompleted(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(model.Task{
		ID:          "task_040",
		Title:       "Close flow",
		Description: "test",
		Status:      TaskStatusApproved,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   beforeUpdate,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.CloseTask(model.User{ID: "u_owner_001"}, "task_040")
	if err != nil {
		t.Fatalf("CloseTask returned error: %v", err)
	}

	if task.Status != TaskStatusCompleted {
		t.Fatalf("expected status %q, got %q", TaskStatusCompleted, task.Status)
	}
	if !task.UpdatedAt.After(beforeUpdate) {
		t.Fatalf("expected updated_at to move forward")
	}
}

func TestCloseTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_041",
		Title:       "Forbidden close",
		Description: "test",
		Status:      TaskStatusApproved,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.CloseTask(model.User{ID: "u_other_001"}, "task_041")
	if err != ErrForbiddenClose {
		t.Fatalf("expected ErrForbiddenClose, got %v", err)
	}
}

func TestCloseTask_RejectsNonApprovedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_042",
		Title:       "Wrong close status",
		Description: "test",
		Status:      TaskStatusSubmitted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.CloseTask(model.User{ID: "u_owner_001"}, "task_042")
	if err != ErrInvalidTaskStatusForClose {
		t.Fatalf("expected ErrInvalidTaskStatusForClose, got %v", err)
	}
}
