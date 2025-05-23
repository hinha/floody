package telemetry

import (
	"context"
	"github.com/hinha/floody/log"
	"github.com/hinha/floody/telemetry/builder"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"time"
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

// WithView adds a view to the meter
//
//	WithView(metric.Instrument{
//		Name: "http.server.duration", // "*" for all
//		Kind: metric.InstrumentKindHistogram,
//	}, metric.Stream{
//		Aggregation: metric.AggregationExplicitBucketHistogram{
//			Boundaries: boundaries,
//		},
//	})

func WithResource(res *resource.Resource) Option {
	return optionFunc(func(o *Options) {
		o.optionMeter = append(o.optionMeter, builder.WithResource(res))
	})
}

// WithExemplarFilter sets the filter for the exemplar
//
//	WithExemplarFilter(exemplar.AlwaysOffFilter)
//
// exemplar.AlwaysOffFilter is the default.
// exemplar.TraceBasedFilter
// exemplar.AlwaysOnFilter
func WithExemplarFilter(filter exemplar.Filter) Option {
	return optionFunc(func(o *Options) {
		o.optionMeter = append(o.optionMeter, builder.WithExamplarFilter(filter))
	})
}

// RetryConfig refer [otlpmetricgrpc.RetryConfig] or [otlptracegrpc.RetryConfig]
type RetryConfig struct {
	Enabled         bool
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
}

func (c *RetryConfig) MetricConfig() otlpmetrichttp.RetryConfig {
	return otlpmetrichttp.RetryConfig{
		Enabled:         c.Enabled,
		InitialInterval: c.InitialInterval,
		MaxInterval:     c.MaxInterval,
		MaxElapsedTime:  c.MaxElapsedTime,
	}
}

func (c *RetryConfig) TraceConfig() otlptracegrpc.RetryConfig {
	return otlptracegrpc.RetryConfig{
		Enabled:         c.Enabled,
		InitialInterval: c.InitialInterval,
		MaxInterval:     c.MaxInterval,
		MaxElapsedTime:  c.MaxElapsedTime,
	}
}

type IFactory interface {
	Build(ctx context.Context, opts ...Option) ([]builder.CloseFunc, error)
	MeterOptions(opt ...metric.MeterOption) *Factory
	TracerOptions(opt ...trace.TracerOption) *Factory
	StartTransaction(ctx context.Context, name string, options ...AppOption[metric.Meter, trace.Tracer]) (context.Context, trace.Span)
	GetConfigs() *Options
	GetLogger() *zap.Logger
}

type Exporter struct {
	Endpoint string
}

type Endpoint struct {
	Endpoint string
	URLPath  string
	WithHTTP bool
	Block    bool
	Debug    bool
	Header   map[string]string
}

type Options struct {
	// EndpointTrace sets the target endpoint the Exporter trace will connect host:port address.
	EndpointTrace Endpoint
	// EndpointMeter sets the target endpoint the Exporter metric will connect host:port address.
	EndpointMeter Endpoint
	// Insecure disables client transport security for the Exporter's gRPC
	// connection, just like grpc.WithInsecure()
	// (https://pkg.go.dev/google.golang.org/grpc#WithInsecure) does.
	Insecure bool
	// TLSCredentials sets the gRPC connection to use Credentials.
	Credentials credentials.TransportCredentials
	// DialOptions sets explicit grpc.DialOptions to use when establishing a gRPC connection.
	// The options here are appended to the internal grpc.DialOptions
	// used, so they will take precedence over any other internal grpc.DialOptions
	// they might conflict with.
	// The [grpc.WithBlock], [grpc.WithTimeout], and [grpc.WithReturnConnectionError]
	// grpc.DialOptions are ignored.
	GrpcDialOptions []grpc.DialOption

	// Timeout sets the max amount of time an Exporter will attempt an export
	Timeout time.Duration
	// HTTPRetry sets the retry policy for transient retryable errors that are
	// returned by the target endpoint.

	// If the target endpoint responds with not only a retryable error, but
	// explicitly returns a backoff time in the response, that time will take
	// precedence over these settings.
	Retry RetryConfig

	// AggregationSelector sets the AggregationSelector the client will use to
	// determine the aggregation to use it for an instrument based on its kind. If
	// this option is not used, the reader will use the DefaultAggregationSelector
	// from the go.opentelemetry.io/otel/sdk/metric package, or the aggregation
	// explicitly passed for a view matching an instrument.
	AggregationSelector sdkmetric.AggregationSelector

	optionMeter []builder.MeterOption
	//optionTracer []builder.TracerOption

	Meter builder.Meter
	//Tracer builder.Tracer

	AppName     string
	Version     string
	Environment string
	// SchemaURL represents the URL of the OpenTelemetry schema being used
	SchemaURL string
	// Instrumentation contains key-value pairs for additional instrumentation configuration
	Instrumentation []attribute.KeyValue
}

type Factory struct {
	*Options
	app        *Application[metric.Meter, trace.Tracer]
	logger     *zap.Logger
	attributes []attribute.KeyValue

	exporterMetricHttp []otlpmetrichttp.Option
	exporterMetricGrpc []otlpmetricgrpc.Option
}

func NewFactory(options *Options) IFactory {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(options.AppName),
		semconv.ServiceVersionKey.String(options.Version),
		attribute.String("service.environment", options.Environment),
	}
	return &Factory{
		Options:    options,
		attributes: attrs,
		app:        &Application[metric.Meter, trace.Tracer]{},
	}
}

