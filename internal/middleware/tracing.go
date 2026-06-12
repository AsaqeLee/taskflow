package middleware

import (
	"net/http"

	"github.com/AsaqeLee/taskflow/internal/requestmeta"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tracer == nil {
			c.Next()
			return
		}

		ctx := propagation.TraceContext{}.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.Request.URL.Path, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		meta := requestmeta.FromContext(c.Request.Context())
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		span.SetName(c.Request.Method + " " + route)
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.String("http.target", c.Request.URL.Path),
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.String("request.id", meta.RequestID),
			attribute.String("trace.id", meta.TraceID),
			attribute.String("user.id", meta.UserID),
		)
		if c.Writer.Status() >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(c.Writer.Status()))
		}
	}
}
