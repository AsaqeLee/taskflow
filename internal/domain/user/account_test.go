package user_test

import (
	"testing"
	"time"

	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

func TestRegister_RejectsEmptyID(t *testing.T) {
	_, err := domainuser.Register("", "Demo", domainuser.RoleHuman, "hash", "", time.Now().UTC())
	if err != domainuser.ErrEmptyUserID {
		t.Fatalf("expected ErrEmptyUserID, got %v", err)
	}
}

func TestAuthorizeDisable_AllowsOwnerToDisableOthers(t *testing.T) {
	target := domainuser.Restore("u_target", "Target", domainuser.RoleHuman, "hash", "", true, nil, "", time.Now().UTC(), time.Now().UTC())
	if err := domainuser.AuthorizeDisable(domainuser.NewActor("u_owner"), domainuser.RoleOwner, target); err != nil {
		t.Fatalf("expected owner to disable others, got %v", err)
	}
}

func TestAuthorizeDisable_RejectsNonOwnerDisablingOthers(t *testing.T) {
	target := domainuser.Restore("u_target", "Target", domainuser.RoleHuman, "hash", "", true, nil, "", time.Now().UTC(), time.Now().UTC())
	if err := domainuser.AuthorizeDisable(domainuser.NewActor("u_other"), domainuser.RoleHuman, target); err != domainuser.ErrForbiddenDisable {
		t.Fatalf("expected ErrForbiddenDisable, got %v", err)
	}
}

func TestDisable_MarksAccountInactive(t *testing.T) {
	now := time.Now().UTC()
	account := domainuser.Restore("u_disable", "Disable Me", domainuser.RoleHuman, "hash", "", true, nil, "", now, now)

	if err := account.Disable(domainuser.NewActor("u_owner"), now); err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	if account.Active() {
		t.Fatalf("expected account to be inactive")
	}
	if account.DisabledBy() != "u_owner" {
		t.Fatalf("expected disabled_by u_owner, got %q", account.DisabledBy())
	}
	if account.DisabledAt() == nil {
		t.Fatalf("expected disabled_at to be set")
	}
}

func TestEnsureDisableable_RejectsAlreadyDisabledAccount(t *testing.T) {
	now := time.Now().UTC()
	account := domainuser.Restore("u_disabled", "Disabled", domainuser.RoleHuman, "hash", "", false, &now, "u_owner", now, now)
	if err := account.EnsureDisableable(); err != domainuser.ErrAlreadyDisabled {
		t.Fatalf("expected ErrAlreadyDisabled, got %v", err)
	}
}
