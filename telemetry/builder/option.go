package builder

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"time"
)

type MeterOption interface {
	apply(builder *MeterBuilder)
}

type optionMeterFunc func(builder *MeterBuilder)

func (o optionMeterFunc) apply(b *MeterBuilder) {
	o(b)
}

// MeterConnector holds the configuration for connecting to a metric backend
type MeterConnector struct {
	Endpoint        string
	EndpointURL     string
	URLPath         string
	Insecure        bool
	Headers         map[string]string
	Timeout         time.Duration
	Debug           bool
	DebugOption     DebugMetric
	ShutdownTimeout time.Duration

	// GRPC exporter options
	ReconnectionPeriod time.Duration
	Compressor         string
	TLSCredentials     credentials.TransportCredentials
	ServiceConfig      string
	DialOption         []grpc.DialOption
	GRPCConn           *grpc.ClientConn
	GRPCRetry          otlpmetricgrpc.RetryConfig

	// HTTP exporter options
	HTTPCompression     otlpmetrichttp.Compression
	HTTPTLSClientConfig *tls.Config
	HTTPRetry           otlpmetrichttp.RetryConfig
	HTTPProxy           otlpmetrichttp.HTTPTransportProxyFunc

	metricHttpOption []otlpmetrichttp.Option
	setHttp          bool
	metricGrpcOption []otlpmetricgrpc.Option
	setGrpc          bool
	readerOptions    []sdkmetric.PeriodicReaderOption
	readerExporter   sdkmetric.Reader
	readerDebug      sdkmetric.Reader
}

// debug creates a debug exporter if Debug is true
func (m *MeterConnector) debug() error {
	if !m.Debug {
		return nil
	}

	apply := m.DebugOption.apply()
	exporter, err := stdoutmetric.New(apply...)
	if err != nil {
		return fmt.Errorf("creating stdout exporter: %w", err)
	}

	m.readerDebug = sdkmetric.NewPeriodicReader(exporter, m.readerOptions...)
	return nil
}

// metricExporter creates the metric exporter based on the configuration
func (m *MeterConnector) metricExporter(ctx context.Context) error {
	if m == nil {
		return errors.New("meter connector is not set")
	} else if m.setHttp && m.setGrpc {
		return errors.New("cannot set both http and grpc exporter")
	} else if m.setHttp {
		exporter, err := otlpmetrichttp.New(ctx, m.metricHttpOption...)
		if err != nil {
			return err
		}

		m.readerExporter = sdkmetric.NewPeriodicReader(exporter, m.readerOptions...)
		return nil
	} else if m.setGrpc {
		exporter, err := otlpmetricgrpc.New(ctx, m.metricGrpcOption...)
		if err != nil {
			return err
		}

		m.readerExporter = sdkmetric.NewPeriodicReader(exporter, m.readerOptions...)
		return nil
	} else {
		return errors.New("exporter is not set")
	}
}

// mergeMeter creates a meter provider with the given options and readers
func (m *MeterConnector) mergeMeter(options []sdkmetric.Option) *sdkmetric.MeterProvider {
	if m.readerExporter != nil {
		options = append(options, sdkmetric.WithReader(m.readerExporter))
	}

	if m.Debug && m.readerDebug != nil {
		options = append(options, sdkmetric.WithReader(m.readerDebug))
	}

	return sdkmetric.NewMeterProvider(options...)
}

// uriConnector sets common options for both HTTP and gRPC exporters
func uriMeterConnector(c *MeterConnector) {
	if c.Endpoint != "" {
		c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithEndpoint(c.Endpoint))
		c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithEndpoint(c.Endpoint))
	}
	if c.EndpointURL != "" {
		c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithEndpointURL(c.EndpointURL))
	}

	if c.Insecure {
		c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithInsecure())
		c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithInsecure())
	}
	if c.Headers != nil {
		c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithHeaders(c.Headers))
		c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithHeaders(c.Headers))
	}
	if c.Timeout != 0 {
		c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithTimeout(c.Timeout))
		c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithTimeout(c.Timeout))
	}
}

