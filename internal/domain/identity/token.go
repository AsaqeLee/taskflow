package identity

import "time"

// RefreshToken is the persisted opaque token used to mint new access tokens.
type RefreshToken struct {
	id                  string
	userID              string
	tokenHash           string
	createdAt           time.Time
	expiresAt           time.Time
	revokedAt           *time.Time
	replacedByTokenHash string
}

// PasswordResetToken is the persisted opaque token used for password reset.
type PasswordResetToken struct {
	id         string
	userID     string
	tokenHash  string
	createdAt  time.Time
	expiresAt  time.Time
	consumedAt *time.Time
}

func IssueRefreshToken(userID, tokenHash string, createdAt, expiresAt time.Time) RefreshToken {
	return RefreshToken{
		userID:    userID,
		tokenHash: tokenHash,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}
}

func RestoreRefreshToken(
	id, userID, tokenHash string,
	createdAt, expiresAt time.Time,
	revokedAt *time.Time,
	replacedByTokenHash string,
) RefreshToken {
	return RefreshToken{
		id:                  id,
		userID:              userID,
		tokenHash:           tokenHash,
		createdAt:           createdAt,
		expiresAt:           expiresAt,
		revokedAt:           revokedAt,
		replacedByTokenHash: replacedByTokenHash,
	}
}

func (t RefreshToken) AssignID(id string) RefreshToken {
	t.id = id
	return t
}

func (t RefreshToken) WithRevocation(revokedAt time.Time, replacedByTokenHash string) RefreshToken {
	t.revokedAt = &revokedAt
	t.replacedByTokenHash = replacedByTokenHash
	return t
}

func (t RefreshToken) ID() string                  { return t.id }
func (t RefreshToken) UserID() string              { return t.userID }
func (t RefreshToken) TokenHash() string           { return t.tokenHash }
func (t RefreshToken) CreatedAt() time.Time        { return t.createdAt }
func (t RefreshToken) ExpiresAt() time.Time        { return t.expiresAt }
func (t RefreshToken) RevokedAt() *time.Time       { return t.revokedAt }
func (t RefreshToken) ReplacedByTokenHash() string { return t.replacedByTokenHash }
func (t RefreshToken) IsExpired(now time.Time) bool {
	return now.After(t.expiresAt)
}
func (t RefreshToken) IsRevoked() bool {
	return t.revokedAt != nil
}

func IssuePasswordResetToken(userID, tokenHash string, createdAt, expiresAt time.Time) PasswordResetToken {
	return PasswordResetToken{
		userID:    userID,
		tokenHash: tokenHash,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}
}

func RestorePasswordResetToken(
	id, userID, tokenHash string,
	createdAt, expiresAt time.Time,
	consumedAt *time.Time,
) PasswordResetToken {
	return PasswordResetToken{
		id:         id,
		userID:     userID,
		tokenHash:  tokenHash,
		createdAt:  createdAt,
		expiresAt:  expiresAt,
		consumedAt: consumedAt,
	}
}

func (t PasswordResetToken) AssignID(id string) PasswordResetToken {
	t.id = id
	return t
}

func (t PasswordResetToken) Consume(consumedAt time.Time) PasswordResetToken {
	t.consumedAt = &consumedAt
	return t
}

func (t PasswordResetToken) ID() string             { return t.id }
func (t PasswordResetToken) UserID() string         { return t.userID }
func (t PasswordResetToken) TokenHash() string      { return t.tokenHash }
func (t PasswordResetToken) CreatedAt() time.Time   { return t.createdAt }
func (t PasswordResetToken) ExpiresAt() time.Time   { return t.expiresAt }
func (t PasswordResetToken) ConsumedAt() *time.Time { return t.consumedAt }
func (t PasswordResetToken) IsExpired(now time.Time) bool {
	return now.After(t.expiresAt)
}
func (t PasswordResetToken) IsConsumed() bool {
	return t.consumedAt != nil
}
