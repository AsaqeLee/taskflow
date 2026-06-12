package main

import (
	"context"
	"log"

	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/observability"
)

func main() {
	cfg := config.Load()
	observability.ConfigureLogger(cfg)

	if cfg.RepositoryDriver != config.RepositoryDriverMongo {
		log.Fatal("migrate requires TASK_REPOSITORY_DRIVER=mongo")
	}

	db, err := database.New(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.ApplyMigrations(context.Background()); err != nil {
		log.Fatal(err)
	}
}
