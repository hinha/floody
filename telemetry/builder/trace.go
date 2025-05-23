package builder

import (
	"context"
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
	shutdownTimeout time.Duration
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
			shutdownTimeout: time.Second * 5,                   // default timeout
			log:             log.DefaultLogger.Named("tracer"), // default logger
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

// WithShutdownTimeout sets the shutdown timeout for the tracer
func (b *TracerBuilder) WithShutdownTimeout(timeout time.Duration) *TracerBuilder {
	b.config.shutdownTimeout = timeout
	return b
}

func (b *TracerBuilder) DebugExporter() *TracerBuilder {
	// Tambahkan stdout exporter
	stdoutExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		b.config.log.Error("failed to create stdout exporter", zap.Error(err))
		return b
	}
	b.config.stdoutExporter = stdoutExporter

	return b
}

// Build creates and returns the TracerProvider along with a cleanup function
func (b *TracerBuilder) Build(ctx context.Context, option ...TraceOption) (*sdktrace.TracerProvider, CloseFunc, error) {
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

	b.config.log.Debug("tracer provider created")

	// Return provider with cleanup function
	return provider, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, b.config.shutdownTimeout)
		defer cancel()
		return provider.Shutdown(ctx)
	}, nil
}
