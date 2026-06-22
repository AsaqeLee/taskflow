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
	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestStartTask_AssigneeCanMoveAssignedTaskToInProgress(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_001",
		"Start flow",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_worker_001",
		now,
		beforeUpdate,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.StartTask(context.Background(), model.User{ID: "u_worker_001"}, "task_001")
	if err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}

	if task.Status != domaintask.StatusInProgress.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusInProgress.String(), task.Status)
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_002",
		"Forbidden start",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.StartTask(context.Background(), model.User{ID: "u_other_001"}, "task_002")
	if err != ErrForbiddenStart {
		t.Fatalf("expected ErrForbiddenStart, got %v", err)
	}
}

func TestAssignTask_RejectsUnknownAssignee(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_assign_unknown",
		"Assign unknown",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.AssignTask(context.Background(), model.User{ID: "u_owner_001"}, "task_assign_unknown", "u_missing_001")
	if err != ErrAssigneeNotFound {
		t.Fatalf("expected ErrAssigneeNotFound, got %v", err)
	}
}

func TestAssignTask_RejectsEmptyAssigneeBeforeLookup(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_assign_empty",
		"Assign empty",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.AssignTask(context.Background(), model.User{ID: "u_owner_001"}, "task_assign_empty", "   ")
	if err != ErrEmptyAssigneeID {
		t.Fatalf("expected ErrEmptyAssigneeID, got %v", err)
	}
}

func TestAssignTask_RejectsUnauthorizedCallerBeforeAssigneeLookup(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_assign_forbidden",
		"Assign forbidden",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.AssignTask(context.Background(), model.User{ID: "u_other_001"}, "task_assign_forbidden", "u_missing_001")
	if err != ErrForbiddenAssign {
		t.Fatalf("expected ErrForbiddenAssign, got %v", err)
	}
}

func TestAssignTask_RejectsInactiveAssignee(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	userRepo := repository.NewMemoryUserRepository()
	seedTaskServiceUsers(t, userRepo)
	svc = NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository(), userRepo)

	now := time.Now().UTC()
	disabled := domainuser.Restore("u_disabled_assignee", "Disabled", domainuser.RoleHuman, "", "", true, nil, "", now, now)
	created, err := userRepo.Create(context.Background(), disabled)
	if err != nil {
		t.Fatalf("create assignee: %v", err)
	}
	if err := created.Disable(domainuser.NewActor("u_owner_001"), now); err != nil {
		t.Fatalf("disable assignee: %v", err)
	}
	if _, err := userRepo.Update(context.Background(), created); err != nil {
		t.Fatalf("persist disabled assignee: %v", err)
	}

	_, err = repo.Create(context.Background(), domaintask.Restore(
		"task_assign_inactive",
		"Assign inactive",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = svc.AssignTask(context.Background(), model.User{ID: "u_owner_001"}, "task_assign_inactive", "u_disabled_assignee")
	if err != ErrAssigneeInactive {
		t.Fatalf("expected ErrAssigneeInactive, got %v", err)
	}
}

func TestStartTask_RejectsNonAssignedStatus(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_003",
		"Wrong status",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_004",
		"Open task",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_010",
		"Submit flow",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_worker_001",
		now,
		beforeUpdate,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, record, err := svc.SubmitTask(context.Background(), model.User{ID: "u_worker_001"}, "task_010", "Implemented the task and attached proof")
	if err != nil {
		t.Fatalf("SubmitTask returned error: %v", err)
	}

	if task.Status != domaintask.StatusSubmitted.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusSubmitted.String(), task.Status)
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

func TestSubmitTaskWithMetadata_PersistsStructuredFeedback(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_010_meta",
		"Agent submit flow",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_agent_001",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, record, err := svc.SubmitTaskWithMetadata(
		context.Background(),
		model.User{ID: "u_agent_001", Role: "agent"},
		"task_010_meta",
		"Initial automation pass finished",
		map[string]string{
			"summary":         "Initial automation pass finished",
			"blocking_reason": "awaiting owner confirmation",
		},
	)
	if err != nil {
		t.Fatalf("SubmitTaskWithMetadata returned error: %v", err)
	}

	if task.Status != domaintask.StatusSubmitted.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusSubmitted.String(), task.Status)
	}
	if record.Metadata["summary"] != "Initial automation pass finished" {
		t.Fatalf("expected metadata summary to be persisted, got %+v", record.Metadata)
	}
	if record.Metadata["blocking_reason"] != "awaiting owner confirmation" {
		t.Fatalf("expected blocking_reason metadata to be persisted, got %+v", record.Metadata)
	}
}

