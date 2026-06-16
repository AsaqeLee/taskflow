package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_100",
		"Start via HTTP",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_test_001",
		now,
		now,
		nil,
		"",
	))
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
	if resp.Task.Status != domaintask.StatusInProgress.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusInProgress.String(), resp.Task.Status)
	}
}

func TestTaskHandler_StartReturnsForbiddenForNonAssignee(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_101",
		"Forbidden start",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_worker_002",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_102",
		"Open task",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_110",
		"Submit via HTTP",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_test_001",
		now,
		now,
		nil,
		"",
	))
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
	if resp.Task.Status != domaintask.StatusSubmitted.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusSubmitted.String(), resp.Task.Status)
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_111",
		"Forbidden submit",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_worker_002",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_112",
		"Wrong status",
		"test",
		domaintask.StatusAssigned,
		"u_owner_001",
		"u_test_001",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_113",
		"Missing content",
		"test",
		domaintask.StatusInProgress,
		"u_owner_001",
		"u_test_001",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_120",
		"Reject via HTTP",
		"test",
		domaintask.StatusSubmitted,
		"u_test_001",
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
	if resp.Task.Status != domaintask.StatusAssigned.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusAssigned.String(), resp.Task.Status)
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_121",
		"Forbidden reject",
		"test",
		domaintask.StatusSubmitted,
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_122",
		"Wrong reject status",
		"test",
		domaintask.StatusInProgress,
		"u_test_001",
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_123",
		"Missing reject content",
		"test",
		domaintask.StatusSubmitted,
		"u_test_001",
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_130",
		"Approve via HTTP",
		"test",
		domaintask.StatusSubmitted,
		"u_test_001",
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
	if resp.Task.Status != domaintask.StatusApproved.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusApproved.String(), resp.Task.Status)
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_131",
		"Forbidden approve",
		"test",
		domaintask.StatusSubmitted,
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_132",
		"Wrong approve status",
		"test",
		domaintask.StatusAssigned,
		"u_test_001",
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_133",
		"Missing approve content",
		"test",
		domaintask.StatusSubmitted,
		"u_test_001",
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_140",
		"Close via HTTP",
		"test",
		domaintask.StatusApproved,
		"u_test_001",
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
	if resp.Task.Status != domaintask.StatusCompleted.String() {
		t.Fatalf("expected status %q, got %q", domaintask.StatusCompleted.String(), resp.Task.Status)
	}
}

func TestTaskHandler_CloseReturnsForbiddenForNonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_141",
		"Forbidden close",
		"test",
		domaintask.StatusApproved,
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_142",
		"Wrong close status",
		"test",
		domaintask.StatusSubmitted,
		"u_test_001",
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_150",
		"List records via HTTP",
		"test",
		domaintask.StatusApproved,
		"u_test_001",
		"u_worker_001",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	first, err := recordRepo.Create(context.Background(), domainrecord.Restore(
		"",
		"task_150",
		"u_worker_001",
		domainrecord.TypeSubmit,
		"submitted",
		now.Add(-time.Minute),
	))
	if err != nil {
		t.Fatalf("seed first record: %v", err)
	}
	second, err := recordRepo.Create(context.Background(), domainrecord.Restore(
		"",
		"task_150",
		"u_test_001",
		domainrecord.TypeApprove,
		"approved",
		now,
	))
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
	if resp.Records[0].ID != first.ID() || resp.Records[1].ID != second.ID() {
		t.Fatalf("expected ordered record ids [%s %s], got [%s %s]", first.ID(), second.ID(), resp.Records[0].ID, resp.Records[1].ID)
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_cancel_1",
		"Cancel Task",
		"test",
		domaintask.StatusOpen,
		"u_test_001",
		"",
		now,
		now,
		nil,
		"",
	))
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
	if resp.Task.Status != domaintask.StatusCancelled.String() {
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_cancel_2",
		"Cancel Task Error",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_react_1",
		"Reactivate Task",
		"test",
		domaintask.StatusCancelled,
		"u_test_001",
		"",
		now,
		now,
		nil,
		"",
	))
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
	if resp.Task.Status != domaintask.StatusOpen.String() {
		t.Fatalf("expected status open, got %q", resp.Task.Status)
	}
}

func TestTaskHandler_DeleteReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryTaskRepository()
	svc := service.NewTaskService(repo, repository.NewMemoryTaskRecordRepository(), repository.NewMemoryAuditLogRepository())
	h := NewTaskHandler(svc)
	now := time.Now().UTC()

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_del_1",
		"Delete Task",
		"test",
		domaintask.StatusOpen,
		"u_test_001",
		"",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_del_2",
		"Delete Task Forbidden",
		"test",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
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

	_, err := repo.Create(context.Background(), domaintask.Restore(
		"task_audit_1",
		"Audit Log API",
		"test",
		domaintask.StatusOpen,
		"u_test_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err = auditRepo.Create(context.Background(), domainaudit.Restore(
		"",
		"task_audit_1",
		"u_test_001",
		domainaudit.ActionCreated,
		"", "", "", "", "",
		"", "",
		now,
	))
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
