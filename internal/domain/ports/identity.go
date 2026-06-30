package ports

import (
	"context"
	"errors"
	"time"

	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
)

var (
	ErrRefreshTokenNotFound       = errors.New("refresh token not found")
	ErrPasswordResetTokenNotFound = errors.New("password reset token not found")
	ErrAPIKeyNotFound             = errors.New("api key not found")
)

type IdentityRepository interface {
	SaveRefreshToken(ctx context.Context, token domainidentity.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (domainidentity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string, revokedAt time.Time, replacedByHash string) error
	RevokeUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error
	SavePasswordResetToken(ctx context.Context, token domainidentity.PasswordResetToken) error
	FindPasswordResetToken(ctx context.Context, tokenHash string) (domainidentity.PasswordResetToken, error)
	ConsumePasswordResetToken(ctx context.Context, tokenHash string, consumedAt time.Time) (domainidentity.PasswordResetToken, error)
	DeletePasswordResetTokensByUser(ctx context.Context, userID string) error
	SaveAPIKey(ctx context.Context, key domainidentity.APIKey) error
	FindAPIKey(ctx context.Context, keyHash string) (domainidentity.APIKey, error)
	ListAPIKeysByUser(ctx context.Context, userID string) ([]domainidentity.APIKey, error)
	TouchAPIKeyLastUsed(ctx context.Context, keyID string, lastUsedAt time.Time) error
	RevokeAPIKey(ctx context.Context, userID, keyID string, revokedAt time.Time) (domainidentity.APIKey, error)
}
