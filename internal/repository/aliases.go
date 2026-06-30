package repository

import "github.com/AsaqeLee/taskflow/internal/domain/ports"

type TaskRepository = ports.TaskRepository
type TaskRecordRepository = ports.TaskRecordRepository
type AuditLogRepository = ports.AuditLogRepository
type UserRepository = ports.UserRepository
type IdentityRepository = ports.IdentityRepository

var (
	ErrTaskNotFound               = ports.ErrTaskNotFound
	ErrTaskRecordNotFound         = ports.ErrTaskRecordNotFound
	ErrUserNotFound               = ports.ErrUserNotFound
	ErrUserNotFoundByToken        = ports.ErrUserNotFoundByToken
	ErrUserAlreadyExists          = ports.ErrUserAlreadyExists
	ErrRefreshTokenNotFound       = ports.ErrRefreshTokenNotFound
	ErrPasswordResetTokenNotFound = ports.ErrPasswordResetTokenNotFound
	ErrAPIKeyNotFound             = ports.ErrAPIKeyNotFound
)
