package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
)

func TestTaskHandler_StartReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_100",
		Title:       "Start via HTTP",
		Description: "test",
		Status:      service.TaskStatusAssigned,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_test_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/start", h.Start)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_100/start", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Task model.Task `json:"task"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.Status != service.TaskStatusInProgress {
		t.Fatalf("expected status %q, got %q", service.TaskStatusInProgress, resp.Task.Status)
	}
}

func TestTaskHandler_StartReturnsForbiddenForNonAssignee(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_101",
		Title:       "Forbidden start",
		Description: "test",
		Status:      service.TaskStatusAssigned,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_002",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/start", h.Start)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_101/start", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_StartReturnsBadRequestForOpenTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_102",
		Title:       "Open task",
		Description: "test",
		Status:      service.TaskStatusOpen,
		CreatorID:   "u_owner_001",
		AssigneeID:  "",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/start", h.Start)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_102/start", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}
