package middleware

import (
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/gin-gonic/gin"
)

const currentUserKey = "currentUser"

func FixedTestUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(currentUserKey, model.User{
			ID:   "u_test_001",
			Name: "Test User",
			Role: "owner",
		})
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
