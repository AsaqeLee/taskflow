package handler

import (
	"errors"
	"net/http"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	service *service.TaskService
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type assignTaskRequest struct {
	AssigneeID string `json:"assignee_id"`
}

type taskRecordRequest struct {
	Content string `json:"content"`
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{service: taskService}
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, err := h.service.CreateTask(currentUser, req.Title, req.Description)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"task": task})
}

func (h *TaskHandler) GetByID(c *gin.Context) {
	task, err := h.service.GetTask(c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (h *TaskHandler) List(c *gin.Context) {
	tasks, err := h.service.ListTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (h *TaskHandler) ListRecords(c *gin.Context) {
	records, err := h.service.ListTaskRecords(c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records})
}

func (h *TaskHandler) UpdateBasic(c *gin.Context) {
	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.UpdateTaskBasic(c.Param("id"), req.Title, req.Description)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (h *TaskHandler) Assign(c *gin.Context) {
	var req assignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, err := h.service.AssignTask(currentUser, c.Param("id"), req.AssigneeID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (h *TaskHandler) Start(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, err := h.service.StartTask(currentUser, c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (h *TaskHandler) Submit(c *gin.Context) {
	var req taskRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, record, err := h.service.SubmitTask(currentUser, c.Param("id"), req.Content)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task, "record": record})
}

func (h *TaskHandler) Reject(c *gin.Context) {
	var req taskRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, record, err := h.service.RejectTask(currentUser, c.Param("id"), req.Content)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task, "record": record})
}

func (h *TaskHandler) Approve(c *gin.Context) {
	var req taskRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, record, err := h.service.ApproveTask(currentUser, c.Param("id"), req.Content)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task, "record": record})
}

func (h *TaskHandler) Close(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, err := h.service.CloseTask(currentUser, c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (h *TaskHandler) Cancel(c *gin.Context) {
	var req taskRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, record, err := h.service.CancelTask(currentUser, c.Param("id"), req.Content)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task, "record": record})
}

func (h *TaskHandler) Reactivate(c *gin.Context) {
	var req taskRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	task, record, err := h.service.ReactivateTask(currentUser, c.Param("id"), req.Content)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task, "record": record})
}

func (h *TaskHandler) Delete(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "current user not found in context"})
		return
	}

	err := h.service.DeleteTask(currentUser, c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task and records successfully deleted"})
}

func (h *TaskHandler) ListAuditLogs(c *gin.Context) {
	logs, err := h.service.ListTaskAuditLogs(c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit_logs": logs})
}

func (h *TaskHandler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidTaskID),
		errors.Is(err, service.ErrEmptyTaskTitle),
		errors.Is(err, service.ErrTooShortTaskTitle),
		errors.Is(err, service.ErrEmptyAssigneeID),
		errors.Is(err, service.ErrEmptyTaskRecordContent),
		errors.Is(err, service.ErrInvalidTaskStatusForAssign),
		errors.Is(err, service.ErrInvalidTaskStatusForStart),
		errors.Is(err, service.ErrInvalidTaskStatusForSubmit),
		errors.Is(err, service.ErrInvalidTaskStatusForReject),
		errors.Is(err, service.ErrInvalidTaskStatusForApprove),
		errors.Is(err, service.ErrInvalidTaskStatusForClose),
		errors.Is(err, service.ErrInvalidTaskStatusForCancel),
		errors.Is(err, service.ErrInvalidTaskStatusForReactivate):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbiddenAssign),
		errors.Is(err, service.ErrForbiddenStart),
		errors.Is(err, service.ErrForbiddenSubmit),
		errors.Is(err, service.ErrForbiddenReject),
		errors.Is(err, service.ErrForbiddenApprove),
		errors.Is(err, service.ErrForbiddenClose),
		errors.Is(err, service.ErrForbiddenCancel),
		errors.Is(err, service.ErrForbiddenReactivate),
		errors.Is(err, service.ErrForbiddenDelete):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrTaskNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
