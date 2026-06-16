package ports

import (
	"context"
	"errors"
	"time"

	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserNotFoundByToken = errors.New("user not found by token")
	ErrUserAlreadyExists   = errors.New("user already exists")
)

type UserRepository interface {
	Create(ctx context.Context, account domainuser.Account) (domainuser.Account, error)
	FindByID(ctx context.Context, id string) (domainuser.Account, error)
	FindByToken(ctx context.Context, token string) (domainuser.Account, error)
	UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) (domainuser.Account, error)
	Disable(ctx context.Context, id, disabledBy string, disabledAt time.Time) (domainuser.Account, error)
	Update(ctx context.Context, account domainuser.Account) (domainuser.Account, error)
}