func (f *Factory) connectorMetric(ctx context.Context, provider *builder.MeterProviderBuilder) *builder.MeterProviderBuilder {
	if f.EndpointMeter.Block {
		f.exporterMetricGrpc = append(f.exporterMetricGrpc, otlpmetricgrpc.WithGRPCConn())
	}

	if f.EndpointMeter.WithHTTP {
		provider = provider.Exporter(ctx)
	} else {
		provider = provider.GrpcExporter(ctx)
	}
	return provider
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

func (f *Factory) Build(ctx context.Context, opts ...Option) ([]builder.CloseFunc, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	for _, opt := range opts {
		opt.apply(f.Options)
	}
	log := f.setupLogger().WithOptions(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).Named("telemetry")
	f.logger = log.With(zap.String("exporter", "otlp"))

	attr := resource.WithAttributes(f.attributes...)

	res, err := resource.New(ctx, attr, resource.WithFromEnv(), resource.WithProcess(), resource.WithOS())
	if err != nil {
		return nil, err
	}
	f.optionMeter = append(f.optionMeter, builder.WithMeterLogger(log))

	meterProvider, meterCloser, err := builder.NewMeterProvider(f.Meter, f.optionMeter...).
		AddMetricOption(sdkmetric.WithResource(res)).
		Exporter(ctx,
			otlpmetrichttp.WithEndpoint(f.EndpointMeter.Endpoint),
			//otlpmetrichttp.WithInsecure(), // disable when use https
		).Build()
	if err != nil {
		panic(err)
	}

	f.app.meterOptions = append(f.app.meterOptions, metric.WithInstrumentationVersion(f.Version),
		metric.WithSchemaURL(f.SchemaURL),
		metric.WithInstrumentationAttributes(f.Instrumentation...))
	f.app.meter = meterProvider.Meter(f.AppName, f.app.meterOptions...)
	if err := f.app.observeMeter(meterProvider); err != nil {
		f.logger.Error("failed to observe meter", zap.Error(err))
	}

	//traceBuild := builder.NewTracerBuilder()
	//if f.EndpointTrace.WithHTTP {
	//	traceBuild = traceBuild.
	//		WithAttributes(f.attributes...).
	//		HTTPExporter(ctx,
	//			otlptracehttp.WithEndpoint(f.EndpointTrace.Endpoint),
	//			otlptracehttp.WithInsecure(), // disable when use https
	//			//otlptracehttp.WithURLPath(f.EndpointTrace.URLPath),
	//			//otlptracehttp.WithHeaders(f.EndpointTrace.Header)
	//		)
	//} else {
	//	traceBuild = traceBuild.
	//		WithAttributes(f.attributes...).
	//		GrpcExporter(ctx,
	//			otlptracegrpc.WithEndpoint(f.EndpointTrace.Endpoint),
	//			//otlptracegrpc.WithInsecure(), // disable when use https
	//		)
	//	if f.EndpointTrace.Debug {
	//		traceBuild = traceBuild.DebugExporter()
	//	}
	//}
	//traceOptions, traceCloser, err := traceBuild.Build(ctx)
	//if err != nil {
	//	panic(err)
	//}
	//

	//f.app.tracer = traceOptions.Tracer(f.AppName, f.app.traceOptions...)

	return []builder.CloseFunc{meterCloser, nil}, nil

	//expTraceOpts := []otlptracegrpc.Option{
	//	otlptracegrpc.WithEndpoint(f.EndpointTrace),
	//	otlptracegrpc.WithHeaders(f.Headers),
	//	otlptracegrpc.WithTimeout(f.Timeout),
	//	otlptracegrpc.WithRetry(f.HTTPRetry.TraceConfig()),
	//}
	//
	//if f.Options != nil {
	//	if f.Insecure {
	//		expTraceOpts = append(expTraceOpts, otlptracegrpc.WithInsecure())
	//	}
	//}
	//
	//log := f.setupLogger().WithOptions(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).Named("telemetry")
	//_, closerTracer, err := builder.NewTracerProvider(f.Tracer, builder.WithTracerLogger(log)).
	//	exporter(ctx, expTraceOpts...).
	//	Build(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//
	//expMeterOpts := []otlpmetrichttp.Option{
	//	otlpmetrichttp.WithEndpoint(f.EndpointMeter),
	//	otlpmetrichttp.WithHeaders(f.Headers),
	//	otlpmetrichttp.WithTimeout(f.Timeout),
	//	otlpmetrichttp.WithRetry(f.HTTPRetry.MetricConfig()),
	//	otlpmetrichttp.WithAggregationSelector(f.AggregationSelector),
	//}
	//
	//if f.Options != nil {
	//	if f.Insecure {
	//		expMeterOpts = append(expMeterOpts, otlpmetrichttp.WithInsecure())
	//	}
	//}
	//
	//f.optionMeter = append(f.optionMeter, builder.WithMeterLogger(log))
	//_, closeMeter, err := builder.NewMeterProvider(f.Meter, f.optionMeter...).
	//	exporter(ctx, expMeterOpts...).
	//	Build()
	//if err != nil {
	//	return nil, err
	//}
	//
	//otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	//f.logger = log.With(zap.String("exporter", "otlp"))
	//
	//return []builder.CloseFunc{closeMeter, closerTracer}, nil
}

func (f *Factory) MeterOptions(opt ...metric.MeterOption) *Factory {
	f.app.meterOptions = append(f.app.meterOptions, opt...)
	return f
}

func (f *Factory) TracerOptions(opt ...trace.TracerOption) *Factory {
	f.app.traceOptions = append(f.app.traceOptions, opt...)
	return f
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

	return f.app.tracer.Start(ctx, name)
}
