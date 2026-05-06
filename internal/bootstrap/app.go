package bootstrap

import (
	"context"
	"fmt"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/router"
	"github.com/gin-gonic/gin"
)

type App struct {
	config   config.Config
	engine   *gin.Engine
	database *database.Client
}

func NewApp(cfg config.Config) (*App, error) {
	db, err := database.New(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	return &App{
		config:   cfg,
		engine:   router.New(),
		database: db,
	}, nil
}

func (a *App) Run() error {
	return a.engine.Run(a.addr())
}

func (a *App) addr() string {
	return fmt.Sprintf(":%s", a.config.Port)
}
