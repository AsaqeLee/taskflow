package config

import (
	"os"
	"path/filepath"
	"testing"
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
