package observability

import (
	"context"

	"github.com/AsaqeLee/taskflow/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func ConfigureTracing(ctx context.Context, cfg config.Config) (trace.Tracer, func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !cfg.TracingEnabled || cfg.TracingEndpoint == "" {
		return otel.Tracer(cfg.TracingServiceName), func(context.Context) error { return nil }, nil
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.TracingEndpoint),
	}
	if cfg.TracingInsecure {
		options = append(options, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", cfg.TracingServiceName),
			attribute.String("service.version", cfg.AppVersion),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	return provider.Tracer(cfg.TracingServiceName), provider.Shutdown, nil
}
