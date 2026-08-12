package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const testMongoURIEnv = "TASKFLOW_MONGO_TEST_URI"

func TestMongoRateLimiterSharesStateAcrossInstances(t *testing.T) {
	collection := newMongoRuntimeCollection(t, "runtime_rate_limits")

	limiterA := NewMongoRateLimiter(collection, 1, time.Minute)
	limiterB := NewMongoRateLimiter(collection, 1, time.Minute)

	// Pin inside a window, away from Truncate(time.Minute) boundaries.
	// Using wall-clock now/now+1s can straddle the minute edge and create two keys.
	now := time.Now().UTC().Truncate(time.Minute).Add(30 * time.Second)
	clientID := "shared-client-" + bson.NewObjectID().Hex()

	allowed, err := limiterA.Allow(context.Background(), clientID, now)
	if err != nil {
		t.Fatalf("limiterA.Allow returned error: %v", err)
	}
	if !allowed {
		t.Fatalf("expected first limiter to allow request")
	}

	// Same window, different limiter instance — must observe shared Mongo state.
	allowed, err = limiterB.Allow(context.Background(), clientID, now.Add(time.Second))
	if err != nil {
		t.Fatalf("limiterB.Allow returned error: %v", err)
	}
	if allowed {
		t.Fatalf("expected second limiter to observe shared limit state and reject request")
	}
}

func TestMongoIdempotencyStoreSharesPendingAndReplayState(t *testing.T) {
	collection := newMongoRuntimeCollection(t, "runtime_idempotency_keys")

	storeA := NewMongoIdempotencyStore(collection, time.Minute)
	storeB := NewMongoIdempotencyStore(collection, time.Minute)

	now := time.Now().UTC()
	decision, _, err := storeA.Reserve(context.Background(), "POST|/tasks|127.0.0.1", "idem-key", "request-sum", now)
	if err != nil {
		t.Fatalf("storeA.Reserve returned error: %v", err)
	}
	if decision != IdempotencyDecisionAccept {
		t.Fatalf("expected first reserve to accept, got %v", decision)
	}

	decision, _, err = storeB.Reserve(context.Background(), "POST|/tasks|127.0.0.1", "idem-key", "request-sum", now.Add(time.Second))
	if err != nil {
		t.Fatalf("storeB.Reserve pending returned error: %v", err)
	}
	if decision != IdempotencyDecisionInProgress {
		t.Fatalf("expected second reserve to observe in-progress state, got %v", decision)
	}

	if err := storeA.Complete(context.Background(), "POST|/tasks|127.0.0.1", "idem-key", "request-sum", IdempotencyRecord{
		Scope:     "POST|/tasks|127.0.0.1",
		Method:    "POST",
		Path:      "/tasks",
		Status:    201,
		Body:      []byte(`{"ok":true}`),
		Headers:   map[string]string{"Content-Type": "application/json"},
		ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("storeA.Complete returned error: %v", err)
	}

	decision, record, err := storeB.Reserve(context.Background(), "POST|/tasks|127.0.0.1", "idem-key", "request-sum", now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("storeB.Reserve replay returned error: %v", err)
	}
	if decision != IdempotencyDecisionReplay {
		t.Fatalf("expected replay decision after complete, got %v", decision)
	}
	if string(record.Body) != `{"ok":true}` {
		t.Fatalf("expected replay body to be preserved, got %s", string(record.Body))
	}
}

func TestMongoIdempotencyMiddlewareReleasesTimedOutRequest(t *testing.T) {
	collection := newMongoRuntimeCollection(t, "runtime_idempotency_timeout")
	store := NewMongoIdempotencyStore(collection, time.Minute)
	var executions atomic.Int32

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestContext())
	router.Use(Idempotency(store, nil))
	// Keep the tiny timeout scoped to handler execution instead of Mongo reserve latency.
	router.Use(Timeout(10 * time.Millisecond))
	router.POST("/slow", func(c *gin.Context) {
		executions.Add(1)
		time.Sleep(25 * time.Millisecond)
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	firstReq := httptest.NewRequest(http.MethodPost, "/slow", strings.NewReader(`{"value":"same"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Idempotency-Key", "slow-key")
	firstResp := httptest.NewRecorder()
	router.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected first request timeout status 504, got %d body=%s", firstResp.Code, firstResp.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/slow", strings.NewReader(`{"value":"same"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Idempotency-Key", "slow-key")
	secondResp := httptest.NewRecorder()
	router.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected second request timeout status 504, got %d body=%s", secondResp.Code, secondResp.Body.String())
	}
	if secondResp.Header().Get("Idempotent-Replayed") != "" {
		t.Fatalf("expected timed out request not to replay cached response")
	}
	if got := executions.Load(); got != 2 {
		t.Fatalf("expected timed out request release idempotency reservation, got %d executions", got)
	}
}

func newMongoRuntimeCollection(t *testing.T, collectionName string) *mongo.Collection {
	t.Helper()

	uri := os.Getenv(testMongoURIEnv)
	if uri == "" {
		t.Skipf("%s not set; skipping Mongo runtime integration test", testMongoURIEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}

	db := client.Database("taskflow_runtime_test_" + bson.NewObjectID().Hex())
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	collection := db.Collection(collectionName)
	if err := collection.Drop(ctx); err != nil && !isNamespaceMissing(err) {
		t.Fatalf("drop collection %s: %v", collectionName, err)
	}

	return collection
}

func isNamespaceMissing(err error) bool {
	var commandErr mongo.CommandError
	return errors.As(err, &commandErr) && commandErr.Code == 26
}
