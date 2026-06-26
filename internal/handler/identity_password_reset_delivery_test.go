package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
)

type recordingPasswordResetDelivery struct {
	notices []PasswordResetNotice
	err     error
}

func (d *recordingPasswordResetDelivery) Deliver(ctx context.Context, notice PasswordResetNotice) error {
	d.notices = append(d.notices, notice)
	return d.err
}

func TestIdentityHandler_RequestPasswordReset_DeliversTokenOutsideDevMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	identityService := service.NewIdentityService(userRepo, identityRepo, false)
	delivery := &recordingPasswordResetDelivery{}
	handler := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		delivery,
		false,
		false,
	)

	if _, err := identityService.Register(context.Background(), "u_reset_webhook", "Reset Webhook", "human", "strong-pass-123"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	router := gin.New()
	router.POST("/auth/password-reset/request", handler.RequestPasswordReset)
	router.POST("/auth/password-reset/confirm", handler.ConfirmPasswordReset)
	router.POST("/auth/login", handler.Login)

	resetReq := httptest.NewRequest(http.MethodPost, "/auth/password-reset/request", strings.NewReader(`{"id":"u_reset_webhook"}`))
	resetReq.Header.Set("Content-Type", "application/json")
	resetResp := httptest.NewRecorder()
	router.ServeHTTP(resetResp, resetReq)
	if resetResp.Code != http.StatusAccepted {
		t.Fatalf("expected reset request status 202, got %d body=%s", resetResp.Code, resetResp.Body.String())
	}

	var resetBody map[string]any
	if err := json.Unmarshal(resetResp.Body.Bytes(), &resetBody); err != nil {
		t.Fatalf("decode reset request response: %v", err)
	}
	if _, ok := resetBody["reset_token"]; ok {
		t.Fatal("did not expect reset_token in non-dev response")
	}
	if len(delivery.notices) != 1 {
		t.Fatalf("expected exactly one password reset delivery, got %d", len(delivery.notices))
	}
	if delivery.notices[0].UserID != "u_reset_webhook" {
		t.Fatalf("expected delivered user id u_reset_webhook, got %q", delivery.notices[0].UserID)
	}
	if delivery.notices[0].Token == "" {
		t.Fatal("expected delivered reset token")
	}

	confirmReq := httptest.NewRequest(
		http.MethodPost,
		"/auth/password-reset/confirm",
		strings.NewReader(`{"id":"u_reset_webhook","token":"`+delivery.notices[0].Token+`","new_password":"new-pass-1234"}`),
	)
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmResp := httptest.NewRecorder()
	router.ServeHTTP(confirmResp, confirmReq)
	if confirmResp.Code != http.StatusOK {
		t.Fatalf("expected reset confirm status 200, got %d body=%s", confirmResp.Code, confirmResp.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"id":"u_reset_webhook","password":"new-pass-1234"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login with reset password succeed, got %d body=%s", loginResp.Code, loginResp.Body.String())
	}
}

func TestIdentityHandler_RequestPasswordReset_AcceptsWhenDeliveryFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	identityService := service.NewIdentityService(userRepo, identityRepo, false)
	delivery := &recordingPasswordResetDelivery{err: errors.New("delivery failed")}
	handler := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		observability.NewMetrics(),
		delivery,
		false,
		false,
	)

	if _, err := identityService.Register(context.Background(), "u_reset_failed", "Reset Failed", "human", "strong-pass-123"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	router := gin.New()
	router.POST("/auth/password-reset/request", handler.RequestPasswordReset)

	resetReq := httptest.NewRequest(http.MethodPost, "/auth/password-reset/request", strings.NewReader(`{"id":"u_reset_failed"}`))
	resetReq.Header.Set("Content-Type", "application/json")
	resetResp := httptest.NewRecorder()
	router.ServeHTTP(resetResp, resetReq)
	if resetResp.Code != http.StatusAccepted {
		t.Fatalf("expected reset request status 202 on delivery failure, got %d body=%s", resetResp.Code, resetResp.Body.String())
	}
	if len(delivery.notices) != 1 {
		t.Fatalf("expected delivery attempt to be recorded, got %d", len(delivery.notices))
	}
}
