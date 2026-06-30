package router

import (
	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	"github.com/AsaqeLee/taskflow/internal/handler"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func New(
	systemHandler *handler.SystemHandler,
	taskHandler *handler.TaskHandler,
	identityHandler *handler.IdentityHandler,
	userRepo ports.UserRepository,
	identityRepo ports.IdentityRepository,
	cfg config.Config,
	tracer trace.Tracer,
	metrics *observability.Metrics,
	rateLimiter middleware.RateLimiter,
	idempotencyStore middleware.IdempotencyStore,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if len(cfg.CORSAllowedOrigins) > 0 {
		r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	}
	r.Use(middleware.Tracing(tracer))
	r.Use(middleware.RequestContext())
	r.Use(middleware.Timeout(cfg.RequestTimeout))
	r.Use(middleware.RateLimit(rateLimiter, metrics, "global_http"))
	r.Use(middleware.StructuredLogger(metrics))

	r.GET("/health", systemHandler.Health)
	r.GET("/livez", systemHandler.Livez)
	r.GET("/readyz", systemHandler.Readyz)
	r.GET("/metrics", systemHandler.Metrics)

	idempotency := middleware.Idempotency(idempotencyStore, metrics)

	// Public routes
	public := r.Group("/")
	public.Use(idempotency)
	public.POST("/auth/login", identityHandler.Login)
	public.POST("/auth/refresh", identityHandler.Refresh)
	public.POST("/auth/password-reset/request", identityHandler.RequestPasswordReset)
	public.POST("/auth/password-reset/confirm", identityHandler.ConfirmPasswordReset)
	if cfg.AllowPublicRegister {
		public.POST("/users", identityHandler.Register)
	}

	// Authenticated routes
	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, cfg.JWTSecret, cfg.DevMode, identityRepo))
	authenticated.Use(idempotency)

	authenticated.GET("/me", identityHandler.Me)
	authenticated.GET("/users", identityHandler.ListUsers)
	if !cfg.AllowPublicRegister {
		authenticated.POST("/users", identityHandler.Register)
	}
	authenticated.POST("/users/:id/api-keys", identityHandler.CreateAPIKey)
	authenticated.GET("/users/:id/api-keys", identityHandler.ListAPIKeys)
	authenticated.POST("/users/:id/api-keys/:keyID/revoke", identityHandler.RevokeAPIKey)
	authenticated.POST("/users/:id/disable", identityHandler.DisableAccount)
	authenticated.POST("/users/:id/revoke-sessions", identityHandler.RevokeSessions)
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
