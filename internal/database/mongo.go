package database

import (
	"context"
	"time"

	"github.com/AsaqeLee/taskflow/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

	return &Client{
		Mongo:  client,
		DBName: cfg.MongoDB,
	}, nil
}
