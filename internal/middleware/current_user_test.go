package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

func TestUserAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewMemoryUserRepository()
	_, err := userRepo.Create(context.Background(), model.User{
		ID:    "u_test_001",
		Name:  "Test Creator",
		Role:  "owner",
		Token: "token_creator",
	})
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

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
			r.Use(UserAuth(userRepo))
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
