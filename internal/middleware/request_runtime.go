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
	"go.opentelemetry.io/otel/trace"
)

const (
	idempotencyStatePending   = "pending"
	idempotencyStateCompleted = "completed"
)

type responseCaptureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *responseCaptureWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if n > 0 {
		_, _ = w.body.Write(data[:n])
	}
	return n, err
}

type deadlineAwareWriter struct {
	gin.ResponseWriter
	ctx context.Context
}

func (w *deadlineAwareWriter) WriteHeader(code int) {
	if w.ctx != nil && w.ctx.Err() != nil {
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *deadlineAwareWriter) WriteHeaderNow() {
	if w.ctx != nil && w.ctx.Err() != nil {
		return
	}
	w.ResponseWriter.WriteHeaderNow()
}

func (w *deadlineAwareWriter) Write(data []byte) (int, error) {
	if w.ctx != nil && w.ctx.Err() != nil {
		return 0, w.ctx.Err()
	}
	return w.ResponseWriter.Write(data)
}

func (w *deadlineAwareWriter) WriteString(value string) (int, error) {
	if w.ctx != nil && w.ctx.Err() != nil {
		return 0, w.ctx.Err()
	}
	return w.ResponseWriter.WriteString(value)
}

func (w *deadlineAwareWriter) Flush() {
	if w.ctx != nil && w.ctx.Err() != nil {
		return
	}
	w.ResponseWriter.Flush()
}

type IdempotencyRecord struct {
	Scope      string
	Method     string
	Path       string
	RequestSum string
	Status     int
	Body       []byte
	Headers    map[string]string
	ExpiresAt  time.Time
	State      string
}

type IdempotencyDecision int

const (
	IdempotencyDecisionAccept IdempotencyDecision = iota
	IdempotencyDecisionReplay
	IdempotencyDecisionConflict
	IdempotencyDecisionInProgress
)

type IdempotencyStore interface {
	Enabled() bool
	TTL() time.Duration
	Reserve(ctx context.Context, scope, key, requestSum string, now time.Time) (IdempotencyDecision, IdempotencyRecord, error)
	Complete(ctx context.Context, scope, key, requestSum string, record IdempotencyRecord) error
	Release(ctx context.Context, scope, key, requestSum string) error
}

type MemoryIdempotencyStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	records map[string]IdempotencyRecord
}

func NewMemoryIdempotencyStore(ttl time.Duration) *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{
		ttl:     ttl,
		records: make(map[string]IdempotencyRecord),
	}
}

func NewIdempotencyStore(ttl time.Duration) IdempotencyStore {
	return NewMemoryIdempotencyStore(ttl)
}

func (s *MemoryIdempotencyStore) Enabled() bool {
	return s != nil && s.ttl > 0
}

func (s *MemoryIdempotencyStore) TTL() time.Duration {
	if s == nil {
		return 0
	}
	return s.ttl
}

func (s *MemoryIdempotencyStore) Reserve(ctx context.Context, scope, key, requestSum string, now time.Time) (IdempotencyDecision, IdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sweepExpiredIdempotencyLocked(now, s.records)

	record, ok := s.records[idempotencyStorageKey(scope, key)]
	if !ok {
		s.records[idempotencyStorageKey(scope, key)] = IdempotencyRecord{
			Scope:      scope,
			RequestSum: requestSum,
			ExpiresAt:  now.Add(s.ttl),
			State:      idempotencyStatePending,
		}
		return IdempotencyDecisionAccept, IdempotencyRecord{}, nil
	}
	if record.RequestSum != requestSum {
		return IdempotencyDecisionConflict, IdempotencyRecord{}, nil
	}
	if record.State == idempotencyStateCompleted {
		return IdempotencyDecisionReplay, record, nil
	}
	return IdempotencyDecisionInProgress, IdempotencyRecord{}, nil
}

func (s *MemoryIdempotencyStore) Complete(ctx context.Context, scope, key, requestSum string, record IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.records[idempotencyStorageKey(scope, key)]
	if !ok || current.RequestSum != requestSum {
		return nil
	}

	record.RequestSum = requestSum
	record.State = idempotencyStateCompleted
	s.records[idempotencyStorageKey(scope, key)] = record
	return nil
}

func (s *MemoryIdempotencyStore) Release(ctx context.Context, scope, key, requestSum string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.records[idempotencyStorageKey(scope, key)]
	if ok && current.RequestSum == requestSum && current.State == idempotencyStatePending {
		delete(s.records, idempotencyStorageKey(scope, key))
	}
	return nil
}

func sweepExpiredIdempotencyLocked(now time.Time, records map[string]IdempotencyRecord) {
	for key, record := range records {
		if now.After(record.ExpiresAt) {
			delete(records, key)
		}
	}
}

type RateLimiter interface {
	Enabled() bool
	Allow(ctx context.Context, clientID string, now time.Time) (bool, error)
}

type rateLimitBucket struct {
	Count     int
	WindowEnd time.Time
}

type MemoryRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]rateLimitBucket
}

func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]rateLimitBucket),
	}
}

func NewRateLimiter(limit int, window time.Duration) RateLimiter {
	return NewMemoryRateLimiter(limit, window)
}

func (l *MemoryRateLimiter) Enabled() bool {
	return l != nil && l.limit > 0 && l.window > 0
}

