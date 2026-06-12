package handler

import (
	"net/http"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/httpapi"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

type IdentityHandler struct {
	userRepo       repository.UserRepository
	jwtSecret      string
	accessTokenTTL time.Duration
	devMode        bool
}

type publicUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewIdentityHandler(userRepo repository.UserRepository, jwtSecret string, accessTokenTTL time.Duration, devMode bool) *IdentityHandler {
	return &IdentityHandler{
		userRepo:       userRepo,
		jwtSecret:      jwtSecret,
		accessTokenTTL: accessTokenTTL,
		devMode:        devMode,
	}
}

func (h *IdentityHandler) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": sanitizeUser(user),
	})
}

type registerRequest struct {
	ID       string `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	ID       string `json:"id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *IdentityHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		if err == auth.ErrWeakPassword {
			httpapi.WriteError(c, http.StatusBadRequest, "weak_password", err.Error())
			return
		}
		httpapi.WriteError(c, http.StatusInternalServerError, "password_hash_failed", "failed to hash password")
		return
	}

	now := time.Now().UTC()
	u := model.User{
		ID:           req.ID,
		Name:         req.Name,
		Role:         req.Role,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if h.devMode {
		u.Token = "dev_" + req.ID
	}

	created, err := h.userRepo.Create(c.Request.Context(), u)
	if err != nil {
		if err == repository.ErrUserAlreadyExists {
			httpapi.WriteError(c, http.StatusConflict, "user_exists", "user already exists")
			return
		}
		httpapi.WriteError(c, http.StatusInternalServerError, "user_create_failed", "failed to create user")
		return
	}

	token, err := auth.GenerateToken(created.ID, created.Role, h.jwtSecret, h.accessTokenTTL)
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate access token")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":                  sanitizeUser(created),
		"access_token":          token,
		"expires_in_seconds":    int(h.accessTokenTTL.Seconds()),
		"token_type":            "Bearer",
		"dev_legacy_token":      created.Token,
		"dev_legacy_token_hint": devTokenHint(created.Token),
	})
}

func (h *IdentityHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), req.ID)
	if err != nil {
		httpapi.Unauthorized(c, "invalid_credentials", auth.ErrInvalidCredentials.Error())
		return
	}

	if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
		httpapi.Unauthorized(c, "invalid_credentials", err.Error())
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Role, h.jwtSecret, h.accessTokenTTL)
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate access token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":               sanitizeUser(user),
		"access_token":       token,
		"expires_in_seconds": int(h.accessTokenTTL.Seconds()),
		"token_type":         "Bearer",
	})
}

func sanitizeUser(user model.User) publicUser {
	return publicUser{
		ID:        user.ID,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func devTokenHint(token string) string {
	if token == "" {
		return ""
	}
	return "available only when DEV_MODE=true"
}
