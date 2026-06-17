package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const identityTestMongoURIEnv = "TASKFLOW_MONGO_TEST_URI"

func TestIdentityHandler_MongoRefreshReuseResetAndSessionRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uri := os.Getenv(identityTestMongoURIEnv)
	if uri == "" {
		t.Skipf("%s not set; skipping Mongo identity integration test", identityTestMongoURIEnv)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("taskflow_identity_test_" + bson.NewObjectID().Hex())
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
	})

	userRepo := repository.NewMongoUserRepository(db.Collection("users"))
	identityRepo := repository.NewMongoIdentityRepository(
		db.Collection("refresh_tokens"),
		db.Collection("password_reset_tokens"),
	)
	metrics := observability.NewMetrics()
	dbClient := &database.Client{Mongo: client, DBName: db.Name()}
	identityService := service.NewIdentityService(userRepo, identityRepo, true, dbClient)
	handler := NewIdentityHandler(
		identityService,
		"test_secret",
		time.Hour,
		24*time.Hour,
		time.Hour,
		middleware.NewMemoryRateLimiter(10, 5*time.Minute),
		middleware.NewMemoryRateLimiter(10, 15*time.Minute),
		metrics,
		true,
		true,
	)

	router := gin.New()
	router.POST("/users", handler.Register)
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/password-reset/request", handler.RequestPasswordReset)
	router.POST("/auth/password-reset/confirm", handler.ConfirmPasswordReset)

	authenticated := router.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.POST("/users/:id/disable", handler.DisableAccount)
	authenticated.POST("/users/:id/revoke-sessions", handler.RevokeSessions)

	doJSON := func(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}

	registerResp := doJSON(http.MethodPost, "/users", `{"id":"u_mongo_identity","name":"Mongo Identity","role":"human","password":"strong-pass-123"}`, nil)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", registerResp.Code, registerResp.Body.String())
	}

	var registered struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(registerResp.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	refreshResp := doJSON(http.MethodPost, "/auth/refresh", `{"refresh_token":"`+registered.RefreshToken+`"}`, nil)
	if refreshResp.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refreshResp.Code, refreshResp.Body.String())
	}

	var refreshed struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(refreshResp.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}

	reuseResp := doJSON(http.MethodPost, "/auth/refresh", `{"refresh_token":"`+registered.RefreshToken+`"}`, nil)
	if reuseResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected reuse to fail with 401, got %d body=%s", reuseResp.Code, reuseResp.Body.String())
	}
	var reuseError struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(reuseResp.Body.Bytes(), &reuseError); err != nil {
		t.Fatalf("decode reuse response: %v", err)
	}
	if reuseError.Error.Code != "refresh_token_reused" {
		t.Fatalf("expected refresh_token_reused code, got %s", reuseError.Error.Code)
	}

	refreshAfterReuseResp := doJSON(http.MethodPost, "/auth/refresh", `{"refresh_token":"`+refreshed.RefreshToken+`"}`, nil)
	if refreshAfterReuseResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected rotated token to be revoked after reuse detection, got %d body=%s", refreshAfterReuseResp.Code, refreshAfterReuseResp.Body.String())
	}

	resetRequestResp := doJSON(http.MethodPost, "/auth/password-reset/request", `{"id":"u_mongo_identity"}`, nil)
	if resetRequestResp.Code != http.StatusAccepted {
		t.Fatalf("expected reset request status 202, got %d body=%s", resetRequestResp.Code, resetRequestResp.Body.String())
	}
	var resetRequestBody struct {
		ResetToken string `json:"reset_token"`
	}
	if err := json.Unmarshal(resetRequestResp.Body.Bytes(), &resetRequestBody); err != nil {
		t.Fatalf("decode reset request response: %v", err)
	}
	if resetRequestBody.ResetToken == "" {
		t.Fatalf("expected dev-mode reset token in response")
	}

	resetConfirmResp := doJSON(http.MethodPost, "/auth/password-reset/confirm", `{"id":"u_mongo_identity","token":"`+resetRequestBody.ResetToken+`","new_password":"new-pass-1234"}`, nil)
	if resetConfirmResp.Code != http.StatusOK {
		t.Fatalf("expected reset confirm status 200, got %d body=%s", resetConfirmResp.Code, resetConfirmResp.Body.String())
	}

	loginResp := doJSON(http.MethodPost, "/auth/login", `{"id":"u_mongo_identity","password":"new-pass-1234"}`, nil)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login with reset password to succeed, got %d body=%s", loginResp.Code, loginResp.Body.String())
	}
	var loggedIn struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loggedIn); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	revokeResp := doJSON(http.MethodPost, "/users/u_mongo_identity/revoke-sessions", "", map[string]string{
		"Authorization": "Bearer " + loggedIn.AccessToken,
	})
	if revokeResp.Code != http.StatusOK {
		t.Fatalf("expected revoke sessions status 200, got %d body=%s", revokeResp.Code, revokeResp.Body.String())
	}

	refreshAfterRevokeResp := doJSON(http.MethodPost, "/auth/refresh", `{"refresh_token":"`+loggedIn.RefreshToken+`"}`, nil)
	if refreshAfterRevokeResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected refresh to fail after session revocation, got %d body=%s", refreshAfterRevokeResp.Code, refreshAfterRevokeResp.Body.String())
	}

	disableResp := doJSON(http.MethodPost, "/users/u_mongo_identity/disable", "", map[string]string{
		"Authorization": "Bearer " + loggedIn.AccessToken,
	})
	if disableResp.Code != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d body=%s", disableResp.Code, disableResp.Body.String())
	}

	loginAfterDisableResp := doJSON(http.MethodPost, "/auth/login", `{"id":"u_mongo_identity","password":"new-pass-1234"}`, nil)
	if loginAfterDisableResp.Code != http.StatusForbidden {
		t.Fatalf("expected disabled login status 403, got %d body=%s", loginAfterDisableResp.Code, loginAfterDisableResp.Body.String())
	}

	if metrics == nil {
		t.Fatalf("expected metrics to be initialized")
	}
}
