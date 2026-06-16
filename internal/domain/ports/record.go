package ports

import (
	"context"
	"errors"

	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
)

var ErrTaskRecordNotFound = errors.New("task record not found")

type TaskRecordRepository interface {
	Create(ctx context.Context, record domainrecord.Record) (domainrecord.Record, error)
	ListByTaskID(ctx context.Context, taskID string) ([]domainrecord.Record, error)
	DeleteByTaskID(ctx context.Context, taskID string) error
}
