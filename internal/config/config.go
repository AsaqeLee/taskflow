package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	defaultLoginRateWindow  = 5 * time.Minute
	defaultResetRateWindow  = 15 * time.Minute
)

const RepositoryDriverMemory = "memory"
const RepositoryDriverMongo = "mongo"

type Config struct {
	Port                           int    `validate:"required,gte=1,lte=65535"`
	MongoURI                       string `validate:"required,uri"`
	MongoDB                        string `validate:"required"`
	RepositoryDriver               string `validate:"required,oneof=memory mongo"`
	JWTSecret                      string `validate:"required"`
	DevMode                        bool
	LogLevel                       string        `validate:"required,oneof=debug info warn error"`
	AccessTokenTTL                 time.Duration `validate:"required,gt=0"`
	RefreshTokenTTL                time.Duration `validate:"required,gt=0"`
	PasswordResetTTL               time.Duration `validate:"required,gt=0"`
	RequestTimeout                 time.Duration `validate:"required,gt=0"`
	ShutdownTimeout                time.Duration `validate:"required,gt=0"`
	ServerReadTimeout              time.Duration `validate:"required,gt=0"`
	ServerWriteTimeout             time.Duration `validate:"required,gt=0"`
	RateLimitRequests              int           `validate:"required,gte=0"`
	RateLimitWindow                time.Duration `validate:"required,gt=0"`
	IdempotencyTTL                 time.Duration `validate:"required,gt=0"`
	LoginRateLimitRequests         int           `validate:"required,gte=0"`
	LoginRateLimitWindow           time.Duration `validate:"required,gt=0"`
	PasswordResetRateLimitRequests int           `validate:"required,gte=0"`
	PasswordResetRateLimitWindow   time.Duration `validate:"required,gt=0"`
	TracingEnabled                 bool
	TracingEndpoint                string
	TracingInsecure                bool
	TracingServiceName             string `validate:"required"`
	AppVersion                     string `validate:"required"`
	CORSAllowedOrigins             []string
	AllowPublicRegister            bool
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
			jwtSecret = newDevJWTSecret()
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
	loginRateLimitWindow := mustParseDuration("LOGIN_RATE_LIMIT_WINDOW", defaultLoginRateWindow)
	loginRateLimitRequests := mustParseInt("LOGIN_RATE_LIMIT_REQUESTS", 10)
	passwordResetRateLimitWindow := mustParseDuration("PASSWORD_RESET_RATE_LIMIT_WINDOW", defaultResetRateWindow)
	passwordResetRateLimitRequests := mustParseInt("PASSWORD_RESET_RATE_LIMIT_REQUESTS", 5)
	logLevel := normalizeLogLevel(getenv("LOG_LEVEL", "info"))
	tracingEnabled, _ := strconv.ParseBool(getenv("TRACING_ENABLED", "false"))
	tracingInsecure, _ := strconv.ParseBool(getenv("TRACING_INSECURE", "true"))
	tracingEndpoint := strings.TrimSpace(getenv("TRACING_ENDPOINT", ""))
	tracingServiceName := strings.TrimSpace(getenv("TRACING_SERVICE_NAME", "taskflow"))
	appVersion := getenv("APP_VERSION", "dev")
	allowPublicRegister := devMode
	if raw := strings.TrimSpace(os.Getenv("ALLOW_PUBLIC_REGISTER")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			log.Fatalf("Invalid ALLOW_PUBLIC_REGISTER configuration: %v", err)
		}
		allowPublicRegister = parsed
	}

	cfg := Config{
		Port:                           port,
		MongoURI:                       mongoURI,
		MongoDB:                        mongoDB,
		RepositoryDriver:               repositoryDriver,
		JWTSecret:                      jwtSecret,
		DevMode:                        devMode,
		LogLevel:                       logLevel,
		AccessTokenTTL:                 accessTokenTTL,
		RefreshTokenTTL:                refreshTokenTTL,
		PasswordResetTTL:               passwordResetTTL,
		RequestTimeout:                 requestTimeout,
		ShutdownTimeout:                shutdownTimeout,
		ServerReadTimeout:              serverReadTimeout,
		ServerWriteTimeout:             serverWriteTimeout,
		RateLimitRequests:              rateLimitRequests,
		RateLimitWindow:                rateLimitWindow,
		IdempotencyTTL:                 idempotencyTTL,
		LoginRateLimitRequests:         loginRateLimitRequests,
		LoginRateLimitWindow:           loginRateLimitWindow,
		PasswordResetRateLimitRequests: passwordResetRateLimitRequests,
		PasswordResetRateLimitWindow:   passwordResetRateLimitWindow,
		TracingEnabled:                 tracingEnabled,
		TracingEndpoint:                tracingEndpoint,
		TracingInsecure:                tracingInsecure,
		TracingServiceName:             tracingServiceName,
		AppVersion:                     appVersion,
		AllowPublicRegister:            allowPublicRegister,
		CORSAllowedOrigins:             parseCSVOrigins(getenv("CORS_ALLOWED_ORIGINS", "")),
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	if !cfg.DevMode {
		if len(cfg.JWTSecret) < 32 {
			log.Fatal("Invalid configuration: JWT_SECRET must be at least 32 characters when DEV_MODE=false")
		}
	}
	if cfg.DevMode && cfg.AppVersion != "dev" && !strings.HasPrefix(cfg.AppVersion, "compose-") {
		log.Printf("WARNING: DEV_MODE=true with APP_VERSION=%s — do not use in intranet/production deployments", cfg.AppVersion)
	}

	return cfg
}

func parseCSVOrigins(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	return origins
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

	data, err := readConfiguredFile(filePath)
	if err != nil {
		log.Fatalf("Invalid %s configuration: %v", fileKey, err)
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		log.Fatalf("Invalid %s configuration: file is empty", fileKey)
	}
	return value
}

func newDevJWTSecret() string {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		log.Fatalf("Failed to generate dev JWT secret: %v", err)
	}
	return hex.EncodeToString(buf[:])
}

func readConfiguredFile(filePath string) ([]byte, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(filePath))
	if cleanPath == "" || cleanPath == "." || cleanPath == string(filepath.Separator) {
		return nil, fmt.Errorf("file path is empty")
	}
	if !filepath.IsAbs(cleanPath) && (cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator))) {
		return nil, fmt.Errorf("path traversal is not allowed")
	}

	root, err := os.OpenRoot(filepath.Dir(cleanPath))
	if err != nil {
		return nil, err
	}
	defer root.Close()

	return root.ReadFile(filepath.Base(cleanPath))
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
