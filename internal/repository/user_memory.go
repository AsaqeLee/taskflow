package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

type MemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[string]domainuser.Account
	nextID int
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:  make(map[string]domainuser.Account),
		nextID: 1,
	}
}

func (r *MemoryUserRepository) Create(ctx context.Context, account domainuser.Account) (domainuser.Account, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainuser.Account{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if account.ID() == "" {
		account = account.AssignID(fmt.Sprintf("u_%03d", r.nextID))
		r.nextID++
	}
	if _, exists := r.users[account.ID()]; exists {
		return domainuser.Account{}, ErrUserAlreadyExists
	}

	now := time.Now().UTC()
	if account.CreatedAt().IsZero() {
		account = domainuser.Restore(
			account.ID(),
			account.Name(),
			account.Role(),
			account.PasswordHash(),
			account.LegacyToken(),
			true,
			nil,
			"",
			now,
			now,
		)
	}

	r.users[account.ID()] = account
	return account, nil
}

func (r *MemoryUserRepository) FindByID(ctx context.Context, id string) (domainuser.Account, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainuser.Account{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	account, exists := r.users[id]
	if !exists {
		return domainuser.Account{}, ErrUserNotFound
	}
	return account, nil
}

func (r *MemoryUserRepository) FindByToken(ctx context.Context, token string) (domainuser.Account, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainuser.Account{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, account := range r.users {
		if account.LegacyToken() == token {
			return account, nil
		}
	}
	return domainuser.Account{}, ErrUserNotFoundByToken
}

func (r *MemoryUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) (domainuser.Account, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainuser.Account{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.users[id]
	if !exists {
		return domainuser.Account{}, ErrUserNotFound
	}

	account = domainuser.Restore(
		account.ID(),
		account.Name(),
		account.Role(),
		passwordHash,
		account.LegacyToken(),
		account.Active(),
		account.DisabledAt(),
		account.DisabledBy(),
		account.CreatedAt(),
		updatedAt,
	)
	r.users[id] = account
	return account, nil
}

func (r *MemoryUserRepository) Disable(ctx context.Context, id, disabledBy string, disabledAt time.Time) (domainuser.Account, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainuser.Account{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.users[id]
	if !exists {
		return domainuser.Account{}, ErrUserNotFound
	}

	account = domainuser.Restore(
		account.ID(),
		account.Name(),
		account.Role(),
		account.PasswordHash(),
		account.LegacyToken(),
		false,
		&disabledAt,
		disabledBy,
		account.CreatedAt(),
		disabledAt,
	)
	r.users[id] = account
	return account, nil
}

func (r *MemoryUserRepository) Update(ctx context.Context, account domainuser.Account) (domainuser.Account, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainuser.Account{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[account.ID()]; !exists {
		return domainuser.Account{}, ErrUserNotFound
	}

	r.users[account.ID()] = account
	return account, nil
}

func (r *MemoryUserRepository) List(ctx context.Context, activeOnly bool) ([]domainuser.Account, error) {
	if err := errIfContextDone(ctx); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domainuser.Account, 0, len(r.users))
	for _, account := range r.users {
		if activeOnly && !account.IsActive() {
			continue
		}
		result = append(result, account)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})
	return result, nil
}
