package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/database"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/handler"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/router"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
)

type App struct {
	config          config.Config
	engine          *gin.Engine
	database        *database.Client
	metrics         *observability.Metrics
	tracingShutdown func(context.Context) error
	taskHandler     *handler.TaskHandler
	identityHandler *handler.IdentityHandler
	systemHandler   *handler.SystemHandler
}

func NewApp(cfg config.Config) (*App, error) {
	ctx := context.Background()
	tracer, tracingShutdown, err := observability.ConfigureTracing(ctx, cfg)
	if err != nil {
		return nil, err
	}

	taskRepo, recordRepo, auditRepo, userRepo, identityRepo, db, err := newRepositories(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.DevMode {
		if err := seedDefaultUsers(ctx, userRepo); err != nil {
			return nil, err
		}
	}

	if db != nil {
		if err := db.ApplyMigrations(ctx); err != nil {
			return nil, err
		}
	}

	metrics := observability.NewMetrics()
	rateLimiter := newRateLimiter(cfg, db)
	idempotencyStore := newIdempotencyStore(cfg, db)
	loginRateLimiter := newScopedRateLimiter(cfg.LoginRateLimitRequests, cfg.LoginRateLimitWindow, db)
	passwordResetRateLimiter := newScopedRateLimiter(cfg.PasswordResetRateLimitRequests, cfg.PasswordResetRateLimitWindow, db)

	taskService := service.NewTaskService(taskRepo, recordRepo, auditRepo, userRepo, db)
	taskHandler := handler.NewTaskHandler(taskService)
	identityService := service.NewIdentityService(userRepo, identityRepo, cfg.DevMode, db)
	identityHandler := handler.NewIdentityHandler(
		identityService,
		cfg.JWTSecret,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		cfg.PasswordResetTTL,
		loginRateLimiter,
		passwordResetRateLimiter,
		metrics,
		cfg.DevMode,
		cfg.AllowPublicRegister,
	)
	systemHandler := handler.NewSystemHandler(db, metrics, cfg.AppVersion)

	return &App{
		config:          cfg,
		engine:          router.New(systemHandler, taskHandler, identityHandler, userRepo, cfg, tracer, metrics, rateLimiter, idempotencyStore),
		database:        db,
		metrics:         metrics,
		tracingShutdown: tracingShutdown,
		taskHandler:     taskHandler,
		identityHandler: identityHandler,
		systemHandler:   systemHandler,
	}, nil
}

func newRepositories(ctx context.Context, cfg config.Config) (
	repository.TaskRepository,
	repository.TaskRecordRepository,
	repository.AuditLogRepository,
	repository.UserRepository,
	repository.IdentityRepository,
	*database.Client,
	error,
) {
	if cfg.RepositoryDriver == config.RepositoryDriverMongo {
		db, err := database.New(ctx, cfg)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		mongoDB := db.Mongo.Database(db.DBName)
		return repository.NewMongoTaskRepository(mongoDB.Collection("tasks")),
			repository.NewMongoTaskRecordRepository(mongoDB.Collection("task_records")),
			repository.NewMongoAuditLogRepository(mongoDB.Collection("audit_logs")),
			repository.NewMongoUserRepository(mongoDB.Collection("users")),
			repository.NewMongoIdentityRepository(
				mongoDB.Collection("refresh_tokens"),
				mongoDB.Collection("password_reset_tokens"),
			),
			db, nil
	}

	return repository.NewMemoryTaskRepository(),
		repository.NewMemoryTaskRecordRepository(),
		repository.NewMemoryAuditLogRepository(),
		repository.NewMemoryUserRepository(),
		repository.NewMemoryIdentityRepository(),
		nil, nil
}

func newRateLimiter(cfg config.Config, db *database.Client) middleware.RateLimiter {
	if db != nil && db.Mongo != nil {
		return middleware.NewMongoRateLimiter(
			db.Mongo.Database(db.DBName).Collection("runtime_rate_limits"),
			cfg.RateLimitRequests,
			cfg.RateLimitWindow,
		)
	}
	return middleware.NewMemoryRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
}

func newIdempotencyStore(cfg config.Config, db *database.Client) middleware.IdempotencyStore {
	if db != nil && db.Mongo != nil {
		return middleware.NewMongoIdempotencyStore(
			db.Mongo.Database(db.DBName).Collection("runtime_idempotency_keys"),
			cfg.IdempotencyTTL,
		)
	}
	return middleware.NewMemoryIdempotencyStore(cfg.IdempotencyTTL)
}

func newScopedRateLimiter(limit int, window time.Duration, db *database.Client) middleware.RateLimiter {
	if db != nil && db.Mongo != nil {
		return middleware.NewMongoRateLimiter(
			db.Mongo.Database(db.DBName).Collection("runtime_rate_limits"),
			limit,
			window,
		)
	}
	return middleware.NewMemoryRateLimiter(limit, window)
}

func seedDefaultUsers(ctx context.Context, userRepo repository.UserRepository) error {
	defaultUsers := []struct {
		ID       string
		Name     string
		Role     string
		Token    string
		Password string
	}{
		{
			ID:       "u_test_001",
			Name:     "Test Creator",
			Role:     "owner",
			Token:    "token_creator",
			Password: "creator-pass-123",
		},
		{
			ID:       "u_test_002",
			Name:     "Test Assignee",
			Role:     "human",
			Token:    "token_assignee",
			Password: "assignee-pass-123",
		},
		{
			ID:       "u_agent_001",
			Name:     "Hermes Agent",
			Role:     "agent",
			Token:    "token_agent",
			Password: "agent-pass-123",
		},
	}

	for _, u := range defaultUsers {
		_, err := userRepo.FindByID(ctx, u.ID)
		if err == nil {
			continue
		}

		passwordHash, hashErr := auth.HashPassword(u.Password)
		if hashErr != nil {
			return fmt.Errorf("failed to hash seed password for %s: %w", u.ID, hashErr)
		}

		role, roleErr := domainuser.ParseRole(u.Role)
		if roleErr != nil {
			return fmt.Errorf("failed to parse seed role for %s: %w", u.ID, roleErr)
		}

		now := time.Now().UTC()
		account, regErr := domainuser.Register(u.ID, u.Name, role, passwordHash, u.Token, now)
		if regErr != nil {
			return fmt.Errorf("failed to build seed account for %s: %w", u.ID, regErr)
		}
		_, err = userRepo.Create(ctx, account)
		if err != nil {
			return fmt.Errorf("failed to seed user %s: %w", u.ID, err)
		}
	}
	return nil
}

func (a *App) Run() error {
	defer func() {
		if a.tracingShutdown != nil {
			_ = a.tracingShutdown(context.Background())
		}
	}()

	server := &http.Server{
		Addr:              a.config.ServerAddress(),
		Handler:           a.engine,
		ReadHeaderTimeout: a.config.ServerReadTimeout,
		ReadTimeout:       a.config.ServerReadTimeout,
		WriteTimeout:      a.config.ServerWriteTimeout,
		IdleTimeout:       a.config.ServerWriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-shutdownSignals:
		ctx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(ctx)
	}
}

func (a *App) addr() string {
	return fmt.Sprintf(":%d", a.config.Port)
}
