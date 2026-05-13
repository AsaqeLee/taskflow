package repository

import (
	"context"
	"errors"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const testMongoURIEnv = "TASKFLOW_MONGO_TEST_URI"
const testMongoDBEnv = "TASKFLOW_MONGO_TEST_DATABASE"

func TestMongoTaskRepository_CreateAndGetByID(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	now := time.Now().UTC()

	created, err := repo.Create(model.Task{
		Title:       "Mongo create",
		Description: "persist me",
		Status:      "open",
		CreatorID:   "u_owner_001",
		AssigneeID:  "",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated id")
	}

	got, err := repo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected id %q, got %q", created.ID, got.ID)
	}
	if got.Title != created.Title {
		t.Fatalf("expected title %q, got %q", created.Title, got.Title)
	}
}

func TestMongoTaskRepository_ListOrdersByCreatedAtAscending(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	firstTime := time.Now().UTC().Add(-time.Hour)
	secondTime := firstTime.Add(time.Minute)

	first, err := repo.Create(model.Task{
		Title:       "first",
		Description: "",
		Status:      "open",
		CreatorID:   "u_owner_001",
		CreatedAt:   firstTime,
		UpdatedAt:   firstTime,
	})
	if err != nil {
		t.Fatalf("Create first task: %v", err)
	}

	second, err := repo.Create(model.Task{
		Title:       "second",
		Description: "",
		Status:      "open",
		CreatorID:   "u_owner_001",
		CreatedAt:   secondTime,
		UpdatedAt:   secondTime,
	})
	if err != nil {
		t.Fatalf("Create second task: %v", err)
	}

	items, err := repo.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(items))
	}
	ids := []string{items[0].ID, items[1].ID}
	expected := []string{first.ID, second.ID}
	if !equalStrings(ids, expected) {
		t.Fatalf("expected ids %v, got %v", expected, ids)
	}
}

func TestMongoTaskRepository_UpdatePersistsChanges(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	now := time.Now().UTC()

	created, err := repo.Create(model.Task{
		Title:       "before update",
		Description: "old",
		Status:      "open",
		CreatorID:   "u_owner_001",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	created.Title = "after update"
	created.Description = "new"
	created.Status = "assigned"
	created.AssigneeID = "u_worker_001"
	created.UpdatedAt = now.Add(time.Minute)

	updated, err := repo.Update(created)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Title != "after update" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}

	got, err := repo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.AssigneeID != "u_worker_001" {
		t.Fatalf("expected assignee to persist, got %q", got.AssigneeID)
	}
	if got.Status != "assigned" {
		t.Fatalf("expected status %q, got %q", "assigned", got.Status)
	}
}

func TestMongoTaskRepository_GetByIDReturnsNotFound(t *testing.T) {
	repo := newTestMongoTaskRepository(t)

	_, err := repo.GetByID("missing")
	if err != ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestMongoTaskRepository_UpdateReturnsNotFound(t *testing.T) {
	repo := newTestMongoTaskRepository(t)

	_, err := repo.Update(model.Task{ID: "missing"})
	if err != ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func newTestMongoTaskRepository(t *testing.T) *MongoTaskRepository {
	t.Helper()

	uri := os.Getenv(testMongoURIEnv)
	if uri == "" {
		t.Skipf("%s not set; skipping Mongo integration test", testMongoURIEnv)
	}

	dbName := os.Getenv(testMongoDBEnv)
	if dbName == "" {
		dbName = "taskflow_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Database(dbName).Collection("tasks").Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	collection := client.Database(dbName).Collection("tasks")
	if err := collection.Drop(ctx); err != nil {
		var cmdErr mongo.CommandError
		if !(errors.As(err, &cmdErr) && cmdErr.Code == 26) {
			t.Fatalf("drop collection: %v", err)
		}
	}

	return NewMongoTaskRepository(collection)
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)
	for i := range gotCopy {
		if gotCopy[i] != wantCopy[i] {
			return false
		}
	}
	return true
}
