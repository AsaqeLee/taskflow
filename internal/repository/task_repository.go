package repository

import (
	"context"
	"errors"

	"github.com/AsaqeLee/taskflow/internal/model"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository interface {
	Create(ctx context.Context, task model.Task) (model.Task, error)
	GetByID(ctx context.Context, id string) (model.Task, error)
	GetByIDIncludingDeleted(ctx context.Context, id string) (model.Task, error)
	List(ctx context.Context) ([]model.Task, error)
	Update(ctx context.Context, task model.Task) (model.Task, error)
	Delete(ctx context.Context, id string) error
}
