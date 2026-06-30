package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/httpapi"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
)

type IdentityHandler struct {
	identityService          *service.IdentityService
	jwtSecret                string
	accessTokenTTL           time.Duration
	refreshTokenTTL          time.Duration
	passwordResetTTL         time.Duration
	loginRateLimiter         middleware.RateLimiter
	passwordResetRateLimiter middleware.RateLimiter
	metrics                  *observability.Metrics
	passwordResetDelivery    PasswordResetDelivery
	devMode                  bool
	allowPublicRegister      bool
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
	identityService *service.IdentityService,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	passwordResetTTL time.Duration,
	loginRateLimiter middleware.RateLimiter,
	passwordResetRateLimiter middleware.RateLimiter,
	metrics *observability.Metrics,
	passwordResetDelivery PasswordResetDelivery,
	devMode bool,
	allowPublicRegister bool,
) *IdentityHandler {
	return &IdentityHandler{
		identityService:          identityService,
		jwtSecret:                jwtSecret,
		accessTokenTTL:           accessTokenTTL,
		refreshTokenTTL:          refreshTokenTTL,
		passwordResetTTL:         passwordResetTTL,
		loginRateLimiter:         loginRateLimiter,
		passwordResetRateLimiter: passwordResetRateLimiter,
		metrics:                  metrics,
		passwordResetDelivery:    passwordResetDelivery,
		devMode:                  devMode,
		allowPublicRegister:      allowPublicRegister,
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

type createAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type apiKeyResource struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

const (
	identityFlowRegister             = "register"
	identityFlowLogin                = "login"
	identityFlowRefresh              = "refresh"
	identityFlowPasswordResetRequest = "password_reset_request"
	identityFlowPasswordResetConfirm = "password_reset_confirm"
	identityFlowDisableAccount       = "disable_account"
	identityFlowRevokeSessions       = "revoke_sessions"
	identityFlowCreateAPIKey         = "create_api_key"
	identityFlowListAPIKeys          = "list_api_keys"
	identityFlowRevokeAPIKey         = "revoke_api_key"
)

func (h *IdentityHandler) ListUsers(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	activeOnly := c.Query("active") == "true"
	users, err := h.identityService.ListUsers(c.Request.Context(), currentUser, activeOnly)
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "users_list_failed", "failed to list users")
		return
	}

	response := make([]publicUser, len(users))
	for i, user := range users {
		response[i] = sanitizeUser(user)
	}
	c.JSON(http.StatusOK, gin.H{"users": response})
}

func (h *IdentityHandler) CreateAPIKey(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		h.recordIdentityEvent(identityFlowCreateAPIKey, "current_user_missing")
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		h.recordIdentityEvent(identityFlowCreateAPIKey, "invalid_user_id")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_user_id", "user id required")
		return
	}

	var req createAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordIdentityEvent(identityFlowCreateAPIKey, "invalid_request")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	key, rawKey, err := h.identityService.CreateAPIKey(c.Request.Context(), currentUser, targetUserID, req.Name, req.ExpiresAt)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbiddenAPIKeyManage):
			h.recordIdentityEvent(identityFlowCreateAPIKey, "forbidden")
			httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can create api keys")
		case errors.Is(err, service.ErrUserNotFound):
			h.recordIdentityEvent(identityFlowCreateAPIKey, "user_not_found")
			httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
		case errors.Is(err, domainuser.ErrAccountDisabled):
			h.recordIdentityEvent(identityFlowCreateAPIKey, "account_disabled")
			httpapi.WriteError(c, http.StatusBadRequest, "account_disabled", "target account is disabled")
		case errors.Is(err, domainuser.ErrEmptyUserID),
			errors.Is(err, domainidentity.ErrEmptyAPIKeyName),
			errors.Is(err, domainidentity.ErrEmptyAPIKeyPrefix),
			errors.Is(err, domainidentity.ErrEmptyAPIKeyHash):
			h.recordIdentityEvent(identityFlowCreateAPIKey, "invalid_request")
			httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		default:
			h.recordIdentityEvent(identityFlowCreateAPIKey, "create_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "api_key_create_failed", "failed to create api key")
		}
		return
	}

	h.recordIdentityEvent(identityFlowCreateAPIKey, "success")
	c.JSON(http.StatusCreated, gin.H{
		"api_key": sanitizeAPIKey(key),
		"key":     rawKey,
	})
}

func (h *IdentityHandler) ListAPIKeys(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		h.recordIdentityEvent(identityFlowListAPIKeys, "current_user_missing")
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		h.recordIdentityEvent(identityFlowListAPIKeys, "invalid_user_id")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_user_id", "user id required")
		return
	}

	keys, err := h.identityService.ListAPIKeys(c.Request.Context(), currentUser, targetUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbiddenAPIKeyManage):
			h.recordIdentityEvent(identityFlowListAPIKeys, "forbidden")
			httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can list api keys")
		case errors.Is(err, service.ErrUserNotFound):
			h.recordIdentityEvent(identityFlowListAPIKeys, "user_not_found")
			httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
		default:
			h.recordIdentityEvent(identityFlowListAPIKeys, "list_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "api_key_list_failed", "failed to list api keys")
		}
		return
	}

	response := make([]apiKeyResource, len(keys))
	for i, key := range keys {
		response[i] = sanitizeAPIKey(key)
	}

	h.recordIdentityEvent(identityFlowListAPIKeys, "success")
	c.JSON(http.StatusOK, gin.H{"api_keys": response})
}

