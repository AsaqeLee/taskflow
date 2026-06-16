package ports

import (
	"context"
	"errors"

	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository interface {
	Create(ctx context.Context, task domaintask.Task) (domaintask.Task, error)
	GetByID(ctx context.Context, id string) (domaintask.Task, error)
	GetByIDIncludingDeleted(ctx context.Context, id string) (domaintask.Task, error)
	List(ctx context.Context) ([]domaintask.Task, error)
	Update(ctx context.Context, task domaintask.Task) (domaintask.Task, error)
}
