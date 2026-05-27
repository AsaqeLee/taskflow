package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/AsaqeLee/taskflow/internal/model"
)

type MemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[string]model.User
	nextID int
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:  make(map[string]model.User),
		nextID: 1,
	}
}

func (r *MemoryUserRepository) Create(user model.User) (model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user.ID == "" {
		user.ID = fmt.Sprintf("u_%03d", r.nextID)
		r.nextID++
	}

	if _, exists := r.users[user.ID]; exists {
		return model.User{}, errors.New("user already exists")
	}

	r.users[user.ID] = user
	return user, nil
}

func (r *MemoryUserRepository) FindByID(id string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return model.User{}, errors.New("user not found")
	}

	return user, nil
}

func (r *MemoryUserRepository) FindByToken(token string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Token == token {
			return user, nil
		}
	}

	return model.User{}, errors.New("user not found by token")
}