func (h *IdentityHandler) RevokeAPIKey(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		h.recordIdentityEvent(identityFlowRevokeAPIKey, "current_user_missing")
		httpapi.WriteError(c, http.StatusInternalServerError, "current_user_missing", "current user not found in context")
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		h.recordIdentityEvent(identityFlowRevokeAPIKey, "invalid_user_id")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_user_id", "user id required")
		return
	}

	keyID := c.Param("keyID")
	if keyID == "" {
		h.recordIdentityEvent(identityFlowRevokeAPIKey, "invalid_key_id")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_key_id", "api key id required")
		return
	}

	key, err := h.identityService.RevokeAPIKey(c.Request.Context(), currentUser, targetUserID, keyID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbiddenAPIKeyManage):
			h.recordIdentityEvent(identityFlowRevokeAPIKey, "forbidden")
			httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can revoke api keys")
		case errors.Is(err, service.ErrUserNotFound):
			h.recordIdentityEvent(identityFlowRevokeAPIKey, "user_not_found")
			httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
		case errors.Is(err, service.ErrAPIKeyNotFound):
			h.recordIdentityEvent(identityFlowRevokeAPIKey, "api_key_not_found")
			httpapi.WriteError(c, http.StatusNotFound, "api_key_not_found", "api key not found")
		default:
			h.recordIdentityEvent(identityFlowRevokeAPIKey, "revoke_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "api_key_revoke_failed", "failed to revoke api key")
		}
		return
	}

	h.recordIdentityEvent(identityFlowRevokeAPIKey, "success")
	c.JSON(http.StatusOK, gin.H{
		"status":  "revoked",
		"api_key": sanitizeAPIKey(key),
	})
}

