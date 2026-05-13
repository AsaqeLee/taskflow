package config

import "os"

const (
	defaultPort             = "8080"
	defaultMongoURI         = "mongodb://localhost:27017"
	defaultMongoDB          = "taskflow"
	defaultRepositoryDriver = RepositoryDriverMemory
)

const RepositoryDriverMemory = "memory"
const RepositoryDriverMongo = "mongo"

type Config struct {
	Port             string
	MongoURI         string
	MongoDB          string
	RepositoryDriver string
}

func Load() Config {
	port := getenv("PORT", defaultPort)
	mongoURI := getenv("MONGODB_URI", defaultMongoURI)
	mongoDB := getenv("MONGODB_DATABASE", defaultMongoDB)
	repositoryDriver := normalizeRepositoryDriver(getenv("TASK_REPOSITORY_DRIVER", defaultRepositoryDriver))

	return Config{
		Port:             port,
		MongoURI:         mongoURI,
		MongoDB:          mongoDB,
		RepositoryDriver: repositoryDriver,
	}
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
