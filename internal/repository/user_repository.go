package repository

import "github.com/AsaqeLee/taskflow/internal/model"

type UserRepository interface {
	Create(user model.User) (model.User, error)
	FindByID(id string) (model.User, error)
	FindByToken(token string) (model.User, error)
}
