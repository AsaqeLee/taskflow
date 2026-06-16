package service

import (
	"testing"

	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/testutil"
)

func newMemoryTaskService(
	t *testing.T,
	repo ports.TaskRepository,
	recordRepo ports.TaskRecordRepository,
	auditRepo ports.AuditLogRepository,
) *TaskService {
	t.Helper()

	userRepo := repository.NewMemoryUserRepository()
	seedTaskServiceUsers(t, userRepo)
	return NewTaskService(repo, recordRepo, auditRepo, userRepo)
}

func seedTaskServiceUsers(t *testing.T, userRepo ports.UserRepository) {
	t.Helper()

	testutil.SeedAccount(t, userRepo, "u_owner_001", "Owner", "owner", "")
	testutil.SeedAccount(t, userRepo, "u_worker_001", "Worker", "human", "")
	testutil.SeedAccount(t, userRepo, "u_other_001", "Other", "human", "")
	testutil.SeedAccount(t, userRepo, "u_creator_rollback", "Rollback Creator", "owner", "")
}
