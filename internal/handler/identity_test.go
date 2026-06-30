package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/AsaqeLee/taskflow/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestIdentityHandler_RegisterAndMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	identityService := service.NewIdentityService(userRepo, identityRepo, true)
	h := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		nil,
		true,
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
	identityService := service.NewIdentityService(userRepo, identityRepo, true)
	h := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		nil,
		true,
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
	identityService := service.NewIdentityService(userRepo, identityRepo, true)
	h := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		nil,
		true,
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

func TestIdentityHandler_ListUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	identityService := service.NewIdentityService(userRepo, identityRepo, true)
	h := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		nil,
		true,
		true,
	)

	r := gin.New()
	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.GET("/users", h.ListUsers)

	if _, err := identityService.Register(context.Background(), "u_owner_list", "Owner", "owner", "strong-pass-123"); err != nil {
		t.Fatalf("seed owner: %v", err)
	}

	_, err := identityService.Register(context.Background(), "u_alice_list", "Alice", "human", "strong-pass-123")
	if err != nil {
		t.Fatalf("seed alice: %v", err)
	}
	_, err = identityService.Register(context.Background(), "u_bob_list", "Bob", "human", "strong-pass-123")
	if err != nil {
		t.Fatalf("seed bob: %v", err)
	}

	ownerListReq := httptest.NewRequest(http.MethodGet, "/users", nil)
	ownerListReq.Header.Set("X-User-ID", "u_owner_list")
	ownerListResp := httptest.NewRecorder()
	r.ServeHTTP(ownerListResp, ownerListReq)
	if ownerListResp.Code != http.StatusOK {
		t.Fatalf("expected owner list 200, got %d body=%s", ownerListResp.Code, ownerListResp.Body.String())
	}

	var ownerList struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.Unmarshal(ownerListResp.Body.Bytes(), &ownerList); err != nil {
		t.Fatalf("decode owner list: %v", err)
	}
	if len(ownerList.Users) < 3 {
		t.Fatalf("expected owner to see all users, got %d", len(ownerList.Users))
	}

	workerListReq := httptest.NewRequest(http.MethodGet, "/users", nil)
	workerListReq.Header.Set("X-User-ID", "u_alice_list")
	workerListResp := httptest.NewRecorder()
	r.ServeHTTP(workerListResp, workerListReq)
	if workerListResp.Code != http.StatusOK {
		t.Fatalf("expected worker list 200, got %d", workerListResp.Code)
	}

	var workerList struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.Unmarshal(workerListResp.Body.Bytes(), &workerList); err != nil {
		t.Fatalf("decode worker list: %v", err)
	}
	if len(workerList.Users) != 1 || workerList.Users[0].ID != "u_alice_list" {
		t.Fatalf("expected worker to see only self, got %+v", workerList.Users)
	}
}

func TestIdentityHandler_ListUsers_ActiveFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	identityService := service.NewIdentityService(userRepo, identityRepo, true)
	h := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		nil,
		true,
		true,
	)

	if _, err := identityService.Register(context.Background(), "u_owner_active", "Owner", "owner", "strong-pass-123"); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	if _, err := identityService.Register(context.Background(), "u_active_human", "Active", "human", "strong-pass-123"); err != nil {
		t.Fatalf("seed active human: %v", err)
	}

	disabled := testutil.SeedAccount(t, userRepo, "u_disabled_human", "Disabled", "human", "")
	actor := domainuser.NewActor("u_owner_active")
	if err := disabled.Disable(actor, time.Now().UTC()); err != nil {
		t.Fatalf("disable account: %v", err)
	}
	if _, err := userRepo.Update(context.Background(), disabled); err != nil {
		t.Fatalf("persist disabled account: %v", err)
	}

	r := gin.New()
	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.GET("/users", h.ListUsers)

	req := httptest.NewRequest(http.MethodGet, "/users?active=true", nil)
	req.Header.Set("X-User-ID", "u_owner_active")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Users []struct {
			ID     string `json:"id"`
			Active bool   `json:"active"`
		} `json:"users"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	for _, user := range body.Users {
		if !user.Active {
			t.Fatalf("expected only active users, got inactive %s", user.ID)
		}
	}
	for _, id := range []string{"u_owner_active", "u_active_human"} {
		found := false
		for _, user := range body.Users {
			if user.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected active user %s in response", id)
		}
	}
	for _, user := range body.Users {
		if user.ID == "u_disabled_human" {
			t.Fatal("disabled user should be filtered out")
		}
	}
}

func TestIdentityHandler_OwnerOnlyRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	identityService := service.NewIdentityService(userRepo, identityRepo, false)
	h := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		nil,
		false,
		false,
	)

	r := gin.New()
	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", false))
	authenticated.POST("/users", h.Register)

	_, err := identityService.Register(context.Background(), "u_owner_gate", "Owner", "owner", "strong-pass-123")
	if err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	_, err = identityService.Register(context.Background(), "u_human_gate", "Human", "human", "strong-pass-123")
	if err != nil {
		t.Fatalf("seed human: %v", err)
	}

	anonReq := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"id":"u_new_001","name":"New","role":"human","password":"strong-pass-123"}`))
	anonReq.Header.Set("Content-Type", "application/json")
	anonResp := httptest.NewRecorder()
	r.ServeHTTP(anonResp, anonReq)
	if anonResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous register 401, got %d", anonResp.Code)
	}

	humanReq := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"id":"u_new_002","name":"New2","role":"human","password":"strong-pass-123"}`))
	humanReq.Header.Set("Content-Type", "application/json")
	humanReq.Header.Set("Authorization", "Bearer "+mustLoginToken(t, identityService, "u_human_gate", "strong-pass-123"))
	humanResp := httptest.NewRecorder()
	r.ServeHTTP(humanResp, humanReq)
	if humanResp.Code != http.StatusForbidden {
		t.Fatalf("expected non-owner register 403, got %d body=%s", humanResp.Code, humanResp.Body.String())
	}

	ownerReq := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"id":"u_new_003","name":"New3","role":"human","password":"strong-pass-123"}`))
	ownerReq.Header.Set("Content-Type", "application/json")
	ownerReq.Header.Set("Authorization", "Bearer "+mustLoginToken(t, identityService, "u_owner_gate", "strong-pass-123"))
	ownerResp := httptest.NewRecorder()
	r.ServeHTTP(ownerResp, ownerReq)
	if ownerResp.Code != http.StatusCreated {
		t.Fatalf("expected owner register 201, got %d body=%s", ownerResp.Code, ownerResp.Body.String())
	}
}

