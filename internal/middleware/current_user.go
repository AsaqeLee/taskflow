package middleware

import (
	"net/http"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	"github.com/AsaqeLee/taskflow/internal/httpapi"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/requestmeta"
	"github.com/gin-gonic/gin"
)

const currentUserKey = "currentUser"

func FixedTestUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := model.User{
			ID:     "u_test_001",
			Name:   "Test User",
			Role:   "owner",
			Token:  "token_creator",
			Active: true,
		}
		c.Set(currentUserKey, user)
		c.Request = c.Request.WithContext(requestmeta.WithUserID(c.Request.Context(), user.ID))
		c.Next()
	}
}

func UserAuth(userRepo ports.UserRepository, jwtSecret string, devMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		var user model.User
		var authenticated bool

		if token != "" {
			// 1. Try to validate token as JWT
			claims, errJwt := auth.ValidateToken(token, jwtSecret)
			if errJwt == nil && claims != nil {
				// JWT is valid, find the user by ID
				account, err := userRepo.FindByID(c.Request.Context(), claims.UserID)
				if err == nil {
					user = model.UserFromAccount(account)
					authenticated = true
				}
			}

			// 2. If JWT fails, and devMode is true, fall back to matching plain/legacy token
			if !authenticated && devMode {
				account, err := userRepo.FindByToken(c.Request.Context(), token)
				if err == nil {
					user = model.UserFromAccount(account)
					authenticated = true
				}
			}
		}

		// 3. If still not authenticated, and devMode is true, check X-User-ID header
		if !authenticated && devMode {
			userID := c.GetHeader("X-User-ID")
			if userID != "" {
				account, err := userRepo.FindByID(c.Request.Context(), userID)
				if err == nil {
					user = model.UserFromAccount(account)
					authenticated = true
				}
			}
		}

		if !authenticated {
			httpapi.Unauthorized(c, "unauthorized", "invalid or missing credentials")
			return
		}
		if !user.Active {
			httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account is disabled")
			return
		}

		c.Set(currentUserKey, user)
		c.Request = c.Request.WithContext(requestmeta.WithUserID(c.Request.Context(), user.ID))
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (model.User, bool) {
	value, exists := c.Get(currentUserKey)
	if !exists {
		return model.User{}, false
	}

	user, ok := value.(model.User)
	if !ok {
		return model.User{}, false
	}

	return user, true
}
