package migrations

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const schemaMigrationsCollection = "schema_migrations"

type Migration struct {
	Version     string
	Description string
	Up          func(ctx context.Context, db *mongo.Database) error
}

type migrationDocument struct {
	Version     string    `bson:"_id"`
	Description string    `bson:"description"`
	AppliedAt   time.Time `bson:"applied_at"`
}

var Definitions = []Migration{
	{
		Version:     "2026061101",
		Description: "create core taskflow indexes",
		Up: func(ctx context.Context, db *mongo.Database) error {
			return createIndexes(ctx, db, map[string][]mongo.IndexModel{
				"tasks": {
					{
						Keys: bson.D{
							{Key: "creator_id", Value: 1},
							{Key: "status", Value: 1},
							{Key: "deleted_at", Value: 1},
						},
					},
					{
						Keys: bson.D{
							{Key: "assignee_id", Value: 1},
							{Key: "status", Value: 1},
							{Key: "deleted_at", Value: 1},
						},
					},
				},
				"task_records": {
					{
						Keys: bson.D{
							{Key: "task_id", Value: 1},
							{Key: "created_at", Value: 1},
						},
					},
				},
				"audit_logs": {
					{
						Keys: bson.D{
							{Key: "task_id", Value: 1},
							{Key: "created_at", Value: 1},
						},
					},
					{
						Keys: bson.D{
							{Key: "request_id", Value: 1},
						},
						Options: options.Index().SetSparse(true),
					},
				},
				"users": {
					{
						Keys: bson.D{
							{Key: "token", Value: 1},
						},
						Options: options.Index().SetUnique(true).SetSparse(true),
					},
				},
			})
		},
	},
	{
		Version:     "2026061201",
		Description: "create identity and runtime shared-state indexes",
		Up: func(ctx context.Context, db *mongo.Database) error {
			return createIndexes(ctx, db, map[string][]mongo.IndexModel{
				"refresh_tokens": {
					{
						Keys:    bson.D{{Key: "token_hash", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{
						Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "expires_at", Value: 1}},
					},
					{
						Keys:    bson.D{{Key: "expires_at", Value: 1}},
						Options: options.Index().SetExpireAfterSeconds(0),
					},
				},
				"password_reset_tokens": {
					{
						Keys:    bson.D{{Key: "token_hash", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{
						Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "expires_at", Value: 1}},
					},
					{
						Keys:    bson.D{{Key: "expires_at", Value: 1}},
						Options: options.Index().SetExpireAfterSeconds(0),
					},
				},
				"runtime_rate_limits": {
					{
						Keys:    bson.D{{Key: "expires_at", Value: 1}},
						Options: options.Index().SetExpireAfterSeconds(0),
					},
				},
				"runtime_idempotency_keys": {
					{
						Keys:    bson.D{{Key: "expires_at", Value: 1}},
						Options: options.Index().SetExpireAfterSeconds(0),
					},
				},
			})
		},
	},
	{
		Version:     "2026062501",
		Description: "add user listing index",
		Up: func(ctx context.Context, db *mongo.Database) error {
			return createIndexes(ctx, db, map[string][]mongo.IndexModel{
				"users": {
					{
						Keys: bson.D{
							{Key: "active", Value: 1},
							{Key: "created_at", Value: 1},
							{Key: "_id", Value: 1},
						},
					},
				},
			})
		},
	},
}

func ApplyAll(ctx context.Context, db *mongo.Database) error {
	applied, err := appliedMigrations(ctx, db.Collection(schemaMigrationsCollection))
	if err != nil {
		return err
	}

	migrationsCollection := db.Collection(schemaMigrationsCollection)
	for _, migration := range Definitions {
		if applied[migration.Version] {
			continue
		}
		if err := migration.Up(ctx, db); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Version, err)
		}
		if _, err := migrationsCollection.InsertOne(ctx, migrationDocument{
			Version:     migration.Version,
			Description: migration.Description,
			AppliedAt:   time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("persist migration %s: %w", migration.Version, err)
		}
	}

	return nil
}

func appliedMigrations(ctx context.Context, collection *mongo.Collection) (map[string]bool, error) {
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	applied := make(map[string]bool)
	for cursor.Next(ctx) {
		var doc migrationDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		applied[doc.Version] = true
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func createIndexes(ctx context.Context, db *mongo.Database, specs map[string][]mongo.IndexModel) error {
	for collectionName, models := range specs {
		if len(models) == 0 {
			continue
		}
		if _, err := db.Collection(collectionName).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("ensure indexes for %s: %w", collectionName, err)
		}
	}
	return nil
}
