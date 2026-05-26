package bootstrap

import (
	"context"
	"testing"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/repository"
)

func TestNewRepositoriesUsesMemoryWhenConfigured(t *testing.T) {
	taskRepo, recordRepo, db, err := newRepositories(context.Background(), config.Config{RepositoryDriver: config.RepositoryDriverMemory})
	if err != nil {
		t.Fatalf("newRepositories returned error: %v", err)
	}
	if db != nil {
		t.Fatalf("expected no database client for memory repository")
	}

	if _, ok := taskRepo.(*repository.MemoryTaskRepository); !ok {
		t.Fatalf("expected MemoryTaskRepository, got %T", taskRepo)
	}
	if _, ok := recordRepo.(*repository.MemoryTaskRecordRepository); !ok {
		t.Fatalf("expected MemoryTaskRecordRepository, got %T", recordRepo)
	}
}

func TestNewRepositoriesReturnsErrorWhenMongoConnectionFails(t *testing.T) {
	_, _, _, err := newRepositories(context.Background(), config.Config{
		RepositoryDriver: config.RepositoryDriverMongo,
		MongoURI:         "mongodb://127.0.0.1:1",
		MongoDB:          "taskflow_test",
	})
	if err == nil {
		t.Fatalf("expected mongo connection error")
	}
}
