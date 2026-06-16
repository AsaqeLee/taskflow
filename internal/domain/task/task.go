package task

import (
	"strings"
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain"
	"github.com/AsaqeLee/taskflow/internal/domain/audit"
	"github.com/AsaqeLee/taskflow/internal/domain/record"
	"github.com/AsaqeLee/taskflow/internal/domain/user"
)

// Task is the aggregate root for task lifecycle management.
type Task struct {
	id          string
	title       string
	description string
	status      Status
	creatorID   string
	assigneeID  string
	createdAt   time.Time
	updatedAt   time.Time
	deletedAt   *time.Time
	deletedBy   string
}

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", domain.ErrEmptyTaskTitle
	}
	if len(title) < 3 {
		return "", domain.ErrTooShortTaskTitle
	}
	return title, nil
}

func validateRecordContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", domain.ErrEmptyTaskRecordContent
	}
	return content, nil
}

// Create initializes a new open task for the given creator.
func Create(creatorID, title, description string, at time.Time) (Task, Transition, error) {
	validTitle, err := validateTitle(title)
	if err != nil {
		return Task{}, Transition{}, err
	}

	task := Task{
		title:       validTitle,
		description: strings.TrimSpace(description),
		status:      StatusOpen,
		creatorID:   creatorID,
		createdAt:   at,
		updatedAt:   at,
	}

	return task, buildCreateTransition(creatorID, at), nil
}

// Restore rehydrates a task from persistence without re-validating invariants.
func Restore(
	id, title, description string,
	status Status,
	creatorID, assigneeID string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
	deletedBy string,
) Task {
	return Task{
		id:          id,
		title:       title,
		description: description,
		status:      status,
		creatorID:   creatorID,
		assigneeID:  assigneeID,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		deletedAt:   deletedAt,
		deletedBy:   deletedBy,
	}
}

func (t Task) ID() string            { return t.id }
func (t Task) Title() string         { return t.title }
func (t Task) Description() string   { return t.description }
func (t Task) Status() Status        { return t.status }
func (t Task) CreatorID() string     { return t.creatorID }
func (t Task) AssigneeID() string    { return t.assigneeID }
func (t Task) CreatedAt() time.Time  { return t.createdAt }
func (t Task) UpdatedAt() time.Time  { return t.updatedAt }
func (t Task) DeletedAt() *time.Time { return t.deletedAt }
func (t Task) DeletedBy() string     { return t.deletedBy }
func (t Task) IsDeleted() bool       { return t.deletedAt != nil }

func (t Task) AssignID(id string) Task {
	t.id = id
	return t
}

func (t *Task) touch(at time.Time) {
	t.updatedAt = at
}

func (t Task) isCreator(actor user.Actor) bool {
	return t.creatorID == actor.ID
}

func (t Task) isAssignee(actor user.Actor) bool {
	return t.assigneeID == actor.ID
}

// UpdateBasic changes title and description. Only the creator may update.
func (t *Task) UpdateBasic(actor user.Actor, title, description string, at time.Time) error {
	if !t.isCreator(actor) {
		return domain.ErrForbiddenUpdate
	}

	validTitle, err := validateTitle(title)
	if err != nil {
		return err
	}

	t.title = validTitle
	t.description = strings.TrimSpace(description)
	t.touch(at)
	return nil
}

// Assign moves an open task to assigned.
func (t *Task) Assign(actor user.Actor, assigneeID string, at time.Time) (Transition, error) {
	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return Transition{}, domain.ErrEmptyAssigneeID
	}
	if !t.isCreator(actor) {
		return Transition{}, domain.ErrForbiddenAssign
	}
	if t.status != StatusOpen {
		return Transition{}, domain.ErrInvalidTaskStatusForAssign
	}

	from := t.status
	t.assigneeID = assigneeID
	t.status = StatusAssigned
	t.touch(at)

	return buildTransition(t.id, actor.ID, audit.ActionAssigned, from, t.status, at, nil), nil
}

// Start moves an assigned task to in_progress. Only the assignee may start.
func (t *Task) Start(actor user.Actor, at time.Time) (Transition, error) {
	if t.status != StatusAssigned {
		return Transition{}, domain.ErrInvalidTaskStatusForStart
	}
	if !t.isAssignee(actor) {
		return Transition{}, domain.ErrForbiddenStart
	}

	from := t.status
	t.status = StatusInProgress
	t.touch(at)

	return buildTransition(t.id, actor.ID, audit.ActionStarted, from, t.status, at, nil), nil
}

