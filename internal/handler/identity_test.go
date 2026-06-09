package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

func TestIdentityHandler_RegisterAndMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	h := NewIdentityHandler(userRepo, "test_secret")

	r := gin.New()
	r.POST("/users", h.Register)

	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.GET("/me", h.Me)

	// 1. Register a new user u_test_003
	body := `{"id": "u_test_003", "name": "Dynamic User", "role": "human"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}

	var registerResp struct {
		User model.User `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &registerResp); err != nil {
		t.Fatalf("failed to parse registration response: %v", err)
	}

	u := registerResp.User
	if u.ID != "u_test_003" || u.Name != "Dynamic User" || u.Role != "human" {
		t.Fatalf("unexpected registered user: %+v", u)
	}
	if u.Token == "" {
		t.Fatalf("expected generated token to be populated")
	}

	// 2. Fetch /me profile with the generated token
	reqMe := httptest.NewRequest(http.MethodGet, "/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+u.Token)
	wMe := httptest.NewRecorder()
	r.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", wMe.Code, wMe.Body.String())
	}

	var meResp struct {
		User model.User `json:"user"`
	}
	if err := json.Unmarshal(wMe.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("failed to parse me response: %v", err)
	}

	if meResp.User.ID != "u_test_003" || meResp.User.Token != u.Token {
		t.Fatalf("unexpected user in profile response: %+v", meResp.User)
	}

	// 3. Fetch /me profile with missing token
	reqMeMissing := httptest.NewRequest(http.MethodGet, "/me", nil)
	wMeMissing := httptest.NewRecorder()
	r.ServeHTTP(wMeMissing, reqMeMissing)

	if wMeMissing.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", wMeMissing.Code)
	}
}
