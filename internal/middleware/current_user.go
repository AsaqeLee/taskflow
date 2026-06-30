package middleware

import (
	"errors"
	"net/http"
	"time"

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

func UserAuth(userRepo ports.UserRepository, jwtSecret string, devMode bool, identityRepos ...ports.IdentityRepository) gin.HandlerFunc {
	var identityRepo ports.IdentityRepository
	if len(identityRepos) > 0 {
		identityRepo = identityRepos[0]
	}

	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		var user model.User
		var authenticated bool
		var apiKeyID string

		if token != "" {
			claims, errJWT := auth.ValidateToken(token, jwtSecret)
			if errJWT == nil && claims != nil {
				account, err := userRepo.FindByID(c.Request.Context(), claims.UserID)
				if err == nil {
					user = model.UserFromAccount(account)
					authenticated = true
				}
			}

			if !authenticated && identityRepo != nil {
				key, err := identityRepo.FindAPIKey(c.Request.Context(), auth.HashOpaqueToken(token))
				switch {
				case err == nil:
					now := time.Now().UTC()
					if !key.IsRevoked() && !key.IsExpired(now) {
						account, accountErr := userRepo.FindByID(c.Request.Context(), key.UserID())
						if accountErr == nil {
							user = model.UserFromAccount(account)
							authenticated = true
							apiKeyID = key.ID()
						}
					}
				case !errors.Is(err, ports.ErrAPIKeyNotFound):
					httpapi.WriteError(c, http.StatusInternalServerError, "api_key_lookup_failed", "failed to resolve api key")
					return
				}
			}

			if !authenticated && devMode {
				account, err := userRepo.FindByToken(c.Request.Context(), token)
				if err == nil {
					user = model.UserFromAccount(account)
					authenticated = true
				}
			}
		}

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
			httpapi.AbortError(c, http.StatusForbidden, "account_disabled", "account disabled")
			return
		}
		if apiKeyID != "" && identityRepo != nil {
			_ = identityRepo.TouchAPIKeyLastUsed(c.Request.Context(), apiKeyID, time.Now().UTC())
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
