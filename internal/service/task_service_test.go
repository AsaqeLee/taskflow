package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestStartTask_AssigneeCanMoveAssignedTaskToInProgress(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_001",
		Title:       "Start flow",
		Description: "test",
		Status:      TaskStatusAssigned,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   beforeUpdate,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.StartTask(context.Background(), model.User{ID: "u_worker_001"}, "task_001")
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}

	if task.Status != TaskStatusInProgress {
		t.Fatalf("expected status %q, got %q", TaskStatusInProgress, task.Status)
	}
	if task.AssigneeID != "u_worker_001" {
		t.Fatalf("expected assignee to remain unchanged, got %q", task.AssigneeID)
	}
	if !task.UpdatedAt.After(beforeUpdate) {
		t.Fatalf("expected updated_at to move forward")
	}
}

func TestStartTask_RejectsNonAssignee(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, err = svc.StartTask(context.Background(), model.User{ID: "u_other_001"}, "task_002")
	if err != ErrForbiddenStart {
		t.Fatalf("expected ErrForbiddenStart, got %v", err)
	}
}

func TestStartTask_RejectsNonAssignedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, err = svc.StartTask(context.Background(), model.User{ID: "u_worker_001"}, "task_003")
	if err != ErrInvalidTaskStatusForStart {
		t.Fatalf("expected ErrInvalidTaskStatusForStart, got %v", err)
	}
}

func TestStartTask_RejectsOpenTaskBeforePermissionCheck(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, err = svc.StartTask(context.Background(), model.User{ID: "u_other_001"}, "task_004")
	if err != ErrInvalidTaskStatusForStart {
		t.Fatalf("expected ErrInvalidTaskStatusForStart, got %v", err)
	}
}

func TestSubmitTask_AssigneeCanMoveInProgressTaskToSubmitted(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), model.Task{
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

	task, record, err := svc.SubmitTask(context.Background(), model.User{ID: "u_worker_001"}, "task_010", "Implemented the task and attached proof")
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
	if record.TaskID != "task_010" {
		t.Fatalf("expected record task_id %q, got %q", "task_010", record.TaskID)
	}
	if record.AuthorID != "u_worker_001" {
		t.Fatalf("expected record author_id %q, got %q", "u_worker_001", record.AuthorID)
	}
	if record.Type != model.TaskRecordTypeSubmit {
		t.Fatalf("expected record type %q, got %q", model.TaskRecordTypeSubmit, record.Type)
	}
	if record.Content != "Implemented the task and attached proof" {
		t.Fatalf("unexpected record content %q", record.Content)
	}
}

func TestSubmitTask_RejectsNonAssignee(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, _, err = svc.SubmitTask(context.Background(), model.User{ID: "u_other_001"}, "task_011", "done")
	if err != ErrForbiddenSubmit {
		t.Fatalf("expected ErrForbiddenSubmit, got %v", err)
	}
}

func TestSubmitTask_RejectsNonInProgressStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, _, err = svc.SubmitTask(context.Background(), model.User{ID: "u_worker_001"}, "task_012", "done")
	if err != ErrInvalidTaskStatusForSubmit {
		t.Fatalf("expected ErrInvalidTaskStatusForSubmit, got %v", err)
	}
}

func TestSubmitTask_RejectsEmptyRecordContent(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_013",
		Title:       "Missing record content",
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

	_, _, err = svc.SubmitTask(context.Background(), model.User{ID: "u_worker_001"}, "task_013", "   ")
	if err != ErrEmptyTaskRecordContent {
		t.Fatalf("expected ErrEmptyTaskRecordContent, got %v", err)
	}
}

func TestRejectTask_OwnerCanMoveSubmittedTaskToAssigned(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), model.Task{
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

	task, record, err := svc.RejectTask(context.Background(), model.User{ID: "u_owner_001"}, "task_020", "Please revise the missing edge cases")
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
	if record.TaskID != "task_020" {
		t.Fatalf("expected record task_id %q, got %q", "task_020", record.TaskID)
	}
	if record.AuthorID != "u_owner_001" {
		t.Fatalf("expected record author_id %q, got %q", "u_owner_001", record.AuthorID)
	}
	if record.Type != model.TaskRecordTypeReject {
		t.Fatalf("expected record type %q, got %q", model.TaskRecordTypeReject, record.Type)
	}
	if record.Content != "Please revise the missing edge cases" {
		t.Fatalf("unexpected record content %q", record.Content)
	}
}

func TestRejectTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, _, err = svc.RejectTask(context.Background(), model.User{ID: "u_other_001"}, "task_021", "needs changes")
	if err != ErrForbiddenReject {
		t.Fatalf("expected ErrForbiddenReject, got %v", err)
	}
}

func TestRejectTask_RejectsNonSubmittedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, _, err = svc.RejectTask(context.Background(), model.User{ID: "u_owner_001"}, "task_022", "needs changes")
	if err != ErrInvalidTaskStatusForReject {
		t.Fatalf("expected ErrInvalidTaskStatusForReject, got %v", err)
	}
}

func TestRejectTask_RejectsEmptyRecordContent(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_023",
		Title:       "Missing reject reason",
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

	_, _, err = svc.RejectTask(context.Background(), model.User{ID: "u_owner_001"}, "task_023", "   ")
	if err != ErrEmptyTaskRecordContent {
		t.Fatalf("expected ErrEmptyTaskRecordContent, got %v", err)
	}
}

func TestApproveTask_OwnerCanMoveSubmittedTaskToApproved(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), model.Task{
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

	task, record, err := svc.ApproveTask(context.Background(), model.User{ID: "u_owner_001"}, "task_030", "Looks good, accepted")
	if err != nil {
		t.Fatalf("ApproveTask returned error: %v", err)
	}

	if task.Status != TaskStatusApproved {
		t.Fatalf("expected status %q, got %q", TaskStatusApproved, task.Status)
	}
	if !task.UpdatedAt.After(beforeUpdate) {
		t.Fatalf("expected updated_at to move forward")
	}
	if record.Type != model.TaskRecordTypeApprove {
		t.Fatalf("expected record type %q, got %q", model.TaskRecordTypeApprove, record.Type)
	}
	if record.Content != "Looks good, accepted" {
		t.Fatalf("unexpected record content %q", record.Content)
	}
}

func TestApproveTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, _, err = svc.ApproveTask(context.Background(), model.User{ID: "u_other_001"}, "task_031", "approved")
	if err != ErrForbiddenApprove {
		t.Fatalf("expected ErrForbiddenApprove, got %v", err)
	}
}

func TestApproveTask_RejectsNonSubmittedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, _, err = svc.ApproveTask(context.Background(), model.User{ID: "u_owner_001"}, "task_032", "approved")
	if err != ErrInvalidTaskStatusForApprove {
		t.Fatalf("expected ErrInvalidTaskStatusForApprove, got %v", err)
	}
}

func TestApproveTask_RejectsEmptyRecordContent(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_033",
		Title:       "Missing approve content",
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

	_, _, err = svc.ApproveTask(context.Background(), model.User{ID: "u_owner_001"}, "task_033", "   ")
	if err != ErrEmptyTaskRecordContent {
		t.Fatalf("expected ErrEmptyTaskRecordContent, got %v", err)
	}
}

func TestCloseTask_OwnerCanMoveApprovedTaskToCompleted(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), model.Task{
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

	task, err := svc.CloseTask(context.Background(), model.User{ID: "u_owner_001"}, "task_040")
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
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, err = svc.CloseTask(context.Background(), model.User{ID: "u_other_001"}, "task_041")
	if err != ErrForbiddenClose {
		t.Fatalf("expected ErrForbiddenClose, got %v", err)
	}
}

func TestCloseTask_RejectsNonApprovedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	_, err = svc.CloseTask(context.Background(), model.User{ID: "u_owner_001"}, "task_042")
	if err != ErrInvalidTaskStatusForClose {
		t.Fatalf("expected ErrInvalidTaskStatusForClose, got %v", err)
	}
}

