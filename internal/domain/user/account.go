package user

import (
	"strings"
	"time"
)

// Account is the aggregate root for identity and account lifecycle rules.
type Account struct {
	id           string
	name         string
	role         Role
	passwordHash string
	legacyToken  string
	active       bool
	disabledAt   *time.Time
	disabledBy   string
	createdAt    time.Time
	updatedAt    time.Time
}

// Register creates a new active account from validated registration input.
func Register(id, name string, role Role, passwordHash, legacyToken string, at time.Time) (Account, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Account{}, ErrEmptyUserID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Account{}, ErrEmptyUserName
	}

	return Account{
		id:           id,
		name:         name,
		role:         role,
		passwordHash: passwordHash,
		legacyToken:  legacyToken,
		active:       true,
		createdAt:    at,
		updatedAt:    at,
	}, nil
}

func Restore(
	id, name string,
	role Role,
	passwordHash, legacyToken string,
	active bool,
	disabledAt *time.Time,
	disabledBy string,
	createdAt, updatedAt time.Time,
) Account {
	return Account{
		id:           id,
		name:         name,
		role:         role,
		passwordHash: passwordHash,
		legacyToken:  legacyToken,
		active:       active,
		disabledAt:   disabledAt,
		disabledBy:   disabledBy,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (a Account) ID() string             { return a.id }
func (a Account) Name() string           { return a.name }
func (a Account) Role() Role             { return a.role }
func (a Account) PasswordHash() string   { return a.passwordHash }
func (a Account) LegacyToken() string    { return a.legacyToken }
func (a Account) Active() bool           { return a.active }
func (a Account) DisabledAt() *time.Time { return a.disabledAt }
func (a Account) DisabledBy() string     { return a.disabledBy }
func (a Account) CreatedAt() time.Time   { return a.createdAt }
func (a Account) UpdatedAt() time.Time   { return a.updatedAt }

func (a Account) AssignID(id string) Account {
	a.id = id
	return a
}

// Actor returns the account as an action performer.
func (a Account) Actor() Actor {
	return NewActor(a.id)
}

// IsActive reports whether the account may authenticate.
func (a Account) IsActive() bool {
	return a.active
}

// AuthorizeDisable checks whether actor may disable the target account.
func AuthorizeDisable(actor Actor, actorRole Role, target Account) error {
	if actorRole.IsOwner() || actor.ID == target.ID() {
		return nil
	}
	return ErrForbiddenDisable
}

// EnsureDisableable validates the account can be disabled.
func (a Account) EnsureDisableable() error {
	if !a.active {
		return ErrAlreadyDisabled
	}
	return nil
}

// Disable marks the account inactive and records who disabled it.
func (a *Account) Disable(actor Actor, at time.Time) error {
	if err := a.EnsureDisableable(); err != nil {
		return err
	}
	a.active = false
	a.disabledAt = &at
	a.disabledBy = actor.ID
	a.updatedAt = at
	return nil
}
