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

func TestTaskHandler_ListRecords_RejectsUnrelatedAuthenticatedUser(t *testing.T) {
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

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated authenticated user to be rejected, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ListAuditLogs_RejectsUnrelatedAuthenticatedUser(t *testing.T) {
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
		"task_access_002",
		"Audit access check",
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
	r.GET("/tasks/:id/audit_logs", h.ListAuditLogs)

	req := httptest.NewRequest(http.MethodGet, "/tasks/task_access_002/audit_logs", nil)
	req.Header.Set("X-User-ID", "u_other_001")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated authenticated user to be rejected, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_GetByID_RejectsUnrelatedAuthenticatedUser(t *testing.T) {
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
		"task_access_003",
		"Task access check",
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
	r.GET("/tasks/:id", h.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/tasks/task_access_003", nil)
	req.Header.Set("X-User-ID", "u_other_001")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated authenticated user to be rejected, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_List_FiltersTasksByParticipationUnlessOwner(t *testing.T) {
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
		"task_access_004",
		"Visible to worker",
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
		t.Fatalf("seed task 1: %v", err)
	}
	_, err = repo.Create(context.Background(), domaintask.Restore(
		"task_access_005",
		"Visible to owner only",
		"",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task 2: %v", err)
	}

	r := gin.New()
	r.Use(middleware.UserAuth(userRepo, "test_secret", true))
	r.GET("/tasks", h.List)

	t.Run("owner sees all", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req.Header.Set("X-User-ID", "u_owner_001")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected owner 200, got %d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Tasks []model.Task `json:"tasks"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode owner response: %v", err)
		}
		if len(resp.Tasks) != 2 {
			t.Fatalf("expected owner to see 2 tasks, got %d", len(resp.Tasks))
		}
	})

	t.Run("worker sees only participating tasks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req.Header.Set("X-User-ID", "u_worker_001")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected worker 200, got %d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Tasks []model.Task `json:"tasks"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode worker response: %v", err)
		}
		if len(resp.Tasks) != 1 || resp.Tasks[0].ID != "task_access_004" {
			t.Fatalf("expected worker to see only participating task, got %+v", resp.Tasks)
		}
	})

	t.Run("unrelated user sees nothing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req.Header.Set("X-User-ID", "u_other_001")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected unrelated user 200, got %d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Tasks []model.Task `json:"tasks"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode unrelated response: %v", err)
		}
		if len(resp.Tasks) != 0 {
			t.Fatalf("expected unrelated user to see 0 tasks, got %d", len(resp.Tasks))
		}
	})
}
