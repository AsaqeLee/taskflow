package router

import (
	"github.com/AsaqeLee/taskflow/internal/handler"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/gin-gonic/gin"
)

func New() *gin.Engine {
	r := gin.Default()
	r.GET("/health", handler.Health)

	authenticated := r.Group("/")
	authenticated.Use(middleware.FixedTestUser())
	authenticated.GET("/me", handler.Me)

	return r
}
