package bootstrap

import (
	"context"
	"testing"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/repository"
)

func TestNewTaskRepositoryUsesMemoryWhenConfigured(t *testing.T) {
	repo, db, err := newTaskRepository(context.Background(), config.Config{RepositoryDriver: config.RepositoryDriverMemory})
	if err != nil {
		t.Fatalf("newTaskRepository returned error: %v", err)
	}
	if db != nil {
		t.Fatalf("expected no database client for memory repository")
	}

	if _, ok := repo.(*repository.MemoryTaskRepository); !ok {
		t.Fatalf("expected MemoryTaskRepository, got %T", repo)
	}
}

func TestNewTaskRepositoryReturnsErrorWhenMongoConnectionFails(t *testing.T) {
	_, _, err := newTaskRepository(context.Background(), config.Config{
		RepositoryDriver: config.RepositoryDriverMongo,
		MongoURI:         "mongodb://127.0.0.1:1",
		MongoDB:          "taskflow_test",
	})
	if err == nil {
		t.Fatalf("expected mongo connection error")
	}
}
