package handler

import (
	"net/http"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/httpapi"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

type IdentityHandler struct {
	userRepo                 repository.UserRepository
	identityRepo             repository.IdentityRepository
	jwtSecret                string
	accessTokenTTL           time.Duration
	refreshTokenTTL          time.Duration
	passwordResetTTL         time.Duration
	loginRateLimiter         middleware.RateLimiter
	passwordResetRateLimiter middleware.RateLimiter
	metrics                  *observability.Metrics
	devMode                  bool
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
	loginRateLimiter middleware.RateLimiter,
	passwordResetRateLimiter middleware.RateLimiter,
	metrics *observability.Metrics,
	devMode bool,
) *IdentityHandler {
	return &IdentityHandler{
		userRepo:                 userRepo,
		identityRepo:             identityRepo,
		jwtSecret:                jwtSecret,
		accessTokenTTL:           accessTokenTTL,
		refreshTokenTTL:          refreshTokenTTL,
		passwordResetTTL:         passwordResetTTL,
		loginRateLimiter:         loginRateLimiter,
		passwordResetRateLimiter: passwordResetRateLimiter,
		metrics:                  metrics,
		devMode:                  devMode,
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

const (
	identityFlowRegister             = "register"
	identityFlowLogin                = "login"
	identityFlowRefresh              = "refresh"
	identityFlowPasswordResetRequest = "password_reset_request"
	identityFlowPasswordResetConfirm = "password_reset_confirm"
	identityFlowDisableAccount       = "disable_account"
	identityFlowRevokeSessions       = "revoke_sessions"
)

func (h *IdentityHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordIdentityEvent(identityFlowRegister, "invalid_request")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		if err == auth.ErrWeakPassword {
			h.recordIdentityEvent(identityFlowRegister, "weak_password")
			httpapi.WriteError(c, http.StatusBadRequest, "weak_password", err.Error())
			return
		}
		h.recordIdentityEvent(identityFlowRegister, "hash_failed")
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
			h.recordIdentityEvent(identityFlowRegister, "user_exists")
			httpapi.WriteError(c, http.StatusConflict, "user_exists", "user already exists")
			return
		}
		h.recordIdentityEvent(identityFlowRegister, "create_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "user_create_failed", "failed to create user")
		return
	}

	if h.writeSessionResponse(c, http.StatusCreated, created, created.Token) {
		h.recordIdentityEvent(identityFlowRegister, "success")
	}
}

func (h *IdentityHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordIdentityEvent(identityFlowLogin, "invalid_request")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !h.allowScopedRateLimit(c, h.loginRateLimiter, identityFlowLogin, req.ID) {
		h.recordIdentityEvent(identityFlowLogin, "rate_limited")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), req.ID)
	if err != nil {
		h.recordIdentityEvent(identityFlowLogin, "invalid_credentials")
		httpapi.Unauthorized(c, "invalid_credentials", auth.ErrInvalidCredentials.Error())
		return
	}
	if !user.Active {
		h.recordIdentityEvent(identityFlowLogin, "account_disabled")
		httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account is disabled")
		return
	}

	if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
		h.recordIdentityEvent(identityFlowLogin, "invalid_credentials")
		httpapi.Unauthorized(c, "invalid_credentials", err.Error())
		return
	}

	if h.writeSessionResponse(c, http.StatusOK, user, "") {
		h.recordIdentityEvent(identityFlowLogin, "success")
	}
}

