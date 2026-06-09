package repository

import (
	"context"

	"github.com/AsaqeLee/taskflow/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user model.User) (model.User, error)
	FindByID(ctx context.Context, id string) (model.User, error)
	FindByToken(ctx context.Context, token string) (model.User, error)
}
