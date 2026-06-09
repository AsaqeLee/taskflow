package config

import (
	"log"
	"os"
	"strconv"

	"github.com/go-playground/validator/v10"
)

const (
	defaultPort             = "8080"
	defaultMongoURI         = "mongodb://localhost:27017"
	defaultMongoDB          = "taskflow"
	defaultRepositoryDriver = RepositoryDriverMemory
)

const RepositoryDriverMemory = "memory"
const RepositoryDriverMongo = "mongo"

type Config struct {
	Port             string `validate:"required,port"`
	MongoURI         string `validate:"required,uri"`
	MongoDB          string `validate:"required"`
	RepositoryDriver string `validate:"required,oneof=memory mongo"`
	JWTSecret        string `validate:"required"`
	DevMode          bool
}

func Load() Config {
	port := getenv("PORT", defaultPort)
	mongoURI := getenv("MONGODB_URI", defaultMongoURI)
	mongoDB := getenv("MONGODB_DATABASE", defaultMongoDB)
	repositoryDriver := normalizeRepositoryDriver(getenv("TASK_REPOSITORY_DRIVER", defaultRepositoryDriver))
	jwtSecret := getenv("JWT_SECRET", "taskflow_default_secret")
	devMode, _ := strconv.ParseBool(getenv("DEV_MODE", "false"))

	cfg := Config{
		Port:             port,
		MongoURI:         mongoURI,
		MongoDB:          mongoDB,
		RepositoryDriver: repositoryDriver,
		JWTSecret:        jwtSecret,
		DevMode:          devMode,
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	return cfg
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeRepositoryDriver(value string) string {
	switch value {
	case RepositoryDriverMongo:
		return RepositoryDriverMongo
	default:
		return RepositoryDriverMemory
	}
}
