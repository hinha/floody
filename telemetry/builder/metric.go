package builder

import (
	"context"
	"errors"
	"github.com/hinha/floody/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"time"
)

const (
	defaultRetryAttempts = 3
	defaultRetryDelay    = time.Second * 2
)

type CloseFunc func(ctx context.Context) error

type Meter struct {
	log            *zap.Logger
	resource       *resource.Resource
	exemplarFilter exemplar.Filter
	reader         *sdkmetric.PeriodicReader // only read property

	ReaderInterval  time.Duration
	ShutdownTimeout time.Duration
}

type MeterProviderBuilder struct {
	Meter

	options  []sdkmetric.Option
	exporter sdkmetric.Exporter
}

func NewMeterProvider(meter Meter, options ...MeterOption) *MeterProviderBuilder {
	for _, o := range options {
		o.apply(meter)
	}
	return &MeterProviderBuilder{
		Meter: meter,
	}
}

func (b *MeterProviderBuilder) AddMetricOption(options ...sdkmetric.Option) *MeterProviderBuilder {
	b.options = append(b.options, options...)
	return b
}

func (b *MeterProviderBuilder) Exporter(ctx context.Context, opts ...otlpmetrichttp.Option) *MeterProviderBuilder {
	var metricExporter *otlpmetrichttp.Exporter
	var err error

	metricExporter, err = otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		b.log.Named("meterBuilder").Fatal("failed to create metric exporter", zap.Error(err))
		return b
	}

	//for attempt := 1; attempt <= defaultRetryAttempts; attempt++ {
	//	metricExporter, err = otlpmetrichttp.New(ctx, opts...)
	//	if err == nil {
	//		break
	//	}
	//	if b.log != nil {
	//		b.log.Named("meterBuilder").Warn("retry creating metric exporter", zap.Int("attempt", attempt), zap.Error(err))
	//	}
	//	time.Sleep(defaultRetryDelay)
	//}
	//
	//if err != nil {
	//	if b.log != nil {
	//		b.log.Named("meterBuilder").Error("failed to create metric exporter", zap.Error(err))
	//	}
	//	return b
	//}

	//healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	//defer cancel()
	//
	//if err := metricExporter.ForceFlush(healthCtx); err != nil {
	//	b.log.Named("meterBuilder").Error("metric exporter not ready", zap.Error(err))
	//}

	b.exporter = metricExporter
	return b
}

func (b *MeterProviderBuilder) GrpcExporter(ctx context.Context, opts ...otlpmetricgrpc.Option) *MeterProviderBuilder {
	metricExporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		b.log.Named("meterBuilder").Fatal("failed to create metric exporter", zap.Error(err))
		return b
	}

	b.exporter = metricExporter
	return b
}

func (b *MeterProviderBuilder) isReady() bool {
	if b.exporter == nil {
		return false
	}
	return true
}

func (b *MeterProviderBuilder) Build() (*sdkmetric.MeterProvider, CloseFunc, error) {
	if !b.isReady() {
		return nil, nil, errors.New("metric exporter is nil, cannot build meter provider")
	}

	if b.ReaderInterval <= 0 {
		b.ReaderInterval = time.Second * 5
	}

	// Use pooling for reader
	reader := sdkmetric.NewPeriodicReader(
		b.exporter,
		sdkmetric.WithInterval(b.ReaderInterval),
		sdkmetric.WithTimeout(b.ReaderInterval/2),
	)
	if reader == nil {
		return nil, nil, errors.New("metric reader is not initialized")
	}
	b.reader = reader

	b.options = append(b.options, sdkmetric.WithReader(reader), sdkmetric.WithResource(b.resource))

	if b.exemplarFilter != nil {
		b.options = append(b.options, sdkmetric.WithExemplarFilter(b.exemplarFilter))
	}

	provider := sdkmetric.NewMeterProvider(b.options...)
	if provider == nil {
		return nil, nil, errors.New("meter provider failed to initialize")
	}

	otel.SetMeterProvider(provider)

	if b.ShutdownTimeout <= 0 {
		b.ShutdownTimeout = time.Second * 5
	}

	if b.log == nil {
		b.log = log.DefaultLogger
	}
	b.log.Named("meterBuilder").Debug("meter provider created")

	return provider, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, b.ShutdownTimeout)
		defer cancel()

		return provider.Shutdown(ctx)
	}, nil
}

func (b *MeterProviderBuilder) Reader() *sdkmetric.PeriodicReader {
	return b.reader
}
