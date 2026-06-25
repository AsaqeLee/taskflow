package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
)

func TestMemoryTaskRepositoryRejectsDoneContext(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now().UTC()
	seeded := domaintask.Restore("task_001", "Seed task", "desc", domaintask.StatusOpen, "u_owner_001", "", now, now, nil, "")
	if _, err := repo.Create(context.Background(), seeded); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	ctx := canceledContext()
	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "create",
			run: func() error {
				_, err := repo.Create(ctx, domaintask.Restore("task_002", "New task", "desc", domaintask.StatusOpen, "u_owner_001", "", now, now, nil, ""))
				return err
			},
		},
		{
			name: "get by id",
			run: func() error {
				_, err := repo.GetByID(ctx, seeded.ID())
				return err
			},
		},
		{
			name: "get by id including deleted",
			run: func() error {
				_, err := repo.GetByIDIncludingDeleted(ctx, seeded.ID())
				return err
			},
		},
		{
			name: "list",
			run: func() error {
				_, err := repo.List(ctx)
				return err
			},
		},
		{
			name: "list visible",
			run: func() error {
				_, err := repo.ListVisibleToUser(ctx, "u_owner_001")
				return err
			},
		},
		{
			name: "update",
			run: func() error {
				_, err := repo.Update(ctx, seeded)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
		})
	}
}

func TestMemoryUserRepositoryRejectsDoneContext(t *testing.T) {
	repo := NewMemoryUserRepository()
	now := time.Now().UTC()
	seeded, err := domainuser.Register("u_001", "Seed User", domainuser.RoleHuman, "hash", "token", now)
	if err != nil {
		t.Fatalf("build user: %v", err)
	}
	if _, err := repo.Create(context.Background(), seeded); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	ctx := canceledContext()
	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "create",
			run: func() error {
				account, err := domainuser.Register("u_002", "New User", domainuser.RoleAgent, "hash", "", now)
				if err != nil {
					return err
				}
				_, err = repo.Create(ctx, account)
				return err
			},
		},
		{
			name: "find by id",
			run: func() error {
				_, err := repo.FindByID(ctx, seeded.ID())
				return err
			},
		},
		{
			name: "find by token",
			run: func() error {
				_, err := repo.FindByToken(ctx, seeded.LegacyToken())
				return err
			},
		},
		{
			name: "update password",
			run: func() error {
				_, err := repo.UpdatePassword(ctx, seeded.ID(), "new-hash", now.Add(time.Minute))
				return err
			},
		},
		{
			name: "disable",
			run: func() error {
				_, err := repo.Disable(ctx, seeded.ID(), "u_owner_001", now.Add(time.Minute))
				return err
			},
		},
		{
			name: "update",
			run: func() error {
				_, err := repo.Update(ctx, seeded)
				return err
			},
		},
		{
			name: "list",
			run: func() error {
				_, err := repo.List(ctx, false)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
		})
	}
}

func TestMemoryIdentityRepositoryRejectsDoneContext(t *testing.T) {
	repo := NewMemoryIdentityRepository()
	now := time.Now().UTC()
	refreshToken := domainidentity.IssueRefreshToken("u_001", "refresh-hash", now, now.Add(time.Hour))
	if err := repo.SaveRefreshToken(context.Background(), refreshToken); err != nil {
		t.Fatalf("seed refresh token: %v", err)
	}
	resetToken := domainidentity.IssuePasswordResetToken("u_001", "reset-hash", now, now.Add(time.Hour))
	if err := repo.SavePasswordResetToken(context.Background(), resetToken); err != nil {
		t.Fatalf("seed password reset token: %v", err)
	}

	ctx := canceledContext()
	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "save refresh token",
			run: func() error {
				return repo.SaveRefreshToken(ctx, domainidentity.IssueRefreshToken("u_002", "refresh-hash-2", now, now.Add(time.Hour)))
			},
		},
		{
			name: "find refresh token",
			run: func() error {
				_, err := repo.FindRefreshToken(ctx, refreshToken.TokenHash())
				return err
			},
		},
		{
			name: "revoke refresh token",
			run: func() error {
				return repo.RevokeRefreshToken(ctx, refreshToken.TokenHash(), now.Add(time.Minute), "")
			},
		},
		{
			name: "revoke user refresh tokens",
			run: func() error {
				return repo.RevokeUserRefreshTokens(ctx, "u_001", now.Add(time.Minute))
			},
		},
		{
			name: "save password reset token",
			run: func() error {
				return repo.SavePasswordResetToken(ctx, domainidentity.IssuePasswordResetToken("u_002", "reset-hash-2", now, now.Add(time.Hour)))
			},
		},
		{
			name: "find password reset token",
			run: func() error {
				_, err := repo.FindPasswordResetToken(ctx, resetToken.TokenHash())
				return err
			},
		},
		{
			name: "consume password reset token",
			run: func() error {
				_, err := repo.ConsumePasswordResetToken(ctx, resetToken.TokenHash(), now.Add(time.Minute))
				return err
			},
		},
		{
			name: "delete password reset tokens by user",
			run: func() error {
				return repo.DeletePasswordResetTokensByUser(ctx, "u_001")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
		})
	}
}

func TestMemoryRecordAndAuditRepositoriesRejectsDoneContext(t *testing.T) {
	recordRepo := NewMemoryTaskRecordRepository()
	auditRepo := NewMemoryAuditLogRepository()
	now := time.Now().UTC()

	record := domainrecord.Restore("record_001", "task_001", "u_001", domainrecord.TypeSubmit, "done", now)
	if _, err := recordRepo.Create(context.Background(), record); err != nil {
		t.Fatalf("seed record: %v", err)
	}
	log := domainaudit.Restore("audit_001", "task_001", "u_001", domainaudit.ActionSubmitted, "req-1", "trace-1", "idem-1", "127.0.0.1", "test-agent", "in_progress", "submitted", now)
	if _, err := auditRepo.Create(context.Background(), log); err != nil {
		t.Fatalf("seed audit log: %v", err)
	}

	ctx := canceledContext()
	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "create record",
			run: func() error {
				_, err := recordRepo.Create(ctx, domainrecord.Restore("record_002", "task_001", "u_001", domainrecord.TypeApprove, "approved", now))
				return err
			},
		},
		{
			name: "list records",
			run: func() error {
				_, err := recordRepo.ListByTaskID(ctx, "task_001")
				return err
			},
		},
		{
			name: "delete records",
			run: func() error {
				return recordRepo.DeleteByTaskID(ctx, "task_001")
			},
		},
		{
			name: "create audit log",
			run: func() error {
				_, err := auditRepo.Create(ctx, domainaudit.Restore("audit_002", "task_001", "u_001", domainaudit.ActionApproved, "req-2", "trace-2", "idem-2", "127.0.0.1", "test-agent", "submitted", "closed", now))
				return err
			},
		},
		{
			name: "list audit logs",
			run: func() error {
				_, err := auditRepo.ListByTaskID(ctx, "task_001")
				return err
			},
		},
		{
			name: "delete audit logs",
			run: func() error {
				return auditRepo.DeleteByTaskID(ctx, "task_001")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
		})
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
