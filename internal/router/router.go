package router

import (
	"github.com/AsaqeLee/taskflow/internal/handler"
	"github.com/gin-gonic/gin"
)

func New() *gin.Engine {
	r := gin.Default()
	r.GET("/health", handler.Health)
	return r
}
