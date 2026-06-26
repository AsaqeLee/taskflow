package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PasswordResetDelivery interface {
	Deliver(ctx context.Context, notice PasswordResetNotice) error
}

type PasswordResetNotice struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type webhookPasswordResetDelivery struct {
	endpoint    string
	bearerToken string
	httpClient  *http.Client
}

type passwordResetWebhookPayload struct {
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewPasswordResetWebhookDelivery(
	endpoint string,
	bearerToken string,
	httpClient *http.Client,
) PasswordResetDelivery {
	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}
	return &webhookPasswordResetDelivery{
		endpoint:    endpoint,
		bearerToken: bearerToken,
		httpClient:  client,
	}
}

func (d *webhookPasswordResetDelivery) Deliver(ctx context.Context, notice PasswordResetNotice) error {
	payload, err := json.Marshal(passwordResetWebhookPayload{
		UserID:    notice.UserID,
		Token:     notice.Token,
		ExpiresAt: notice.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("marshal password reset webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build password reset webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if d.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+d.bearerToken)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deliver password reset webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("password reset webhook returned status %d", resp.StatusCode)
	}

	return nil
}
