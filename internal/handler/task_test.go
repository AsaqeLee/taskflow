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

func TestTaskHandler_SubmitReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_110",
		Title:       "Submit via HTTP",
		Description: "test",
		Status:      service.TaskStatusInProgress,
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
	r.POST("/tasks/:id/submit", h.Submit)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_110/submit", strings.NewReader(""))
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
	if resp.Task.Status != service.TaskStatusSubmitted {
		t.Fatalf("expected status %q, got %q", service.TaskStatusSubmitted, resp.Task.Status)
	}
}

func TestTaskHandler_SubmitReturnsForbiddenForNonAssignee(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_111",
		Title:       "Forbidden submit",
		Description: "test",
		Status:      service.TaskStatusInProgress,
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
	r.POST("/tasks/:id/submit", h.Submit)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_111/submit", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_SubmitReturnsBadRequestForNonInProgressTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_112",
		Title:       "Wrong status",
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
	r.POST("/tasks/:id/submit", h.Submit)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_112/submit", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_RejectReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_120",
		Title:       "Reject via HTTP",
		Description: "test",
		Status:      service.TaskStatusSubmitted,
		CreatorID:   "u_test_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/reject", h.Reject)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_120/reject", strings.NewReader(""))
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
	if resp.Task.Status != service.TaskStatusAssigned {
		t.Fatalf("expected status %q, got %q", service.TaskStatusAssigned, resp.Task.Status)
	}
}

func TestTaskHandler_RejectReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_121",
		Title:       "Forbidden reject",
		Description: "test",
		Status:      service.TaskStatusSubmitted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/reject", h.Reject)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_121/reject", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_RejectReturnsBadRequestForNonSubmittedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_122",
		Title:       "Wrong reject status",
		Description: "test",
		Status:      service.TaskStatusInProgress,
		CreatorID:   "u_test_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/reject", h.Reject)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_122/reject", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ApproveReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_130",
		Title:       "Approve via HTTP",
		Description: "test",
		Status:      service.TaskStatusSubmitted,
		CreatorID:   "u_test_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/approve", h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_130/approve", strings.NewReader(""))
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
	if resp.Task.Status != service.TaskStatusApproved {
		t.Fatalf("expected status %q, got %q", service.TaskStatusApproved, resp.Task.Status)
	}
}

func TestTaskHandler_ApproveReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_131",
		Title:       "Forbidden approve",
		Description: "test",
		Status:      service.TaskStatusSubmitted,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/approve", h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_131/approve", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ApproveReturnsBadRequestForNonSubmittedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_132",
		Title:       "Wrong approve status",
		Description: "test",
		Status:      service.TaskStatusAssigned,
		CreatorID:   "u_test_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/approve", h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_132/approve", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_CloseReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_140",
		Title:       "Close via HTTP",
		Description: "test",
		Status:      service.TaskStatusApproved,
		CreatorID:   "u_test_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/close", h.Close)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_140/close", strings.NewReader(""))
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
	if resp.Task.Status != service.TaskStatusCompleted {
		t.Fatalf("expected status %q, got %q", service.TaskStatusCompleted, resp.Task.Status)
	}
}

func TestTaskHandler_CloseReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_141",
		Title:       "Forbidden close",
		Description: "test",
		Status:      service.TaskStatusApproved,
		CreatorID:   "u_owner_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/close", h.Close)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_141/close", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_CloseReturnsBadRequestForNonApprovedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(model.Task{
		ID:          "task_142",
		Title:       "Wrong close status",
		Description: "test",
		Status:      service.TaskStatusSubmitted,
		CreatorID:   "u_test_001",
		AssigneeID:  "u_worker_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/close", h.Close)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_142/close", strings.NewReader(""))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}
