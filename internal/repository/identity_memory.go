package repository

import (
	"context"
	"sync"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
)

type MemoryIdentityRepository struct {
	mu                  sync.RWMutex
	refreshTokens       map[string]model.RefreshToken
	passwordResetTokens map[string]model.PasswordResetToken
}

func NewMemoryIdentityRepository() *MemoryIdentityRepository {
	return &MemoryIdentityRepository{
		refreshTokens:       make(map[string]model.RefreshToken),
		passwordResetTokens: make(map[string]model.PasswordResetToken),
	}
}

func (r *MemoryIdentityRepository) SaveRefreshToken(ctx context.Context, token model.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sweepExpiredRefreshTokensLocked(time.Now().UTC(), r.refreshTokens)
	r.refreshTokens[token.TokenHash] = token
	return nil
}

func (r *MemoryIdentityRepository) FindRefreshToken(ctx context.Context, tokenHash string) (model.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	sweepExpiredRefreshTokensLocked(now, r.refreshTokens)
	token, exists := r.refreshTokens[tokenHash]
	if !exists {
		return model.RefreshToken{}, ErrRefreshTokenNotFound
	}
	return token, nil
}

func (r *MemoryIdentityRepository) RevokeRefreshToken(ctx context.Context, tokenHash string, revokedAt time.Time, replacedByHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	token, exists := r.refreshTokens[tokenHash]
	if !exists {
		return ErrRefreshTokenNotFound
	}

	token.RevokedAt = &revokedAt
	token.ReplacedByTokenHash = replacedByHash
	r.refreshTokens[tokenHash] = token
	return nil
}

func (r *MemoryIdentityRepository) RevokeUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenHash, token := range r.refreshTokens {
		if token.UserID != userID || token.RevokedAt != nil {
			continue
		}
		token.RevokedAt = &revokedAt
		r.refreshTokens[tokenHash] = token
	}
	return nil
}

func (r *MemoryIdentityRepository) SavePasswordResetToken(ctx context.Context, token model.PasswordResetToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sweepExpiredPasswordResetTokensLocked(time.Now().UTC(), r.passwordResetTokens)
	r.passwordResetTokens[token.TokenHash] = token
	return nil
}

func (r *MemoryIdentityRepository) FindPasswordResetToken(ctx context.Context, tokenHash string) (model.PasswordResetToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	sweepExpiredPasswordResetTokensLocked(now, r.passwordResetTokens)
	token, exists := r.passwordResetTokens[tokenHash]
	if !exists {
		return model.PasswordResetToken{}, ErrPasswordResetTokenNotFound
	}
	return token, nil
}

func (r *MemoryIdentityRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string, consumedAt time.Time) (model.PasswordResetToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	sweepExpiredPasswordResetTokensLocked(now, r.passwordResetTokens)
	token, exists := r.passwordResetTokens[tokenHash]
	if !exists {
		return model.PasswordResetToken{}, ErrPasswordResetTokenNotFound
	}
	token.ConsumedAt = &consumedAt
	r.passwordResetTokens[tokenHash] = token
	return token, nil
}

func (r *MemoryIdentityRepository) DeletePasswordResetTokensByUser(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenHash, token := range r.passwordResetTokens {
		if token.UserID == userID {
			delete(r.passwordResetTokens, tokenHash)
		}
	}
	return nil
}

func sweepExpiredRefreshTokensLocked(now time.Time, tokens map[string]model.RefreshToken) {
	for tokenHash, token := range tokens {
		if now.After(token.ExpiresAt) {
			delete(tokens, tokenHash)
		}
	}
}

func sweepExpiredPasswordResetTokensLocked(now time.Time, tokens map[string]model.PasswordResetToken) {
	for tokenHash, token := range tokens {
		if now.After(token.ExpiresAt) {
			delete(tokens, tokenHash)
		}
	}
}
