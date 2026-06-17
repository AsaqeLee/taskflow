package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/AsaqeLee/taskflow/internal/bootstrap"
	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/repository"
)

func main() {
	usersFile := flag.String("users", "", "path to users JSON file (required)")
	flag.Parse()

	if *usersFile == "" {
		if env := os.Getenv("USERS_FILE"); env != "" {
			*usersFile = env
		}
	}
	if *usersFile == "" {
		log.Fatal("users file is required: pass -users or set USERS_FILE")
	}

	cfg := config.Load()
	observability.ConfigureLogger(cfg)

	ctx := context.Background()
	var userRepo repository.UserRepository

	switch cfg.RepositoryDriver {
	case config.RepositoryDriverMongo:
		db, err := database.New(ctx, cfg)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.ApplyMigrations(ctx); err != nil {
			log.Fatal(err)
		}
		mongoDB := db.Mongo.Database(cfg.MongoDB)
		userRepo = repository.NewMongoUserRepository(mongoDB.Collection("users"))
	default:
		userRepo = repository.NewMemoryUserRepository()
	}

	if err := bootstrap.SeedUsersFromFile(ctx, userRepo, *usersFile); err != nil {
		log.Fatal(err)
	}
	log.Printf("bootstrap completed from %s", *usersFile)
}
