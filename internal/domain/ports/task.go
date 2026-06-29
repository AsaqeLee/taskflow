package ports

import (
	"context"
	"errors"

	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskListQuery struct {
	Query  string
	Status string
	Limit  int
	Offset int
}

type TaskListResult struct {
	Tasks []domaintask.Task
	Total int
}

type TaskRepository interface {
	Create(ctx context.Context, task domaintask.Task) (domaintask.Task, error)
	GetByID(ctx context.Context, id string) (domaintask.Task, error)
	GetByIDIncludingDeleted(ctx context.Context, id string) (domaintask.Task, error)
	List(ctx context.Context) ([]domaintask.Task, error)
	ListVisibleToUser(ctx context.Context, userID string) ([]domaintask.Task, error)
	Search(ctx context.Context, query TaskListQuery) (TaskListResult, error)
	SearchVisibleToUser(ctx context.Context, userID string, query TaskListQuery) (TaskListResult, error)
	Update(ctx context.Context, task domaintask.Task) (domaintask.Task, error)
}
