package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadConfiguredFile(t *testing.T) {
	t.Run("reads from clean path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "secret.txt")
		if err := os.WriteFile(path, []byte("secret-value"), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		data, err := readConfiguredFile(path)
		if err != nil {
			t.Fatalf("readConfiguredFile returned error: %v", err)
		}
		if string(data) != "secret-value" {
			t.Fatalf("expected secret-value, got %q", string(data))
		}
	})

	t.Run("rejects parent traversal", func(t *testing.T) {
		if _, err := readConfiguredFile("../secret.txt"); err == nil {
			t.Fatal("expected parent traversal to be rejected")
		}
	})
}

func TestNewDevJWTSecret(t *testing.T) {
	first := newDevJWTSecret()
	second := newDevJWTSecret()

	if len(first) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(first))
	}
	if first == second {
		t.Fatal("expected generated dev secrets to differ")
	}
}

func TestValidateProductionConfig(t *testing.T) {
	t.Run("accepts strict production config", func(t *testing.T) {
		if err := validateProductionConfig(validProductionConfig()); err != nil {
			t.Fatalf("validateProductionConfig returned error: %v", err)
		}
	})

	t.Run("rejects short jwt secret outside dev mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.StrictProductionConfig = false
		cfg.JWTSecret = "short-secret"
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected short JWT secret to be rejected")
		}
	})

	t.Run("rejects dev mode when strict production enabled", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.DevMode = true
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected dev mode to be rejected")
		}
	})

	t.Run("rejects memory repository in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.RepositoryDriver = RepositoryDriverMemory
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected memory repository to be rejected")
		}
	})

	t.Run("rejects placeholder jwt secret in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.JWTSecret = placeholderJWTSecretCompose
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected placeholder JWT secret to be rejected")
		}
	})

	t.Run("rejects placeholder app version in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.AppVersion = placeholderAppVersionLocal
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected placeholder app version to be rejected")
		}
	})

	t.Run("rejects public register in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.AllowPublicRegister = true
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected public register to be rejected")
		}
	})

	t.Run("rejects missing password reset webhook in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.PasswordResetWebhookURL = ""
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected missing password reset webhook to be rejected")
		}
	})

	t.Run("rejects non-https password reset webhook in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.PasswordResetWebhookURL = "http://mailer.internal/hooks/password-reset"
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected non-https password reset webhook to be rejected")
		}
	})

	t.Run("rejects missing password reset webhook auth token in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.PasswordResetWebhookAuthToken = ""
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected missing password reset webhook auth token to be rejected")
		}
	})

	t.Run("rejects invalid password reset webhook url", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.StrictProductionConfig = false
		cfg.PasswordResetWebhookURL = "not-a-url"
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected invalid password reset webhook URL to be rejected")
		}
	})

	t.Run("rejects localhost cors origin in strict mode", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.CORSAllowedOrigins = []string{"http://localhost:5173"}
		if err := validateProductionConfig(cfg); err == nil {
			t.Fatal("expected localhost CORS origin to be rejected")
		}
	})
}

func validProductionConfig() Config {
	return Config{
		Port:                           8080,
		MongoURI:                       "mongodb://mongo:27017/?replicaSet=rs0",
		MongoDB:                        "taskflow",
		RepositoryDriver:               RepositoryDriverMongo,
		JWTSecret:                      "0123456789abcdef0123456789abcdef",
		DevMode:                        false,
		LogLevel:                       "info",
		AccessTokenTTL:                 2 * time.Hour,
		RefreshTokenTTL:                7 * 24 * time.Hour,
		PasswordResetTTL:               time.Hour,
		PasswordResetWebhookURL:        "https://mailer.internal/hooks/password-reset",
		PasswordResetWebhookAuthToken:  "reset-webhook-token",
		RequestTimeout:                 15 * time.Second,
		ShutdownTimeout:                10 * time.Second,
		ServerReadTimeout:              10 * time.Second,
		ServerWriteTimeout:             30 * time.Second,
		RateLimitRequests:              120,
		RateLimitWindow:                time.Minute,
		IdempotencyTTL:                 10 * time.Minute,
		LoginRateLimitRequests:         10,
		LoginRateLimitWindow:           5 * time.Minute,
		PasswordResetRateLimitRequests: 5,
		PasswordResetRateLimitWindow:   15 * time.Minute,
		TracingEnabled:                 false,
		TracingEndpoint:                "",
		TracingInsecure:                true,
		TracingServiceName:             "taskflow",
		AppVersion:                     "0d7edf3",
		CORSAllowedOrigins:             []string{"https://taskflow.internal"},
		AllowPublicRegister:            false,
		StrictProductionConfig:         true,
	}
}
