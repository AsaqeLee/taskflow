package middleware

import (
	"net/http"

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

func UserAuth(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		var user model.User
		var err error
		var found bool

		if token != "" {
			user, err = userRepo.FindByToken(token)
			if err == nil {
				found = true
			}
		}

		if !found {
			userID := c.GetHeader("X-User-ID")
			if userID != "" {
				user, err = userRepo.FindByID(userID)
				if err == nil {
					found = true
				}
			}
		}

		if !found {
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
