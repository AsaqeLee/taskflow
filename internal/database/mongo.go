package database

import (
	"context"
	"fmt"
	"time"

	"github.com/AsaqeLee/taskflow/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const connectTimeout = 5 * time.Second

type Client struct {
	Mongo  *mongo.Client
	DBName string
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &Client{
		Mongo:  client,
		DBName: cfg.MongoDB,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.Mongo == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	return c.Mongo.Ping(ctx, readpref.Primary())
}

func (c *Client) EnsureIndexes(ctx context.Context) error {
	if c == nil || c.Mongo == nil {
		return nil
	}

	db := c.Mongo.Database(c.DBName)
	indexSpecs := map[string][]mongo.IndexModel{
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
	}

	for collectionName, models := range indexSpecs {
		collection := db.Collection(collectionName)
		if _, err := collection.Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("ensure indexes for %s: %w", collectionName, err)
		}
	}

	return nil
}

func (c *Client) RunTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := c.Mongo.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx context.Context) (interface{}, error) {
		return nil, fn(sessCtx)
	})
	return err
}
