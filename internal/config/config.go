package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	defaultPort             = 8080
	defaultMongoURI         = "mongodb://localhost:27017"
	defaultMongoDB          = "taskflow"
	defaultRepositoryDriver = RepositoryDriverMemory
	defaultAccessTokenTTL   = 2 * time.Hour
	defaultRefreshTokenTTL  = 7 * 24 * time.Hour
	defaultPasswordResetTTL = time.Hour
	defaultRequestTimeout   = 15 * time.Second
	defaultShutdownTimeout  = 10 * time.Second
	defaultServerReadTimout = 10 * time.Second
	defaultServerWriteTime  = 30 * time.Second
	defaultRateLimitWindow  = time.Minute
	defaultIdempotencyTTL   = 10 * time.Minute
)

const RepositoryDriverMemory = "memory"
const RepositoryDriverMongo = "mongo"

type Config struct {
	Port               int    `validate:"required,gte=1,lte=65535"`
	MongoURI           string `validate:"required,uri"`
	MongoDB            string `validate:"required"`
	RepositoryDriver   string `validate:"required,oneof=memory mongo"`
	JWTSecret          string `validate:"required"`
	DevMode            bool
	LogLevel           string        `validate:"required,oneof=debug info warn error"`
	AccessTokenTTL     time.Duration `validate:"required,gt=0"`
	RefreshTokenTTL    time.Duration `validate:"required,gt=0"`
	PasswordResetTTL   time.Duration `validate:"required,gt=0"`
	RequestTimeout     time.Duration `validate:"required,gt=0"`
	ShutdownTimeout    time.Duration `validate:"required,gt=0"`
	ServerReadTimeout  time.Duration `validate:"required,gt=0"`
	ServerWriteTimeout time.Duration `validate:"required,gt=0"`
	RateLimitRequests  int           `validate:"required,gte=0"`
	RateLimitWindow    time.Duration `validate:"required,gt=0"`
	IdempotencyTTL     time.Duration `validate:"required,gt=0"`
	TracingEnabled     bool
	TracingEndpoint    string
	TracingInsecure    bool
	TracingServiceName string `validate:"required"`
	AppVersion         string `validate:"required"`
}

func Load() Config {
	devMode, _ := strconv.ParseBool(getenv("DEV_MODE", "false"))
	portStr := getenv("PORT", strconv.Itoa(defaultPort))
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid PORT configuration: %v", err)
	}

	mongoURI := getenvWithFile("MONGODB_URI", "MONGODB_URI_FILE", defaultMongoURI)
	mongoDB := getenv("MONGODB_DATABASE", defaultMongoDB)
	repositoryDriver := normalizeRepositoryDriver(getenv("TASK_REPOSITORY_DRIVER", RepositoryDriverMemory))
	jwtSecret := strings.TrimSpace(getenvWithFile("JWT_SECRET", "JWT_SECRET_FILE", ""))
	if jwtSecret == "" {
		if devMode {
			jwtSecret = "taskflow_dev_only_secret"
		} else {
			log.Fatal("Invalid configuration: JWT_SECRET or JWT_SECRET_FILE is required")
		}
	}

	accessTokenTTL := mustParseDuration("ACCESS_TOKEN_TTL", defaultAccessTokenTTL)
	refreshTokenTTL := mustParseDuration("REFRESH_TOKEN_TTL", defaultRefreshTokenTTL)
	passwordResetTTL := mustParseDuration("PASSWORD_RESET_TTL", defaultPasswordResetTTL)
	requestTimeout := mustParseDuration("REQUEST_TIMEOUT", defaultRequestTimeout)
	shutdownTimeout := mustParseDuration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	serverReadTimeout := mustParseDuration("SERVER_READ_TIMEOUT", defaultServerReadTimout)
	serverWriteTimeout := mustParseDuration("SERVER_WRITE_TIMEOUT", defaultServerWriteTime)
	rateLimitWindow := mustParseDuration("RATE_LIMIT_WINDOW", defaultRateLimitWindow)
	idempotencyTTL := mustParseDuration("IDEMPOTENCY_TTL", defaultIdempotencyTTL)
	rateLimitRequests := mustParseInt("RATE_LIMIT_REQUESTS", 120)
	logLevel := normalizeLogLevel(getenv("LOG_LEVEL", "info"))
	tracingEnabled, _ := strconv.ParseBool(getenv("TRACING_ENABLED", "false"))
	tracingInsecure, _ := strconv.ParseBool(getenv("TRACING_INSECURE", "true"))
	tracingEndpoint := strings.TrimSpace(getenv("TRACING_ENDPOINT", ""))
	tracingServiceName := strings.TrimSpace(getenv("TRACING_SERVICE_NAME", "taskflow"))
	appVersion := getenv("APP_VERSION", "dev")

	cfg := Config{
		Port:               port,
		MongoURI:           mongoURI,
		MongoDB:            mongoDB,
		RepositoryDriver:   repositoryDriver,
		JWTSecret:          jwtSecret,
		DevMode:            devMode,
		LogLevel:           logLevel,
		AccessTokenTTL:     accessTokenTTL,
		RefreshTokenTTL:    refreshTokenTTL,
		PasswordResetTTL:   passwordResetTTL,
		RequestTimeout:     requestTimeout,
		ShutdownTimeout:    shutdownTimeout,
		ServerReadTimeout:  serverReadTimeout,
		ServerWriteTimeout: serverWriteTimeout,
		RateLimitRequests:  rateLimitRequests,
		RateLimitWindow:    rateLimitWindow,
		IdempotencyTTL:     idempotencyTTL,
		TracingEnabled:     tracingEnabled,
		TracingEndpoint:    tracingEndpoint,
		TracingInsecure:    tracingInsecure,
		TracingServiceName: tracingServiceName,
		AppVersion:         appVersion,
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

func getenvWithFile(key, fileKey, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	filePath := strings.TrimSpace(os.Getenv(fileKey))
	if filePath == "" {
		return fallback
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Invalid %s configuration: %v", fileKey, err)
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		log.Fatalf("Invalid %s configuration: file is empty", fileKey)
	}
	return value
}

func mustParseDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("Invalid %s configuration: %v", key, err)
	}
	return duration
}

func mustParseInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Invalid %s configuration: %v", key, err)
	}
	return number
}

func normalizeRepositoryDriver(value string) string {
	switch value {
	case RepositoryDriverMongo:
		return RepositoryDriverMongo
	default:
		return RepositoryDriverMemory
	}
}

func normalizeLogLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug", "info", "warn", "error":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		log.Fatalf("Invalid LOG_LEVEL configuration: %v", errors.New("must be one of debug, info, warn, error"))
		return ""
	}
}

func (c Config) ServerAddress() string {
	return fmt.Sprintf(":%d", c.Port)
}
