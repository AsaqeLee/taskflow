package model

import (
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

func UserFromAccount(account domainuser.Account) User {
	return User{
		ID:           account.ID(),
		Name:         account.Name(),
		Role:         account.Role().String(),
		PasswordHash: account.PasswordHash(),
		Token:        account.LegacyToken(),
		Active:       account.Active(),
		DisabledAt:   account.DisabledAt(),
		DisabledBy:   account.DisabledBy(),
		CreatedAt:    account.CreatedAt(),
		UpdatedAt:    account.UpdatedAt(),
	}
}

func AccountFromUser(user User) (domainuser.Account, error) {
	role, err := domainuser.ParseRole(user.Role)
	if err != nil {
		return domainuser.Account{}, err
	}
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
	), nil
}
