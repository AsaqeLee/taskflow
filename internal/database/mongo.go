package database

import (
	"context"
	"time"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/migrations"
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

func (c *Client) ApplyMigrations(ctx context.Context) error {
	if c == nil || c.Mongo == nil {
		return nil
	}

	return migrations.ApplyAll(ctx, c.Mongo.Database(c.DBName))
}

func (c *Client) EnsureIndexes(ctx context.Context) error {
	return c.ApplyMigrations(ctx)
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
