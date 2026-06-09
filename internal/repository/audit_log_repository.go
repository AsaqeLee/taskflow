package repository

import (
	"context"

	"github.com/AsaqeLee/taskflow/internal/model"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log model.AuditLog) (model.AuditLog, error)
	ListByTaskID(ctx context.Context, taskID string) ([]model.AuditLog, error)
	DeleteByTaskID(ctx context.Context, taskID string) error
}
