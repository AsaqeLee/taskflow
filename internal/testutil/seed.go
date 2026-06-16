package testutil

import (
	"context"
	"testing"
	"time"

	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
)

type TaskRepository interface {
	Create(ctx context.Context, task domaintask.Task) (domaintask.Task, error)
}

func SeedTask(
	t *testing.T,
	repo TaskRepository,
	id, title, description string,
	status domaintask.Status,
	creatorID, assigneeID string,
	at time.Time,
) domaintask.Task {
	t.Helper()

	task := domaintask.Restore(id, title, description, status, creatorID, assigneeID, at, at, nil, "")
	created, err := repo.Create(context.Background(), task)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return created
}
