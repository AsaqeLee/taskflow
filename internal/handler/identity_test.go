package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

func TestIdentityHandler_RegisterAndMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	h := NewIdentityHandler(
		userRepo,
		identityRepo,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		true,
	)

	r := gin.New()
	r.POST("/users", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/password-reset/request", h.RequestPasswordReset)
	r.POST("/auth/password-reset/confirm", h.ConfirmPasswordReset)

	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.GET("/me", h.Me)
	authenticated.POST("/users/:id/disable", h.DisableAccount)
	authenticated.POST("/users/:id/revoke-sessions", h.RevokeSessions)

	// 1. Register a new user u_test_003
	body := `{"id": "u_test_003", "name": "Dynamic User", "role": "human", "password": "strong-pass-123"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}

	var registerResp struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"user"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &registerResp); err != nil {
		t.Fatalf("failed to parse registration response: %v", err)
	}

	u := registerResp.User
	if u.ID != "u_test_003" || u.Name != "Dynamic User" || u.Role != "human" {
		t.Fatalf("unexpected registered user: %+v", u)
	}
	if registerResp.AccessToken == "" {
		t.Fatalf("expected generated access token to be populated")
	}
	if registerResp.RefreshToken == "" {
		t.Fatalf("expected generated refresh token to be populated")
	}

	// 2. Fetch /me profile with the generated token
	reqMe := httptest.NewRequest(http.MethodGet, "/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+registerResp.AccessToken)
	wMe := httptest.NewRecorder()
	r.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", wMe.Code, wMe.Body.String())
	}

	var meResp struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(wMe.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("failed to parse me response: %v", err)
	}

	if meResp.User.ID != "u_test_003" || meResp.User.Name != "Dynamic User" {
		t.Fatalf("unexpected user in profile response: %+v", meResp.User)
	}

	// 3. Login with password should issue a fresh JWT
	reqLogin := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"id":"u_test_003","password":"strong-pass-123"}`))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d body=%s", wLogin.Code, wLogin.Body.String())
	}

	// 4. Fetch /me profile with missing token
	reqMeMissing := httptest.NewRequest(http.MethodGet, "/me", nil)
	wMeMissing := httptest.NewRecorder()
	r.ServeHTTP(wMeMissing, reqMeMissing)

	if wMeMissing.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", wMeMissing.Code)
	}
}

func TestIdentityHandler_RefreshResetAndDisable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	h := NewIdentityHandler(
		userRepo,
		identityRepo,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		true,
	)

	r := gin.New()
	r.POST("/users", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/password-reset/request", h.RequestPasswordReset)
	r.POST("/auth/password-reset/confirm", h.ConfirmPasswordReset)

	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.POST("/users/:id/disable", h.DisableAccount)
	authenticated.POST("/users/:id/revoke-sessions", h.RevokeSessions)

	registerReq := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"id":"u_reset_001","name":"Reset User","role":"human","password":"strong-pass-123"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	r.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d body=%s", registerResp.Code, registerResp.Body.String())
	}

	var registered struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(registerResp.Body.Bytes(), &registered); err != nil {
		t.Fatalf("parse register response: %v", err)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"`+registered.RefreshToken+`"}`))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshResp := httptest.NewRecorder()
	r.ServeHTTP(refreshResp, refreshReq)
	if refreshResp.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d body=%s", refreshResp.Code, refreshResp.Body.String())
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/auth/password-reset/request", strings.NewReader(`{"id":"u_reset_001"}`))
	resetReq.Header.Set("Content-Type", "application/json")
	resetResp := httptest.NewRecorder()
	r.ServeHTTP(resetResp, resetReq)
	if resetResp.Code != http.StatusAccepted {
		t.Fatalf("expected reset request status 202, got %d body=%s", resetResp.Code, resetResp.Body.String())
	}

	var resetRequested struct {
		ResetToken string `json:"reset_token"`
	}
	if err := json.Unmarshal(resetResp.Body.Bytes(), &resetRequested); err != nil {
		t.Fatalf("parse password reset request response: %v", err)
	}
	if resetRequested.ResetToken == "" {
		t.Fatalf("expected dev-mode reset token in response")
	}

	confirmReq := httptest.NewRequest(http.MethodPost, "/auth/password-reset/confirm", strings.NewReader(`{"id":"u_reset_001","token":"`+resetRequested.ResetToken+`","new_password":"new-pass-1234"}`))
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmResp := httptest.NewRecorder()
	r.ServeHTTP(confirmResp, confirmReq)
	if confirmResp.Code != http.StatusOK {
		t.Fatalf("expected password reset confirm status 200, got %d body=%s", confirmResp.Code, confirmResp.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"id":"u_reset_001","password":"new-pass-1234"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	r.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login with reset password to succeed, got %d body=%s", loginResp.Code, loginResp.Body.String())
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/users/u_reset_001/disable", nil)
	disableReq.Header.Set("Authorization", "Bearer "+registered.AccessToken)
	disableResp := httptest.NewRecorder()
	r.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d body=%s", disableResp.Code, disableResp.Body.String())
	}

	loginAfterDisableReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"id":"u_reset_001","password":"new-pass-1234"}`))
	loginAfterDisableReq.Header.Set("Content-Type", "application/json")
	loginAfterDisableResp := httptest.NewRecorder()
	r.ServeHTTP(loginAfterDisableResp, loginAfterDisableReq)
	if loginAfterDisableResp.Code != http.StatusForbidden {
		t.Fatalf("expected disabled login status 403, got %d body=%s", loginAfterDisableResp.Code, loginAfterDisableResp.Body.String())
	}
}