func TestSubmitTask_RejectsNonAssignee(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_011",
		"Forbidden submit",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_012",
		"Wrong status",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_013",
		"Missing record content",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_020",
		"Reject flow",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		beforeUpdate,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, record, err := svc.RejectTask(context.Background(), model.User{ID: "u_owner_001"}, "task_020", "Please revise the missing edge cases")
	if err != nil {
		t.Fatalf("RejectTask returned error: %v", err)
	}

	if task.Status != domaintask.StatusAssigned.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusAssigned.String(), task.Status)
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_021",
		"Forbidden reject",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_022",
		"Wrong reject status",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_023",
		"Missing reject reason",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_030",
		"Approve flow",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		beforeUpdate,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, record, err := svc.ApproveTask(context.Background(), model.User{ID: "u_owner_001"}, "task_030", "Looks good, accepted")
	if err != nil {
		t.Fatalf("ApproveTask returned error: %v", err)
	}

	if task.Status != domaintask.StatusApproved.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusApproved.String(), task.Status)
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_031",
		"Forbidden approve",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_032",
		"Wrong approve status",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_033",
		"Missing approve content",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()
	beforeUpdate := now.Add(-time.Second)

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_040",
		"Close flow",
		"test",
		domaintask.StatusApproved,
		"u_owner_001",
		"u_worker_001",
		now,
		beforeUpdate,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task, err := svc.CloseTask(context.Background(), model.User{ID: "u_owner_001"}, "task_040")
	if err != nil {
		t.Fatalf("CloseTask returned error: %v", err)
	}

	if task.Status != domaintask.StatusCompleted.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusCompleted.String(), task.Status)
	}
	if !task.UpdatedAt.After(beforeUpdate) {
		t.Fatalf("expected updated_at to move forward")
	}
}

func TestCloseTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_041",
		"Forbidden close",
		"test",
		domaintask.StatusApproved,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_042",
		"Wrong close status",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_050",
		"List records",
		"test",
		domaintask.StatusSubmitted,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	first, err := recordRepo.Create(context.Background(), domainrecord.Restore(
		"",
		"task_050",
		"u_worker_001",
		domainrecord.TypeSubmit,
		"first",
		now.Add(-time.Minute),
	))
	if err != nil {
		t.Fatalf("seed first record: %v", err)
	}
	second, err := recordRepo.Create(context.Background(), domainrecord.Restore(
		"",
		"task_050",
		"u_owner_001",
		domainrecord.TypeApprove,
		"second",
		now,
	))
	if err != nil {
		t.Fatalf("seed second record: %v", err)
	}
	_, err = recordRepo.Create(context.Background(), domainrecord.Restore(
		"",
		"task_other",
		"u_other_001",
		domainrecord.TypeSubmit,
		"ignored",
		now.Add(time.Minute),
	))
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
	if records[0].ID != first.ID() || records[1].ID != second.ID() {
		t.Fatalf("expected ordered record ids [%s %s], got [%s %s]", first.ID(), second.ID(), records[0].ID, records[1].ID)
	}
}

