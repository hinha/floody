package telemetry

import (
	"context"
	"fmt"
	"github.com/hinha/floody/log"
	"github.com/hinha/floody/telemetry/builder"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

type IFactory interface {
	Build(ctx context.Context, opts ...Option) ([]builder.CloseFunc, error)
	StartTransaction(ctx context.Context, name string, options ...AppOption[metric.Meter, trace.Tracer]) (context.Context, trace.Span)
	GetConfigs() *Options
	GetLogger() *zap.Logger
}

type Options struct {
	readerOptions   []sdkmetric.PeriodicReaderOption
	processorOption []sdktrace.BatchSpanProcessorOption
	attributes      []attribute.KeyValue

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
	app        *Application[metric.Meter, trace.Tracer]
	logger     *zap.Logger
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
	}
}

func (f *Factory) setupLogger() log.LoggerI {
	return log.NewLogger(log.NewDevelopmentConfig())
}

func (f *Factory) GetLogger() *zap.Logger {
	return f.logger
}

func (f *Factory) GetConfigs() *Options {
	return f.Options
}

func (f *Factory) configureLogger() *zap.Logger {
	baseLogger := f.setupLogger()

	loggerOpts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	return baseLogger.
		WithOptions(loggerOpts...).
		Named(LoggerNameTelemetry).
		With(zap.String("exporter", LoggerNameExporter))
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
		obsMeter, meterCloser, err := meterProvider.
			AddMetricOption(f.MeterOption...).
			WithAttributes(f.attributes...).
			WithLogger(f.logger.Named("meter\t")).
			WithServiceName(f.AppName).
			Build(ctx, connectorMetric)
		if err != nil {
			return nil, err
		}
		closersFn = append(closersFn, meterCloser)

		SetMetricTelemetry(obsMeter.Meter(f.AppName,
			metric.WithInstrumentationVersion(f.Version), metric.WithInstrumentationAttributes(f.attributes...)))
		f.app.setMeter(GetMetricTelemetry())
		if err := f.app.observeMeter(obsMeter); err != nil {
			f.logger.Error("failed to observe meter", zap.Error(err))
		}
	}

	var connectorTracer builder.TraceOption
	if f.TraceHTTP {
		connectorTracer = builder.WithTraceHttp(f.TraceConnector, f.processorOption...)
	} else {
		connectorTracer = builder.WithTraceGrpc(f.TraceConnector, f.processorOption...)
	}
	traceProvider := builder.NewTracerBuilder()
	obsTrace, traceCloser, err := traceProvider.
		WithLogger(f.logger.Named("trace\t")).
		WithAttributes(f.attributes...).
		WithServiceName(f.AppName).
		Build(ctx, connectorTracer)
	if err != nil {
		return nil, err
	}
	closersFn = append(closersFn, traceCloser)

	SetTraceTelemetry(obsTrace.Tracer(f.AppName,
		trace.WithInstrumentationVersion(f.Version),
		trace.WithInstrumentationAttributes(f.attributes...)))
	f.app.setTracer(GetTraceTelemetry())

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	f.logger.Info("telemetry factory build")
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