// WithMeterHttp configures the meter to use HTTP exporter
func WithMeterHttp(c *MeterConnector, readerOpt ...sdkmetric.PeriodicReaderOption) MeterOption {
	return optionMeterFunc(func(b *MeterBuilder) {
		if c == nil && b == nil {
			return
		}
		b.client = c
		uriMeterConnector(c)
		c.readerOptions = readerOpt

		if c.HTTPCompression != 0 {
			c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithCompression(c.HTTPCompression))
		}
		if c.URLPath != "" {
			c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithURLPath(c.URLPath))
		}
		if c.HTTPTLSClientConfig != nil {
			c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithTLSClientConfig(c.HTTPTLSClientConfig))
		}
		if c.HTTPProxy != nil {
			c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithProxy(c.HTTPProxy))
		}
		c.metricHttpOption = append(c.metricHttpOption, otlpmetrichttp.WithRetry(c.HTTPRetry))
		c.setHttp = true
	})
}

// WithMeterGrpc configures the meter to use gRPC exporter
func WithMeterGrpc(c *MeterConnector, readerOpt ...sdkmetric.PeriodicReaderOption) MeterOption {
	return optionMeterFunc(func(b *MeterBuilder) {
		if c == nil && b == nil {
			return
		}
		b.client = c
		uriMeterConnector(c)
		c.readerOptions = readerOpt

		if c.ReconnectionPeriod != 0 {
			c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithReconnectionPeriod(c.ReconnectionPeriod))
		}
		if c.Compressor != "" {
			c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithCompressor(c.Compressor))
		}
		if c.TLSCredentials != nil {
			c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithTLSCredentials(c.TLSCredentials))
		}
		if c.ServiceConfig != "" {
			c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithServiceConfig(c.ServiceConfig))
		}
		if c.DialOption != nil {
			c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithDialOption(c.DialOption...))
		}
		if c.GRPCConn != nil {
			c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithGRPCConn(c.GRPCConn))
		}
		c.metricGrpcOption = append(c.metricGrpcOption, otlpmetricgrpc.WithRetry(c.GRPCRetry))
		c.setGrpc = true
	})
}

type TraceOption interface {
	apply(builder *TracerBuilder)
}

type optionTraceFunc func(builder *TracerBuilder)

func (o optionTraceFunc) apply(b *TracerBuilder) {
	o(b)
}

type TraceConnector struct {
	Endpoint        string
	EndpointURL     string
	URLPath         string
	Insecure        bool
	Headers         map[string]string
	Timeout         time.Duration
	Debug           bool
	DebugOption     DebugTrace
	ShutdownTimeout time.Duration

	// GRPC exporter options
	ReconnectionPeriod time.Duration
	Compressor         string
	TLSCredentials     credentials.TransportCredentials
	ServiceConfig      string
	DialOption         []grpc.DialOption
	GRPCConn           *grpc.ClientConn
	GRPCRetry          otlptracegrpc.RetryConfig

	// HTTP exporter options
	HTTPCompression     otlptracehttp.Compression
	HTTPTLSClientConfig *tls.Config
	HTTPRetry           otlptracehttp.RetryConfig
	HTTPProxy           otlptracehttp.HTTPTransportProxyFunc

	traceHttpOption   []otlptracehttp.Option
	setHttp           bool
	traceGrpcOption   []otlptracegrpc.Option
	setGrpc           bool
	batchOption       []sdktrace.BatchSpanProcessorOption
	processorExporter sdktrace.SpanProcessor
	processorDebug    sdktrace.SpanProcessor
}

