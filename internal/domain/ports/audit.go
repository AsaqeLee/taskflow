package ports

import (
	"context"

	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log domainaudit.Log) (domainaudit.Log, error)
	ListByTaskID(ctx context.Context, taskID string) ([]domainaudit.Log, error)
	DeleteByTaskID(ctx context.Context, taskID string) error
}
