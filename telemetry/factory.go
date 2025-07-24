package telemetry

import (
	"context"
	"fmt"
	"github.com/hinha/floody/telemetry/builder"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"os"
	"time"
)

const (
	LoggerNameTelemetry = "telemetry"
	LoggerNameExporter  = "otlp"
)

// An Option configures a Logger.
type Option interface {
	apply(*Options)
}

// optionFunc wraps a func. so it satisfies the Option interface.
type optionFunc func(*Options)

func (f optionFunc) apply(log *Options) {
	f(log)
}

func WithAttribute(attrs ...attribute.KeyValue) Option {
	return optionFunc(func(options *Options) {
		attrDefault := []attribute.KeyValue{
			semconv.ServiceNameKey.String(options.AppName),
			semconv.ServiceVersionKey.String(options.Version),
			semconv.DeploymentEnvironmentKey.String(options.Environment),
		}

		options.attributes = append(attrDefault, attrs...)
	})
}

func WithMeterReaderOption(readers ...sdkmetric.PeriodicReaderOption) Option {
	return optionFunc(func(options *Options) {
		options.readerOptions = readers
	})
}

func WithNewResource(res *resource.Resource) Option {
	return optionFunc(func(options *Options) {
		options.resource = res
	})
}

func WithTraceProviderOption(tpo ...sdktrace.TracerProviderOption) Option {
	return optionFunc(func(options *Options) {
		options.traceProviderOptions = tpo
	})
}

type IFactory interface {
	Build(ctx context.Context, opts ...Option) ([]builder.CloseFunc, error)
	StartTransaction(ctx context.Context, name string, options ...AppOption[metric.Meter, trace.Tracer]) (context.Context, trace.Span)
	GetConfigs() *Options
	GetLogger() zerolog.Logger
	SetLogger(fn func() zerolog.LevelWriter)
}

type Options struct {
	readerOptions        []sdkmetric.PeriodicReaderOption
	processorOption      []sdktrace.BatchSpanProcessorOption
	traceProviderOptions []sdktrace.TracerProviderOption
	attributes           []attribute.KeyValue
	resource             *resource.Resource

	AppName     string
	Version     string
	Environment string

	MeterHTTP      bool
	MeterConnector *builder.MeterConnector
	MeterOption    []sdkmetric.Option

	TraceHTTP      bool
	TraceConnector *builder.TraceConnector
}

type Factory struct {
	*Options

	Logger     func() zerolog.LevelWriter
	app        *Application[metric.Meter, trace.Tracer]
	logger     zerolog.Logger
	attributes []attribute.KeyValue
}

func NewFactory(options *Options) IFactory {
	if options == nil {
		options = &Options{
			AppName:     "floody",
			Version:     "0.0.1",
			Environment: "development",
		}
	}
	return &Factory{
		Options: options,
		app:     &Application[metric.Meter, trace.Tracer]{},
		Logger: func() zerolog.LevelWriter {
			return zerolog.MultiLevelWriter(zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339,
			})
		},
	}
}

func (f *Factory) SetLogger(fn func() zerolog.LevelWriter) {
	f.Logger = fn
}

func (f *Factory) setupLogger() zerolog.Logger {
	return zerolog.New(f.Logger()).With().Timestamp().Caller().Logger()
}

func (f *Factory) GetLogger() zerolog.Logger {
	return f.logger
}

func (f *Factory) GetConfigs() *Options {
	return f.Options
}

func (f *Factory) configureLogger() zerolog.Logger {
	baseLogger := f.setupLogger()
	logger := baseLogger.With().Str("main", LoggerNameTelemetry).Logger()
	loggerWithExporter := logger.With().Str("exporter", LoggerNameExporter).Logger()

	return loggerWithExporter
}

func (f *Factory) Build(ctx context.Context, opts ...Option) ([]builder.CloseFunc, error) {
	for _, opt := range opts {
		opt.apply(f.Options)
	}

	if f.MeterConnector == nil && f.TraceConnector == nil {
		return nil, fmt.Errorf("both MeterConnector and TraceConnector cannot be nil")
	}

	f.logger = f.configureLogger()

	var closersFn []builder.CloseFunc

	if f.MeterConnector != nil {
		var connectorMetric builder.MeterOption
		if f.MeterHTTP {
			connectorMetric = builder.WithMeterHttp(f.MeterConnector, f.readerOptions...)
		} else {
			connectorMetric = builder.WithMeterGrpc(f.MeterConnector, f.readerOptions...)
		}
		meterProvider := builder.NewMeterProvider()
		meterLogger := f.logger.With().Str("components", "meter").Logger()

		obsMeter, meterCloser, err := meterProvider.
			AddMetricOption(f.MeterOption...).
			WithAttributes(f.Options.attributes...).
			WithLogger(meterLogger).
			WithServiceName(f.AppName).
			Build(ctx, connectorMetric)
		if err != nil {
			return nil, err
		}
		closersFn = append(closersFn, meterCloser)

		SetMetricTelemetry(obsMeter.Meter(f.AppName,
			metric.WithInstrumentationVersion(f.Version), metric.WithInstrumentationAttributes(f.Options.attributes...)))
		f.app.setMeter(GetMetricTelemetry())
		if err := f.app.observeMeter(obsMeter); err != nil {
			f.logger.Error().Err(err).Msg("failed to observe meter")
		}
	}

	var connectorTracer builder.TraceOption
	if f.TraceHTTP {
		connectorTracer = builder.WithTraceHttp(f.TraceConnector, f.processorOption...)
	} else {
		connectorTracer = builder.WithTraceGrpc(f.TraceConnector, f.processorOption...)
	}
	traceProvider := builder.NewTracerBuilder()
	// Create a logger with trace component
	traceLogger := f.logger.With().Str("components", "trace").Logger()

	obsTrace, traceCloser, err := traceProvider.
		WithLogger(traceLogger).
		WithResource(f.Options.resource).
		WithAttributes(f.Options.attributes...).
		WithServiceName(f.AppName).
		Build(ctx, connectorTracer)
	if err != nil {
		return nil, err
	}
	closersFn = append(closersFn, traceCloser)

	SetTraceTelemetry(obsTrace.Tracer(f.AppName,
		trace.WithInstrumentationVersion(f.Version),
		trace.WithInstrumentationAttributes(f.Options.attributes...)))
	f.app.setTracer(GetTraceTelemetry())

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return closersFn, nil
}

func (f *Factory) StartTransaction(
	ctx context.Context,
	name string,
	options ...AppOption[metric.Meter, trace.Tracer],
) (context.Context, trace.Span) {

	// Apply additional options that modify the application
	for _, opt := range options {
		opt.apply(f.app)
	}

	return f.app.tracer.Start(ctx, name, f.app.spanOptions...)
}