func WithTraceGrpc(c *TraceConnector, batchOption ...sdktrace.BatchSpanProcessorOption) TraceOption {
	return optionTraceFunc(func(builder *TracerBuilder) {
		if c == nil && builder == nil {
			return
		}
		builder.client = c
		uriConnector(c)
		c.batchOption = batchOption

		if c.ReconnectionPeriod != 0 {
			c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithReconnectionPeriod(c.ReconnectionPeriod))
		}
		if c.Compressor != "" {
			c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithCompressor(c.Compressor))
		}
		if c.TLSCredentials != nil {
			c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithTLSCredentials(c.TLSCredentials))
		}
		if c.ServiceConfig != "" {
			c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithServiceConfig(c.ServiceConfig))
		}
		if c.DialOption != nil {
			c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithDialOption(c.DialOption...))
		}
		if c.GRPCConn != nil {
			c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithGRPCConn(c.GRPCConn))
		}
		c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithRetry(c.GRPCRetry))

		// Ensure endpoint is properly set
		if c.Endpoint != "" {
			c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithEndpoint(c.Endpoint))
		}
		c.setGrpc = true
	})
}

func WithTraceHttp(c *TraceConnector, batchOption ...sdktrace.BatchSpanProcessorOption) TraceOption {
	return optionTraceFunc(func(builder *TracerBuilder) {
		if c == nil && builder == nil {
			return
		}
		builder.client = c
		uriConnector(c)
		c.batchOption = batchOption

		if c.HTTPCompression != 0 {
			c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithCompression(c.HTTPCompression))
		}
		if c.URLPath != "" {
			c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithURLPath(c.URLPath))
		}
		if c.HTTPTLSClientConfig != nil {
			c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithTLSClientConfig(c.HTTPTLSClientConfig))
		}
		if c.HTTPProxy != nil {
			c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithProxy(c.HTTPProxy))
		}
		c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithRetry(c.HTTPRetry))
		c.setHttp = true
	})
}

func uriConnector(c *TraceConnector) {
	if c.Endpoint != "" {
		c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithEndpoint(c.Endpoint))
		c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithEndpoint(c.Endpoint))
	}
	if c.EndpointURL != "" {
		c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithEndpointURL(c.EndpointURL))
		c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithEndpointURL(c.EndpointURL))
	}

	if c.Insecure {
		c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithInsecure())
		c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithInsecure())
	}
	if c.Headers != nil {
		c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithHeaders(c.Headers))
		c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithHeaders(c.Headers))
	}
	if c.Timeout != 0 {
		c.traceHttpOption = append(c.traceHttpOption, otlptracehttp.WithTimeout(c.Timeout))
		c.traceGrpcOption = append(c.traceGrpcOption, otlptracegrpc.WithTimeout(c.Timeout))
	}
}

func (m *TraceConnector) debug() error {
	if !m.Debug {
		return nil
	}

	apply := m.DebugOption.apply()
	exporter, err := stdouttrace.New(apply...)
	if err != nil {
		return fmt.Errorf("creating stdout exporter: %w", err)
	}

	m.processorDebug = sdktrace.NewBatchSpanProcessor(exporter)
	return nil
}

func (m *TraceConnector) spanExporter(ctx context.Context) error {
	if m == nil {
		return errors.New("meter connector is not set")
	} else if m.setHttp && m.setGrpc {
		return errors.New("cannot set both http and grpc exporter")
	} else if m.setHttp {
		exporter, err := otlptracehttp.New(ctx, m.traceHttpOption...)
		if err != nil {
			return err
		}

		m.processorExporter = sdktrace.NewBatchSpanProcessor(exporter, m.batchOption...)
		return nil
	} else if m.setGrpc {
		exporter, err := otlptracegrpc.New(ctx, m.traceGrpcOption...)
		if err != nil {
			return err
		}

		m.processorExporter = sdktrace.NewBatchSpanProcessor(exporter, m.batchOption...)
		return nil
	} else {
		return errors.New("exporter is not set")
	}
}

func (m *TraceConnector) mergeSpan(src ...sdktrace.TracerProviderOption) *sdktrace.TracerProvider {
	src = append(src, sdktrace.WithSpanProcessor(m.processorExporter))
	if m.Debug {
		src = append(src, sdktrace.WithSpanProcessor(m.processorDebug))
	}
	return sdktrace.NewTracerProvider(src...)
}