func TestListTaskRecords_ReturnsTaskRecordsOrderedByCreatedAt(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_050",
		Title:       "List records",
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

	first, err := recordRepo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_050",
		AuthorID:  "u_worker_001",
		Type:      model.TaskRecordTypeSubmit,
		Content:   "first",
		CreatedAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("seed first record: %v", err)
	}
	second, err := recordRepo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_050",
		AuthorID:  "u_owner_001",
		Type:      model.TaskRecordTypeApprove,
		Content:   "second",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed second record: %v", err)
	}
	_, err = recordRepo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_other",
		AuthorID:  "u_other_001",
		Type:      model.TaskRecordTypeSubmit,
		Content:   "ignored",
		CreatedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("seed other task record: %v", err)
	}

	records, err := svc.ListTaskRecords(context.Background(), "task_050")
	if err != nil {
		t.Fatalf("ListTaskRecords returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != first.ID || records[1].ID != second.ID {
		t.Fatalf("expected ordered record ids [%s %s], got [%s %s]", first.ID, second.ID, records[0].ID, records[1].ID)
	}
}

func TestListTaskRecords_ReturnsNotFoundForUnknownTask(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())

	_, err := svc.ListTaskRecords(context.Background(), "missing")
	if err != repository.ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestCancelTask_OwnerCanCancelActiveTasks(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	states := []string{TaskStatusOpen, TaskStatusAssigned, TaskStatusInProgress, TaskStatusSubmitted}
	for i, startStatus := range states {
		taskID := fmt.Sprintf("task_cancel_%d", i)
		_, err := repo.Create(context.Background(), model.Task{
			ID:          taskID,
			Title:       "Active Task " + startStatus,
			Status:      startStatus,
			CreatorID:   "u_owner_001",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		if err != nil {
			t.Fatalf("seed task: %v", err)
		}

		updatedTask, record, err := svc.CancelTask(context.Background(), model.User{ID: "u_owner_001"}, taskID, "Cancelling this task due to scope change")
		if err != nil {
			t.Fatalf("CancelTask from %s returned error: %v", startStatus, err)
		}

		if updatedTask.Status != TaskStatusCancelled {
			t.Fatalf("expected status %q, got %q", TaskStatusCancelled, updatedTask.Status)
		}
		if record.Type != model.TaskRecordTypeCancel {
			t.Fatalf("expected record type %q, got %q", model.TaskRecordTypeCancel, record.Type)
		}
		if record.Content != "Cancelling this task due to scope change" {
			t.Fatalf("unexpected record content %q", record.Content)
		}
	}
}

func TestCancelTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_cancel_err",
		Title:     "Cancel Error",
		Status:    TaskStatusOpen,
		CreatorID: "u_owner_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, _, err = svc.CancelTask(context.Background(), model.User{ID: "u_other_001"}, "task_cancel_err", "cancel")
	if err != ErrForbiddenCancel {
		t.Fatalf("expected ErrForbiddenCancel, got %v", err)
	}
}

func TestCancelTask_RejectsCompletedTask(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_cancel_comp",
		Title:     "Cancel Completed",
		Status:    TaskStatusCompleted,
		CreatorID: "u_owner_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, _, err = svc.CancelTask(context.Background(), model.User{ID: "u_owner_001"}, "task_cancel_comp", "cancel")
	if err != ErrInvalidTaskStatusForCancel {
		t.Fatalf("expected ErrInvalidTaskStatusForCancel, got %v", err)
	}
}

func TestReactivateTask_OwnerCanReactivateCancelledOrCompletedTask(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	// 1. Reactivate task with no assignee (should go to open)
	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_react_open",
		Title:       "React Open",
		Status:      TaskStatusCancelled,
		CreatorID:   "u_owner_001",
		AssigneeID:  "",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task1, rec1, err := svc.ReactivateTask(context.Background(), model.User{ID: "u_owner_001"}, "task_react_open", "reactivating open task")
	if err != nil {
		t.Fatalf("ReactivateTask open: %v", err)
	}
	if task1.Status != TaskStatusOpen {
		t.Fatalf("expected status open, got %q", task1.Status)
	}
	if rec1.Type != model.TaskRecordTypeReactivate {
		t.Fatalf("expected record type reactivate, got %q", rec1.Type)
	}

	// 2. Reactivate task with assignee (should go to assigned)
	_, err = repo.Create(context.Background(), model.Task{
		ID:          "task_react_assign",
		Title:       "React Assign",
		Status:      TaskStatusCompleted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task2, rec2, err := svc.ReactivateTask(context.Background(), model.User{ID: "u_owner_001"}, "task_react_assign", "reactivating assigned task")
	if err != nil {
		t.Fatalf("ReactivateTask assigned: %v", err)
	}
	if task2.Status != TaskStatusAssigned {
		t.Fatalf("expected status assigned, got %q", task2.Status)
	}
	if rec2.Content != "reactivating assigned task" {
		t.Fatalf("unexpected record content %q", rec2.Content)
	}
}

func TestReactivateTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_react_err",
		Title:     "React Error",
		Status:    TaskStatusCancelled,
		CreatorID: "u_owner_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, _, err = svc.ReactivateTask(context.Background(), model.User{ID: "u_other_001"}, "task_react_err", "reactivate")
	if err != ErrForbiddenReactivate {
		t.Fatalf("expected ErrForbiddenReactivate, got %v", err)
	}
}

func TestDeleteTask_OwnerCanDeleteTaskAndRecords(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_del",
		Title:     "To Delete",
		Status:    TaskStatusOpen,
		CreatorID: "u_owner_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = recordRepo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_del",
		AuthorID:  "u_owner_001",
		Type:      model.TaskRecordTypeSubmit,
		Content:   "record content",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed record: %v", err)
	}

	err = svc.DeleteTask(context.Background(), model.User{ID: "u_owner_001"}, "task_del")
	if err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}

	// Verify task is gone
	_, err = repo.GetByID(context.Background(), "task_del")
	if err != repository.ErrTaskNotFound {
		t.Fatalf("expected task to be deleted, got error: %v", err)
	}

	// Verify records are gone
	records, err := recordRepo.ListByTaskID(context.Background(), "task_del")
	if err != nil {
		t.Fatalf("ListByTaskID returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected all records to be deleted, got %d", len(records))
	}
}

func TestDeleteTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_del_err",
		Title:     "Delete Error",
		Status:    TaskStatusOpen,
		CreatorID: "u_owner_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	err = svc.DeleteTask(context.Background(), model.User{ID: "u_other_001"}, "task_del_err")
	if err != ErrForbiddenDelete {
		t.Fatalf("expected ErrForbiddenDelete, got %v", err)
	}
}

func TestTaskService_AuditLogsAreCreatedThroughoutLifecycle(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	auditRepo := repository.NewMemoryAuditLogRepository()
	svc := NewTaskService(repo, recordRepo, auditRepo)

	creator := model.User{ID: "u_creator_1"}
	worker := model.User{ID: "u_worker_1"}

	// 1. Create Task -> should log task_created
	task, err := svc.CreateTask(context.Background(), creator, "Audit Test Task", "E2E audit logging verification")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	logs, err := svc.ListTaskAuditLogs(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("ListTaskAuditLogs: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != model.AuditActionCreated {
		t.Fatalf("expected 1 log with action %q, got %d logs: %+v", model.AuditActionCreated, len(logs), logs)
	}

	// 2. Assign Task -> should log task_assigned
	task, err = svc.AssignTask(context.Background(), creator, task.ID, worker.ID)
	if err != nil {
		t.Fatalf("AssignTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if len(logs) != 2 || logs[1].Action != model.AuditActionAssigned {
		t.Fatalf("expected log[1] to be %q, got %+v", model.AuditActionAssigned, logs)
	}

	// 3. Start Task -> should log task_started
	task, err = svc.StartTask(context.Background(), worker, task.ID)
	if err != nil {
		t.Fatalf("StartTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if len(logs) != 3 || logs[2].Action != model.AuditActionStarted {
		t.Fatalf("expected log[2] to be %q, got %+v", model.AuditActionStarted, logs)
	}

	// 4. Submit Task -> should log task_submitted
	task, _, err = svc.SubmitTask(context.Background(), worker, task.ID, "completed first pass")
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if len(logs) != 4 || logs[3].Action != model.AuditActionSubmitted {
		t.Fatalf("expected log[3] to be %q", model.AuditActionSubmitted)
	}

	// 5. Reject Task -> should log task_rejected
	task, _, err = svc.RejectTask(context.Background(), creator, task.ID, "need more docs")
	if err != nil {
		t.Fatalf("RejectTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if len(logs) != 5 || logs[4].Action != model.AuditActionRejected {
		t.Fatalf("expected log[4] to be %q", model.AuditActionRejected)
	}

	// 6. Start Again
	task, err = svc.StartTask(context.Background(), worker, task.ID)
	if err != nil {
		t.Fatalf("StartTask again: %v", err)
	}

	// 7. Submit Again
	task, _, err = svc.SubmitTask(context.Background(), worker, task.ID, "completed second pass")
	if err != nil {
		t.Fatalf("SubmitTask again: %v", err)
	}

	// 8. Approve Task -> should log task_approved
	task, _, err = svc.ApproveTask(context.Background(), creator, task.ID, "approved")
	if err != nil {
		t.Fatalf("ApproveTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if logs[7].Action != model.AuditActionApproved {
		t.Fatalf("expected log[7] to be %q", model.AuditActionApproved)
	}

	// 9. Close Task -> should log task_closed
	task, err = svc.CloseTask(context.Background(), creator, task.ID)
	if err != nil {
		t.Fatalf("CloseTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if logs[8].Action != model.AuditActionClosed {
		t.Fatalf("expected log[8] to be %q", model.AuditActionClosed)
	}

	// 10. Reactivate Task -> should log task_reopened
	task, _, err = svc.ReactivateTask(context.Background(), creator, task.ID, "needs minor bug fix")
	if err != nil {
		t.Fatalf("ReactivateTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if logs[9].Action != model.AuditActionReopened {
		t.Fatalf("expected log[9] to be %q", model.AuditActionReopened)
	}

	// 11. Cancel Task -> should log task_cancelled
	task, _, err = svc.CancelTask(context.Background(), creator, task.ID, "cancel it")
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	logs, _ = svc.ListTaskAuditLogs(context.Background(), task.ID)
	if logs[10].Action != model.AuditActionCancelled {
		t.Fatalf("expected log[10] to be %q", model.AuditActionCancelled)
	}

	// 12. Delete Task -> should cascade delete everything
	err = svc.DeleteTask(context.Background(), creator, task.ID)
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	// Verify audit logs are completely deleted
	records, err := auditRepo.ListByTaskID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("ListByTaskID: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 audit logs, got %d", len(records))
	}
}

type mockFailingAuditLogRepository struct {
	repository.AuditLogRepository
}

func (m *mockFailingAuditLogRepository) Create(ctx context.Context, log model.AuditLog) (model.AuditLog, error) {
	return model.AuditLog{}, errors.New("simulated audit log write failure")
}

func TestTaskService_TransactionRollbackOnFailure(t *testing.T) {
	uri := os.Getenv("TASKFLOW_MONGO_TEST_URI")
	if uri == "" {
		t.Skip("TASKFLOW_MONGO_TEST_URI not set; skipping transaction rollback integration test")
	}

	dbName := os.Getenv("TASKFLOW_MONGO_TEST_DATABASE")
	if dbName == "" {
		dbName = "taskflow_transaction_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		_ = client.Database(dbName).Collection("tasks").Drop(context.Background())
		_ = client.Database(dbName).Collection("audit_logs").Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}()

	_ = client.Database(dbName).Collection("tasks").Drop(ctx)
	_ = client.Database(dbName).Collection("audit_logs").Drop(ctx)

	mongoDB := client.Database(dbName)
	taskRepo := repository.NewMongoTaskRepository(mongoDB.Collection("tasks"))
	realAuditRepo := repository.NewMongoAuditLogRepository(mongoDB.Collection("audit_logs"))
	failingAuditRepo := &mockFailingAuditLogRepository{AuditLogRepository: realAuditRepo}
	recordRepo := repository.NewMongoTaskRecordRepository(mongoDB.Collection("task_records"))

	dbClient := &database.Client{Mongo: client, DBName: dbName}
	svc := NewTaskService(taskRepo, recordRepo, failingAuditRepo, dbClient)

	creator := model.User{ID: "u_creator_rollback"}

	// Attempt to create a task. The CreateTask flow does:
	// 1. taskRepo.Create
	// 2. auditRepo.Create (which will fail due to our mock)
	// Both should be rolled back!
	_, err = svc.CreateTask(ctx, creator, "Rollback Task", "Should not be persisted")
	if err == nil || !strings.Contains(err.Error(), "simulated audit log write failure") {
		t.Fatalf("expected simulated audit log write failure, got: %v", err)
	}

	// Verify that the task was NOT created in the database
	tasks, err := taskRepo.List(ctx)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("transaction rollback failed: expected 0 tasks, got %d", len(tasks))
	}
}
