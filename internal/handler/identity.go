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
	userRepo         repository.UserRepository
	identityRepo     repository.IdentityRepository
	jwtSecret        string
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	passwordResetTTL time.Duration
	devMode          bool
}

type publicUser struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	Active     bool       `json:"active"`
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
	DisabledBy string     `json:"disabled_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func NewIdentityHandler(
	userRepo repository.UserRepository,
	identityRepo repository.IdentityRepository,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	passwordResetTTL time.Duration,
	devMode bool,
) *IdentityHandler {
	return &IdentityHandler{
		userRepo:         userRepo,
		identityRepo:     identityRepo,
		jwtSecret:        jwtSecret,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
		passwordResetTTL: passwordResetTTL,
		devMode:          devMode,
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

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type passwordResetRequest struct {
	ID string `json:"id" binding:"required"`
}

type passwordResetConfirmRequest struct {
	ID          string `json:"id" binding:"required"`
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
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
		Active:       true,
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

	h.writeSessionResponse(c, http.StatusCreated, created, created.Token)
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
	if !user.Active {
		httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account is disabled")
		return
	}

	if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
		httpapi.Unauthorized(c, "invalid_credentials", err.Error())
		return
	}

	h.writeSessionResponse(c, http.StatusOK, user, "")
}

func (h *IdentityHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	now := time.Now().UTC()
	current, err := h.identityRepo.FindRefreshToken(c.Request.Context(), auth.HashOpaqueToken(req.RefreshToken))
	if err != nil || current.RevokedAt != nil || now.After(current.ExpiresAt) {
		httpapi.Unauthorized(c, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), current.UserID)
	if err != nil {
		httpapi.Unauthorized(c, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}
	if !user.Active {
		httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account is disabled")
		return
	}

	newRefreshToken, newRefreshHash, err := h.newRefreshToken()
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to rotate refresh token")
		return
	}
	if err := h.identityRepo.RevokeRefreshToken(c.Request.Context(), current.TokenHash, now, newRefreshHash); err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_rotation_failed", "failed to revoke prior refresh token")
		return
	}
	if err := h.identityRepo.SaveRefreshToken(c.Request.Context(), model.RefreshToken{
		UserID:    user.ID,
		TokenHash: newRefreshHash,
		CreatedAt: now,
		ExpiresAt: now.Add(h.refreshTokenTTL),
	}); err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_rotation_failed", "failed to persist refresh token")
		return
	}

	accessToken, err := auth.GenerateToken(user.ID, user.Role, h.jwtSecret, h.accessTokenTTL)
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate access token")
		return
	}

	c.JSON(http.StatusOK, h.sessionResponse(user, accessToken, newRefreshToken, ""))
}

func (h *IdentityHandler) RequestPasswordReset(c *gin.Context) {
	var req passwordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	response := gin.H{
		"status": "accepted",
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), req.ID)
	if err == nil && user.Active {
		now := time.Now().UTC()
		if err := h.identityRepo.DeletePasswordResetTokensByUser(c.Request.Context(), user.ID); err == nil {
			rawToken, tokenHash, tokenErr := h.newPasswordResetToken()
			if tokenErr == nil {
				_ = h.identityRepo.SavePasswordResetToken(c.Request.Context(), model.PasswordResetToken{
					UserID:    user.ID,
					TokenHash: tokenHash,
					CreatedAt: now,
					ExpiresAt: now.Add(h.passwordResetTTL),
				})
				if h.devMode {
					response["reset_token"] = rawToken
				}
			}
		}
	}

	c.JSON(http.StatusAccepted, response)
}

func (h *IdentityHandler) ConfirmPasswordReset(c *gin.Context) {
	var req passwordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resetToken, err := h.identityRepo.FindPasswordResetToken(c.Request.Context(), auth.HashOpaqueToken(req.Token))
	if err != nil ||
		resetToken.UserID != req.ID ||
		resetToken.ConsumedAt != nil ||
		time.Now().UTC().After(resetToken.ExpiresAt) {
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_reset_token", "password reset token is invalid or expired")
		return
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		if err == auth.ErrWeakPassword {
			httpapi.WriteError(c, http.StatusBadRequest, "weak_password", err.Error())
			return
		}
		httpapi.WriteError(c, http.StatusInternalServerError, "password_hash_failed", "failed to hash password")
		return
	}

	now := time.Now().UTC()
	updated, err := h.userRepo.UpdatePassword(c.Request.Context(), req.ID, passwordHash, now)
	if err != nil {
		if err == repository.ErrUserNotFound {
			httpapi.WriteError(c, http.StatusBadRequest, "invalid_reset_token", "password reset token is invalid or expired")
			return
		}
		httpapi.WriteError(c, http.StatusInternalServerError, "password_reset_failed", "failed to update password")
		return
	}
	if _, err := h.identityRepo.ConsumePasswordResetToken(c.Request.Context(), auth.HashOpaqueToken(req.Token), now); err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "password_reset_failed", "failed to consume password reset token")
		return
	}
	_ = h.identityRepo.RevokeUserRefreshTokens(c.Request.Context(), req.ID, now)
	_ = h.identityRepo.DeletePasswordResetTokensByUser(c.Request.Context(), req.ID)

	c.JSON(http.StatusOK, gin.H{
		"status": "password_reset",
		"user":   sanitizeUser(updated),
	})
}

func (h *IdentityHandler) DisableAccount(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_user_id", "user id is required")
		return
	}
	if currentUser.Role != "owner" && currentUser.ID != targetUserID {
		httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can disable other accounts")
		return
	}

	now := time.Now().UTC()
	disabled, err := h.userRepo.Disable(c.Request.Context(), targetUserID, currentUser.ID, now)
	if err != nil {
		if err == repository.ErrUserNotFound {
			httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		httpapi.WriteError(c, http.StatusInternalServerError, "account_disable_failed", "failed to disable account")
		return
	}
	_ = h.identityRepo.RevokeUserRefreshTokens(c.Request.Context(), targetUserID, now)
	_ = h.identityRepo.DeletePasswordResetTokensByUser(c.Request.Context(), targetUserID)

	c.JSON(http.StatusOK, gin.H{
		"user": sanitizeUser(disabled),
	})
}

func (h *IdentityHandler) writeSessionResponse(c *gin.Context, status int, user model.User, devLegacyToken string) {
	now := time.Now().UTC()
	refreshToken, refreshHash, err := h.newRefreshToken()
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate refresh token")
		return
	}
	if err := h.identityRepo.SaveRefreshToken(c.Request.Context(), model.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		CreatedAt: now,
		ExpiresAt: now.Add(h.refreshTokenTTL),
	}); err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to persist refresh token")
		return
	}

	accessToken, err := auth.GenerateToken(user.ID, user.Role, h.jwtSecret, h.accessTokenTTL)
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate access token")
		return
	}

	c.JSON(status, h.sessionResponse(user, accessToken, refreshToken, devLegacyToken))
}

func (h *IdentityHandler) sessionResponse(user model.User, accessToken, refreshToken, devLegacyToken string) gin.H {
	response := gin.H{
		"user":                       sanitizeUser(user),
		"access_token":               accessToken,
		"refresh_token":              refreshToken,
		"expires_in_seconds":         int(h.accessTokenTTL.Seconds()),
		"refresh_expires_in_seconds": int(h.refreshTokenTTL.Seconds()),
		"token_type":                 "Bearer",
	}
	if devLegacyToken != "" {
		response["dev_legacy_token"] = devLegacyToken
		response["dev_legacy_token_hint"] = devTokenHint(devLegacyToken)
	}
	return response
}

func (h *IdentityHandler) newRefreshToken() (string, string, error) {
	rawToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", "", err
	}
	return rawToken, auth.HashOpaqueToken(rawToken), nil
}

func (h *IdentityHandler) newPasswordResetToken() (string, string, error) {
	rawToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", "", err
	}
	return rawToken, auth.HashOpaqueToken(rawToken), nil
}

func sanitizeUser(user model.User) publicUser {
	return publicUser{
		ID:         user.ID,
		Name:       user.Name,
		Role:       user.Role,
		Active:     user.Active,
		DisabledAt: user.DisabledAt,
		DisabledBy: user.DisabledBy,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}

func devTokenHint(token string) string {
	if token == "" {
		return ""
	}
	return "available only when DEV_MODE=true"
}
