package bootstrap

import (
	"context"
	"fmt"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/handler"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/router"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
)

type App struct {
	config      config.Config
	engine      *gin.Engine
	database    *database.Client
	taskHandler *handler.TaskHandler
}

func NewApp(cfg config.Config) (*App, error) {
	taskRepo, db, err := newTaskRepository(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	taskService := service.NewTaskService(taskRepo)
	taskHandler := handler.NewTaskHandler(taskService)

	return &App{
		config:      cfg,
		engine:      router.New(taskHandler),
		database:    db,
		taskHandler: taskHandler,
	}, nil
}

func newTaskRepository(ctx context.Context, cfg config.Config) (repository.TaskRepository, *database.Client, error) {
	if cfg.RepositoryDriver == config.RepositoryDriverMongo {
		db, err := database.New(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		return repository.NewMongoTaskRepository(db.Mongo.Database(db.DBName).Collection("tasks")), db, nil
	}

	return repository.NewMemoryTaskRepository(), nil, nil
}

func (a *App) Run() error {
	return a.engine.Run(a.addr())
}

func (a *App) addr() string {
	return fmt.Sprintf(":%s", a.config.Port)
}