func TestIdentityHandler_RefreshReuseAndSessionRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	h := NewIdentityHandler(
		userRepo,
		identityRepo,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		true,
	)

	r := gin.New()
	r.POST("/users", h.Register)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/login", h.Login)

	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.POST("/users/:id/revoke-sessions", h.RevokeSessions)

	registerReq := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"id":"u_rotate_001","name":"Rotate User","role":"human","password":"strong-pass-123"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	r.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d body=%s", registerResp.Code, registerResp.Body.String())
	}

	var registered struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(registerResp.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"`+registered.RefreshToken+`"}`))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshResp := httptest.NewRecorder()
	r.ServeHTTP(refreshResp, refreshReq)
	if refreshResp.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d body=%s", refreshResp.Code, refreshResp.Body.String())
	}

	var rotated struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(refreshResp.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}

	reuseReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"`+registered.RefreshToken+`"}`))
	reuseReq.Header.Set("Content-Type", "application/json")
	reuseResp := httptest.NewRecorder()
	r.ServeHTTP(reuseResp, reuseReq)
	if reuseResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected reused refresh token to fail, got %d body=%s", reuseResp.Code, reuseResp.Body.String())
	}

	refreshAfterReuseReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"`+rotated.RefreshToken+`"}`))
	refreshAfterReuseReq.Header.Set("Content-Type", "application/json")
	refreshAfterReuseResp := httptest.NewRecorder()
	r.ServeHTTP(refreshAfterReuseResp, refreshAfterReuseReq)
	if refreshAfterReuseResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected rotated refresh token to be revoked after reuse detection, got %d body=%s", refreshAfterReuseResp.Code, refreshAfterReuseResp.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"id":"u_rotate_001","password":"strong-pass-123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	r.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d body=%s", loginResp.Code, loginResp.Body.String())
	}

	var loggedIn struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loggedIn); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/users/u_rotate_001/revoke-sessions", nil)
	revokeReq.Header.Set("Authorization", "Bearer "+loggedIn.AccessToken)
	revokeResp := httptest.NewRecorder()
	r.ServeHTTP(revokeResp, revokeReq)
	if revokeResp.Code != http.StatusOK {
		t.Fatalf("expected revoke sessions status 200, got %d body=%s", revokeResp.Code, revokeResp.Body.String())
	}

	refreshAfterRevokeReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"`+loggedIn.RefreshToken+`"}`))
	refreshAfterRevokeReq.Header.Set("Content-Type", "application/json")
	refreshAfterRevokeResp := httptest.NewRecorder()
	r.ServeHTTP(refreshAfterRevokeResp, refreshAfterRevokeReq)
	if refreshAfterRevokeResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked refresh token to fail, got %d body=%s", refreshAfterRevokeResp.Code, refreshAfterRevokeResp.Body.String())
	}
}
