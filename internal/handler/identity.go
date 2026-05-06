package handler

import (
	"net/http"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/gin-gonic/gin"
)

func Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "current user not found in context",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
