package repository

import "github.com/AsaqeLee/taskflow/internal/model"

type AuditLogRepository interface {
	Create(log model.AuditLog) (model.AuditLog, error)
	ListByTaskID(taskID string) ([]model.AuditLog, error)
	DeleteByTaskID(taskID string) error
}
