package bootstrap

import (
	"context"
	"fmt"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/handler"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/router"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
)

type App struct {
	config          config.Config
	engine          *gin.Engine
	database        *database.Client
	taskHandler     *handler.TaskHandler
	identityHandler *handler.IdentityHandler
}

func NewApp(cfg config.Config) (*App, error) {
	ctx := context.Background()
	taskRepo, recordRepo, auditRepo, userRepo, db, err := newRepositories(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := seedDefaultUsers(ctx, userRepo); err != nil {
		return nil, err
	}

	taskService := service.NewTaskService(taskRepo, recordRepo, auditRepo, db)
	taskHandler := handler.NewTaskHandler(taskService)
	identityHandler := handler.NewIdentityHandler(userRepo, cfg.JWTSecret)

	return &App{
		config:          cfg,
		engine:          router.New(taskHandler, identityHandler, userRepo, cfg),
		database:        db,
		taskHandler:     taskHandler,
		identityHandler: identityHandler,
	}, nil
}

func newRepositories(ctx context.Context, cfg config.Config) (
	repository.TaskRepository,
	repository.TaskRecordRepository,
	repository.AuditLogRepository,
	repository.UserRepository,
	*database.Client,
	error,
) {
	if cfg.RepositoryDriver == config.RepositoryDriverMongo {
		db, err := database.New(ctx, cfg)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		mongoDB := db.Mongo.Database(db.DBName)
		return repository.NewMongoTaskRepository(mongoDB.Collection("tasks")),
			repository.NewMongoTaskRecordRepository(mongoDB.Collection("task_records")),
			repository.NewMongoAuditLogRepository(mongoDB.Collection("audit_logs")),
			repository.NewMongoUserRepository(mongoDB.Collection("users")),
			db, nil
	}

	return repository.NewMemoryTaskRepository(),
		repository.NewMemoryTaskRecordRepository(),
		repository.NewMemoryAuditLogRepository(),
		repository.NewMemoryUserRepository(),
		nil, nil
}

func seedDefaultUsers(ctx context.Context, userRepo repository.UserRepository) error {
	defaultUsers := []model.User{
		{
			ID:    "u_test_001",
			Name:  "Test Creator",
			Role:  "owner",
			Token: "token_creator",
		},
		{
			ID:    "u_test_002",
			Name:  "Test Assignee",
			Role:  "human",
			Token: "token_assignee",
		},
		{
			ID:    "u_agent_001",
			Name:  "Hermes Agent",
			Role:  "agent",
			Token: "token_agent",
		},
	}

	for _, u := range defaultUsers {
		_, err := userRepo.FindByID(ctx, u.ID)
		if err != nil {
			_, err = userRepo.Create(ctx, u)
			if err != nil {
				return fmt.Errorf("failed to seed user %s: %w", u.ID, err)
			}
		}
	}
	return nil
}

func (a *App) Run() error {
	return a.engine.Run(a.addr())
}

func (a *App) addr() string {
	return fmt.Sprintf(":%d", a.config.Port)
}
