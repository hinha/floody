package builder

import (
	"context"
	"errors"
	"github.com/hinha/floody/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"time"
)

// TracerConfig holds the configuration for the tracer
type TracerConfig struct {
	log             *zap.Logger
	name            string
	tag             string
	attributes      []attribute.KeyValue
	stdoutExporter  sdktrace.SpanExporter
	traceHttpOption []otlptracehttp.Option
}

// TracerBuilder implements the builder pattern for creating a tracer
type TracerBuilder struct {
	config TracerConfig
	client *TraceConnector
}

// NewTracerBuilder creates a new TracerBuilder instance
func NewTracerBuilder() *TracerBuilder {
	return &TracerBuilder{
		config: TracerConfig{
			log: log.DefaultLogger.Named("tracer"), // default logger
		},
	}
}

// WithLogger sets the logger for the tracer
func (b *TracerBuilder) WithLogger(logger *zap.Logger) *TracerBuilder {
	b.config.log = logger
	return b
}

// WithName sets the name for the tracer
func (b *TracerBuilder) WithName(name string) *TracerBuilder {
	b.config.name = name
	return b
}

// WithTag sets the tag for the tracer
func (b *TracerBuilder) WithTag(tag string) *TracerBuilder {
	b.config.tag = tag
	return b
}

// WithAttributes sets the attributes for the tracer
func (b *TracerBuilder) WithAttributes(attrs ...attribute.KeyValue) *TracerBuilder {
	b.config.attributes = append(b.config.attributes, attrs...)
	return b
}

// Build creates and returns the TracerProvider along with a cleanup function
func (b *TracerBuilder) Build(ctx context.Context, option ...TraceOption) (*sdktrace.TracerProvider, CloseFunc, error) {
	if option == nil || len(option) == 0 {
		return nil, nil, errors.New("config must be set")
	}

	for _, o := range option {
		o.apply(b)
	}

	// Create resource with attributes
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithHostID(),
		resource.WithTelemetrySDK(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithAttributes(b.config.attributes...),
	)
	if err != nil {
		return nil, nil, err
	}

	err = b.client.spanExporter(ctx)
	if err != nil {
		b.config.log.Error("failed to create exporter", zap.Error(err))
		return nil, nil, err
	}
	err = b.client.debug()
	if err != nil {
		b.config.log.Error("failed to create debug exporter", zap.Error(err))
		return nil, nil, err
	}

	sampler := sdktrace.WithSampler(sdktrace.AlwaysSample())
	rsc := sdktrace.WithResource(res)

	provider := b.client.mergeSpan(sampler, rsc)

	// Set global TracerProvider
	otel.SetTracerProvider(provider)

	if b.client.ShutdownTimeout < 0 {
		b.client.ShutdownTimeout = 5 * time.Second
	}

	b.config.log.Debug("tracer provider created")

	// Return provider with cleanup function
	return provider, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, b.client.ShutdownTimeout)
		defer cancel()
		return provider.Shutdown(ctx)
	}, nil
}

type DebugTrace struct {
	Timestamps  bool
	PrettyPrint bool
}

func (d *DebugTrace) apply() (opts []stdouttrace.Option) {
	if !d.Timestamps {
		opts = append(opts, stdouttrace.WithoutTimestamps())
	}
	if d.PrettyPrint {
		opts = append(opts, stdouttrace.WithPrettyPrint())
	}

	return
}