func (h *IdentityHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordIdentityEvent(identityFlowRefresh, "invalid_request")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	now := time.Now().UTC()
	current, err := h.identityRepo.FindRefreshToken(c.Request.Context(), auth.HashOpaqueToken(req.RefreshToken))
	if err != nil || now.After(current.ExpiresAt) {
		h.recordIdentityEvent(identityFlowRefresh, "invalid_refresh_token")
		httpapi.Unauthorized(c, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}
	if current.RevokedAt != nil {
		if current.ReplacedByTokenHash != "" {
			_ = h.identityRepo.RevokeUserRefreshTokens(c.Request.Context(), current.UserID, now)
			h.recordIdentityEvent(identityFlowRefresh, "reuse_detected")
			httpapi.Unauthorized(c, "refresh_token_reused", "refresh token reuse detected; active sessions revoked")
			return
		}
		h.recordIdentityEvent(identityFlowRefresh, "invalid_refresh_token")
		httpapi.Unauthorized(c, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), current.UserID)
	if err != nil {
		h.recordIdentityEvent(identityFlowRefresh, "invalid_refresh_token")
		httpapi.Unauthorized(c, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}
	if !user.Active {
		h.recordIdentityEvent(identityFlowRefresh, "account_disabled")
		httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account is disabled")
		return
	}

	newRefreshToken, newRefreshHash, err := h.newRefreshToken()
	if err != nil {
		h.recordIdentityEvent(identityFlowRefresh, "rotation_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to rotate refresh token")
		return
	}
	if err := h.identityRepo.RevokeRefreshToken(c.Request.Context(), current.TokenHash, now, newRefreshHash); err != nil {
		h.recordIdentityEvent(identityFlowRefresh, "rotation_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "token_rotation_failed", "failed to revoke prior refresh token")
		return
	}
	if err := h.identityRepo.SaveRefreshToken(c.Request.Context(), model.RefreshToken{
		UserID:    user.ID,
		TokenHash: newRefreshHash,
		CreatedAt: now,
		ExpiresAt: now.Add(h.refreshTokenTTL),
	}); err != nil {
		h.recordIdentityEvent(identityFlowRefresh, "rotation_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "token_rotation_failed", "failed to persist refresh token")
		return
	}

	accessToken, err := auth.GenerateToken(user.ID, user.Role, h.jwtSecret, h.accessTokenTTL)
	if err != nil {
		h.recordIdentityEvent(identityFlowRefresh, "rotation_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate access token")
		return
	}

	h.recordIdentityEvent(identityFlowRefresh, "success")
	c.JSON(http.StatusOK, h.sessionResponse(user, accessToken, newRefreshToken, ""))
}

func (h *IdentityHandler) RequestPasswordReset(c *gin.Context) {
	var req passwordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordIdentityEvent(identityFlowPasswordResetRequest, "invalid_request")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !h.allowScopedRateLimit(c, h.passwordResetRateLimiter, identityFlowPasswordResetRequest, req.ID) {
		h.recordIdentityEvent(identityFlowPasswordResetRequest, "rate_limited")
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

	h.recordIdentityEvent(identityFlowPasswordResetRequest, "accepted")
	c.JSON(http.StatusAccepted, response)
}

func (h *IdentityHandler) ConfirmPasswordReset(c *gin.Context) {
	var req passwordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordIdentityEvent(identityFlowPasswordResetConfirm, "invalid_request")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !h.allowScopedRateLimit(c, h.passwordResetRateLimiter, identityFlowPasswordResetConfirm, req.ID) {
		h.recordIdentityEvent(identityFlowPasswordResetConfirm, "rate_limited")
		return
	}

	resetToken, err := h.identityRepo.FindPasswordResetToken(c.Request.Context(), auth.HashOpaqueToken(req.Token))
	if err != nil ||
		resetToken.UserID != req.ID ||
		resetToken.ConsumedAt != nil ||
		time.Now().UTC().After(resetToken.ExpiresAt) {
		h.recordIdentityEvent(identityFlowPasswordResetConfirm, "invalid_reset_token")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_reset_token", "password reset token is invalid or expired")
		return
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		if err == auth.ErrWeakPassword {
			h.recordIdentityEvent(identityFlowPasswordResetConfirm, "weak_password")
			httpapi.WriteError(c, http.StatusBadRequest, "weak_password", err.Error())
			return
		}
		h.recordIdentityEvent(identityFlowPasswordResetConfirm, "hash_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "password_hash_failed", "failed to hash password")
		return
	}

	now := time.Now().UTC()
	updated, err := h.userRepo.UpdatePassword(c.Request.Context(), req.ID, passwordHash, now)
	if err != nil {
		if err == repository.ErrUserNotFound {
			h.recordIdentityEvent(identityFlowPasswordResetConfirm, "invalid_reset_token")
			httpapi.WriteError(c, http.StatusBadRequest, "invalid_reset_token", "password reset token is invalid or expired")
			return
		}
		h.recordIdentityEvent(identityFlowPasswordResetConfirm, "update_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "password_reset_failed", "failed to update password")
		return
	}
	if _, err := h.identityRepo.ConsumePasswordResetToken(c.Request.Context(), auth.HashOpaqueToken(req.Token), now); err != nil {
		h.recordIdentityEvent(identityFlowPasswordResetConfirm, "consume_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "password_reset_failed", "failed to consume password reset token")
		return
	}
	_ = h.identityRepo.RevokeUserRefreshTokens(c.Request.Context(), req.ID, now)
	_ = h.identityRepo.DeletePasswordResetTokensByUser(c.Request.Context(), req.ID)

	h.recordIdentityEvent(identityFlowPasswordResetConfirm, "success")
	c.JSON(http.StatusOK, gin.H{
		"status": "password_reset",
		"user":   sanitizeUser(updated),
	})
}

func (h *IdentityHandler) DisableAccount(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		h.recordIdentityEvent(identityFlowDisableAccount, "current_user_missing")
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		h.recordIdentityEvent(identityFlowDisableAccount, "invalid_user_id")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_user_id", "user id is required")
		return
	}
	if currentUser.Role != "owner" && currentUser.ID != targetUserID {
		h.recordIdentityEvent(identityFlowDisableAccount, "forbidden")
		httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can disable other accounts")
		return
	}

	now := time.Now().UTC()
	disabled, err := h.userRepo.Disable(c.Request.Context(), targetUserID, currentUser.ID, now)
	if err != nil {
		if err == repository.ErrUserNotFound {
			h.recordIdentityEvent(identityFlowDisableAccount, "user_not_found")
			httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		h.recordIdentityEvent(identityFlowDisableAccount, "disable_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "account_disable_failed", "failed to disable account")
		return
	}
	_ = h.identityRepo.RevokeUserRefreshTokens(c.Request.Context(), targetUserID, now)
	_ = h.identityRepo.DeletePasswordResetTokensByUser(c.Request.Context(), targetUserID)

	h.recordIdentityEvent(identityFlowDisableAccount, "success")
	c.JSON(http.StatusOK, gin.H{
		"user": sanitizeUser(disabled),
	})
}

func (h *IdentityHandler) RevokeSessions(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		h.recordIdentityEvent(identityFlowRevokeSessions, "current_user_missing")
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		h.recordIdentityEvent(identityFlowRevokeSessions, "invalid_user_id")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_user_id", "user id is required")
		return
	}
	if currentUser.Role != "owner" && currentUser.ID != targetUserID {
		h.recordIdentityEvent(identityFlowRevokeSessions, "forbidden")
		httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can revoke other users' sessions")
		return
	}

	if _, err := h.userRepo.FindByID(c.Request.Context(), targetUserID); err != nil {
		h.recordIdentityEvent(identityFlowRevokeSessions, "user_not_found")
		httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	now := time.Now().UTC()
	if err := h.identityRepo.RevokeUserRefreshTokens(c.Request.Context(), targetUserID, now); err != nil {
		h.recordIdentityEvent(identityFlowRevokeSessions, "revoke_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "session_revoke_failed", "failed to revoke user sessions")
		return
	}

	h.recordIdentityEvent(identityFlowRevokeSessions, "success")
	c.JSON(http.StatusOK, gin.H{
		"status":          "sessions_revoked",
		"revoked_user_id": targetUserID,
		"revoked_by":      currentUser.ID,
	})
}

func (h *IdentityHandler) writeSessionResponse(c *gin.Context, status int, user model.User, devLegacyToken string) bool {
	now := time.Now().UTC()
	refreshToken, refreshHash, err := h.newRefreshToken()
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate refresh token")
		return false
	}
	if err := h.identityRepo.SaveRefreshToken(c.Request.Context(), model.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		CreatedAt: now,
		ExpiresAt: now.Add(h.refreshTokenTTL),
	}); err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to persist refresh token")
		return false
	}

	accessToken, err := auth.GenerateToken(user.ID, user.Role, h.jwtSecret, h.accessTokenTTL)
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate access token")
		return false
	}

	c.JSON(status, h.sessionResponse(user, accessToken, refreshToken, devLegacyToken))
	return true
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

func (h *IdentityHandler) allowScopedRateLimit(c *gin.Context, limiter middleware.RateLimiter, scope, subject string) bool {
	if limiter == nil || !limiter.Enabled() {
		return true
	}

	now := time.Now().UTC()
	keys := []string{scope + "|ip|" + c.ClientIP()}
	if subject != "" {
		keys = append(keys, scope+"|subject|"+subject)
	}

	for _, key := range keys {
		allowed, err := limiter.Allow(c.Request.Context(), key, now)
		if err != nil {
			h.recordRateLimitDecision(scope, "error")
			httpapi.WriteError(c, http.StatusInternalServerError, "rate_limit_failed", "failed to evaluate rate limit")
			return false
		}
		if !allowed {
			h.recordRateLimitDecision(scope, "rejected")
			httpapi.AbortError(c, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
			return false
		}
	}

	h.recordRateLimitDecision(scope, "allowed")
	return true
}

func (h *IdentityHandler) recordIdentityEvent(flow, outcome string) {
	if h.metrics == nil {
		return
	}
	h.metrics.ObserveIdentityEvent(flow, outcome)
}

func (h *IdentityHandler) recordRateLimitDecision(scope, decision string) {
	if h.metrics == nil {
		return
	}
	h.metrics.ObserveRateLimitDecision(scope, decision)
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
