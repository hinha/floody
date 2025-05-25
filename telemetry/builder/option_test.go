package builder

import (
	"context"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"testing"
)

func TestOptionMeterFunc_apply(t *testing.T) {
	// Create a builder
	builder := NewMeterProvider()

	// Create a test function that modifies the builder
	called := false
	testFunc := func(b *MeterBuilder) {
		called = true
		b.config.serviceName = "test-service"
	}

	// Create an optionMeterFunc from the test function
	option := optionMeterFunc(testFunc)

	// Apply the option to the builder
	option.apply(builder)

	// Check that the function was called and the builder was modified
	if !called {
		t.Error("expected function to be called")
	}

	if builder.config.serviceName != "test-service" {
		t.Errorf("expected service name to be set to 'test-service', got %s", builder.config.serviceName)
	}
}

func TestOptionTraceFunc_apply(t *testing.T) {
	// Create a builder
	builder := NewTracerBuilder()

	// Create a test function that modifies the builder
	called := false
	testFunc := func(b *TracerBuilder) {
		called = true
		b.config.serviceName = "test-service"
	}

	// Create an optionTraceFunc from the test function
	option := optionTraceFunc(testFunc)

	// Apply the option to the builder
	option.apply(builder)

	// Check that the function was called and the builder was modified
	if !called {
		t.Error("expected function to be called")
	}

	if builder.config.serviceName != "test-service" {
		t.Errorf("expected service name to be set to 'test-service', got %s", builder.config.serviceName)
	}
}

func TestMeterConnector_mergeMeter(t *testing.T) {
	// Create a MeterConnector
	connector := &MeterConnector{}

	// Test with no readers
	provider := connector.mergeMeter(nil)
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Test with options
	options := []sdkmetric.Option{
		sdkmetric.WithReader(nil),
	}
	provider = connector.mergeMeter(options)
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Test with debug reader
	connector.Debug = true
	connector.readerDebug = sdkmetric.NewPeriodicReader(nil)
	provider = connector.mergeMeter(nil)
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Test with exporter reader
	connector.readerExporter = sdkmetric.NewPeriodicReader(nil)
	provider = connector.mergeMeter(nil)
	if provider == nil {
		t.Error("expected non-nil provider")
	}
}

func TestTraceConnector_mergeSpan(t *testing.T) {
	// Create a TraceConnector
	connector := &TraceConnector{}

	// Test with no options
	provider := connector.mergeSpan()
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Test with options
	options := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}
	provider = connector.mergeSpan(options...)
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Test with processor
	connector.processorExporter = sdktrace.NewBatchSpanProcessor(nil)
	provider = connector.mergeSpan()
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Test with debug processor
	connector.Debug = true
	connector.processorDebug = sdktrace.NewBatchSpanProcessor(nil)
	provider = connector.mergeSpan()
	if provider == nil {
		t.Error("expected non-nil provider")
	}
}

