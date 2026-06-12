package repository

import (
	"context"
	"errors"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserNotFoundByToken = errors.New("user not found by token")
var ErrUserAlreadyExists = errors.New("user already exists")
var ErrRefreshTokenNotFound = errors.New("refresh token not found")
var ErrPasswordResetTokenNotFound = errors.New("password reset token not found")

type UserRepository interface {
	Create(ctx context.Context, user model.User) (model.User, error)
	FindByID(ctx context.Context, id string) (model.User, error)
	FindByToken(ctx context.Context, token string) (model.User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) (model.User, error)
	Disable(ctx context.Context, id, disabledBy string, disabledAt time.Time) (model.User, error)
}

type IdentityRepository interface {
	SaveRefreshToken(ctx context.Context, token model.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (model.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string, revokedAt time.Time, replacedByHash string) error
	RevokeUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error
	SavePasswordResetToken(ctx context.Context, token model.PasswordResetToken) error
	FindPasswordResetToken(ctx context.Context, tokenHash string) (model.PasswordResetToken, error)
	ConsumePasswordResetToken(ctx context.Context, tokenHash string, consumedAt time.Time) (model.PasswordResetToken, error)
	DeletePasswordResetTokensByUser(ctx context.Context, userID string) error
}
