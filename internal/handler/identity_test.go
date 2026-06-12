package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

func TestIdentityHandler_RegisterAndMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	h := NewIdentityHandler(userRepo, "test_secret", time.Hour, true)

	r := gin.New()
	r.POST("/users", h.Register)
	r.POST("/auth/login", h.Login)

	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.GET("/me", h.Me)

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
		AccessToken string `json:"access_token"`
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
