package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

type IdentityHandler struct {
	userRepo repository.UserRepository
}

func NewIdentityHandler(userRepo repository.UserRepository) *IdentityHandler {
	return &IdentityHandler{userRepo: userRepo}
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

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	token := hex.EncodeToString(tokenBytes)

	u := model.User{
		ID:    req.ID,
		Name:  req.Name,
		Role:  req.Role,
		Token: token,
	}

	created, err := h.userRepo.Create(u)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists or failed to create"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": created,
	})
}
