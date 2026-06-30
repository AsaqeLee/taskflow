package identity

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyAPIKeyUserID = errors.New("api key user id is required")
	ErrEmptyAPIKeyName   = errors.New("api key name is required")
	ErrEmptyAPIKeyPrefix = errors.New("api key prefix is required")
	ErrEmptyAPIKeyHash   = errors.New("api key hash is required")
)

// APIKey is a persisted bearer credential for unattended agents and integrations.
type APIKey struct {
	id         string
	userID     string
	name       string
	keyPrefix  string
	keyHash    string
	createdAt  time.Time
	expiresAt  *time.Time
	lastUsedAt *time.Time
	revokedAt  *time.Time
}

func IssueAPIKey(userID, name, keyPrefix, keyHash string, createdAt time.Time, expiresAt *time.Time) (APIKey, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return APIKey{}, ErrEmptyAPIKeyUserID
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return APIKey{}, ErrEmptyAPIKeyName
	}

	keyPrefix = strings.TrimSpace(keyPrefix)
	if keyPrefix == "" {
		return APIKey{}, ErrEmptyAPIKeyPrefix
	}

	keyHash = strings.TrimSpace(keyHash)
	if keyHash == "" {
		return APIKey{}, ErrEmptyAPIKeyHash
	}

	return APIKey{
		userID:    userID,
		name:      name,
		keyPrefix: keyPrefix,
		keyHash:   keyHash,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}, nil
}

func RestoreAPIKey(
	id, userID, name, keyPrefix, keyHash string,
	createdAt time.Time,
	expiresAt, lastUsedAt, revokedAt *time.Time,
) APIKey {
	return APIKey{
		id:         id,
		userID:     userID,
		name:       name,
		keyPrefix:  keyPrefix,
		keyHash:    keyHash,
		createdAt:  createdAt,
		expiresAt:  expiresAt,
		lastUsedAt: lastUsedAt,
		revokedAt:  revokedAt,
	}
}

func (k APIKey) AssignID(id string) APIKey {
	k.id = id
	return k
}

func (k APIKey) MarkUsed(at time.Time) APIKey {
	k.lastUsedAt = &at
	return k
}

func (k APIKey) Revoke(at time.Time) APIKey {
	k.revokedAt = &at
	return k
}

func (k APIKey) ID() string {
	return k.id
}

func (k APIKey) UserID() string {
	return k.userID
}

func (k APIKey) Name() string {
	return k.name
}

func (k APIKey) KeyPrefix() string {
	return k.keyPrefix
}

func (k APIKey) KeyHash() string {
	return k.keyHash
}

func (k APIKey) CreatedAt() time.Time {
	return k.createdAt
}

func (k APIKey) ExpiresAt() *time.Time {
	return k.expiresAt
}

func (k APIKey) LastUsedAt() *time.Time {
	return k.lastUsedAt
}

func (k APIKey) RevokedAt() *time.Time {
	return k.revokedAt
}

func (k APIKey) IsExpired(now time.Time) bool {
	return k.expiresAt != nil && now.After(*k.expiresAt)
}

func (k APIKey) IsRevoked() bool {
	return k.revokedAt != nil
}
