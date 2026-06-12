package requestmeta

import (
	"context"
	"time"
)

type key string

const contextKey key = "requestMeta"

type Meta struct {
	RequestID      string
	TraceID        string
	SpanID         string
	IdempotencyKey string
	SourceIP       string
	UserAgent      string
	UserID         string
	StartedAt      time.Time
}

func WithContext(ctx context.Context, meta Meta) context.Context {
	return context.WithValue(ctx, contextKey, meta)
}

func FromContext(ctx context.Context) Meta {
	value := ctx.Value(contextKey)
	if value == nil {
		return Meta{}
	}

	meta, ok := value.(Meta)
	if !ok {
		return Meta{}
	}

	return meta
}

func WithUserID(ctx context.Context, userID string) context.Context {
	meta := FromContext(ctx)
	meta.UserID = userID
	return WithContext(ctx, meta)
}

func RequestID(ctx context.Context) string {
	return FromContext(ctx).RequestID
}
