package migrations

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const testMongoURIEnv = "TASKFLOW_MONGO_TEST_URI"

func TestDefinitionsAreSortedAndUnique(t *testing.T) {
	if len(Definitions) == 0 {
		t.Fatalf("expected at least one migration definition")
	}

	versions := make([]string, 0, len(Definitions))
	for _, definition := range Definitions {
		if definition.Version == "" {
			t.Fatalf("found migration with empty version")
		}
		if definition.Description == "" {
			t.Fatalf("migration %s is missing description", definition.Version)
		}
		if definition.Up == nil {
			t.Fatalf("migration %s is missing Up function", definition.Version)
		}
		if slices.Contains(versions, definition.Version) {
			t.Fatalf("duplicate migration version %s", definition.Version)
		}
		versions = append(versions, definition.Version)
	}

	sortedVersions := append([]string(nil), versions...)
	slices.Sort(sortedVersions)
	if !slices.Equal(versions, sortedVersions) {
		t.Fatalf("expected migration versions to be lexically sorted, got %v", versions)
	}
}

func TestApplyAllAppliesMigrationsOnce(t *testing.T) {
	uri := os.Getenv(testMongoURIEnv)
	if uri == "" {
		t.Skipf("%s not set; skipping Mongo migration integration test", testMongoURIEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	defer client.Disconnect(context.Background())

	dbName := "taskflow_migrations_test_" + bson.NewObjectID().Hex()
	db := client.Database(dbName)
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
	})

	if err := ApplyAll(ctx, db); err != nil {
		t.Fatalf("ApplyAll first run returned error: %v", err)
	}
	if err := ApplyAll(ctx, db); err != nil {
		t.Fatalf("ApplyAll second run returned error: %v", err)
	}

	count, err := db.Collection(schemaMigrationsCollection).CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("count schema migrations: %v", err)
	}
	if count != int64(len(Definitions)) {
		t.Fatalf("expected %d applied migrations, got %d", len(Definitions), count)
	}

	if !collectionHasIndexKey(ctx, t, db.Collection("refresh_tokens"), "token_hash") {
		t.Fatalf("expected refresh_tokens to have token_hash index")
	}
	if !collectionHasIndexKey(ctx, t, db.Collection("runtime_idempotency_keys"), "expires_at") {
		t.Fatalf("expected runtime_idempotency_keys to have expires_at index")
	}
}

func collectionHasIndexKey(ctx context.Context, t *testing.T, collection *mongo.Collection, key string) bool {
	t.Helper()

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		t.Fatalf("list indexes for %s: %v", collection.Name(), err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			t.Fatalf("decode index doc for %s: %v", collection.Name(), err)
		}

		keyDoc, ok := doc["key"].(bson.M)
		if ok {
			if _, exists := keyDoc[key]; exists {
				return true
			}
			continue
		}

		keyList, ok := doc["key"].(bson.D)
		if ok {
			for _, item := range keyList {
				if item.Key == key {
					return true
				}
			}
			continue
		}

		keyMap, ok := doc["key"].(map[string]any)
		if ok {
			if _, exists := keyMap[key]; exists {
				return true
			}
		}
	}
	if err := cursor.Err(); err != nil {
		t.Fatalf("iterate indexes for %s: %v", collection.Name(), err)
	}

	return false
}
