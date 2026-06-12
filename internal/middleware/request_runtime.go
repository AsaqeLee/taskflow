package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AsaqeLee/taskflow/internal/httpapi"
	"github.com/AsaqeLee/taskflow/internal/observability"
	"github.com/AsaqeLee/taskflow/internal/requestmeta"
	"github.com/gin-gonic/gin"
)

type responseCaptureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *responseCaptureWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

type idempotencyRecord struct {
	Scope      string
	Method     string
	Path       string
	RequestSum string
	Status     int
	Body       []byte
	Headers    map[string]string
	ExpiresAt  time.Time
}

type IdempotencyStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	records map[string]idempotencyRecord
}

func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	return &IdempotencyStore{
		ttl:     ttl,
		records: make(map[string]idempotencyRecord),
	}
}

type rateLimitBucket struct {
	Count     int
	WindowEnd time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]rateLimitBucket
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]rateLimitBucket),
	}
}

func RequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newID()
		}

		traceID := traceIDFromHeaders(c)
		if traceID == "" {
			traceID = requestID
		}

		meta := requestmeta.Meta{
			RequestID:      requestID,
			TraceID:        traceID,
			IdempotencyKey: c.GetHeader("Idempotency-Key"),
			SourceIP:       c.ClientIP(),
			UserAgent:      c.Request.UserAgent(),
			StartedAt:      time.Now().UTC(),
		}

		c.Header("X-Request-ID", requestID)
		c.Header("X-Trace-ID", traceID)
		c.Request = c.Request.WithContext(requestmeta.WithContext(c.Request.Context(), meta))
		c.Next()
	}
}

func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			httpapi.WriteError(c, http.StatusGatewayTimeout, "request_timeout", "request exceeded timeout")
		}
	}
}

func StructuredLogger(metrics *observability.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		duration := time.Since(startedAt)
		route := c.FullPath()
		if metrics != nil {
			metrics.ObserveRequest(c.Request.Method, route, c.Writer.Status(), duration)
		}

		meta := requestmeta.FromContext(c.Request.Context())
		slog.Info(
			"http_request",
			slog.String("request_id", meta.RequestID),
			slog.String("trace_id", meta.TraceID),
			slog.String("user_id", meta.UserID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("route", route),
			slog.Int("status", c.Writer.Status()),
			slog.Int("bytes", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
			slog.Duration("duration", duration),
		)
	}
}

func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil || limiter.limit <= 0 || limiter.window <= 0 {
			c.Next()
			return
		}

		if !limiter.Allow(c.ClientIP(), time.Now().UTC()) {
			httpapi.AbortError(c, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
			return
		}

		c.Next()
	}
}

func (l *RateLimiter) Allow(clientID string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.clients[clientID]
	if !ok || now.After(bucket.WindowEnd) {
		l.clients[clientID] = rateLimitBucket{
			Count:     1,
			WindowEnd: now.Add(l.window),
		}
		return true
	}

	if bucket.Count >= l.limit {
		return false
	}

	bucket.Count++
	l.clients[clientID] = bucket
	return true
}

func Idempotency(store *IdempotencyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil || store.ttl <= 0 {
			c.Next()
			return
		}
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPatch && c.Request.Method != http.MethodDelete {
			c.Next()
			return
		}

		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if key == "" {
			c.Next()
			return
		}

		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			httpapi.AbortError(c, http.StatusBadRequest, "invalid_request_body", "failed to read request body")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))

		scope := c.Request.Method + "|" + c.Request.URL.Path + "|" + c.ClientIP()
		sum := sha256.Sum256(payload)
		requestSum := hex.EncodeToString(sum[:])

		if replayed, ok := store.Lookup(scope, key, requestSum); ok {
			for headerKey, headerValue := range replayed.Headers {
				c.Header(headerKey, headerValue)
			}
			c.Header("Idempotent-Replayed", "true")
			c.Data(replayed.Status, headerContentType(replayed.Headers), replayed.Body)
			c.Abort()
			return
		}
		if store.Conflicts(scope, key, requestSum) {
			httpapi.AbortError(c, http.StatusConflict, "idempotency_conflict", "idempotency key was already used with a different payload")
			return
		}

		capture := &responseCaptureWriter{ResponseWriter: c.Writer}
		c.Writer = capture
		c.Next()

		if c.Writer.Status() >= 500 {
			return
		}

		headers := map[string]string{
			"Content-Type": c.Writer.Header().Get("Content-Type"),
		}
		store.Save(scope, key, requestSum, idempotencyRecord{
			Scope:      scope,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			RequestSum: requestSum,
			Status:     c.Writer.Status(),
			Body:       append([]byte(nil), capture.body.Bytes()...),
			Headers:    headers,
			ExpiresAt:  time.Now().UTC().Add(store.ttl),
		})
	}
}

func (s *IdempotencyStore) Lookup(scope, key, requestSum string) (idempotencyRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sweepExpiredLocked(time.Now().UTC(), s.records)

	record, ok := s.records[scope+"|"+key]
	if !ok {
		return idempotencyRecord{}, false
	}
	if record.RequestSum != requestSum {
		return idempotencyRecord{}, false
	}
	return record, true
}

func (s *IdempotencyStore) Conflicts(scope, key, requestSum string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sweepExpiredLocked(time.Now().UTC(), s.records)

	record, ok := s.records[scope+"|"+key]
	if !ok {
		return false
	}
	return record.RequestSum != requestSum
}

func (s *IdempotencyStore) Save(scope, key, requestSum string, record idempotencyRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sweepExpiredLocked(time.Now().UTC(), s.records)
	record.RequestSum = requestSum
	s.records[scope+"|"+key] = record
}

func sweepExpiredLocked(now time.Time, records map[string]idempotencyRecord) {
	for key, record := range records {
		if now.After(record.ExpiresAt) {
			delete(records, key)
		}
	}
}

func headerContentType(headers map[string]string) string {
	if value := headers["Content-Type"]; value != "" {
		return value
	}
	return "application/json; charset=utf-8"
}

func traceIDFromHeaders(c *gin.Context) string {
	traceparent := strings.TrimSpace(c.GetHeader("traceparent"))
	if traceparent != "" {
		parts := strings.Split(traceparent, "-")
		if len(parts) >= 4 && len(parts[1]) == 32 {
			return parts[1]
		}
	}

	traceID := strings.TrimSpace(c.GetHeader("X-Trace-ID"))
	if traceID != "" {
		return traceID
	}

	return ""
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf[:])
}
