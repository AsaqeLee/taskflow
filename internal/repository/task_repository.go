package repository

import (
	"errors"

	"github.com/AsaqeLee/taskflow/internal/model"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository interface {
	Create(task model.Task) (model.Task, error)
	GetByID(id string) (model.Task, error)
	List() ([]model.Task, error)
	Update(task model.Task) (model.Task, error)
}
