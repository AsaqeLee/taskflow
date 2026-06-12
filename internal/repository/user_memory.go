package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

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

func (r *MemoryUserRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user.ID == "" {
		user.ID = fmt.Sprintf("u_%03d", r.nextID)
		r.nextID++
	}

	if _, exists := r.users[user.ID]; exists {
		return model.User{}, ErrUserAlreadyExists
	}
	if user.CreatedAt.IsZero() {
		now := time.Now().UTC()
		user.CreatedAt = now
		user.UpdatedAt = now
	}
	if !user.Active && user.DisabledAt == nil {
		user.Active = true
	}

	r.users[user.ID] = user
	return user, nil
}

func (r *MemoryUserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return model.User{}, ErrUserNotFound
	}

	return user, nil
}

func (r *MemoryUserRepository) FindByToken(ctx context.Context, token string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Token == token {
			return user, nil
		}
	}

	return model.User{}, ErrUserNotFoundByToken
}

func (r *MemoryUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) (model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return model.User{}, ErrUserNotFound
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = updatedAt
	r.users[id] = user
	return user, nil
}

func (r *MemoryUserRepository) Disable(ctx context.Context, id, disabledBy string, disabledAt time.Time) (model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return model.User{}, ErrUserNotFound
	}

	user.Active = false
	user.DisabledBy = disabledBy
	user.DisabledAt = &disabledAt
	user.UpdatedAt = disabledAt
	r.users[id] = user
	return user, nil
}