func TestIdentityHandler_APIKeyLifecycleAndAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	identityService := service.NewIdentityService(userRepo, identityRepo, false)
	h := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		nil,
		false,
		false,
	)

	testutil.SeedAccount(t, userRepo, "u_owner_api", "Owner API", "owner", "owner_api_token")
	testutil.SeedAccount(t, userRepo, "u_agent_api", "Hermes Agent", "agent", "")

	r := gin.New()
	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true, identityRepo))
	authenticated.GET("/me", h.Me)
	authenticated.POST("/users/:id/api-keys", h.CreateAPIKey)
	authenticated.GET("/users/:id/api-keys", h.ListAPIKeys)
	authenticated.POST("/users/:id/api-keys/:keyID/revoke", h.RevokeAPIKey)

	createReq := httptest.NewRequest(http.MethodPost, "/users/u_agent_api/api-keys", strings.NewReader(`{"name":"Hermes Prod"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer owner_api_token")
	createResp := httptest.NewRecorder()
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected create api key 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	var created struct {
		APIKey struct {
			ID string `json:"id"`
		} `json:"api_key"`
		Key string `json:"key"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create api key response: %v", err)
	}
	if created.APIKey.ID == "" {
		t.Fatalf("expected api key id in response")
	}
	if created.Key == "" {
		t.Fatalf("expected raw api key in response")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+created.Key)
	meResp := httptest.NewRecorder()
	r.ServeHTTP(meResp, meReq)
	if meResp.Code != http.StatusOK {
		t.Fatalf("expected api key auth 200, got %d body=%s", meResp.Code, meResp.Body.String())
	}

	var meBody struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(meResp.Body.Bytes(), &meBody); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if meBody.User.ID != "u_agent_api" {
		t.Fatalf("expected api key to authenticate u_agent_api, got %q", meBody.User.ID)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/users/u_agent_api/api-keys", nil)
	listReq.Header.Set("Authorization", "Bearer owner_api_token")
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list api keys 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}

	var listed struct {
		APIKeys []struct {
			ID         string     `json:"id"`
			LastUsedAt *time.Time `json:"last_used_at"`
		} `json:"api_keys"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list api keys response: %v", err)
	}
	if len(listed.APIKeys) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(listed.APIKeys))
	}
	if listed.APIKeys[0].ID != created.APIKey.ID {
		t.Fatalf("expected listed api key id %q, got %q", created.APIKey.ID, listed.APIKeys[0].ID)
	}
	if listed.APIKeys[0].LastUsedAt == nil {
		t.Fatalf("expected api key last_used_at to be updated after authenticated request")
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/users/u_agent_api/api-keys/"+created.APIKey.ID+"/revoke", nil)
	revokeReq.Header.Set("Authorization", "Bearer owner_api_token")
	revokeResp := httptest.NewRecorder()
	r.ServeHTTP(revokeResp, revokeReq)
	if revokeResp.Code != http.StatusOK {
		t.Fatalf("expected revoke api key 200, got %d body=%s", revokeResp.Code, revokeResp.Body.String())
	}

	reuseReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	reuseReq.Header.Set("Authorization", "Bearer "+created.Key)
	reuseResp := httptest.NewRecorder()
	r.ServeHTTP(reuseResp, reuseReq)
	if reuseResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked api key 401, got %d body=%s", reuseResp.Code, reuseResp.Body.String())
	}
}

func mustLoginToken(t *testing.T, identityService *service.IdentityService, id, password string) string {
	t.Helper()
	user, err := identityService.Authenticate(context.Background(), id, password)
	if err != nil {
		t.Fatalf("authenticate %s: %v", id, err)
	}
	token, err := auth.GenerateToken(user.ID, user.Role, "test_secret", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}
