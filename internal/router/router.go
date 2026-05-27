package router

import (
	"github.com/AsaqeLee/taskflow/internal/handler"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

func New(
	taskHandler *handler.TaskHandler,
	identityHandler *handler.IdentityHandler,
	userRepo repository.UserRepository,
) *gin.Engine {
	r := gin.Default()
	r.GET("/health", handler.Health)

	// Public routes
	r.POST("/users", identityHandler.Register)

	// Authenticated routes
	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo))

	authenticated.GET("/me", identityHandler.Me)
	authenticated.POST("/tasks", taskHandler.Create)
	authenticated.GET("/tasks", taskHandler.List)
	authenticated.GET("/tasks/:id", taskHandler.GetByID)
	authenticated.GET("/tasks/:id/records", taskHandler.ListRecords)
	authenticated.PATCH("/tasks/:id", taskHandler.UpdateBasic)
	authenticated.POST("/tasks/:id/assign", taskHandler.Assign)
	authenticated.POST("/tasks/:id/start", taskHandler.Start)
	authenticated.POST("/tasks/:id/submit", taskHandler.Submit)
	authenticated.POST("/tasks/:id/reject", taskHandler.Reject)
	authenticated.POST("/tasks/:id/approve", taskHandler.Approve)
	authenticated.POST("/tasks/:id/close", taskHandler.Close)
	authenticated.POST("/tasks/:id/cancel", taskHandler.Cancel)
	authenticated.POST("/tasks/:id/reactivate", taskHandler.Reactivate)
	authenticated.DELETE("/tasks/:id", taskHandler.Delete)
	authenticated.GET("/tasks/:id/audit_logs", taskHandler.ListAuditLogs)

	return r
}
