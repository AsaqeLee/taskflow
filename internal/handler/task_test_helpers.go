package handler

import (
	"testing"

	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/AsaqeLee/taskflow/internal/testutil"
)

func newMemoryTaskHandler(
	t *testing.T,
	repo ports.TaskRepository,
	recordRepo ports.TaskRecordRepository,
	auditRepo ports.AuditLogRepository,
) *TaskHandler {
	t.Helper()

	userRepo := repository.NewMemoryUserRepository()
	testutil.SeedAccount(t, userRepo, "u_owner_001", "Owner", "owner", "")
	testutil.SeedAccount(t, userRepo, "u_worker_001", "Worker", "human", "")
	testutil.SeedAccount(t, userRepo, "u_other_001", "Other", "human", "")
	return NewTaskHandler(service.NewTaskService(repo, recordRepo, auditRepo, userRepo))
}