// Submit moves an in-progress task to submitted and records execution content.
func (t *Task) Submit(actor user.Actor, content string, at time.Time) (Transition, error) {
	if t.status != StatusInProgress {
		return Transition{}, domain.ErrInvalidTaskStatusForSubmit
	}
	if !t.isAssignee(actor) {
		return Transition{}, domain.ErrForbiddenSubmit
	}

	validContent, err := validateRecordContent(content)
	if err != nil {
		return Transition{}, err
	}

	from := t.status
	t.status = StatusSubmitted
	t.touch(at)

	draft, err := record.NewDraft(actor.ID, record.TypeSubmit, validContent, at)
	if err != nil {
		return Transition{}, err
	}

	return buildTransition(t.id, actor.ID, audit.ActionSubmitted, from, t.status, at, &draft), nil
}

// Reject moves a submitted task back to assigned.
func (t *Task) Reject(actor user.Actor, content string, at time.Time) (Transition, error) {
	if t.status != StatusSubmitted {
		return Transition{}, domain.ErrInvalidTaskStatusForReject
	}
	if !t.isCreator(actor) {
		return Transition{}, domain.ErrForbiddenReject
	}

	validContent, err := validateRecordContent(content)
	if err != nil {
		return Transition{}, err
	}

	from := t.status
	t.status = StatusAssigned
	t.touch(at)

	draft, err := record.NewDraft(actor.ID, record.TypeReject, validContent, at)
	if err != nil {
		return Transition{}, err
	}

	return buildTransition(t.id, actor.ID, audit.ActionRejected, from, t.status, at, &draft), nil
}

// Approve moves a submitted task to approved.
func (t *Task) Approve(actor user.Actor, content string, at time.Time) (Transition, error) {
	if t.status != StatusSubmitted {
		return Transition{}, domain.ErrInvalidTaskStatusForApprove
	}
	if !t.isCreator(actor) {
		return Transition{}, domain.ErrForbiddenApprove
	}

	validContent, err := validateRecordContent(content)
	if err != nil {
		return Transition{}, err
	}

	from := t.status
	t.status = StatusApproved
	t.touch(at)

	draft, err := record.NewDraft(actor.ID, record.TypeApprove, validContent, at)
	if err != nil {
		return Transition{}, err
	}

	return buildTransition(t.id, actor.ID, audit.ActionApproved, from, t.status, at, &draft), nil
}

// Close moves an approved task to completed.
func (t *Task) Close(actor user.Actor, at time.Time) (Transition, error) {
	if t.status != StatusApproved {
		return Transition{}, domain.ErrInvalidTaskStatusForClose
	}
	if !t.isCreator(actor) {
		return Transition{}, domain.ErrForbiddenClose
	}

	from := t.status
	t.status = StatusCompleted
	t.touch(at)

	return buildTransition(t.id, actor.ID, audit.ActionClosed, from, t.status, at, nil), nil
}

// Cancel terminates an active task before completion.
func (t *Task) Cancel(actor user.Actor, content string, at time.Time) (Transition, error) {
	if t.status != StatusOpen && t.status != StatusAssigned &&
		t.status != StatusInProgress && t.status != StatusSubmitted {
		return Transition{}, domain.ErrInvalidTaskStatusForCancel
	}
	if !t.isCreator(actor) {
		return Transition{}, domain.ErrForbiddenCancel
	}

	validContent, err := validateRecordContent(content)
	if err != nil {
		return Transition{}, err
	}

	from := t.status
	t.status = StatusCancelled
	t.touch(at)

	draft, err := record.NewDraft(actor.ID, record.TypeCancel, validContent, at)
	if err != nil {
		return Transition{}, err
	}

	return buildTransition(t.id, actor.ID, audit.ActionCancelled, from, t.status, at, &draft), nil
}

// Reactivate restores a cancelled or completed task.
func (t *Task) Reactivate(actor user.Actor, content string, at time.Time) (Transition, error) {
	if t.status != StatusCancelled && t.status != StatusCompleted {
		return Transition{}, domain.ErrInvalidTaskStatusForReactivate
	}
	if !t.isCreator(actor) {
		return Transition{}, domain.ErrForbiddenReactivate
	}

	validContent, err := validateRecordContent(content)
	if err != nil {
		return Transition{}, err
	}

	from := t.status
	if t.assigneeID != "" {
		t.status = StatusAssigned
	} else {
		t.status = StatusOpen
	}
	t.touch(at)

	draft, err := record.NewDraft(actor.ID, record.TypeReactivate, validContent, at)
	if err != nil {
		return Transition{}, err
	}

	return buildTransition(t.id, actor.ID, audit.ActionReopened, from, t.status, at, &draft), nil
}

// MarkDeleted soft-deletes the task. Only the creator may delete.
func (t *Task) MarkDeleted(actor user.Actor, at time.Time) (Transition, error) {
	if !t.isCreator(actor) {
		return Transition{}, domain.ErrForbiddenDelete
	}

	from := t.status
	t.status = StatusDeleted
	t.deletedAt = &at
	t.deletedBy = actor.ID
	t.touch(at)

	return buildTransition(t.id, actor.ID, audit.ActionDeleted, from, StatusDeleted, at, nil), nil
}