func TestMeterConnector_debug(t *testing.T) {
	// Create a MeterConnector
	connector := &MeterConnector{}

	// Test with Debug = false
	err := connector.debug(zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test with Debug = true
	connector.Debug = true
	err = connector.debug(zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTraceConnector_debug(t *testing.T) {
	// Create a TraceConnector
	connector := &TraceConnector{}

	// Test with Debug = false
	err := connector.debug(zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test with Debug = true
	connector.Debug = true
	err = connector.debug(zap.NewNop())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMeterConnector_metricExporter(t *testing.T) {
	// Create a MeterConnector
	connector := &MeterConnector{}

	// Test with nil connector
	var nilConnector *MeterConnector
	err := nilConnector.metricExporter(context.Background())
	if err == nil {
		t.Error("expected error with nil connector")
	}

	// Test with no exporter set
	err = connector.metricExporter(context.Background())
	if err == nil {
		t.Error("expected error with no exporter set")
	}

	// Test with both HTTP and gRPC set
	connector.setHttp = true
	connector.setGrpc = true
	err = connector.metricExporter(context.Background())
	if err == nil {
		t.Error("expected error with both HTTP and gRPC set")
	}
}

func TestTraceConnector_spanExporter(t *testing.T) {
	// Create a TraceConnector
	connector := &TraceConnector{}

	// Test with nil connector
	var nilConnector *TraceConnector
	err := nilConnector.spanExporter(context.Background())
	if err == nil {
		t.Error("expected error with nil connector")
	}

	// Test with no exporter set
	err = connector.spanExporter(context.Background())
	if err == nil {
		t.Error("expected error with no exporter set")
	}

	// Test with both HTTP and gRPC set
	connector.setHttp = true
	connector.setGrpc = true
	err = connector.spanExporter(context.Background())
	if err == nil {
		t.Error("expected error with both HTTP and gRPC set")
	}
}

func TestWithMeterHttp(t *testing.T) {
	// Create a MeterConnector
	connector := &MeterConnector{}

	// Create a MeterBuilder
	builder := NewMeterProvider()

	// Apply the WithMeterHttp option
	option := WithMeterHttp(connector)
	option.apply(builder)

	// Check that the connector is set on the builder
	if builder.client != connector {
		t.Error("expected connector to be set on builder")
	}

	// Check that the connector is configured for HTTP
	if !connector.setHttp {
		t.Error("expected connector to be configured for HTTP")
	}
}

func TestWithMeterGrpc(t *testing.T) {
	// Create a MeterConnector
	connector := &MeterConnector{}

	// Create a MeterBuilder
	builder := NewMeterProvider()

	// Apply the WithMeterGrpc option
	option := WithMeterGrpc(connector)
	option.apply(builder)

	// Check that the connector is set on the builder
	if builder.client != connector {
		t.Error("expected connector to be set on builder")
	}

	// Check that the connector is configured for gRPC
	if !connector.setGrpc {
		t.Error("expected connector to be configured for gRPC")
	}
}

func TestWithTraceHttp(t *testing.T) {
	// Create a TraceConnector
	connector := &TraceConnector{}

	// Create a TracerBuilder
	builder := NewTracerBuilder()

	// Apply the WithTraceHttp option
	option := WithTraceHttp(connector)
	option.apply(builder)

	// Check that the connector is set on the builder
	if builder.client != connector {
		t.Error("expected connector to be set on builder")
	}

	// Check that the connector is configured for HTTP
	if !connector.setHttp {
		t.Error("expected connector to be configured for HTTP")
	}
}

func TestWithTraceGrpc(t *testing.T) {
	// Create a TraceConnector
	connector := &TraceConnector{}

	// Create a TracerBuilder
	builder := NewTracerBuilder()

	// Apply the WithTraceGrpc option
	option := WithTraceGrpc(connector)
	option.apply(builder)

	// Check that the connector is set on the builder
	if builder.client != connector {
		t.Error("expected connector to be set on builder")
	}

	// Check that the connector is configured for gRPC
	if !connector.setGrpc {
		t.Error("expected connector to be configured for gRPC")
	}
}

func TestUriMeterConnector(t *testing.T) {
	// Create a MeterConnector
	connector := &MeterConnector{
		Endpoint:    "localhost:4317",
		EndpointURL: "http://localhost:4318",
		Insecure:    true,
		Headers:     map[string]string{"key": "value"},
	}

	// Call uriMeterConnector
	uriMeterConnector(connector)

	// Check that the options were set correctly
	if len(connector.metricHttpOption) == 0 {
		t.Error("expected HTTP options to be set")
	}

	if len(connector.metricGrpcOption) == 0 {
		t.Error("expected gRPC options to be set")
	}
}

func TestUriConnector(t *testing.T) {
	// Create a TraceConnector
	connector := &TraceConnector{
		Endpoint:    "localhost:4317",
		EndpointURL: "http://localhost:4318",
		Insecure:    true,
		Headers:     map[string]string{"key": "value"},
	}

	// Call uriConnector
	uriConnector(connector)

	// Check that the options were set correctly
	if len(connector.traceHttpOption) == 0 {
		t.Error("expected HTTP options to be set")
	}

	if len(connector.traceGrpcOption) == 0 {
		t.Error("expected gRPC options to be set")
	}
}
