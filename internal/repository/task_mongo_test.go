package repository

import (
	"context"
	"errors"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const testMongoURIEnv = "TASKFLOW_MONGO_TEST_URI"
const testMongoDBEnv = "TASKFLOW_MONGO_TEST_DATABASE"

func TestMongoTaskRepository_CreateAndGetByID(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	now := time.Now().UTC()

	created, err := repo.Create(context.Background(), domaintask.Restore(
		"",
		"Mongo create",
		"persist me",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID() == "" {
		t.Fatalf("expected generated id")
	}

	got, err := repo.GetByID(context.Background(), created.ID())
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.ID() != created.ID() {
		t.Fatalf("expected id %q, got %q", created.ID(), got.ID())
	}
	if got.Title() != created.Title() {
		t.Fatalf("expected title %q, got %q", created.Title(), got.Title())
	}
}

func TestMongoTaskRepository_ListOrdersByCreatedAtAscending(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	firstTime := time.Now().UTC().Add(-time.Hour)
	secondTime := firstTime.Add(time.Minute)

	first, err := repo.Create(context.Background(), domaintask.Restore(
		"",
		"first",
		"",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		firstTime,
		firstTime,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("Create first task: %v", err)
	}

	second, err := repo.Create(context.Background(), domaintask.Restore(
		"",
		"second",
		"",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		secondTime,
		secondTime,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("Create second task: %v", err)
	}

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(items))
	}
	ids := []string{items[0].ID(), items[1].ID()}
	expected := []string{first.ID(), second.ID()}
	if !equalStrings(ids, expected) {
		t.Fatalf("expected ids %v, got %v", expected, ids)
	}
}

func TestMongoTaskRepository_ListVisibleToUserFiltersCreatorAndAssignee(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	now := time.Now().UTC()

	seed := []domaintask.Task{
		domaintask.Restore("task_creator", "creator task", "", domaintask.StatusOpen, "u_owner_001", "", now, now, nil, ""),
		domaintask.Restore("task_assignee", "assignee task", "", domaintask.StatusAssigned, "u_other_001", "u_owner_001", now.Add(time.Minute), now.Add(time.Minute), nil, ""),
		domaintask.Restore("task_other", "other task", "", domaintask.StatusOpen, "u_other_001", "u_else_001", now.Add(2*time.Minute), now.Add(2*time.Minute), nil, ""),
	}

	for _, task := range seed {
		if _, err := repo.Create(context.Background(), task); err != nil {
			t.Fatalf("seed task %s: %v", task.ID(), err)
		}
	}

	items, err := repo.ListVisibleToUser(context.Background(), "u_owner_001")
	if err != nil {
		t.Fatalf("ListVisibleToUser returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 visible tasks, got %d", len(items))
	}
	if items[0].ID() != "task_creator" || items[1].ID() != "task_assignee" {
		t.Fatalf("unexpected visible task ids: [%s %s]", items[0].ID(), items[1].ID())
	}
}

func TestMongoTaskRepository_UpdatePersistsChanges(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	now := time.Now().UTC()

	created, err := repo.Create(context.Background(), domaintask.Restore(
		"",
		"before update",
		"old",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		now,
		now,
		nil,
		"",
	))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	updatedTask := domaintask.Restore(
		created.ID(),
		"after update",
		"new",
		domaintask.StatusAssigned,
		created.CreatorID(),
		"u_worker_001",
		created.CreatedAt(),
		now.Add(time.Minute),
		nil,
		"",
	)

	updated, err := repo.Update(context.Background(), updatedTask)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Title() != "after update" {
		t.Fatalf("expected updated title, got %q", updated.Title())
	}

	got, err := repo.GetByID(context.Background(), created.ID())
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.AssigneeID() != "u_worker_001" {
		t.Fatalf("expected assignee to persist, got %q", got.AssigneeID())
	}
	if got.Status() != domaintask.StatusAssigned {
		t.Fatalf("expected status %q, got %q", domaintask.StatusAssigned, got.Status())
	}
}

func TestMongoTaskRepository_GetByIDReturnsNotFound(t *testing.T) {
	repo := newTestMongoTaskRepository(t)

	_, err := repo.GetByID(context.Background(), "missing")
	if err != ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestMongoTaskRepository_UpdateReturnsNotFound(t *testing.T) {
	repo := newTestMongoTaskRepository(t)

	_, err := repo.Update(context.Background(), domaintask.Restore(
		"missing",
		"title",
		"",
		domaintask.StatusOpen,
		"u_owner_001",
		"",
		time.Now().UTC(),
		time.Now().UTC(),
		nil,
		"",
	))
	if err != ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestMongoTaskRepository_SearchAppliesFiltersAndPagination(t *testing.T) {
	repo := newTestMongoTaskRepository(t)
	now := time.Now().UTC()

	seed := []domaintask.Task{
		domaintask.Restore("task_alpha_001", "Alpha first", "", domaintask.StatusAssigned, "u_owner_001", "u_worker_001", now, now, nil, ""),
		domaintask.Restore("task_alpha_002", "Beta", "alpha detail", domaintask.StatusAssigned, "u_owner_001", "u_worker_001", now.Add(time.Minute), now.Add(time.Minute), nil, ""),
		domaintask.Restore("task_other_001", "Gamma", "", domaintask.StatusOpen, "u_owner_001", "", now.Add(2*time.Minute), now.Add(2*time.Minute), nil, ""),
	}
	for _, task := range seed {
		if _, err := repo.Create(context.Background(), task); err != nil {
			t.Fatalf("seed task %s: %v", task.ID(), err)
		}
	}

	result, err := repo.Search(context.Background(), ports.TaskListQuery{
		Query:  "alpha",
		Status: domaintask.StatusAssigned.String(),
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Tasks) != 1 || result.Tasks[0].ID() != "task_alpha_002" {
		t.Fatalf("unexpected search result: %+v", result.Tasks)
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
