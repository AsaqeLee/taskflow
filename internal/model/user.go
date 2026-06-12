package model

import "time"

type User struct {
	ID           string    `json:"id" bson:"_id"`
	Name         string    `json:"name" bson:"name"`
	Role         string    `json:"role" bson:"role"`
	PasswordHash string    `json:"-" bson:"password_hash"`
	Token        string    `json:"token,omitempty" bson:"token,omitempty"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}
