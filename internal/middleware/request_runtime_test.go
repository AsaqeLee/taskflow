package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestIdempotencyMiddlewareReplaysStoredResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := NewIdempotencyStore(time.Minute)
	var executions atomic.Int32

	r := gin.New()
	r.Use(RequestContext())
	r.Use(Idempotency(store))
	r.POST("/echo", func(c *gin.Context) {
		executions.Add(1)
		c.JSON(http.StatusCreated, gin.H{"ok": true, "count": executions.Load()})
	})

	firstReq := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"value":"same"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Idempotency-Key", "abc123")
	firstResp := httptest.NewRecorder()
	r.ServeHTTP(firstResp, firstReq)

	if firstResp.Code != http.StatusCreated {
		t.Fatalf("expected first request status 201, got %d body=%s", firstResp.Code, firstResp.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"value":"same"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Idempotency-Key", "abc123")
	secondResp := httptest.NewRecorder()
	r.ServeHTTP(secondResp, secondReq)

	if secondResp.Code != http.StatusCreated {
		t.Fatalf("expected replayed request status 201, got %d body=%s", secondResp.Code, secondResp.Body.String())
	}
	if secondResp.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("expected replay response header to be set")
	}
	if executions.Load() != 1 {
		t.Fatalf("expected handler to execute once, got %d", executions.Load())
	}

	var body map[string]any
	if err := json.Unmarshal(secondResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode replayed body: %v", err)
	}
	if body["count"] != float64(1) {
		t.Fatalf("expected replayed response body, got %v", body)
	}
}

func TestIdempotencyMiddlewareRejectsConflictingPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := NewIdempotencyStore(time.Minute)

	r := gin.New()
	r.Use(RequestContext())
	r.Use(Idempotency(store))
	r.POST("/echo", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	firstReq := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"value":"same"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Idempotency-Key", "abc123")
	firstResp := httptest.NewRecorder()
	r.ServeHTTP(firstResp, firstReq)

	secondReq := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"value":"different"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Idempotency-Key", "abc123")
	secondResp := httptest.NewRecorder()
	r.ServeHTTP(secondResp, secondReq)

	if secondResp.Code != http.StatusConflict {
		t.Fatalf("expected conflict status 409, got %d body=%s", secondResp.Code, secondResp.Body.String())
	}
}

func TestRateLimitMiddlewareRejectsExcessRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RequestContext())
	r.Use(RateLimit(NewRateLimiter(1, time.Minute)))
	r.GET("/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	firstReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	firstResp := httptest.NewRecorder()
	r.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected first request status 200, got %d body=%s", firstResp.Code, firstResp.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	secondResp := httptest.NewRecorder()
	r.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status 429, got %d body=%s", secondResp.Code, secondResp.Body.String())
	}
}

func TestMemoryIdempotencyStoreMarksPendingRequestsInProgress(t *testing.T) {
	store := NewMemoryIdempotencyStore(time.Minute)
	now := time.Now().UTC()

	firstDecision, _, err := store.Reserve(context.Background(), "POST|/tasks|127.0.0.1", "same-key", "payload-sum", now)
	if err != nil {
		t.Fatalf("first reserve returned error: %v", err)
	}
	if firstDecision != IdempotencyDecisionAccept {
		t.Fatalf("expected first decision to accept, got %v", firstDecision)
	}

	secondDecision, _, err := store.Reserve(context.Background(), "POST|/tasks|127.0.0.1", "same-key", "payload-sum", now.Add(time.Second))
	if err != nil {
		t.Fatalf("second reserve returned error: %v", err)
	}
	if secondDecision != IdempotencyDecisionInProgress {
		t.Fatalf("expected second decision to report in progress, got %v", secondDecision)
	}
}
