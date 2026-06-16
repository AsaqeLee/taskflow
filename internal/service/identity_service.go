package service

import (
	"context"
	"errors"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/model"
)

var (
	ErrWeakPassword            = auth.ErrWeakPassword
	ErrInvalidCredentials      = auth.ErrInvalidCredentials
	ErrInvalidRefreshToken     = errors.New("refresh token is invalid or expired")
	ErrRefreshTokenReused      = errors.New("refresh token reuse detected")
	ErrInvalidPasswordResetTok = errors.New("password reset token is invalid or expired")
	ErrForbiddenSessionRevoke  = errors.New("current user cannot revoke this account's sessions")
)

type IdentityService struct {
	users        ports.UserRepository
	identityRepo ports.IdentityRepository
	devMode      bool
}

func NewIdentityService(users ports.UserRepository, identityRepo ports.IdentityRepository, devMode bool) *IdentityService {
	return &IdentityService{
		users:        users,
		identityRepo: identityRepo,
		devMode:      devMode,
	}
}

func (s *IdentityService) Register(ctx context.Context, id, name, role, password string) (model.User, error) {
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return model.User{}, err
	}

	parsedRole, err := domainuser.ParseRole(role)
	if err != nil {
		return model.User{}, err
	}

	legacyToken := ""
	if s.devMode {
		legacyToken = "dev_" + id
	}

	now := time.Now().UTC()
	account, err := domainuser.Register(id, name, parsedRole, passwordHash, legacyToken, now)
	if err != nil {
		return model.User{}, err
	}

	created, err := s.users.Create(ctx, account)
	if err != nil {
		return model.User{}, err
	}

	return model.UserFromAccount(created), nil
}

func (s *IdentityService) IssueRefreshToken(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	rawToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", err
	}

	token := domainidentity.IssueRefreshToken(
		userID,
		auth.HashOpaqueToken(rawToken),
		now,
		now.Add(ttl),
	)
	if err := s.identityRepo.SaveRefreshToken(ctx, token); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *IdentityService) RotateRefreshToken(ctx context.Context, rawRefreshToken string, ttl time.Duration) (model.User, string, error) {
	now := time.Now().UTC()
	current, err := s.identityRepo.FindRefreshToken(ctx, auth.HashOpaqueToken(rawRefreshToken))
	if err != nil || current.IsExpired(now) {
		return model.User{}, "", ErrInvalidRefreshToken
	}
	if current.IsRevoked() {
		if current.ReplacedByTokenHash() != "" {
			_ = s.identityRepo.RevokeUserRefreshTokens(ctx, current.UserID(), now)
			return model.User{}, "", ErrRefreshTokenReused
		}
		return model.User{}, "", ErrInvalidRefreshToken
	}

	user, err := s.FindAccount(ctx, current.UserID())
	if err != nil {
		return model.User{}, "", ErrInvalidRefreshToken
	}
	if err := s.EnsureActive(user); err != nil {
		return model.User{}, "", err
	}

	newRawToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return model.User{}, "", err
	}
	newHash := auth.HashOpaqueToken(newRawToken)
	if err := s.identityRepo.RevokeRefreshToken(ctx, current.TokenHash(), now, newHash); err != nil {
		return model.User{}, "", err
	}
	if err := s.identityRepo.SaveRefreshToken(ctx, domainidentity.IssueRefreshToken(user.ID, newHash, now, now.Add(ttl))); err != nil {
		return model.User{}, "", err
	}

	return user, newRawToken, nil
}

func (s *IdentityService) RequestPasswordReset(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	user, err := s.FindAccount(ctx, userID)
	if err != nil || !user.Active {
		return "", nil
	}

	now := time.Now().UTC()
	if err := s.identityRepo.DeletePasswordResetTokensByUser(ctx, user.ID); err != nil {
		return "", err
	}

	rawToken, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", err
	}
	token := domainidentity.IssuePasswordResetToken(
		user.ID,
		auth.HashOpaqueToken(rawToken),
		now,
		now.Add(ttl),
	)
	if err := s.identityRepo.SavePasswordResetToken(ctx, token); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *IdentityService) ConfirmPasswordReset(ctx context.Context, userID, rawToken, newPassword string) (model.User, error) {
	now := time.Now().UTC()
	resetToken, err := s.identityRepo.FindPasswordResetToken(ctx, auth.HashOpaqueToken(rawToken))
	if err != nil ||
		resetToken.UserID() != userID ||
		resetToken.IsConsumed() ||
		resetToken.IsExpired(now) {
		return model.User{}, ErrInvalidPasswordResetTok
	}

	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return model.User{}, err
	}

	updatedAccount, err := s.users.UpdatePassword(ctx, userID, passwordHash, now)
	if err != nil {
		if errors.Is(err, ports.ErrUserNotFound) {
			return model.User{}, ErrInvalidPasswordResetTok
		}
		return model.User{}, err
	}
	if _, err := s.identityRepo.ConsumePasswordResetToken(ctx, auth.HashOpaqueToken(rawToken), now); err != nil {
		if errors.Is(err, ports.ErrPasswordResetTokenNotFound) {
			return model.User{}, ErrInvalidPasswordResetTok
		}
		return model.User{}, err
	}

	_ = s.identityRepo.RevokeUserRefreshTokens(ctx, userID, now)
	_ = s.identityRepo.DeletePasswordResetTokensByUser(ctx, userID)
	return model.UserFromAccount(updatedAccount), nil
}

func (s *IdentityService) DisableAccount(ctx context.Context, actor model.User, targetUserID string) (model.User, error) {
	target, err := s.users.FindByID(ctx, targetUserID)
	if err != nil {
		return model.User{}, err
	}

	actorRole, err := domainuser.ParseRole(actor.Role)
	if err != nil {
		return model.User{}, err
	}
	if err := domainuser.AuthorizeDisable(domainuser.NewActor(actor.ID), actorRole, target); err != nil {
		return model.User{}, err
	}
	if err := target.EnsureDisableable(); err != nil {
		return model.User{}, err
	}

	now := time.Now().UTC()
	disabled, err := s.users.Disable(ctx, targetUserID, actor.ID, now)
	if err != nil {
		return model.User{}, err
	}

	_ = s.identityRepo.RevokeUserRefreshTokens(ctx, targetUserID, now)
	_ = s.identityRepo.DeletePasswordResetTokensByUser(ctx, targetUserID)

	return model.UserFromAccount(disabled), nil
}

func (s *IdentityService) RevokeSessions(ctx context.Context, actor model.User, targetUserID string) error {
	actorRole, err := domainuser.ParseRole(actor.Role)
	if err != nil {
		return err
	}
	if !actorRole.IsOwner() && actor.ID != targetUserID {
		return ErrForbiddenSessionRevoke
	}
	if _, err := s.users.FindByID(ctx, targetUserID); err != nil {
		return err
	}
	return s.identityRepo.RevokeUserRefreshTokens(ctx, targetUserID, time.Now().UTC())
}

func (s *IdentityService) FindAccount(ctx context.Context, id string) (model.User, error) {
	account, err := s.users.FindByID(ctx, id)
	if err != nil {
		return model.User{}, err
	}
	return model.UserFromAccount(account), nil
}

func (s *IdentityService) FindAccountByToken(ctx context.Context, token string) (model.User, error) {
	account, err := s.users.FindByToken(ctx, token)
	if err != nil {
		return model.User{}, err
	}
	return model.UserFromAccount(account), nil
}

func (s *IdentityService) EnsureActive(account model.User) error {
	if !account.Active {
		return domainuser.ErrAccountDisabled
	}
	return nil
}
