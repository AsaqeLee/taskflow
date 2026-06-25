package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/domain"
	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/model"
)

var (
	ErrInvalidTaskID                  = domain.ErrInvalidTaskID
	ErrEmptyTaskTitle                 = domain.ErrEmptyTaskTitle
	ErrTooShortTaskTitle              = domain.ErrTooShortTaskTitle
	ErrEmptyAssigneeID                = domain.ErrEmptyAssigneeID
	ErrEmptyTaskRecordContent         = domain.ErrEmptyTaskRecordContent
	ErrForbiddenRead                  = domain.ErrForbiddenRead
	ErrForbiddenUpdate                = domain.ErrForbiddenUpdate
	ErrForbiddenAssign                = domain.ErrForbiddenAssign
	ErrInvalidTaskStatusForAssign     = domain.ErrInvalidTaskStatusForAssign
	ErrForbiddenStart                 = domain.ErrForbiddenStart
	ErrInvalidTaskStatusForStart      = domain.ErrInvalidTaskStatusForStart
	ErrForbiddenSubmit                = domain.ErrForbiddenSubmit
	ErrInvalidTaskStatusForSubmit     = domain.ErrInvalidTaskStatusForSubmit
	ErrForbiddenReject                = domain.ErrForbiddenReject
	ErrInvalidTaskStatusForReject     = domain.ErrInvalidTaskStatusForReject
	ErrForbiddenApprove               = domain.ErrForbiddenApprove
	ErrInvalidTaskStatusForApprove    = domain.ErrInvalidTaskStatusForApprove
	ErrForbiddenClose                 = domain.ErrForbiddenClose
	ErrInvalidTaskStatusForClose      = domain.ErrInvalidTaskStatusForClose
	ErrForbiddenCancel                = domain.ErrForbiddenCancel
	ErrInvalidTaskStatusForCancel     = domain.ErrInvalidTaskStatusForCancel
	ErrForbiddenReactivate            = domain.ErrForbiddenReactivate
	ErrInvalidTaskStatusForReactivate = domain.ErrInvalidTaskStatusForReactivate
	ErrForbiddenDelete                = domain.ErrForbiddenDelete
	ErrInvalidTaskStatus              = domain.ErrInvalidTaskStatus
	ErrAssigneeNotFound               = domain.ErrAssigneeNotFound
	ErrAssigneeInactive               = domain.ErrAssigneeInactive
	ErrTaskNotFound                   = ports.ErrTaskNotFound
)

type TaskService struct {
	repo         ports.TaskRepository
	recordRepo   ports.TaskRecordRepository
	auditRepo    ports.AuditLogRepository
	userRepo     ports.UserRepository
	eventApplier taskEventApplier
	dbClient     *database.Client
}

const (
	taskActionAssign     = "assign"
	taskActionStart      = "start"
	taskActionSubmit     = "submit"
	taskActionReject     = "reject"
	taskActionApprove    = "approve"
	taskActionClose      = "close"
	taskActionCancel     = "cancel"
	taskActionReactivate = "reactivate"
	taskActionDelete     = "delete"
)

