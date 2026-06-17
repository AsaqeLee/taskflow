package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/AsaqeLee/taskflow/internal/testutil"
	"github.com/gin-gonic/gin"
)

// V0 policy: any authenticated user may read task records if they know task id.
func TestTaskHandler_ListRecords_AllowsUnrelatedAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	auditRepo := repository.NewMemoryAuditLogRepository()
	userRepo := repository.NewMemoryUserRepository()
	testutil.SeedAccount(t, userRepo, "u_owner_001", "Owner", "owner", "")
	testutil.SeedAccount(t, userRepo, "u_worker_001", "Worker", "human", "")
	testutil.SeedAccount(t, userRepo, "u_other_001", "Other", "human", "")
	h := NewTaskHandler(service.NewTaskService(repo, recordRepo, auditRepo, userRepo))
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_access_001",
		"Access policy check",
		"",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.UserAuth(userRepo, "test_secret", true))
	r.GET("/tasks/:id/records", h.ListRecords)

	req := httptest.NewRequest(http.MethodGet, "/tasks/task_access_001/records", nil)
	req.Header.Set("X-User-ID", "u_other_001")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected unrelated authenticated user to read records in V0, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Records []model.TaskRecord `json:"records"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}