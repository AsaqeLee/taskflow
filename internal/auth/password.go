package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const MinPasswordLength = 8

var ErrWeakPassword = errors.New("password must be at least 8 characters")
var ErrInvalidCredentials = errors.New("invalid credentials")

func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func ComparePassword(hash, password string) error {
	if hash == "" {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}
