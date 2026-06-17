package user

import "errors"

var (
	ErrEmptyUserID         = errors.New("user id is required")
	ErrEmptyUserName       = errors.New("user name is required")
	ErrInvalidRole         = errors.New("user role is invalid")
	ErrAccountDisabled     = errors.New("account is disabled")
	ErrForbiddenDisable    = errors.New("current user cannot disable this account")
	ErrAlreadyDisabled     = errors.New("account is already disabled")
	ErrForbiddenUserCreate = errors.New("only owners can create users")
)
