package repository

import (
	"context"
	"errors"

	"github.com/AsaqeLee/taskflow/internal/model"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserNotFoundByToken = errors.New("user not found by token")
var ErrUserAlreadyExists = errors.New("user already exists")

type UserRepository interface {
	Create(ctx context.Context, user model.User) (model.User, error)
	FindByID(ctx context.Context, id string) (model.User, error)
	FindByToken(ctx context.Context, token string) (model.User, error)
}