func NewTaskService(
	repo ports.TaskRepository,
	recordRepo ports.TaskRecordRepository,
	auditRepo ports.AuditLogRepository,
	userRepo ports.UserRepository,
	dbClient ...*database.Client,
) *TaskService {
	var db *database.Client
	if len(dbClient) > 0 {
		db = dbClient[0]
	}
	return &TaskService{
		repo:         repo,
		recordRepo:   recordRepo,
		auditRepo:    auditRepo,
		userRepo:     userRepo,
		eventApplier: newTaskEventApplier(recordRepo, auditRepo),
		dbClient:     db,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, currentUser model.User, title, description string) (model.Task, error) {
	actor := domainuser.NewActor(currentUser.ID)
	now := time.Now().UTC()

	var createdTask domaintask.Task
	var transition domaintask.Transition
	var err error

	runOps := func(txCtx context.Context) error {
		task, change, createErr := domaintask.Create(actor.ID, title, description, now)
		if createErr != nil {
			return createErr
		}

		createdTask, err = s.repo.Create(txCtx, task)
		if err != nil {
			return err
		}
		transition = change.Bind(createdTask.ID(), actor.ID)
		_, _, err = s.eventApplier.apply(txCtx, transition)
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
	return taskViewForUser(createdTask, currentUser), nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (model.Task, error) {
	if strings.TrimSpace(id) == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	return model.TaskFromDomain(task), nil
}

func (s *TaskService) GetTaskForUser(ctx context.Context, currentUser model.User, id string) (model.Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if !canReadTask(task, currentUser) {
		return model.Task{}, ErrForbiddenRead
	}
	return taskViewForUser(task, currentUser), nil
}

func (s *TaskService) ListTasks(ctx context.Context) ([]model.Task, error) {
	tasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return model.TasksFromDomain(tasks), nil
}

func (s *TaskService) ListTasksForUser(ctx context.Context, currentUser model.User) ([]model.Task, error) {
	if isOwner(currentUser) {
		tasks, err := s.repo.List(ctx)
		if err != nil {
			return nil, err
		}
		return model.TasksFromDomain(tasks), nil
	}

	tasks, err := s.repo.ListVisibleToUser(ctx, currentUser.ID)
	if err != nil {
		return nil, err
	}
	return taskViewsForUser(tasks, currentUser), nil
}

func (s *TaskService) ListTaskRecords(ctx context.Context, taskID string) ([]model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	if _, err := s.repo.GetByIDIncludingDeleted(ctx, taskID); err != nil {
		return nil, err
	}

	records, err := s.recordRepo.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return model.TaskRecordsFromDomain(records), nil
}

func (s *TaskService) ListTaskRecordsForUser(ctx context.Context, currentUser model.User, taskID string) ([]model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	task, err := s.repo.GetByIDIncludingDeleted(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !canReadTask(task, currentUser) {
		return nil, ErrForbiddenRead
	}

	records, err := s.recordRepo.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return model.TaskRecordsFromDomain(records), nil
}

func (s *TaskService) UpdateTaskBasic(ctx context.Context, currentUser model.User, id, title, description string) (model.Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.Task{}, ErrInvalidTaskID
	}

	actor := domainuser.NewActor(currentUser.ID)
	now := time.Now().UTC()
	var updatedTask domaintask.Task
	var err error

	runOps := func(txCtx context.Context) error {
		task, loadErr := s.repo.GetByID(txCtx, id)
		if loadErr != nil {
			return loadErr
		}
		if updateErr := task.UpdateBasic(actor, title, description, now); updateErr != nil {
			return updateErr
		}
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
	return taskViewForUser(updatedTask, currentUser), nil
}

func (s *TaskService) AssignTask(ctx context.Context, currentUser model.User, taskID, assigneeID string) (model.Task, error) {
	return s.runTransition(ctx, taskID, func(txCtx context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		change, err := task.Assign(actor, assigneeID, at)
		if err != nil {
			return domaintask.Transition{}, err
		}
		if err := s.ensureAssignableUser(txCtx, assigneeID); err != nil {
			return domaintask.Transition{}, err
		}
		return change, nil
	}, currentUser)
}

func (s *TaskService) ensureAssignableUser(ctx context.Context, assigneeID string) error {
	assignee, err := s.userRepo.FindByID(ctx, assigneeID)
	if err != nil {
		if errors.Is(err, ports.ErrUserNotFound) {
			return domain.ErrAssigneeNotFound
		}
		return err
	}
	if !assignee.IsActive() {
		return domain.ErrAssigneeInactive
	}
	return nil
}

func (s *TaskService) StartTask(ctx context.Context, currentUser model.User, taskID string) (model.Task, error) {
	return s.runTransition(ctx, taskID, func(_ context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		return task.Start(actor, at)
	}, currentUser)
}

func (s *TaskService) SubmitTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	return s.SubmitTaskWithMetadata(ctx, currentUser, taskID, content, nil)
}

func (s *TaskService) SubmitTaskWithMetadata(
	ctx context.Context,
	currentUser model.User,
	taskID,
	content string,
	metadata map[string]string,
) (model.Task, model.TaskRecord, error) {
	task, record, err := s.runTransitionWithRecord(ctx, taskID, func(_ context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		change, err := task.Submit(actor, content, at)
		if err != nil {
			return domaintask.Transition{}, err
		}
		return change.WithRecordMetadata(metadata), nil
	}, currentUser)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return task, record, nil
}

func (s *TaskService) RejectTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	return s.RejectTaskWithMetadata(ctx, currentUser, taskID, content, nil)
}

func (s *TaskService) RejectTaskWithMetadata(
	ctx context.Context,
	currentUser model.User,
	taskID,
	content string,
	metadata map[string]string,
) (model.Task, model.TaskRecord, error) {
	task, record, err := s.runTransitionWithRecord(ctx, taskID, func(_ context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		change, err := task.Reject(actor, content, at)
		if err != nil {
			return domaintask.Transition{}, err
		}
		return change.WithRecordMetadata(metadata), nil
	}, currentUser)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return task, record, nil
}

func (s *TaskService) ApproveTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	return s.ApproveTaskWithMetadata(ctx, currentUser, taskID, content, nil)
}

func (s *TaskService) ApproveTaskWithMetadata(
	ctx context.Context,
	currentUser model.User,
	taskID,
	content string,
	metadata map[string]string,
) (model.Task, model.TaskRecord, error) {
	task, record, err := s.runTransitionWithRecord(ctx, taskID, func(_ context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		change, err := task.Approve(actor, content, at)
		if err != nil {
			return domaintask.Transition{}, err
		}
		return change.WithRecordMetadata(metadata), nil
	}, currentUser)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return task, record, nil
}

func (s *TaskService) CloseTask(ctx context.Context, currentUser model.User, taskID string) (model.Task, error) {
	return s.runTransition(ctx, taskID, func(_ context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		return task.Close(actor, at)
	}, currentUser)
}

func (s *TaskService) CancelTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	task, record, err := s.runTransitionWithRecord(ctx, taskID, func(_ context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		return task.Cancel(actor, content, at)
	}, currentUser)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return task, record, nil
}

