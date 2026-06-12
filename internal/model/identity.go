package model

import "time"

type RefreshToken struct {
	ID                  string     `json:"id" bson:"_id"`
	UserID              string     `json:"user_id" bson:"user_id"`
	TokenHash           string     `json:"-" bson:"token_hash"`
	CreatedAt           time.Time  `json:"created_at" bson:"created_at"`
	ExpiresAt           time.Time  `json:"expires_at" bson:"expires_at"`
	RevokedAt           *time.Time `json:"revoked_at,omitempty" bson:"revoked_at,omitempty"`
	ReplacedByTokenHash string     `json:"-" bson:"replaced_by_token_hash,omitempty"`
}

type PasswordResetToken struct {
	ID         string     `json:"id" bson:"_id"`
	UserID     string     `json:"user_id" bson:"user_id"`
	TokenHash  string     `json:"-" bson:"token_hash"`
	CreatedAt  time.Time  `json:"created_at" bson:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at" bson:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty" bson:"consumed_at,omitempty"`
}
