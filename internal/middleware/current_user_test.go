package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestUserAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	testutil.SeedAccount(t, userRepo, "u_test_001", "Test Creator", "owner", "token_creator")

	tests := []struct {
		name           string
		setupHeaders   func(req *http.Request)
		expectedStatus int
		expectedUserID string
	}{
		{
			name: "Valid Bearer Token",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token_creator")
			},
			expectedStatus: http.StatusOK,
			expectedUserID: "u_test_001",
		},
		{
			name: "Valid Token Without Bearer Prefix",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "token_creator")
			},
			expectedStatus: http.StatusOK,
			expectedUserID: "u_test_001",
		},
		{
			name: "Valid X-User-ID Fallback Header",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-User-ID", "u_test_001")
			},
			expectedStatus: http.StatusOK,
			expectedUserID: "u_test_001",
		},
		{
			name: "Missing Authentication Credentials",
			setupHeaders: func(req *http.Request) {
				// No headers set
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid Token",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid_token")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid X-User-ID",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-User-ID", "invalid_id")
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(UserAuth(userRepo, "test_secret", true))
			r.GET("/test-auth", func(c *gin.Context) {
				user, exists := CurrentUser(c)
				if !exists {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "user missing from context"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"userID": user.ID})
			})

			req := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
			if tt.setupHeaders != nil {
				tt.setupHeaders(req)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp["userID"] != tt.expectedUserID {
					t.Fatalf("expected user ID %q, got %q", tt.expectedUserID, resp["userID"])
				}
			}
		})
	}
}

func TestUserAuthMiddleware_JWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	userID := "u_test_jwt"
	testutil.SeedAccount(t, userRepo, userID, "JWT User", "owner", "some_legacy_token")

	secret := "my_jwt_test_secret"

	// 1. Generate valid token
	validToken, err := auth.GenerateToken(userID, "owner", secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// 2. Generate expired token
	expiredToken, err := auth.GenerateToken(userID, "owner", secret, -time.Hour)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	// 3. Generate token with different secret
	wrongSecretToken, err := auth.GenerateToken(userID, "owner", "wrong_secret", time.Hour)
	if err != nil {
		t.Fatalf("failed to generate wrong secret token: %v", err)
	}

	tests := []struct {
		name           string
		devMode        bool
		setupHeaders   func(req *http.Request)
		expectedStatus int
	}{
		{
			name:    "Valid JWT in Production Mode",
			devMode: false,
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+validToken)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "Expired JWT in Production Mode",
			devMode: false,
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+expiredToken)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "Wrong Secret JWT in Production Mode",
			devMode: false,
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+wrongSecretToken)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "X-User-ID Blocked in Production Mode",
			devMode: false,
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-User-ID", userID)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "Legacy Plain Token Blocked in Production Mode",
			devMode: false,
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer some_legacy_token")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "Legacy Plain Token Allowed in Dev Mode",
			devMode: true,
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer some_legacy_token")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "X-User-ID Allowed in Dev Mode",
			devMode: true,
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-User-ID", userID)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(UserAuth(userRepo, secret, tt.devMode))
			r.GET("/test-auth", func(c *gin.Context) {
				user, exists := CurrentUser(c)
				if !exists {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "user missing"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"userID": user.ID})
			})

			req := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
			if tt.setupHeaders != nil {
				tt.setupHeaders(req)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d body=%s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestUserAuthMiddleware_RejectsDisabledAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	created := testutil.SeedAccount(t, userRepo, "u_disabled_001", "Disabled User", "human", "disabled_legacy_token")

	now := time.Now().UTC()
	account, err := userRepo.FindByID(context.Background(), created.ID())
	if err != nil {
		t.Fatalf("failed to load user: %v", err)
	}
	if err := account.Disable(domainuser.NewActor("u_admin"), now); err != nil {
		t.Fatalf("failed to disable user: %v", err)
	}
	if _, err := userRepo.Update(context.Background(), account); err != nil {
		t.Fatalf("failed to persist disabled user: %v", err)
	}

	secret := "my_jwt_test_secret"
	accessToken, err := auth.GenerateToken(created.ID(), created.Role().String(), secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	r := gin.New()
	r.Use(UserAuth(userRepo, secret, false))
	r.GET("/test-auth", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected disabled account to be rejected with 403, got %d body=%s", w.Code, w.Body.String())
	}
}
