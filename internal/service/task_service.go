package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/requestmeta"
)

const TaskStatusOpen = "open"
const TaskStatusAssigned = "assigned"
const TaskStatusInProgress = "in_progress"
const TaskStatusSubmitted = "submitted"
const TaskStatusApproved = "approved"
const TaskStatusCompleted = "completed"
const TaskStatusCancelled = "cancelled"
const TaskStatusDeleted = "deleted"

var ErrInvalidTaskID = errors.New("task id is required")
var ErrEmptyTaskTitle = errors.New("task title is required")
var ErrTooShortTaskTitle = errors.New("task title must be at least 3 characters")
var ErrEmptyAssigneeID = errors.New("assignee id is required")
var ErrEmptyTaskRecordContent = errors.New("task record content is required")
var ErrForbiddenUpdate = errors.New("current user cannot update task")
var ErrForbiddenAssign = errors.New("current user cannot assign task")
var ErrInvalidTaskStatusForAssign = errors.New("task status does not allow assign")
var ErrForbiddenStart = errors.New("current user cannot start task")
var ErrInvalidTaskStatusForStart = errors.New("task status does not allow start")
var ErrForbiddenSubmit = errors.New("current user cannot submit task")
var ErrInvalidTaskStatusForSubmit = errors.New("task status does not allow submit")
var ErrForbiddenReject = errors.New("current user cannot reject task")
var ErrInvalidTaskStatusForReject = errors.New("task status does not allow reject")
var ErrForbiddenApprove = errors.New("current user cannot approve task")
var ErrInvalidTaskStatusForApprove = errors.New("task status does not allow approve")
var ErrForbiddenClose = errors.New("current user cannot close task")
var ErrInvalidTaskStatusForClose = errors.New("task status does not allow close")
var ErrForbiddenCancel = errors.New("current user cannot cancel task")
var ErrInvalidTaskStatusForCancel = errors.New("task status does not allow cancel")
var ErrForbiddenReactivate = errors.New("current user cannot reactivate task")
var ErrInvalidTaskStatusForReactivate = errors.New("task status does not allow reactivate")
var ErrForbiddenDelete = errors.New("current user cannot delete task")

type TaskService struct {
	repo       repository.TaskRepository
	recordRepo repository.TaskRecordRepository
	auditRepo  repository.AuditLogRepository
	dbClient   *database.Client
}

func NewTaskService(
	repo repository.TaskRepository,
	recordRepo repository.TaskRecordRepository,
	auditRepo repository.AuditLogRepository,
	dbClient ...*database.Client,
) *TaskService {
	var db *database.Client
	if len(dbClient) > 0 {
		db = dbClient[0]
	}
	return &TaskService{
		repo:       repo,
		recordRepo: recordRepo,
		auditRepo:  auditRepo,
		dbClient:   db,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, currentUser model.User, title, description string) (model.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Task{}, ErrEmptyTaskTitle
	}
	if len(title) < 3 {
		return model.Task{}, ErrTooShortTaskTitle
	}

	var createdTask model.Task
	var err error

	runOps := func(txCtx context.Context) error {
		now := time.Now().UTC()
		task := model.Task{
			Title:       title,
			Description: strings.TrimSpace(description),
			Status:      TaskStatusOpen,
			CreatorID:   currentUser.ID,
			AssigneeID:  "",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		createdTask, err = s.repo.Create(txCtx, task)
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, createdTask.ID, currentUser.ID, model.AuditActionCreated, "", createdTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, err
	}
	return createdTask, nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (model.Task, error) {
	if strings.TrimSpace(id) == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	return s.repo.GetByID(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context) ([]model.Task, error) {
	return s.repo.List(ctx)
}

func (s *TaskService) ListTaskRecords(ctx context.Context, taskID string) ([]model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	if _, err := s.repo.GetByIDIncludingDeleted(ctx, taskID); err != nil {
		return nil, err
	}

	return s.recordRepo.ListByTaskID(ctx, taskID)
}

func (s *TaskService) UpdateTaskBasic(ctx context.Context, currentUser model.User, id, title, description string) (model.Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return model.Task{}, ErrEmptyTaskTitle
	}
	if len(title) < 3 {
		return model.Task{}, ErrTooShortTaskTitle
	}

	var updatedTask model.Task
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if task.CreatorID != currentUser.ID {
			return ErrForbiddenUpdate
		}

		task.Title = title
		task.Description = strings.TrimSpace(description)
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		return err
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, err
	}
	return updatedTask, nil
}

func (s *TaskService) AssignTask(ctx context.Context, currentUser model.User, taskID, assigneeID string) (model.Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return model.Task{}, ErrEmptyAssigneeID
	}

	var updatedTask model.Task
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.CreatorID != currentUser.ID {
			return ErrForbiddenAssign
		}

		if task.Status != TaskStatusOpen {
			return ErrInvalidTaskStatusForAssign
		}

		task.AssigneeID = assigneeID
		task.Status = TaskStatusAssigned
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionAssigned, TaskStatusOpen, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, err
	}
	return updatedTask, nil
}

func (s *TaskService) StartTask(ctx context.Context, currentUser model.User, taskID string) (model.Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	var updatedTask model.Task
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.Status != TaskStatusAssigned {
			return ErrInvalidTaskStatusForStart
		}

		if task.AssigneeID != currentUser.ID {
			return ErrForbiddenStart
		}

		task.Status = TaskStatusInProgress
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionStarted, TaskStatusAssigned, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, err
	}
	return updatedTask, nil
}

