package testutil

import (
	"context"
	"testing"
	"time"

	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

type UserRepository interface {
	Create(ctx context.Context, account domainuser.Account) (domainuser.Account, error)
}

func SeedAccount(t *testing.T, repo UserRepository, id, name, role, legacyToken string) domainuser.Account {
	t.Helper()

	parsedRole, err := domainuser.ParseRole(role)
	if err != nil {
		t.Fatalf("parse role: %v", err)
	}

	now := time.Now().UTC()
	account := domainuser.Restore(id, name, parsedRole, "", legacyToken, true, nil, "", now, now)
	created, err := repo.Create(context.Background(), account)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return created
}
