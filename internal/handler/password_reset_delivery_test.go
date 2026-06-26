package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookPasswordResetDelivery_Deliver(t *testing.T) {
	var gotAuth string
	var gotBody passwordResetWebhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	delivery := NewPasswordResetWebhookDelivery(server.URL, "top-secret", server.Client())
	expiresAt := time.Now().UTC().Add(time.Hour).Round(time.Second)
	err := delivery.Deliver(context.Background(), PasswordResetNotice{
		UserID:    "u_reset_webhook",
		Token:     "opaque-reset-token",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}
	if gotAuth != "Bearer top-secret" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotBody.UserID != "u_reset_webhook" {
		t.Fatalf("expected user id u_reset_webhook, got %q", gotBody.UserID)
	}
	if gotBody.Token != "opaque-reset-token" {
		t.Fatalf("expected reset token payload, got %q", gotBody.Token)
	}
	if !gotBody.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expires_at %s, got %s", expiresAt, gotBody.ExpiresAt)
	}
}

func TestWebhookPasswordResetDelivery_DeliverReturnsErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	delivery := NewPasswordResetWebhookDelivery(server.URL, "", server.Client())
	err := delivery.Deliver(context.Background(), PasswordResetNotice{
		UserID:    "u_reset_webhook",
		Token:     "opaque-reset-token",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	if err == nil {
		t.Fatal("expected delivery error for non-2xx response")
	}
}