func (l *MemoryRateLimiter) Allow(ctx context.Context, clientID string, now time.Time) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.clients[clientID]
	if !ok || now.After(bucket.WindowEnd) {
		l.clients[clientID] = rateLimitBucket{
			Count:     1,
			WindowEnd: now.Add(l.window),
		}
		return true, nil
	}

	if bucket.Count >= l.limit {
		return false, nil
	}

	bucket.Count++
	l.clients[clientID] = bucket
	return true, nil
}

func RequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newID()
		}

		traceID := ""
		spanID := ""
		if spanContext := trace.SpanFromContext(c.Request.Context()).SpanContext(); spanContext.IsValid() {
			traceID = spanContext.TraceID().String()
			spanID = spanContext.SpanID().String()
		}
		if traceID == "" {
			traceID = traceIDFromHeaders(c)
		}
		if traceID == "" {
			traceID = requestID
		}

		meta := requestmeta.Meta{
			RequestID:      requestID,
			TraceID:        traceID,
			SpanID:         spanID,
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

		writer := c.Writer
		c.Writer = &deadlineAwareWriter{ResponseWriter: writer, ctx: ctx}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		timedOut := ctx.Err() == context.DeadlineExceeded && !writer.Written()
		c.Writer = writer

		if timedOut {
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
			slog.String("span_id", meta.SpanID),
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

func RateLimit(limiter RateLimiter, metrics *observability.Metrics, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil || !limiter.Enabled() {
			c.Next()
			return
		}

		allowed, err := limiter.Allow(c.Request.Context(), c.ClientIP(), time.Now().UTC())
		if err != nil {
			if metrics != nil {
				metrics.ObserveRateLimitDecision(scope, "error")
			}
			httpapi.AbortError(c, http.StatusInternalServerError, "rate_limit_failed", "failed to evaluate rate limit")
			return
		}
		if !allowed {
			if metrics != nil {
				metrics.ObserveRateLimitDecision(scope, "rejected")
			}
			httpapi.AbortError(c, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
			return
		}
		if metrics != nil {
			metrics.ObserveRateLimitDecision(scope, "allowed")
		}

		c.Next()
	}
}

func Idempotency(store IdempotencyStore, metrics *observability.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil || !store.Enabled() {
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

		scope := idempotencyScope(c)
		sum := sha256.Sum256(payload)
		requestSum := hex.EncodeToString(sum[:])

		decision, record, err := store.Reserve(c.Request.Context(), scope, key, requestSum, time.Now().UTC())
		if err != nil {
			if metrics != nil {
				metrics.ObserveIdempotencyDecision("error")
			}
			httpapi.AbortError(c, http.StatusInternalServerError, "idempotency_failed", "failed to reserve idempotency key")
			return
		}

		switch decision {
		case IdempotencyDecisionReplay:
			if metrics != nil {
				metrics.ObserveIdempotencyDecision("replayed")
			}
			for headerKey, headerValue := range record.Headers {
				c.Header(headerKey, headerValue)
			}
			c.Header("Idempotent-Replayed", "true")
			c.Data(record.Status, headerContentType(record.Headers), record.Body)
			c.Abort()
			return
		case IdempotencyDecisionConflict:
			if metrics != nil {
				metrics.ObserveIdempotencyDecision("conflict")
			}
			httpapi.AbortError(c, http.StatusConflict, "idempotency_conflict", "idempotency key was already used with a different payload")
			return
		case IdempotencyDecisionInProgress:
			if metrics != nil {
				metrics.ObserveIdempotencyDecision("in_progress")
			}
			httpapi.AbortError(c, http.StatusConflict, "idempotency_in_progress", "idempotency key is already being processed")
			return
		}
		if metrics != nil {
			metrics.ObserveIdempotencyDecision("accepted")
		}

		capture := &responseCaptureWriter{ResponseWriter: c.Writer}
		c.Writer = capture
		c.Next()
		if c.Request.Context().Err() != nil {
			if metrics != nil {
				metrics.ObserveIdempotencyDecision("released")
			}
			_ = store.Release(c.Request.Context(), scope, key, requestSum)
			return
		}

		if c.Writer.Status() >= 500 {
			if metrics != nil {
				metrics.ObserveIdempotencyDecision("released")
			}
			_ = store.Release(c.Request.Context(), scope, key, requestSum)
			return
		}

		headers := map[string]string{
			"Content-Type": c.Writer.Header().Get("Content-Type"),
		}
		_ = store.Complete(c.Request.Context(), scope, key, requestSum, IdempotencyRecord{
			Scope:      scope,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			RequestSum: requestSum,
			Status:     c.Writer.Status(),
			Body:       append([]byte(nil), capture.body.Bytes()...),
			Headers:    headers,
			ExpiresAt:  time.Now().UTC().Add(store.TTL()),
		})
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

func idempotencyStorageKey(scope, key string) string {
	return scope + "|" + key
}

func idempotencyScope(c *gin.Context) string {
	meta := requestmeta.FromContext(c.Request.Context())
	identity := "ip:" + c.ClientIP()
	if strings.TrimSpace(meta.UserID) != "" {
		identity = "user:" + meta.UserID
	}
	return c.Request.Method + "|" + c.Request.URL.Path + "|" + identity
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf[:])
}
