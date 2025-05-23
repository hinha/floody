package builder

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"time"
)

// MeterOption applies a configuration option value to a MeterProvider.
type MeterOption interface {
	apply(Meter)
}

// optionMeterFunc applies a set of options to a meter.
type optionMeterFunc func(Meter)

// apply returns a meter with option(s) applied.
func (o optionMeterFunc) apply(mtr Meter) {
	o(mtr)
}

func WithMeterLogger(log *zap.Logger) MeterOption {
	return optionMeterFunc(func(mtr Meter) {
		mtr.log = log
	})
}

func WithResource(res *resource.Resource) MeterOption {
	return optionMeterFunc(func(mtr Meter) {
		mtr.resource = res
	})
}

func WithExamplarFilter(filter exemplar.Filter) MeterOption {
	return optionMeterFunc(func(mtr Meter) {
		mtr.exemplarFilter = filter
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
	Endpoint       string
	EndpointURL    string
	URLPath        string
	Insecure       bool
	Headers        map[string]string
	Timeout        time.Duration
	Debug          bool
	DebugTimestamp bool

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

		otlptracegrpc.WithEndpoint(c.Endpoint)
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

	var stdoutOptions stdouttrace.Option
	if !m.DebugTimestamp {
		stdoutOptions = stdouttrace.WithoutTimestamps()
	}

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint(), stdoutOptions)
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