func (s *TaskService) SubmitTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	var updatedTask model.Task
	var record model.TaskRecord
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.Status != TaskStatusInProgress {
			return ErrInvalidTaskStatusForSubmit
		}

		if task.AssigneeID != currentUser.ID {
			return ErrForbiddenSubmit
		}

		task.Status = TaskStatusSubmitted
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		record, err = s.recordRepo.Create(txCtx, model.TaskRecord{
			TaskID:    task.ID,
			AuthorID:  currentUser.ID,
			Type:      model.TaskRecordTypeSubmit,
			Content:   content,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionSubmitted, TaskStatusInProgress, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return updatedTask, record, nil
}

func (s *TaskService) RejectTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	var updatedTask model.Task
	var record model.TaskRecord
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.Status != TaskStatusSubmitted {
			return ErrInvalidTaskStatusForReject
		}

		if task.CreatorID != currentUser.ID {
			return ErrForbiddenReject
		}

		task.Status = TaskStatusAssigned
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		record, err = s.recordRepo.Create(txCtx, model.TaskRecord{
			TaskID:    task.ID,
			AuthorID:  currentUser.ID,
			Type:      model.TaskRecordTypeReject,
			Content:   content,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionRejected, TaskStatusSubmitted, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return updatedTask, record, nil
}

func (s *TaskService) ApproveTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	var updatedTask model.Task
	var record model.TaskRecord
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.Status != TaskStatusSubmitted {
			return ErrInvalidTaskStatusForApprove
		}

		if task.CreatorID != currentUser.ID {
			return ErrForbiddenApprove
		}

		task.Status = TaskStatusApproved
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		record, err = s.recordRepo.Create(txCtx, model.TaskRecord{
			TaskID:    task.ID,
			AuthorID:  currentUser.ID,
			Type:      model.TaskRecordTypeApprove,
			Content:   content,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionApproved, TaskStatusSubmitted, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return updatedTask, record, nil
}

func (s *TaskService) CloseTask(ctx context.Context, currentUser model.User, taskID string) (model.Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	var updatedTask model.Task
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.Status != TaskStatusApproved {
			return ErrInvalidTaskStatusForClose
		}

		if task.CreatorID != currentUser.ID {
			return ErrForbiddenClose
		}

		task.Status = TaskStatusCompleted
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionClosed, TaskStatusApproved, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, err
	}
	return updatedTask, nil
}

func (s *TaskService) CancelTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	var updatedTask model.Task
	var record model.TaskRecord
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.Status != TaskStatusOpen && task.Status != TaskStatusAssigned &&
			task.Status != TaskStatusInProgress && task.Status != TaskStatusSubmitted {
			return ErrInvalidTaskStatusForCancel
		}

		if task.CreatorID != currentUser.ID {
			return ErrForbiddenCancel
		}

		task.Status = TaskStatusCancelled
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		record, err = s.recordRepo.Create(txCtx, model.TaskRecord{
			TaskID:    task.ID,
			AuthorID:  currentUser.ID,
			Type:      model.TaskRecordTypeCancel,
			Content:   content,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionCancelled, task.Status, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return updatedTask, record, nil
}

func (s *TaskService) ReactivateTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	var updatedTask model.Task
	var record model.TaskRecord
	var err error

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.Status != TaskStatusCancelled && task.Status != TaskStatusCompleted {
			return ErrInvalidTaskStatusForReactivate
		}

		if task.CreatorID != currentUser.ID {
			return ErrForbiddenReactivate
		}

		if task.AssigneeID != "" {
			task.Status = TaskStatusAssigned
		} else {
			task.Status = TaskStatusOpen
		}
		task.UpdatedAt = time.Now().UTC()

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		record, err = s.recordRepo.Create(txCtx, model.TaskRecord{
			TaskID:    task.ID,
			AuthorID:  currentUser.ID,
			Type:      model.TaskRecordTypeReactivate,
			Content:   content,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		return s.createAuditLog(txCtx, updatedTask.ID, currentUser.ID, model.AuditActionReopened, task.Status, updatedTask.Status)
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return updatedTask, record, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, currentUser model.User, taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ErrInvalidTaskID
	}

	runOps := func(txCtx context.Context) error {
		task, err := s.repo.GetByID(txCtx, taskID)
		if err != nil {
			return err
		}

		if task.CreatorID != currentUser.ID {
			return ErrForbiddenDelete
		}

		previousStatus := task.Status
		now := time.Now().UTC()
		task.Status = TaskStatusDeleted
		task.DeletedAt = &now
		task.DeletedBy = currentUser.ID
		task.UpdatedAt = now

		if _, err := s.repo.Update(txCtx, task); err != nil {
			return err
		}

		return s.createAuditLog(txCtx, taskID, currentUser.ID, model.AuditActionDeleted, previousStatus, TaskStatusDeleted)
	}

	if s.dbClient != nil {
		return s.dbClient.RunTransaction(ctx, runOps)
	}
	return runOps(ctx)
}

func (s *TaskService) ListTaskAuditLogs(ctx context.Context, taskID string) ([]model.AuditLog, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	if _, err := s.repo.GetByIDIncludingDeleted(ctx, taskID); err != nil {
		return nil, err
	}

	return s.auditRepo.ListByTaskID(ctx, taskID)
}

func (s *TaskService) createAuditLog(ctx context.Context, taskID, actorID, action, fromStatus, toStatus string) error {
	meta := requestmeta.FromContext(ctx)
	_, err := s.auditRepo.Create(ctx, model.AuditLog{
		TaskID:         taskID,
		ActorID:        actorID,
		Action:         action,
		RequestID:      meta.RequestID,
		TraceID:        meta.TraceID,
		IdempotencyKey: meta.IdempotencyKey,
		SourceIP:       meta.SourceIP,
		UserAgent:      meta.UserAgent,
		FromStatus:     fromStatus,
		ToStatus:       toStatus,
		CreatedAt:      time.Now().UTC(),
	})
	return err
}
