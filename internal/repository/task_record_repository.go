package repository

import (
	"errors"
	"context"

	"github.com/AsaqeLee/taskflow/internal/model"
)

var ErrTaskRecordNotFound = errors.New("task record not found")

type TaskRecordRepository interface {
	Create(ctx context.Context, record model.TaskRecord) (model.TaskRecord, error)
	ListByTaskID(ctx context.Context, taskID string) ([]model.TaskRecord, error)
	DeleteByTaskID(ctx context.Context, taskID string) error
}
