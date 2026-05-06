package config

import "os"

const (
	defaultPort     = "8080"
	defaultMongoURI = "mongodb://localhost:27017"
	defaultMongoDB  = "taskflow"
)

type Config struct {
	Port     string
	MongoURI string
	MongoDB  string
}

func Load() Config {
	port := getenv("PORT", defaultPort)
	mongoURI := getenv("MONGODB_URI", defaultMongoURI)
	mongoDB := getenv("MONGODB_DATABASE", defaultMongoDB)

	return Config{
		Port:     port,
		MongoURI: mongoURI,
		MongoDB:  mongoDB,
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
