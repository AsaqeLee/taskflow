package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongoUserRepository_ListOrdersByCreatedAtAndIDAscending(t *testing.T) {
	repo := newTestMongoUserRepository(t)
	now := time.Now().UTC()

	for _, account := range []domainuser.Account{
		domainuser.Restore("u_b", "Bravo", domainuser.RoleHuman, "hash", "", true, nil, "", now, now),
		domainuser.Restore("u_a", "Alpha", domainuser.RoleHuman, "hash", "", true, nil, "", now, now),
		domainuser.Restore("u_older", "Older", domainuser.RoleHuman, "hash", "", true, nil, "", now.Add(-time.Hour), now.Add(-time.Hour)),
	} {
		if _, err := repo.Create(context.Background(), account); err != nil {
			t.Fatalf("seed account %s: %v", account.ID(), err)
		}
	}

	items, err := repo.List(context.Background(), false)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 users, got %d", len(items))
	}

	got := []string{items[0].ID(), items[1].ID(), items[2].ID()}
	want := []string{"u_older", "u_a", "u_b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected user ordering: got %v want %v", got, want)
		}
	}
}

func TestMongoUserRepository_ListFiltersInactiveUsers(t *testing.T) {
	repo := newTestMongoUserRepository(t)
	now := time.Now().UTC()

	active := domainuser.Restore("u_active", "Active", domainuser.RoleHuman, "hash", "", true, nil, "", now, now)
	disabledAt := now.Add(time.Minute)
	inactive := domainuser.Restore("u_inactive", "Inactive", domainuser.RoleHuman, "hash", "", false, &disabledAt, "u_owner", now, disabledAt)

	for _, account := range []domainuser.Account{active, inactive} {
		if _, err := repo.Create(context.Background(), account); err != nil {
			t.Fatalf("seed account %s: %v", account.ID(), err)
		}
	}

	items, err := repo.List(context.Background(), true)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 active user, got %d", len(items))
	}
	if items[0].ID() != "u_active" {
		t.Fatalf("expected active user only, got %q", items[0].ID())
	}
}

func newTestMongoUserRepository(t *testing.T) *MongoUserRepository {
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
		_ = client.Database(dbName).Collection("users").Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	collection := client.Database(dbName).Collection("users")
	if err := collection.Drop(ctx); err != nil {
		var cmdErr mongo.CommandError
		if !(errors.As(err, &cmdErr) && cmdErr.Code == 26) {
			t.Fatalf("drop collection: %v", err)
		}
	}

	return NewMongoUserRepository(collection)
}