func TestListTaskRecords_ReturnsNotFoundForUnknownTask(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())

	_, err := svc.ListTaskRecords(context.Background(), "missing")
	if err != repository.ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestCancelTask_OwnerCanCancelActiveTasks(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	states := []string{
		domaintask.StatusOpen.String(),
		domaintask.StatusAssigned.String(),
		domaintask.StatusInProgress.String(),
		domaintask.StatusSubmitted.String(),
	}
	for i, startStatus := range states {
		taskID := fmt.Sprintf("task_cancel_%d", i)
		status, err := domaintask.ParseStatus(startStatus)
		if err != nil {
			t.Fatalf("parse status: %v", err)
		}
		_, err = repo.Create(context.Background(), domaintask.Restore(
			taskID,
			"Active Task "+startStatus,
			"test",
			status,
			"u_owner_001",
			"",
			now,
			now,
			nil,
			"",
		))
		if err != nil {
			t.Fatalf("seed task: %v", err)
		}

		updatedTask, record, err := svc.CancelTask(context.Background(), model.User{ID: "u_owner_001"}, taskID, "Cancelling this task due to scope change")
		if err != nil {
			t.Fatalf("CancelTask from %s returned error: %v", startStatus, err)
		}

		if updatedTask.Status != domaintask.StatusCancelled.String() {
			t.Fatalf("expected status %q, got %q", domaintask.StatusCancelled.String(), updatedTask.Status)
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_cancel_err",
		"Cancel Error",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_cancel_comp",
		"Cancel Completed",
		"test",
		domaintask.StatusCompleted,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	// 1. Reactivate task with no assignee (should go to open)
	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_react_open",
		"React Open",
		"test",
		domaintask.StatusCancelled,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task1, rec1, err := svc.ReactivateTask(context.Background(), model.User{ID: "u_owner_001"}, "task_react_open", "reactivating open task")
	if err != nil {
		t.Fatalf("ReactivateTask open: %v", err)
	}
	if task1.Status != domaintask.StatusOpen.String() {
		t.Fatalf("expected status open, got %q", task1.Status)
	}
	if rec1.Type != model.TaskRecordTypeReactivate {
		t.Fatalf("expected record type reactivate, got %q", rec1.Type)
	}

	// 2. Reactivate task with assignee (should go to assigned)
	_, err = repo.Create(context.Background(), domaintask.Restore(
		"task_react_assign",
		"React Assign",
		"test",
		domaintask.StatusCompleted,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task2, rec2, err := svc.ReactivateTask(context.Background(), model.User{ID: "u_owner_001"}, "task_react_assign", "reactivating assigned task")
	if err != nil {
		t.Fatalf("ReactivateTask assigned: %v", err)
	}
	if task2.Status != domaintask.StatusAssigned.String() {
		t.Fatalf("expected status assigned, got %q", task2.Status)
	}
	if rec2.Content != "reactivating assigned task" {
		t.Fatalf("unexpected record content %q", rec2.Content)
	}
}

func TestReactivateTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_react_err",
		"React Error",
		"test",
		domaintask.StatusCancelled,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, _, err = svc.ReactivateTask(context.Background(), model.User{ID: "u_other_001"}, "task_react_err", "reactivate")
	if err != ErrForbiddenReactivate {
		t.Fatalf("expected ErrForbiddenReactivate, got %v", err)
	}
}

func TestDeleteTask_OwnerSoftDeletesTaskAndRetainsRecords(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := newMemoryTaskService(t, repo, recordRepo, repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_del",
		"To Delete",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = recordRepo.Create(context.Background(), domainrecord.Restore(
		"",
		"task_del",
		"u_owner_001",
		domainrecord.TypeSubmit,
		"record content",
		now,
	))
	if err != nil {
		t.Fatalf("seed record: %v", err)
	}

	err = svc.DeleteTask(context.Background(), model.User{ID: "u_owner_001"}, "task_del")
	if err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}

	// Verify task is hidden from active reads
	_, err = repo.GetByID(context.Background(), "task_del")
	if err != repository.ErrTaskNotFound {
		t.Fatalf("expected task to be deleted, got error: %v", err)
	}

	// Verify task is still retained internally
	deletedTask, err := repo.GetByIDIncludingDeleted(context.Background(), "task_del")
	if err != nil {
		t.Fatalf("GetByIDIncludingDeleted returned error: %v", err)
	}
	if deletedTask.DeletedAt() == nil {
		t.Fatalf("expected deleted_at to be populated")
	}
	if deletedTask.DeletedBy() != "u_owner_001" {
		t.Fatalf("expected deleted_by to be populated, got %q", deletedTask.DeletedBy())
	}
	if deletedTask.Status() != domaintask.StatusDeleted {
		t.Fatalf("expected deleted status, got %q", deletedTask.Status())
	}

	// Verify records are retained for auditability
	records, err := recordRepo.ListByTaskID(context.Background(), "task_del")
	if err != nil {
		t.Fatalf("ListByTaskID returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected records to be retained, got %d", len(records))
	}
}

func TestDeleteTask_RejectsNonOwner(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	svc := newMemoryTaskService(t, repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_del_err",
		"Delete Error",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
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
	svc := newMemoryTaskService(t, repo, recordRepo, auditRepo)

	creator := model.User{ID: "u_owner_001"}
	worker := model.User{ID: "u_worker_001"}

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

	// 12. Delete Task -> should retain audit trail and append task_deleted
	err = svc.DeleteTask(context.Background(), creator, task.ID)
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	// Verify audit logs are retained
	records, err := auditRepo.ListByTaskID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("ListByTaskID: %v", err)
	}
	if len(records) != 12 {
		t.Fatalf("expected 12 audit logs, got %d", len(records))
	}
	if records[11].Action().String() != model.AuditActionDeleted {
		t.Fatalf("expected final audit action %q, got %q", model.AuditActionDeleted, records[11].Action().String())
	}
}

type mockFailingAuditLogRepository struct {
	repository.AuditLogRepository
}

func (m *mockFailingAuditLogRepository) Create(ctx context.Context, log domainaudit.Log) (domainaudit.Log, error) {
	return domainaudit.Log{}, errors.New("simulated audit log write failure")
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

	userRepo := repository.NewMongoUserRepository(mongoDB.Collection("users"))
	seedTaskServiceUsers(t, userRepo)

	dbClient := &database.Client{Mongo: client, DBName: dbName}
	svc := NewTaskService(taskRepo, recordRepo, failingAuditRepo, userRepo, dbClient)

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