func (s *TaskService) ReactivateTask(ctx context.Context, currentUser model.User, taskID, content string) (model.Task, model.TaskRecord, error) {
	task, record, err := s.runTransitionWithRecord(ctx, taskID, func(_ context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error) {
		return task.Reactivate(actor, content, at)
	}, currentUser)
	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	return task, record, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, currentUser model.User, taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ErrInvalidTaskID
	}

	actor := domainuser.NewActor(currentUser.ID)
	now := time.Now().UTC()
	var err error

	runOps := func(txCtx context.Context) error {
		task, loadErr := s.repo.GetByID(txCtx, taskID)
		if loadErr != nil {
			return loadErr
		}

		transition, deleteErr := task.MarkDeleted(actor, now)
		if deleteErr != nil {
			return deleteErr
		}

		if _, err = s.repo.Update(txCtx, task); err != nil {
			return err
		}
		_, _, err = s.eventApplier.apply(txCtx, transition.Bind(task.ID(), actor.ID))
		return err
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

	logs, err := s.auditRepo.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return model.AuditLogsFromDomain(logs), nil
}

func (s *TaskService) ListTaskAuditLogsForUser(ctx context.Context, currentUser model.User, taskID string) ([]model.AuditLog, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	task, err := s.repo.GetByIDIncludingDeleted(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !canReadTask(task, currentUser) {
		return nil, ErrForbiddenRead
	}

	logs, err := s.auditRepo.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return model.AuditLogsFromDomain(logs), nil
}

type transitionFunc func(ctx context.Context, task *domaintask.Task, actor domainuser.Actor, at time.Time) (domaintask.Transition, error)

func (s *TaskService) runTransition(
	ctx context.Context,
	taskID string,
	transition transitionFunc,
	currentUser model.User,
) (model.Task, error) {
	task, _, err := s.runTransitionWithRecord(ctx, taskID, transition, currentUser)
	return task, err
}

func (s *TaskService) runTransitionWithRecord(
	ctx context.Context,
	taskID string,
	transition transitionFunc,
	currentUser model.User,
) (model.Task, model.TaskRecord, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return model.Task{}, model.TaskRecord{}, ErrInvalidTaskID
	}

	actor := domainuser.NewActor(currentUser.ID)
	now := time.Now().UTC()

	var updatedTask domaintask.Task
	var savedRecord domainrecord.Record
	var hasRecord bool
	var err error

	runOps := func(txCtx context.Context) error {
		task, loadErr := s.repo.GetByID(txCtx, taskID)
		if loadErr != nil {
			return loadErr
		}

		change, transitionErr := transition(txCtx, &task, actor, now)
		if transitionErr != nil {
			return transitionErr
		}

		updatedTask, err = s.repo.Update(txCtx, task)
		if err != nil {
			return err
		}

		savedRecord, hasRecord, err = s.eventApplier.apply(txCtx, change.Bind(updatedTask.ID(), actor.ID))
		return err
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if err != nil {
		return model.Task{}, model.TaskRecord{}, err
	}
	if hasRecord {
		return taskViewForUser(updatedTask, currentUser), model.TaskRecordFromDomain(savedRecord), nil
	}
	return taskViewForUser(updatedTask, currentUser), model.TaskRecord{}, nil
}

func canReadTask(task domaintask.Task, currentUser model.User) bool {
	if isOwner(currentUser) {
		return true
	}
	return task.CreatorID() == currentUser.ID || task.AssigneeID() == currentUser.ID
}

func isOwner(currentUser model.User) bool {
	role, err := domainuser.ParseRole(currentUser.Role)
	return err == nil && role.IsOwner()
}

func taskViewForUser(task domaintask.Task, currentUser model.User) model.Task {
	view := model.TaskFromDomain(task)
	view.AvailableActions = availableActionsForUser(task, currentUser)
	return view
}

func taskViewsForUser(tasks []domaintask.Task, currentUser model.User) []model.Task {
	result := make([]model.Task, len(tasks))
	for i, task := range tasks {
		result[i] = taskViewForUser(task, currentUser)
	}
	return result
}

func availableActionsForUser(task domaintask.Task, currentUser model.User) []string {
	actions := make([]string, 0, 3)

	if task.CreatorID() == currentUser.ID {
		switch task.Status() {
		case domaintask.StatusOpen:
			actions = append(actions, taskActionAssign, taskActionCancel, taskActionDelete)
		case domaintask.StatusAssigned:
			actions = append(actions, taskActionCancel, taskActionDelete)
		case domaintask.StatusInProgress:
			actions = append(actions, taskActionCancel)
		case domaintask.StatusSubmitted:
			actions = append(actions, taskActionApprove, taskActionReject)
		case domaintask.StatusApproved:
			actions = append(actions, taskActionClose)
		case domaintask.StatusCancelled, domaintask.StatusCompleted:
			actions = append(actions, taskActionReactivate)
		}
	}

	if task.AssigneeID() == currentUser.ID {
		switch task.Status() {
		case domaintask.StatusAssigned:
			actions = append(actions, taskActionStart)
		case domaintask.StatusInProgress:
			actions = append(actions, taskActionSubmit)
		}
	}

	return dedupeActions(actions)
}

func dedupeActions(actions []string) []string {
	if len(actions) == 0 {
		return nil
	}

	result := make([]string, 0, len(actions))
	seen := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		result = append(result, action)
	}
	return result
}
