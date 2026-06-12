package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/httpapi"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	db         *database.Client
	metrics    *observability.Metrics
	appVersion string
}

func NewSystemHandler(db *database.Client, metrics *observability.Metrics, appVersion string) *SystemHandler {
	return &SystemHandler{
		db:         db,
		metrics:    metrics,
		appVersion: appVersion,
	}
}

func (h *SystemHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"app_version": h.appVersion,
	})
}

func (h *SystemHandler) Livez(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

func (h *SystemHandler) Readyz(c *gin.Context) {
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := h.db.Ping(ctx); err != nil {
			httpapi.WriteError(c, http.StatusServiceUnavailable, "not_ready", "database is not ready")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

func (h *SystemHandler) Metrics(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4")
	if h.metrics == nil {
		c.String(http.StatusOK, "")
		return
	}
	if err := h.metrics.WritePrometheus(c.Writer, h.appVersion); err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "metrics_write_failed", "failed to render metrics")
		return
	}
}
