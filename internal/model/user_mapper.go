package model

import (
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

func UserFromAccount(account domainuser.Account) User {
	role, _ := domainuser.ParseRole(account.Role().String())
	return User{
		ID:           account.ID(),
		Name:         account.Name(),
		Role:         role.String(),
		PasswordHash: account.PasswordHash(),
		Token:        account.LegacyToken(),
		Active:       account.Active(),
		DisabledAt:   account.DisabledAt(),
		DisabledBy:   account.DisabledBy(),
		CreatedAt:    account.CreatedAt(),
		UpdatedAt:    account.UpdatedAt(),
	}
}

func AccountFromUser(user User) domainuser.Account {
	role, _ := domainuser.ParseRole(user.Role)
	return domainuser.Restore(
		user.ID,
		user.Name,
		role,
		user.PasswordHash,
		user.Token,
		user.Active,
		user.DisabledAt,
		user.DisabledBy,
		user.CreatedAt,
		user.UpdatedAt,
	)
}
