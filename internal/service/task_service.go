package service

import (
	"errors"
	"strings"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
)

const TaskStatusOpen = "open"
const TaskStatusAssigned = "assigned"
const TaskStatusInProgress = "in_progress"
const TaskStatusSubmitted = "submitted"
const TaskStatusApproved = "approved"
const TaskStatusCompleted = "completed"
const TaskStatusCancelled = "cancelled"

var ErrInvalidTaskID = errors.New("task id is required")
var ErrEmptyTaskTitle = errors.New("task title is required")
var ErrTooShortTaskTitle = errors.New("task title must be at least 3 characters")
var ErrEmptyAssigneeID = errors.New("assignee id is required")
var ErrEmptyTaskRecordContent = errors.New("task record content is required")
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
}

func NewTaskService(repo repository.TaskRepository, recordRepo repository.TaskRecordRepository, auditRepo repository.AuditLogRepository) *TaskService {
	return &TaskService{repo: repo, recordRepo: recordRepo, auditRepo: auditRepo}
}

func (s *TaskService) CreateTask(currentUser model.User, title, description string) (model.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Task{}, ErrEmptyTaskTitle
	}
	if len(title) < 3 {
		return model.Task{}, ErrTooShortTaskTitle
	}

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

	createdTask, err := s.repo.Create(task)
	if err != nil {
		return model.Task{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    createdTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionCreated,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, err
	}

	return createdTask, nil
}

func (s *TaskService) GetTask(id string) (model.Task, error) {
	if strings.TrimSpace(id) == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	return s.repo.GetByID(id)
}

func (s *TaskService) ListTasks() ([]model.Task, error) {
	return s.repo.List()
}

func (s *TaskService) ListTaskRecords(taskID string) ([]model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	if _, err := s.repo.GetByID(taskID); err != nil {
		return nil, err
	}

	return s.recordRepo.ListByTaskID(taskID)
}

func (s *TaskService) UpdateTaskBasic(id, title, description string) (model.Task, error) {
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

	task, err := s.repo.GetByID(id)
	if err != nil {
		return model.Task{}, err
	}

	task.Title = title
	task.Description = strings.TrimSpace(description)
	task.UpdatedAt = time.Now().UTC()

	return s.repo.Update(task)
}

func (s *TaskService) AssignTask(currentUser model.User, taskID, assigneeID string) (model.Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return model.Task{}, ErrEmptyAssigneeID
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, err
	}

	if task.CreatorID != currentUser.ID {
		return model.Task{}, ErrForbiddenAssign
	}

	if task.Status != TaskStatusOpen {
		return model.Task{}, ErrInvalidTaskStatusForAssign
	}

	task.AssigneeID = assigneeID
	task.Status = TaskStatusAssigned
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionAssigned,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, err
	}

	return updatedTask, nil
}

func (s *TaskService) StartTask(currentUser model.User, taskID string) (model.Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, err
	}

	if task.Status != TaskStatusAssigned {
		return model.Task{}, ErrInvalidTaskStatusForStart
	}

	if task.AssigneeID != currentUser.ID {
		return model.Task{}, ErrForbiddenStart
	}

	task.Status = TaskStatusInProgress
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionStarted,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, err
	}

	return updatedTask, nil
}

