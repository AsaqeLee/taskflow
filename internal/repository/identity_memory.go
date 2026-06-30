package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
)

type MemoryIdentityRepository struct {
	mu                  sync.RWMutex
	refreshTokens       map[string]domainidentity.RefreshToken
	passwordResetTokens map[string]domainidentity.PasswordResetToken
	apiKeys             map[string]domainidentity.APIKey
	nextAPIKeyID        int
}

func NewMemoryIdentityRepository() *MemoryIdentityRepository {
	return &MemoryIdentityRepository{
		refreshTokens:       make(map[string]domainidentity.RefreshToken),
		passwordResetTokens: make(map[string]domainidentity.PasswordResetToken),
		apiKeys:             make(map[string]domainidentity.APIKey),
		nextAPIKeyID:        1,
	}
}

func (r *MemoryIdentityRepository) SaveRefreshToken(ctx context.Context, token domainidentity.RefreshToken) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	sweepExpiredRefreshTokensLocked(time.Now().UTC(), r.refreshTokens)
	r.refreshTokens[token.TokenHash()] = token
	return nil
}

func (r *MemoryIdentityRepository) FindRefreshToken(ctx context.Context, tokenHash string) (domainidentity.RefreshToken, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainidentity.RefreshToken{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	sweepExpiredRefreshTokensLocked(now, r.refreshTokens)

	token, exists := r.refreshTokens[tokenHash]
	if !exists {
		return domainidentity.RefreshToken{}, ErrRefreshTokenNotFound
	}

	return token, nil
}

func (r *MemoryIdentityRepository) RevokeRefreshToken(ctx context.Context, tokenHash string, revokedAt time.Time, replacedByHash string) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	token, exists := r.refreshTokens[tokenHash]
	if !exists {
		return ErrRefreshTokenNotFound
	}

	r.refreshTokens[tokenHash] = token.WithRevocation(revokedAt, replacedByHash)
	return nil
}

func (r *MemoryIdentityRepository) RevokeUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenHash, token := range r.refreshTokens {
		if token.UserID() != userID || token.IsRevoked() {
			continue
		}
		r.refreshTokens[tokenHash] = token.WithRevocation(revokedAt, "")
	}

	return nil
}

func (r *MemoryIdentityRepository) SavePasswordResetToken(ctx context.Context, token domainidentity.PasswordResetToken) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	sweepExpiredPasswordResetTokensLocked(time.Now().UTC(), r.passwordResetTokens)
	r.passwordResetTokens[token.TokenHash()] = token
	return nil
}

func (r *MemoryIdentityRepository) FindPasswordResetToken(ctx context.Context, tokenHash string) (domainidentity.PasswordResetToken, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainidentity.PasswordResetToken{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	sweepExpiredPasswordResetTokensLocked(now, r.passwordResetTokens)

	token, exists := r.passwordResetTokens[tokenHash]
	if !exists {
		return domainidentity.PasswordResetToken{}, ErrPasswordResetTokenNotFound
	}

	return token, nil
}

func (r *MemoryIdentityRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string, consumedAt time.Time) (domainidentity.PasswordResetToken, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainidentity.PasswordResetToken{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	sweepExpiredPasswordResetTokensLocked(now, r.passwordResetTokens)

	token, exists := r.passwordResetTokens[tokenHash]
	if !exists {
		return domainidentity.PasswordResetToken{}, ErrPasswordResetTokenNotFound
	}

	token = token.Consume(consumedAt)
	r.passwordResetTokens[tokenHash] = token
	return token, nil
}

func (r *MemoryIdentityRepository) DeletePasswordResetTokensByUser(ctx context.Context, userID string) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenHash, token := range r.passwordResetTokens {
		if token.UserID() == userID {
			delete(r.passwordResetTokens, tokenHash)
		}
	}

	return nil
}

func (r *MemoryIdentityRepository) SaveAPIKey(ctx context.Context, key domainidentity.APIKey) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if key.ID() == "" {
		key = key.AssignID(fmt.Sprintf("ak_%03d", r.nextAPIKeyID))
		r.nextAPIKeyID++
	}

	r.apiKeys[key.KeyHash()] = key
	return nil
}

func (r *MemoryIdentityRepository) FindAPIKey(ctx context.Context, keyHash string) (domainidentity.APIKey, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainidentity.APIKey{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	key, exists := r.apiKeys[keyHash]
	if !exists {
		return domainidentity.APIKey{}, ErrAPIKeyNotFound
	}

	return key, nil
}

func (r *MemoryIdentityRepository) ListAPIKeysByUser(ctx context.Context, userID string) ([]domainidentity.APIKey, error) {
	if err := errIfContextDone(ctx); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]domainidentity.APIKey, 0, len(r.apiKeys))
	for _, key := range r.apiKeys {
		if key.UserID() == userID {
			keys = append(keys, key)
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].CreatedAt().Equal(keys[j].CreatedAt()) {
			return keys[i].ID() < keys[j].ID()
		}
		return keys[i].CreatedAt().After(keys[j].CreatedAt())
	})

	return keys, nil
}

func (r *MemoryIdentityRepository) TouchAPIKeyLastUsed(ctx context.Context, keyID string, lastUsedAt time.Time) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for keyHash, key := range r.apiKeys {
		if key.ID() != keyID {
			continue
		}
		r.apiKeys[keyHash] = key.MarkUsed(lastUsedAt)
		return nil
	}

	return ErrAPIKeyNotFound
}

func (r *MemoryIdentityRepository) RevokeAPIKey(ctx context.Context, userID, keyID string, revokedAt time.Time) (domainidentity.APIKey, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainidentity.APIKey{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for keyHash, key := range r.apiKeys {
		if key.ID() != keyID || key.UserID() != userID {
			continue
		}
		if key.IsRevoked() {
			return key, nil
		}
		key = key.Revoke(revokedAt)
		r.apiKeys[keyHash] = key
		return key, nil
	}

	return domainidentity.APIKey{}, ErrAPIKeyNotFound
}

func sweepExpiredRefreshTokensLocked(now time.Time, tokens map[string]domainidentity.RefreshToken) {
	for tokenHash, token := range tokens {
		if token.IsExpired(now) {
			delete(tokens, tokenHash)
		}
	}
}

func sweepExpiredPasswordResetTokensLocked(now time.Time, tokens map[string]domainidentity.PasswordResetToken) {
	for tokenHash, token := range tokens {
		if token.IsExpired(now) {
			delete(tokens, tokenHash)
		}
	}
}
