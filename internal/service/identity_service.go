package service

import (
	"context"
	"errors"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/database"
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
	ErrForbiddenAPIKeyManage   = errors.New("current user cannot manage api keys")
	ErrUserNotFound            = ports.ErrUserNotFound
	ErrUserAlreadyExists       = ports.ErrUserAlreadyExists
	ErrAPIKeyNotFound          = ports.ErrAPIKeyNotFound
	ErrForbiddenUserCreate     = domainuser.ErrForbiddenUserCreate
)

const apiKeyTokenPrefix = "tfk_"

type IdentityService struct {
	users        ports.UserRepository
	identityRepo ports.IdentityRepository
	dbClient     *database.Client
	devMode      bool
}

func NewIdentityService(
	users ports.UserRepository,
	identityRepo ports.IdentityRepository,
	devMode bool,
	dbClient ...*database.Client,
) *IdentityService {
	var db *database.Client
	if len(dbClient) > 0 {
		db = dbClient[0]
	}

	return &IdentityService{
		users:        users,
		identityRepo: identityRepo,
		dbClient:     db,
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

func (s *IdentityService) Authenticate(ctx context.Context, id, password string) (model.User, error) {
	user, err := s.FindAccount(ctx, id)
	if err != nil {
		if errors.Is(err, ports.ErrUserNotFound) {
			return model.User{}, ErrInvalidCredentials
		}
		return model.User{}, err
	}

	if err := s.EnsureActive(user); err != nil {
		return model.User{}, err
	}

	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		return model.User{}, ErrInvalidCredentials
	}

	return user, nil
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
	tokenHash := auth.HashOpaqueToken(rawRefreshToken)

	var user model.User
	var newRawToken string
	var revokeUserID string
	var err error

	runOps := func(txCtx context.Context) error {
		current, findErr := s.identityRepo.FindRefreshToken(txCtx, tokenHash)
		if findErr != nil || current.IsExpired(now) {
			return ErrInvalidRefreshToken
		}
		if current.IsRevoked() {
			if current.ReplacedByTokenHash() != "" {
				revokeUserID = current.UserID()
				return ErrRefreshTokenReused
			}
			return ErrInvalidRefreshToken
		}

		account, accountErr := s.users.FindByID(txCtx, current.UserID())
		if accountErr != nil {
			return ErrInvalidRefreshToken
		}

		user = model.UserFromAccount(account)
		if activeErr := s.EnsureActive(user); activeErr != nil {
			return activeErr
		}

		generated, genErr := auth.GenerateOpaqueToken()
		if genErr != nil {
			return genErr
		}

		newHash := auth.HashOpaqueToken(generated)
		if revokeErr := s.identityRepo.RevokeRefreshToken(txCtx, current.TokenHash(), now, newHash); revokeErr != nil {
			return revokeErr
		}
		if saveErr := s.identityRepo.SaveRefreshToken(txCtx, domainidentity.IssueRefreshToken(user.ID, newHash, now, now.Add(ttl))); saveErr != nil {
			return saveErr
		}

		newRawToken = generated
		return nil
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}

	if errors.Is(err, ErrRefreshTokenReused) {
		if revokeUserID == "" {
			return model.User{}, "", ErrRefreshTokenReused
		}
		if revokeErr := s.identityRepo.RevokeUserRefreshTokens(ctx, revokeUserID, now); revokeErr != nil {
			return model.User{}, "", revokeErr
		}
		return model.User{}, "", ErrRefreshTokenReused
	}
	if err != nil {
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
	tokenHash := auth.HashOpaqueToken(rawToken)

	var updated model.User
	var err error

	runOps := func(txCtx context.Context) error {
		resetToken, findErr := s.identityRepo.FindPasswordResetToken(txCtx, tokenHash)
		if findErr != nil || resetToken.UserID() != userID || resetToken.IsConsumed() || resetToken.IsExpired(now) {
			return ErrInvalidPasswordResetTok
		}

		passwordHash, hashErr := auth.HashPassword(newPassword)
		if hashErr != nil {
			return hashErr
		}

		updatedAccount, updateErr := s.users.UpdatePassword(txCtx, userID, passwordHash, now)
		if updateErr != nil {
			if errors.Is(updateErr, ports.ErrUserNotFound) {
				return ErrInvalidPasswordResetTok
			}
			return updateErr
		}

		if _, consumeErr := s.identityRepo.ConsumePasswordResetToken(txCtx, tokenHash, now); consumeErr != nil {
			if errors.Is(consumeErr, ports.ErrPasswordResetTokenNotFound) {
				return ErrInvalidPasswordResetTok
			}
			return consumeErr
		}
		if revokeErr := s.identityRepo.RevokeUserRefreshTokens(txCtx, userID, now); revokeErr != nil {
			return revokeErr
		}
		if deleteErr := s.identityRepo.DeletePasswordResetTokensByUser(txCtx, userID); deleteErr != nil {
			return deleteErr
		}

		updated = model.UserFromAccount(updatedAccount)
		return nil
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}
	if err != nil {
		return model.User{}, err
	}

	return updated, nil
}

func (s *IdentityService) DisableAccount(ctx context.Context, actor model.User, targetUserID string) (model.User, error) {
	actorRole, err := domainuser.ParseRole(actor.Role)
	if err != nil {
		return model.User{}, err
	}

	now := time.Now().UTC()
	var disabled domainuser.Account

	runOps := func(txCtx context.Context) error {
		target, findErr := s.users.FindByID(txCtx, targetUserID)
		if findErr != nil {
			return findErr
		}

		if authErr := domainuser.AuthorizeDisable(domainuser.NewActor(actor.ID), actorRole, target); authErr != nil {
			return authErr
		}
		if disableErr := target.Disable(domainuser.NewActor(actor.ID), now); disableErr != nil {
			return disableErr
		}

		updated, disableErr := s.users.Disable(txCtx, targetUserID, actor.ID, now)
		if disableErr != nil {
			return disableErr
		}
		if revokeErr := s.identityRepo.RevokeUserRefreshTokens(txCtx, targetUserID, now); revokeErr != nil {
			return revokeErr
		}
		if deleteErr := s.identityRepo.DeletePasswordResetTokensByUser(txCtx, targetUserID); deleteErr != nil {
			return deleteErr
		}

		disabled = updated
		return nil
	}

	if s.dbClient != nil {
		err = s.dbClient.RunTransaction(ctx, runOps)
	} else {
		err = runOps(ctx)
	}
	if err != nil {
		return model.User{}, err
	}

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

func (s *IdentityService) CreateAPIKey(
	ctx context.Context,
	actor model.User,
	targetUserID, name string,
	expiresAt *time.Time,
) (domainidentity.APIKey, string, error) {
	if err := ensureOwner(actor); err != nil {
		return domainidentity.APIKey{}, "", err
	}

	targetUser, err := s.FindAccount(ctx, targetUserID)
	if err != nil {
		return domainidentity.APIKey{}, "", err
	}
	if err := s.EnsureActive(targetUser); err != nil {
		return domainidentity.APIKey{}, "", err
	}

	rawSecret, err := auth.GenerateOpaqueToken()
	if err != nil {
		return domainidentity.APIKey{}, "", err
	}

	now := time.Now().UTC()
	rawKey := apiKeyTokenPrefix + rawSecret
	key, err := domainidentity.IssueAPIKey(
		targetUser.ID,
		name,
		apiKeyPrefix(rawKey),
		auth.HashOpaqueToken(rawKey),
		now,
		expiresAt,
	)
	if err != nil {
		return domainidentity.APIKey{}, "", err
	}

	if err := s.identityRepo.SaveAPIKey(ctx, key); err != nil {
		return domainidentity.APIKey{}, "", err
	}

	saved, err := s.identityRepo.FindAPIKey(ctx, key.KeyHash())
	if err != nil {
		return domainidentity.APIKey{}, "", err
	}

	return saved, rawKey, nil
}

func (s *IdentityService) ListAPIKeys(ctx context.Context, actor model.User, targetUserID string) ([]domainidentity.APIKey, error) {
	if err := ensureOwner(actor); err != nil {
		return nil, err
	}

	if _, err := s.users.FindByID(ctx, targetUserID); err != nil {
		return nil, err
	}

	return s.identityRepo.ListAPIKeysByUser(ctx, targetUserID)
}

func (s *IdentityService) RevokeAPIKey(ctx context.Context, actor model.User, targetUserID, keyID string) (domainidentity.APIKey, error) {
	if err := ensureOwner(actor); err != nil {
		return domainidentity.APIKey{}, err
	}

	if _, err := s.users.FindByID(ctx, targetUserID); err != nil {
		return domainidentity.APIKey{}, err
	}

	return s.identityRepo.RevokeAPIKey(ctx, targetUserID, keyID, time.Now().UTC())
}

func (s *IdentityService) ListUsers(ctx context.Context, actor model.User, activeOnly bool) ([]model.User, error) {
	actorRole, err := domainuser.ParseRole(actor.Role)
	if err != nil {
		return nil, err
	}

	if actorRole.IsOwner() {
		accounts, err := s.users.List(ctx, activeOnly)
		if err != nil {
			return nil, err
		}

		users := make([]model.User, len(accounts))
		for i, account := range accounts {
			users[i] = model.UserFromAccount(account)
		}
		return users, nil
	}

	self, err := s.FindAccount(ctx, actor.ID)
	if err != nil {
		return nil, err
	}

	return []model.User{self}, nil
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

func ensureOwner(actor model.User) error {
	role, err := domainuser.ParseRole(actor.Role)
	if err != nil {
		return err
	}
	if !role.IsOwner() {
		return ErrForbiddenAPIKeyManage
	}
	return nil
}

func apiKeyPrefix(rawKey string) string {
	if len(rawKey) <= 12 {
		return rawKey
	}
	return rawKey[:12]
}
