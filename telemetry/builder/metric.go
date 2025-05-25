package builder

import (
	"context"
	"errors"
	"github.com/hinha/floody/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"io"
	"time"
)

// CloseFunc is a function that closes the provider
type CloseFunc func(ctx context.Context) error

// MeterConfig holds the configuration for the meter
type MeterConfig struct {
	log         *zap.Logger
	resource    *resource.Resource
	attributes  []attribute.KeyValue
	serviceName string
}

// MeterBuilder implements the builder pattern for creating a meter provider
type MeterBuilder struct {
	config  MeterConfig
	client  *MeterConnector
	options []sdkmetric.Option
}

// NewMeterProvider creates a new MeterBuilder instance
func NewMeterProvider() *MeterBuilder {
	return &MeterBuilder{
		config: MeterConfig{
			log: log.DefaultLogger.Named("meter"), // default logger
		},
	}
}

// WithLogger sets the logger for the meter
func (b *MeterBuilder) WithLogger(logger *zap.Logger) *MeterBuilder {
	b.config.log = logger
	return b
}

// WithAttributes sets the attributes for the meter
func (b *MeterBuilder) WithAttributes(attrs ...attribute.KeyValue) *MeterBuilder {
	b.config.attributes = append(b.config.attributes, attrs...)
	return b
}

// WithServiceName sets the service name for the meter
func (b *MeterBuilder) WithServiceName(serviceName string) *MeterBuilder {
	b.config.serviceName = serviceName
	return b
}

// AddMetricOption adds SDK metric options to the meter provider.
// These options will be applied when building the meter provider.
// Examples include sdkmetric.WithResource(), sdkmetric.WithReader(), etc.
func (b *MeterBuilder) AddMetricOption(options ...sdkmetric.Option) *MeterBuilder {
	b.options = append(b.options, options...)
	return b
}

// Build creates and returns the MeterProvider along with a cleanup function
func (b *MeterBuilder) Build(ctx context.Context, option ...MeterOption) (*sdkmetric.MeterProvider, CloseFunc, error) {
	if option == nil || len(option) == 0 {
		return nil, nil, errors.New("config must be set")
	}
	for _, o := range option {
		o.apply(b)
	}

	// Create resource with attributes if not already set
	if b.config.resource == nil {
		// Add service name to attributes if set
		attrs := b.config.attributes
		if b.config.serviceName != "" {
			attrs = append(attrs, semconv.ServiceNameKey.String(b.config.serviceName))
		}

		res, err := resource.New(ctx,
			resource.WithFromEnv(),
			resource.WithHost(),
			resource.WithHostID(),
			resource.WithTelemetrySDK(),
			resource.WithOS(),
			resource.WithProcess(),
			resource.WithContainer(),
			resource.WithAttributes(attrs...),
		)
		if err != nil {
			return nil, nil, err
		}
		b.config.resource = res
	}

	err := b.client.metricExporter(ctx)
	if err != nil {
		b.config.log.Error("failed to create exporter", zap.Error(err))
		return nil, nil, err
	}

	err = b.client.debug()
	if err != nil {
		b.config.log.Error("failed to create debug exporter", zap.Error(err))
		return nil, nil, err
	}

	// Create provider options
	providerOptions := []sdkmetric.Option{
		sdkmetric.WithResource(b.config.resource),
	}

	// Apply any additional options that were added via AddMetricOption
	if len(b.options) > 0 {
		providerOptions = append(providerOptions, b.options...)
	}

	// Create provider with readers
	provider := b.client.mergeMeter(providerOptions)

	// Set global MeterProvider
	otel.SetMeterProvider(provider)

	if b.client.ShutdownTimeout < 0 {
		b.client.ShutdownTimeout = 5 * time.Second
	}

	b.config.log.Info("meter provider created")

	// Return provider with cleanup function
	return provider, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, b.client.ShutdownTimeout)
		defer cancel()
		return provider.Shutdown(ctx)
	}, nil
}

type DebugMetric struct {
	Encoder             stdoutmetric.Encoder
	Writer              io.Writer
	PrettyPrint         bool
	TemporalitySelector sdkmetric.TemporalitySelector
	AggregationSelector sdkmetric.AggregationSelector
	Timestamps          bool
}

func (d *DebugMetric) apply() (opts []stdoutmetric.Option) {
	if d.Encoder != nil {
		opts = append(opts, stdoutmetric.WithEncoder(d.Encoder))
	}
	if d.Writer != nil {
		opts = append(opts, stdoutmetric.WithWriter(d.Writer))
	}
	if d.PrettyPrint {
		opts = append(opts, stdoutmetric.WithPrettyPrint())
	}
	if d.TemporalitySelector != nil {
		opts = append(opts, stdoutmetric.WithTemporalitySelector(d.TemporalitySelector))
	}
	if d.AggregationSelector != nil {
		opts = append(opts, stdoutmetric.WithAggregationSelector(d.AggregationSelector))
	}
	if !d.Timestamps {
		opts = append(opts, stdoutmetric.WithoutTimestamps())
	}
	return
}
