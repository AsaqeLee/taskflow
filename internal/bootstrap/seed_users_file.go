package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AsaqeLee/taskflow/internal/auth"
	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

type bootstrapUserSpec struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

func SeedUsersFromFile(ctx context.Context, userRepo ports.UserRepository, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read users file: %w", err)
	}

	var specs []bootstrapUserSpec
	if err := json.Unmarshal(data, &specs); err != nil {
		return fmt.Errorf("parse users file: %w", err)
	}
	if len(specs) == 0 {
		return fmt.Errorf("users file is empty")
	}

	for _, spec := range specs {
		if err := seedUserSpec(ctx, userRepo, spec); err != nil {
			return err
		}
	}
	return nil
}

func seedUserSpec(ctx context.Context, userRepo ports.UserRepository, spec bootstrapUserSpec) error {
	_, err := userRepo.FindByID(ctx, spec.ID)
	if err == nil {
		return nil
	}
	if err != ports.ErrUserNotFound {
		return fmt.Errorf("lookup user %s: %w", spec.ID, err)
	}

	passwordHash, err := auth.HashPassword(spec.Password)
	if err != nil {
		return fmt.Errorf("hash password for %s: %w", spec.ID, err)
	}

	role, err := domainuser.ParseRole(spec.Role)
	if err != nil {
		return fmt.Errorf("parse role for %s: %w", spec.ID, err)
	}

	now := time.Now().UTC()
	account, err := domainuser.Register(spec.ID, spec.Name, role, passwordHash, "", now)
	if err != nil {
		return fmt.Errorf("build account for %s: %w", spec.ID, err)
	}

	if _, err := userRepo.Create(ctx, account); err != nil {
		if err == ports.ErrUserAlreadyExists {
			return nil
		}
		return fmt.Errorf("create user %s: %w", spec.ID, err)
	}
	return nil
}