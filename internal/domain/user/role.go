package user

import "strings"

// Role is the system permission role for an account.
type Role string

const (
	RoleOwner Role = "owner"
	RoleHuman Role = "human"
	RoleAgent Role = "agent"
)

func ParseRole(value string) (Role, error) {
	role := Role(strings.TrimSpace(value))
	switch role {
	case RoleOwner, RoleHuman, RoleAgent:
		return role, nil
	default:
		return "", ErrInvalidRole
	}
}

func (r Role) String() string {
	return string(r)
}

func (r Role) IsOwner() bool {
	return r == RoleOwner
}
