package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/database"
	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func newTestIdentityService(t *testing.T) (*IdentityService, *repository.MemoryUserRepository, *repository.MemoryIdentityRepository) {
	t.Helper()

	userRepo := repository.NewMemoryUserRepository()
	identityRepo := repository.NewMemoryIdentityRepository()
	return NewIdentityService(userRepo, identityRepo, false), userRepo, identityRepo
}

func TestIdentityService_Authenticate_ValidCredentials(t *testing.T) {
	svc, userRepo, _ := newTestIdentityService(t)

	password := "strong-pass-123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	now := time.Now().UTC()
	_, err = userRepo.Create(context.Background(), domainuser.Restore(
		"u_auth_001", "Auth User", domainuser.RoleHuman, hash, "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	user, err := svc.Authenticate(context.Background(), "u_auth_001", password)
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if user.ID != "u_auth_001" {
		t.Fatalf("expected user id u_auth_001, got %q", user.ID)
	}
}

func TestIdentityService_Authenticate_RejectsInvalidPassword(t *testing.T) {
	svc, userRepo, _ := newTestIdentityService(t)

	hash, err := auth.HashPassword("correct-pass")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	now := time.Now().UTC()
	_, err = userRepo.Create(context.Background(), domainuser.Restore(
		"u_auth_002", "Auth User", domainuser.RoleHuman, hash, "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = svc.Authenticate(context.Background(), "u_auth_002", "wrong-pass")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestIdentityService_Authenticate_RejectsDisabledAccount(t *testing.T) {
	svc, userRepo, _ := newTestIdentityService(t)

	hash, err := auth.HashPassword("strong-pass-123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	now := time.Now().UTC()
	account := domainuser.Restore(
		"u_auth_003", "Disabled User", domainuser.RoleHuman, hash, "", true, nil, "", now, now,
	)
	created, err := userRepo.Create(context.Background(), account)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := created.Disable(domainuser.NewActor("u_admin"), now); err != nil {
		t.Fatalf("disable account: %v", err)
	}
	if _, err := userRepo.Update(context.Background(), created); err != nil {
		t.Fatalf("persist disabled account: %v", err)
	}

	_, err = svc.Authenticate(context.Background(), "u_auth_003", "strong-pass-123")
	if err != domainuser.ErrAccountDisabled {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}
}

func TestIdentityService_RotateRefreshToken_IssuesReplacement(t *testing.T) {
	svc, userRepo, _ := newTestIdentityService(t)
	now := time.Now().UTC()
	_, err := userRepo.Create(context.Background(), domainuser.Restore(
		"u_refresh_001", "Refresh User", domainuser.RoleHuman, "", "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	rawToken, err := svc.IssueRefreshToken(context.Background(), "u_refresh_001", time.Hour)
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	user, newToken, err := svc.RotateRefreshToken(context.Background(), rawToken, time.Hour)
	if err != nil {
		t.Fatalf("RotateRefreshToken returned error: %v", err)
	}
	if user.ID != "u_refresh_001" {
		t.Fatalf("expected user id u_refresh_001, got %q", user.ID)
	}
	if newToken == "" || newToken == rawToken {
		t.Fatalf("expected a new refresh token, got %q", newToken)
	}

	_, _, err = svc.RotateRefreshToken(context.Background(), rawToken, time.Hour)
	if err != ErrRefreshTokenReused {
		t.Fatalf("expected ErrRefreshTokenReused, got %v", err)
	}

	_, _, err = svc.RotateRefreshToken(context.Background(), newToken, time.Hour)
	if err != ErrInvalidRefreshToken {
		t.Fatalf("expected rotated token to be revoked after reuse detection, got %v", err)
	}
}

func TestIdentityService_DisableAccount_UsesDomainDisable(t *testing.T) {
	svc, userRepo, _ := newTestIdentityService(t)
	now := time.Now().UTC()

	_, err := userRepo.Create(context.Background(), domainuser.Restore(
		"u_owner_disable", "Owner", domainuser.RoleOwner, "", "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	_, err = userRepo.Create(context.Background(), domainuser.Restore(
		"u_target_disable", "Target", domainuser.RoleHuman, "", "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	rawToken, err := svc.IssueRefreshToken(context.Background(), "u_target_disable", time.Hour)
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	disabled, err := svc.DisableAccount(context.Background(), model.User{ID: "u_owner_disable", Role: "owner"}, "u_target_disable")
	if err != nil {
		t.Fatalf("DisableAccount returned error: %v", err)
	}
	if disabled.Active {
		t.Fatalf("expected disabled account to be inactive")
	}
	if disabled.DisabledBy != "u_owner_disable" {
		t.Fatalf("expected disabled_by u_owner_disable, got %q", disabled.DisabledBy)
	}

	_, _, err = svc.RotateRefreshToken(context.Background(), rawToken, time.Hour)
	if err != ErrInvalidRefreshToken {
		t.Fatalf("expected refresh to fail after disable, got %v", err)
	}
}

func TestIdentityService_DisableAccount_UsesScopedDisablePersistence(t *testing.T) {
	now := time.Now().UTC()
	userRepo := &trackingDisableUserRepo{MemoryUserRepository: repository.NewMemoryUserRepository()}
	identityRepo := repository.NewMemoryIdentityRepository()
	svc := NewIdentityService(userRepo, identityRepo, false)

	_, err := userRepo.Create(context.Background(), domainuser.Restore(
		"u_owner_disable_scope", "Owner", domainuser.RoleOwner, "", "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	_, err = userRepo.Create(context.Background(), domainuser.Restore(
		"u_target_disable_scope", "Target", domainuser.RoleHuman, "", "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	if _, err := svc.DisableAccount(context.Background(), model.User{ID: "u_owner_disable_scope", Role: "owner"}, "u_target_disable_scope"); err != nil {
		t.Fatalf("DisableAccount returned error: %v", err)
	}
	if !userRepo.disableCalled {
		t.Fatalf("expected Disable to be used for persistence")
	}
	if userRepo.updateCalled {
		t.Fatalf("did not expect full Update persistence path")
	}
}

func TestIdentityService_ConfirmPasswordReset_ReturnsCleanupFailure(t *testing.T) {
	now := time.Now().UTC()
	userRepo := repository.NewMemoryUserRepository()
	baseRepo := repository.NewMemoryIdentityRepository()
	identityRepo := &failingCleanupIdentityRepo{
		MemoryIdentityRepository: baseRepo,
		revokeErr:                errors.New("revoke failed"),
	}
	svc := NewIdentityService(userRepo, identityRepo, false)

	hash, err := auth.HashPassword("strong-pass-123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	_, err = userRepo.Create(context.Background(), domainuser.Restore(
		"u_reset_cleanup", "Reset User", domainuser.RoleHuman, hash, "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	rawToken, err := svc.RequestPasswordReset(context.Background(), "u_reset_cleanup", time.Hour)
	if err != nil {
		t.Fatalf("RequestPasswordReset returned error: %v", err)
	}
	if rawToken == "" {
		t.Fatalf("expected password reset token")
	}

	_, err = svc.ConfirmPasswordReset(context.Background(), "u_reset_cleanup", rawToken, "new-pass-1234")
	if !errors.Is(err, identityRepo.revokeErr) {
		t.Fatalf("expected cleanup failure %v, got %v", identityRepo.revokeErr, err)
	}
}

func TestIdentityService_APIKeyLifecycle(t *testing.T) {
	svc, userRepo, identityRepo := newTestIdentityService(t)

	now := time.Now().UTC()
	owner, err := userRepo.Create(context.Background(), domainuser.Restore(
		"u_owner_api",
		"Owner API",
		domainuser.RoleOwner,
		"",
		"",
		true,
		nil,
		"",
		now,
		now,
	))
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	_, err = userRepo.Create(context.Background(), domainuser.Restore(
		"u_agent_api",
		"Hermes Agent",
		domainuser.RoleAgent,
		"",
		"",
		true,
		nil,
		"",
		now,
		now,
	))
	if err != nil {
		t.Fatalf("create target user: %v", err)
	}
	human, err := userRepo.Create(context.Background(), domainuser.Restore(
		"u_human_api",
		"Human API",
		domainuser.RoleHuman,
		"",
		"",
		true,
		nil,
		"",
		now,
		now,
	))
	if err != nil {
		t.Fatalf("create human: %v", err)
	}

	actor := model.UserFromAccount(owner)
	key, rawKey, err := svc.CreateAPIKey(context.Background(), actor, "u_agent_api", "Hermes Prod", nil)
	if err != nil {
		t.Fatalf("CreateAPIKey returned error: %v", err)
	}
	if key.ID() == "" {
		t.Fatalf("expected persisted api key id")
	}
	if rawKey == "" {
		t.Fatalf("expected raw api key")
	}

	stored, err := identityRepo.FindAPIKey(context.Background(), auth.HashOpaqueToken(rawKey))
	if err != nil {
		t.Fatalf("FindAPIKey returned error: %v", err)
	}
	if stored.ID() != key.ID() {
		t.Fatalf("expected stored key id %q, got %q", key.ID(), stored.ID())
	}

	listed, err := svc.ListAPIKeys(context.Background(), actor, "u_agent_api")
	if err != nil {
		t.Fatalf("ListAPIKeys returned error: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(listed))
	}
	if listed[0].ID() != key.ID() {
		t.Fatalf("expected listed key id %q, got %q", key.ID(), listed[0].ID())
	}

	revoked, err := svc.RevokeAPIKey(context.Background(), actor, "u_agent_api", key.ID())
	if err != nil {
		t.Fatalf("RevokeAPIKey returned error: %v", err)
	}
	if revoked.RevokedAt() == nil {
		t.Fatalf("expected revoked api key timestamp")
	}

	_, _, err = svc.CreateAPIKey(context.Background(), model.UserFromAccount(human), "u_agent_api", "forbidden", nil)
	if !errors.Is(err, ErrForbiddenAPIKeyManage) {
		t.Fatalf("expected ErrForbiddenAPIKeyManage, got %v", err)
	}
}

func TestIdentityService_RotateRefreshToken_ReuseRevokesActiveSessionsInMongoTransaction(t *testing.T) {
	uri := os.Getenv("TASKFLOW_MONGO_TEST_URI")
	if uri == "" {
		t.Skip("TASKFLOW_MONGO_TEST_URI not set; skipping Mongo identity transaction test")
	}

	dbName := os.Getenv("TASKFLOW_MONGO_TEST_DATABASE")
	if dbName == "" {
		dbName = "taskflow_identity_transaction_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		_ = client.Database(dbName).Collection("users").Drop(context.Background())
		_ = client.Database(dbName).Collection("refresh_tokens").Drop(context.Background())
		_ = client.Database(dbName).Collection("password_reset_tokens").Drop(context.Background())
		_ = client.Database(dbName).Collection("api_keys").Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}()

	mongoDB := client.Database(dbName)
	_ = mongoDB.Collection("users").Drop(ctx)
	_ = mongoDB.Collection("refresh_tokens").Drop(ctx)
	_ = mongoDB.Collection("password_reset_tokens").Drop(ctx)
	_ = mongoDB.Collection("api_keys").Drop(ctx)

	userRepo := repository.NewMongoUserRepository(mongoDB.Collection("users"))
	identityRepo := repository.NewMongoIdentityRepository(
		mongoDB.Collection("refresh_tokens"),
		mongoDB.Collection("password_reset_tokens"),
		mongoDB.Collection("api_keys"),
	)
	dbClient := &database.Client{Mongo: client, DBName: dbName}
	svc := NewIdentityService(userRepo, identityRepo, false, dbClient)

	now := time.Now().UTC()
	_, err = userRepo.Create(ctx, domainuser.Restore(
		"u_refresh_mongo", "Refresh Mongo", domainuser.RoleHuman, "", "", true, nil, "", now, now,
	))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	rawToken, err := svc.IssueRefreshToken(ctx, "u_refresh_mongo", time.Hour)
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	_, rotatedToken, err := svc.RotateRefreshToken(ctx, rawToken, time.Hour)
	if err != nil {
		t.Fatalf("initial rotation failed: %v", err)
	}
	if rotatedToken == "" {
		t.Fatalf("expected rotated refresh token")
	}

	_, _, err = svc.RotateRefreshToken(ctx, rawToken, time.Hour)
	if err != ErrRefreshTokenReused {
		t.Fatalf("expected ErrRefreshTokenReused, got %v", err)
	}

	_, _, err = svc.RotateRefreshToken(ctx, rotatedToken, time.Hour)
	if err != ErrInvalidRefreshToken {
		t.Fatalf("expected rotated token to be revoked after reuse detection, got %v", err)
	}
}

type trackingDisableUserRepo struct {
	*repository.MemoryUserRepository
	disableCalled bool
	updateCalled  bool
}

func (r *trackingDisableUserRepo) Disable(ctx context.Context, id, disabledBy string, disabledAt time.Time) (domainuser.Account, error) {
	r.disableCalled = true
	return r.MemoryUserRepository.Disable(ctx, id, disabledBy, disabledAt)
}

func (r *trackingDisableUserRepo) Update(ctx context.Context, account domainuser.Account) (domainuser.Account, error) {
	r.updateCalled = true
	return domainuser.Account{}, errors.New("unexpected full account update")
}

type failingCleanupIdentityRepo struct {
	*repository.MemoryIdentityRepository
	revokeErr error
	deleteErr error
}

func (r *failingCleanupIdentityRepo) RevokeUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	return r.MemoryIdentityRepository.RevokeUserRefreshTokens(ctx, userID, revokedAt)
}

func (r *failingCleanupIdentityRepo) DeletePasswordResetTokensByUser(ctx context.Context, userID string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return r.MemoryIdentityRepository.DeletePasswordResetTokensByUser(ctx, userID)
}

var _ ports.UserRepository = (*trackingDisableUserRepo)(nil)
var _ ports.IdentityRepository = (*failingCleanupIdentityRepo)(nil)
var _ = domainidentity.RefreshToken{}
