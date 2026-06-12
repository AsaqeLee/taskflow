package handler

import (
	"context"
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
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_110/submit", strings.NewReader(`{"content":"Delivered the requested output"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Task   model.Task       `json:"task"`
		Record model.TaskRecord `json:"record"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.Status != service.TaskStatusSubmitted {
		t.Fatalf("expected status %q, got %q", service.TaskStatusSubmitted, resp.Task.Status)
	}
	if resp.Record.Type != model.TaskRecordTypeSubmit {
		t.Fatalf("expected record type %q, got %q", model.TaskRecordTypeSubmit, resp.Record.Type)
	}
	if resp.Record.Content != "Delivered the requested output" {
		t.Fatalf("unexpected record content %q", resp.Record.Content)
	}
}

func TestTaskHandler_SubmitReturnsForbiddenForNonAssignee(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_111/submit", strings.NewReader(`{"content":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_SubmitReturnsBadRequestForNonInProgressTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_112/submit", strings.NewReader(`{"content":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_SubmitReturnsBadRequestForEmptyContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_113",
		Title:       "Missing content",
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_113/submit", strings.NewReader(`{"content":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_RejectReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_120/reject", strings.NewReader(`{"content":"Please revise the handoff"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Task   model.Task       `json:"task"`
		Record model.TaskRecord `json:"record"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.Status != service.TaskStatusAssigned {
		t.Fatalf("expected status %q, got %q", service.TaskStatusAssigned, resp.Task.Status)
	}
	if resp.Record.Type != model.TaskRecordTypeReject {
		t.Fatalf("expected record type %q, got %q", model.TaskRecordTypeReject, resp.Record.Type)
	}
	if resp.Record.Content != "Please revise the handoff" {
		t.Fatalf("unexpected record content %q", resp.Record.Content)
	}
}

func TestTaskHandler_RejectReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_121/reject", strings.NewReader(`{"content":"needs changes"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_RejectReturnsBadRequestForNonSubmittedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_122/reject", strings.NewReader(`{"content":"needs changes"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_RejectReturnsBadRequestForEmptyContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_123",
		Title:       "Missing reject content",
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_123/reject", strings.NewReader(`{"content":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ApproveReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_130/approve", strings.NewReader(`{"content":"Accepted after review"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Task   model.Task       `json:"task"`
		Record model.TaskRecord `json:"record"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.Status != service.TaskStatusApproved {
		t.Fatalf("expected status %q, got %q", service.TaskStatusApproved, resp.Task.Status)
	}
	if resp.Record.Type != model.TaskRecordTypeApprove {
		t.Fatalf("expected record type %q, got %q", model.TaskRecordTypeApprove, resp.Record.Type)
	}
	if resp.Record.Content != "Accepted after review" {
		t.Fatalf("unexpected record content %q", resp.Record.Content)
	}
}

func TestTaskHandler_ApproveReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_131/approve", strings.NewReader(`{"content":"approved"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ApproveReturnsBadRequestForNonSubmittedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_132/approve", strings.NewReader(`{"content":"approved"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ApproveReturnsBadRequestForEmptyContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_133",
		Title:       "Missing approve content",
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

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_133/approve", strings.NewReader(`{"content":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_CloseReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
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

func TestTaskHandler_ListRecordsReturnsRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	recordRepo := repository.NewMemoryTaskRecordRepository()
	svc := service.NewTaskService(repo, recordRepo, repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:          "task_150",
		Title:       "List records via HTTP",
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

	first, err := recordRepo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_150",
		AuthorID:  "u_worker_001",
		Type:      model.TaskRecordTypeSubmit,
		Content:   "submitted",
		CreatedAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("seed first record: %v", err)
	}
	second, err := recordRepo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_150",
		AuthorID:  "u_test_001",
		Type:      model.TaskRecordTypeApprove,
		Content:   "approved",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed second record: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.GET("/tasks/:id/records", h.ListRecords)

	req := httptest.NewRequest(http.MethodGet, "/tasks/task_150/records", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Records []model.TaskRecord `json:"records"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(resp.Records))
	}
	if resp.Records[0].ID != first.ID || resp.Records[1].ID != second.ID {
		t.Fatalf("expected ordered record ids [%s %s], got [%s %s]", first.ID, second.ID, resp.Records[0].ID, resp.Records[1].ID)
	}
}

func TestTaskHandler_ListRecordsReturnsNotFoundForMissingTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.GET("/tasks/:id/records", h.ListRecords)

	req := httptest.NewRequest(http.MethodGet, "/tasks/missing/records", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_CancelReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_cancel_1",
		Title:     "Cancel Task",
		Status:    service.TaskStatusOpen,
		CreatorID: "u_test_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/cancel", h.Cancel)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_cancel_1/cancel", strings.NewReader(`{"content":"Scope changed"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Task   model.Task       `json:"task"`
		Record model.TaskRecord `json:"record"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.Status != service.TaskStatusCancelled {
		t.Fatalf("expected status cancelled, got %q", resp.Task.Status)
	}
	if resp.Record.Type != model.TaskRecordTypeCancel {
		t.Fatalf("expected record type cancel, got %q", resp.Record.Type)
	}
}

func TestTaskHandler_CancelReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_cancel_2",
		Title:     "Cancel Task Error",
		Status:    service.TaskStatusOpen,
		CreatorID: "u_owner_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/cancel", h.Cancel)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_cancel_2/cancel", strings.NewReader(`{"content":"Scope changed"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ReactivateReturnsUpdatedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_react_1",
		Title:     "Reactivate Task",
		Status:    service.TaskStatusCancelled,
		CreatorID: "u_test_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.POST("/tasks/:id/reactivate", h.Reactivate)

	req := httptest.NewRequest(http.MethodPost, "/tasks/task_react_1/reactivate", strings.NewReader(`{"content":"Need to revisit"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Task   model.Task       `json:"task"`
		Record model.TaskRecord `json:"record"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.Status != service.TaskStatusOpen {
		t.Fatalf("expected status open, got %q", resp.Task.Status)
	}
}

func TestTaskHandler_DeleteReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_del_1",
		Title:     "Delete Task",
		Status:    service.TaskStatusOpen,
		CreatorID: "u_test_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.DELETE("/tasks/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/task_del_1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Verify task is gone
	_, err = repo.GetByID(context.Background(), "task_del_1")
	if err != repository.ErrTaskNotFound {
		t.Fatalf("expected task to be deleted")
	}
}

func TestTaskHandler_DeleteReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_del_2",
		Title:     "Delete Task Forbidden",
		Status:    service.TaskStatusOpen,
		CreatorID: "u_owner_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.DELETE("/tasks/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/task_del_2", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTaskHandler_ListAuditLogsReturnsLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	auditRepo := repository.NewMemoryAuditLogRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), auditRepo)
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), model.Task{
		ID:        "task_audit_1",
		Title:     "Audit Log API",
		Status:    service.TaskStatusOpen,
		CreatorID: "u_test_001",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = auditRepo.Create(context.Background(), model.AuditLog{
		TaskID:    "task_audit_1",
		ActorID:   "u_test_001",
		Action:    model.AuditActionCreated,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed audit log: %v", err)
	}

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.GET("/tasks/:id/audit_logs", h.ListAuditLogs)

	req := httptest.NewRequest(http.MethodGet, "/tasks/task_audit_1/audit_logs", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		AuditLogs []model.AuditLog `json:"audit_logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.AuditLogs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(resp.AuditLogs))
	}
	if resp.AuditLogs[0].Action != model.AuditActionCreated {
		t.Fatalf("expected action %q, got %q", model.AuditActionCreated, resp.AuditLogs[0].Action)
	}
}

func TestTaskHandler_ListAuditLogsReturnsNotFoundForMissingTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)

	r := gin.New()
	r.Use(middleware.FixedTestUser())
	r.GET("/tasks/:id/audit_logs", h.ListAuditLogs)

	req := httptest.NewRequest(http.MethodGet, "/tasks/missing/audit_logs", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
}