func (s *TaskService) SubmitTask(currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	if task.Status != TaskStatusInProgress {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskStatusForSubmit
	}

	if task.AssigneeID != currentUser.ID {
		return model.Task{}, model.TaskRecord{}, ErrForbiddenSubmit
	}

	task.Status = TaskStatusSubmitted
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	record, err := s.recordRepo.Create(model.TaskRecord{
		TaskID:    task.ID,
		AuthorID:  currentUser.ID,
		Type:      model.TaskRecordTypeSubmit,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionSubmitted,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	return updatedTask, record, nil
}

func (s *TaskService) RejectTask(currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	if task.Status != TaskStatusSubmitted {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskStatusForReject
	}

	if task.CreatorID != currentUser.ID {
		return model.Task{}, model.TaskRecord{}, ErrForbiddenReject
	}

	task.Status = TaskStatusAssigned
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	record, err := s.recordRepo.Create(model.TaskRecord{
		TaskID:    task.ID,
		AuthorID:  currentUser.ID,
		Type:      model.TaskRecordTypeReject,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionRejected,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	return updatedTask, record, nil
}

func (s *TaskService) ApproveTask(currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	if task.Status != TaskStatusSubmitted {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskStatusForApprove
	}

	if task.CreatorID != currentUser.ID {
		return model.Task{}, model.TaskRecord{}, ErrForbiddenApprove
	}

	task.Status = TaskStatusApproved
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	record, err := s.recordRepo.Create(model.TaskRecord{
		TaskID:    task.ID,
		AuthorID:  currentUser.ID,
		Type:      model.TaskRecordTypeApprove,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionApproved,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	return updatedTask, record, nil
}

func (s *TaskService) CloseTask(currentUser model.User, taskID string) (model.Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, err
	}

	if task.Status != TaskStatusApproved {
		return model.Task{}, ErrInvalidTaskStatusForClose
	}

	if task.CreatorID != currentUser.ID {
		return model.Task{}, ErrForbiddenClose
	}

	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionClosed,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, err
	}

	return updatedTask, nil
}

func (s *TaskService) CancelTask(currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	if task.Status != TaskStatusOpen && task.Status != TaskStatusAssigned &&
		task.Status != TaskStatusInProgress && task.Status != TaskStatusSubmitted {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskStatusForCancel
	}

	if task.CreatorID != currentUser.ID {
		return model.Task{}, model.TaskRecord{}, ErrForbiddenCancel
	}

	task.Status = TaskStatusCancelled
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	record, err := s.recordRepo.Create(model.TaskRecord{
		TaskID:    task.ID,
		AuthorID:  currentUser.ID,
		Type:      model.TaskRecordTypeCancel,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionCancelled,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	return updatedTask, record, nil
}

func (s *TaskService) ReactivateTask(currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return model.Task{}, model.TaskRecord{}, ErrEmptyTaskRecordContent
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	if task.Status != TaskStatusCancelled && task.Status != TaskStatusCompleted {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskStatusForReactivate
	}

	if task.CreatorID != currentUser.ID {
		return model.Task{}, model.TaskRecord{}, ErrForbiddenReactivate
	}

	if task.AssigneeID != "" {
		task.Status = TaskStatusAssigned
	} else {
		task.Status = TaskStatusOpen
	}
	task.UpdatedAt = time.Now().UTC()

	updatedTask, err := s.repo.Update(task)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	record, err := s.recordRepo.Create(model.TaskRecord{
		TaskID:    task.ID,
		AuthorID:  currentUser.ID,
		Type:      model.TaskRecordTypeReactivate,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	_, err = s.auditRepo.Create(model.AuditLog{
		TaskID:    updatedTask.ID,
		ActorID:   currentUser.ID,
		Action:    model.AuditActionReopened,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}

	return updatedTask, record, nil
}

func (s *TaskService) DeleteTask(currentUser model.User, taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ErrInvalidTaskID
	}

	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return err
	}

	if task.CreatorID != currentUser.ID {
		return ErrForbiddenDelete
	}

	if err := s.repo.Delete(taskID); err != nil {
		return err
	}

	if err := s.recordRepo.DeleteByTaskID(taskID); err != nil {
		return err
	}

	return s.auditRepo.DeleteByTaskID(taskID)
}

func (s *TaskService) ListTaskAuditLogs(taskID string) ([]model.AuditLog, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	if _, err := s.repo.GetByID(taskID); err != nil {
		return nil, err
	}

	return s.auditRepo.ListByTaskID(taskID)
}
