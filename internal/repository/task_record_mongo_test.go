package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongoTaskRecordRepository_CreateAndListByTaskID(t *testing.T) {
	repo := newTestMongoTaskRecordRepository(t)
	now := time.Now().UTC()

	first, err := repo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_200",
		AuthorID:  "u_worker_001",
		Type:      model.TaskRecordTypeSubmit,
		Content:   "submitted",
		CreatedAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("Create first record: %v", err)
	}
	if first.ID == "" {
		t.Fatalf("expected generated id for first record")
	}

	second, err := repo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_200",
		AuthorID:  "u_owner_001",
		Type:      model.TaskRecordTypeApprove,
		Content:   "approved",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create second record: %v", err)
	}
	if second.ID == "" {
		t.Fatalf("expected generated id for second record")
	}

	_, err = repo.Create(context.Background(), model.TaskRecord{
		TaskID:    "task_other",
		AuthorID:  "u_other_001",
		Type:      model.TaskRecordTypeSubmit,
		Content:   "ignored",
		CreatedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Create other task record: %v", err)
	}

	records, err := repo.ListByTaskID(context.Background(), "task_200")
	if err != nil {
		t.Fatalf("ListByTaskID returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != first.ID || records[1].ID != second.ID {
		t.Fatalf("expected ordered record ids [%s %s], got [%s %s]", first.ID, second.ID, records[0].ID, records[1].ID)
	}
}

func TestMongoTaskRecordRepository_ListByTaskIDReturnsEmptySliceWhenMissing(t *testing.T) {
	repo := newTestMongoTaskRecordRepository(t)

	records, err := repo.ListByTaskID(context.Background(), "missing")
	if err != nil {
		t.Fatalf("ListByTaskID returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected empty result, got %d records", len(records))
	}
}

func newTestMongoTaskRecordRepository(t *testing.T) *MongoTaskRecordRepository {
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
		_ = client.Database(dbName).Collection("task_records").Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	collection := client.Database(dbName).Collection("task_records")
	if err := collection.Drop(ctx); err != nil {
		var cmdErr mongo.CommandError
		if !(errors.As(err, &cmdErr) && cmdErr.Code == 26) {
			t.Fatalf("drop collection: %v", err)
		}
	}

	return NewMongoTaskRecordRepository(collection)
}
