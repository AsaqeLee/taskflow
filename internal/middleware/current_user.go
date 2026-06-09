package middleware

import (
	"net/http"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/gin-gonic/gin"
)

const currentUserKey = "currentUser"

func FixedTestUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(currentUserKey, model.User{
			ID:    "u_test_001",
			Name:  "Test User",
			Role:  "owner",
			Token: "token_creator",
		})
		c.Next()
	}
}

func UserAuth(userRepo repository.UserRepository, jwtSecret string, devMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		var user model.User
		var err error
		var authenticated bool

		if token != "" {
			// 1. Try to validate token as JWT
			claims, errJwt := auth.ValidateToken(token, jwtSecret)
			if errJwt == nil && claims != nil {
				// JWT is valid, find the user by ID
				user, err = userRepo.FindByID(c.Request.Context(), claims.UserID)
				if err == nil {
					authenticated = true
				}
			}

			// 2. If JWT fails, and devMode is true, fall back to matching plain/legacy token
			if !authenticated && devMode {
				user, err = userRepo.FindByToken(c.Request.Context(), token)
				if err == nil {
					authenticated = true
				}
			}
		}

		// 3. If still not authenticated, and devMode is true, check X-User-ID header
		if !authenticated && devMode {
			userID := c.GetHeader("X-User-ID")
			if userID != "" {
				user, err = userRepo.FindByID(c.Request.Context(), userID)
				if err == nil {
					authenticated = true
				}
			}
		}

		if !authenticated {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid or missing token/user id"})
			c.Abort()
			return
		}

		c.Set(currentUserKey, user)
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
