package bootstrap

import (
	"fmt"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/router"
	"github.com/gin-gonic/gin"
)

type App struct {
	config config.Config
	engine *gin.Engine
}

func NewApp(cfg config.Config) *App {
	return &App{
		config: cfg,
		engine: router.New(),
	}
}

func (a *App) Run() error {
	return a.engine.Run(a.addr())
}

func (a *App) addr() string {
	return fmt.Sprintf(":%s", a.config.Port)
}