func (h *IdentityHandler) Register(c *gin.Context) {
	if !h.allowPublicRegister {
		currentUser, ok := middleware.CurrentUser(c)
		if !ok {
			h.recordIdentityEvent(identityFlowRegister, "unauthorized")
			httpapi.Unauthorized(c, "unauthorized", "authentication required to create users")
			return
		}
		actorRole, err := domainuser.ParseRole(currentUser.Role)
		if err != nil || !actorRole.IsOwner() {
			h.recordIdentityEvent(identityFlowRegister, "forbidden")
			httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can create users")
			return
		}
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordIdentityEvent(identityFlowRegister, "invalid_request")
		httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	created, err := h.identityService.Register(c.Request.Context(), req.ID, req.Name, req.Role, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWeakPassword):
			h.recordIdentityEvent(identityFlowRegister, "weak_password")
			httpapi.WriteError(c, http.StatusBadRequest, "weak_password", err.Error())
		case errors.Is(err, domainuser.ErrEmptyUserID),
			errors.Is(err, domainuser.ErrEmptyUserName),
			errors.Is(err, domainuser.ErrInvalidRole):
			h.recordIdentityEvent(identityFlowRegister, "invalid_request")
			httpapi.WriteError(c, http.StatusBadRequest, "invalid_request", err.Error())
		case errors.Is(err, service.ErrUserAlreadyExists):
			h.recordIdentityEvent(identityFlowRegister, "user_exists")
			httpapi.WriteError(c, http.StatusConflict, "user_exists", "user already exists")
		default:
			h.recordIdentityEvent(identityFlowRegister, "create_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "user_create_failed", "failed to create user")
		}
		return
	}

	if h.allowPublicRegister {
		if h.writeSessionResponse(c, http.StatusCreated, created, created.Token) {
			h.recordIdentityEvent(identityFlowRegister, "success")
		}
		return
	}

	h.recordIdentityEvent(identityFlowRegister, "success")
	c.JSON(http.StatusCreated, gin.H{"user": sanitizeUser(created)})
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

	user, err := h.identityService.Authenticate(c.Request.Context(), req.ID, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			h.recordIdentityEvent(identityFlowLogin, "invalid_credentials")
			httpapi.Unauthorized(c, "invalid_credentials", err.Error())
		case errors.Is(err, domainuser.ErrAccountDisabled):
			h.recordIdentityEvent(identityFlowLogin, "account_disabled")
			httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account is disabled")
		default:
			h.recordIdentityEvent(identityFlowLogin, "invalid_credentials")
			httpapi.Unauthorized(c, "invalid_credentials", service.ErrInvalidCredentials.Error())
		}
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

	user, newRefreshToken, err := h.identityService.RotateRefreshToken(c.Request.Context(), req.RefreshToken, h.refreshTokenTTL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRefreshToken):
			h.recordIdentityEvent(identityFlowRefresh, "invalid_refresh_token")
			httpapi.Unauthorized(c, "invalid_refresh_token", err.Error())
		case errors.Is(err, service.ErrRefreshTokenReused):
			h.recordIdentityEvent(identityFlowRefresh, "reuse_detected")
			httpapi.Unauthorized(c, "refresh_token_reused", "refresh token reuse detected; active sessions revoked")
		case errors.Is(err, domainuser.ErrAccountDisabled):
			h.recordIdentityEvent(identityFlowRefresh, "account_disabled")
			httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account is disabled")
		default:
			h.recordIdentityEvent(identityFlowRefresh, "rotation_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "token_rotation_failed", "failed to rotate refresh token")
		}
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

	rawToken, err := h.identityService.RequestPasswordReset(c.Request.Context(), req.ID, h.passwordResetTTL)
	if err != nil {
		h.recordIdentityEvent(identityFlowPasswordResetRequest, "reset_request_failed")
		httpapi.WriteError(c, http.StatusInternalServerError, "password_reset_failed", "failed to create password reset token")
		return
	}
	if rawToken != "" {
		if h.devMode {
			response["reset_token"] = rawToken
		} else if h.passwordResetDelivery != nil {
			err := h.passwordResetDelivery.Deliver(c.Request.Context(), PasswordResetNotice{
				UserID:    req.ID,
				Token:     rawToken,
				ExpiresAt: time.Now().UTC().Add(h.passwordResetTTL),
			})
			if err != nil {
				slog.Error("password_reset_delivery_failed", slog.String("user_id", req.ID), slog.Any("error", err))
				h.recordIdentityEvent(identityFlowPasswordResetRequest, "delivery_failed")
				c.JSON(http.StatusAccepted, response)
				return
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

	updated, err := h.identityService.ConfirmPasswordReset(c.Request.Context(), req.ID, req.Token, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPasswordResetTok):
			h.recordIdentityEvent(identityFlowPasswordResetConfirm, "invalid_reset_token")
			httpapi.WriteError(c, http.StatusBadRequest, "invalid_reset_token", "password reset token is invalid or expired")
		case errors.Is(err, service.ErrWeakPassword):
			h.recordIdentityEvent(identityFlowPasswordResetConfirm, "weak_password")
			httpapi.WriteError(c, http.StatusBadRequest, "weak_password", err.Error())
		default:
			h.recordIdentityEvent(identityFlowPasswordResetConfirm, "update_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "password_reset_failed", "failed to update password")
		}
		return
	}

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
	disabled, err := h.identityService.DisableAccount(c.Request.Context(), currentUser, targetUserID)
	if err != nil {
		switch {
		case errors.Is(err, domainuser.ErrForbiddenDisable):
			h.recordIdentityEvent(identityFlowDisableAccount, "forbidden")
			httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can disable other accounts")
		case errors.Is(err, domainuser.ErrAlreadyDisabled):
			h.recordIdentityEvent(identityFlowDisableAccount, "already_disabled")
			httpapi.WriteError(c, http.StatusBadRequest, "already_disabled", err.Error())
		case errors.Is(err, service.ErrUserNotFound):
			h.recordIdentityEvent(identityFlowDisableAccount, "user_not_found")
			httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
		default:
			h.recordIdentityEvent(identityFlowDisableAccount, "disable_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "account_disable_failed", "failed to disable account")
		}
		return
	}

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
	if err := h.identityService.RevokeSessions(c.Request.Context(), currentUser, targetUserID); err != nil {
		switch {
		case errors.Is(err, service.ErrForbiddenSessionRevoke):
			h.recordIdentityEvent(identityFlowRevokeSessions, "forbidden")
			httpapi.AbortError(c, http.StatusForbidden, "forbidden", "only owners can revoke other users' sessions")
		case errors.Is(err, service.ErrUserNotFound):
			h.recordIdentityEvent(identityFlowRevokeSessions, "user_not_found")
			httpapi.WriteError(c, http.StatusNotFound, "user_not_found", "user not found")
		default:
			h.recordIdentityEvent(identityFlowRevokeSessions, "revoke_failed")
			httpapi.WriteError(c, http.StatusInternalServerError, "session_revoke_failed", "failed to revoke user sessions")
		}
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
	refreshToken, err := h.identityService.IssueRefreshToken(c.Request.Context(), user.ID, h.refreshTokenTTL)
	if err != nil {
		httpapi.WriteError(c, http.StatusInternalServerError, "token_generation_failed", "failed to generate refresh token")
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

func sanitizeAPIKey(key domainidentity.APIKey) apiKeyResource {
	return apiKeyResource{
		ID:         key.ID(),
		UserID:     key.UserID(),
		Name:       key.Name(),
		KeyPrefix:  key.KeyPrefix(),
		CreatedAt:  key.CreatedAt(),
		ExpiresAt:  key.ExpiresAt(),
		LastUsedAt: key.LastUsedAt(),
		RevokedAt:  key.RevokedAt(),
	}
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
