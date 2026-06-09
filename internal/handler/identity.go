package handler

import (
	"net/http"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

type IdentityHandler struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

func NewIdentityHandler(userRepo repository.UserRepository, jwtSecret string) *IdentityHandler {
	return &IdentityHandler{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (h *IdentityHandler) Me(c *gin.Context) {
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

type registerRequest struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
	Role string `json:"role" binding:"required"`
}

func (h *IdentityHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := auth.GenerateToken(req.ID, req.Role, h.jwtSecret, 2*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	u := model.User{
		ID:    req.ID,
		Name:  req.Name,
		Role:  req.Role,
		Token: token,
	}

	created, err := h.userRepo.Create(c.Request.Context(), u)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists or failed to create"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": created,
	})
}
