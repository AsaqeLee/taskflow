package repository

import (
	"context"
	"sync"
	"time"

	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
)

type MemoryIdentityRepository struct {
	mu                  sync.RWMutex
	refreshTokens       map[string]domainidentity.RefreshToken
	passwordResetTokens map[string]domainidentity.PasswordResetToken
}

func NewMemoryIdentityRepository() *MemoryIdentityRepository {
	return &MemoryIdentityRepository{
		refreshTokens:       make(map[string]domainidentity.RefreshToken),
		passwordResetTokens: make(map[string]domainidentity.PasswordResetToken),
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

	token = token.WithRevocation(revokedAt, replacedByHash)
	r.refreshTokens[tokenHash] = token
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
		token = token.WithRevocation(revokedAt, "")
		r.refreshTokens[tokenHash] = token
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
