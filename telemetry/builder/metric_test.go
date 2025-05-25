package builder

import (
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
	"io"
	"testing"
)

func TestNewMeterProvider(t *testing.T) {
	provider := NewMeterProvider()
	if provider == nil {
		t.Error("expected non-nil provider")
	}

	// Check default logger is set
	if provider.config.log == nil {
		t.Error("expected default logger to be set")
	}
}

func TestMeterBuilder_WithLogger(t *testing.T) {
	logger := zap.NewNop()
	provider := NewMeterProvider().WithLogger(logger)

	if provider.config.log != logger {
		t.Error("expected logger to be set")
	}
}

func TestMeterBuilder_WithAttributes(t *testing.T) {
	attr1 := attribute.String("key1", "value1")
	attr2 := attribute.Int("key2", 123)

	provider := NewMeterProvider().WithAttributes(attr1, attr2)

	if len(provider.config.attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(provider.config.attributes))
	}

	if provider.config.attributes[0] != attr1 {
		t.Errorf("expected attribute %v, got %v", attr1, provider.config.attributes[0])
	}

	if provider.config.attributes[1] != attr2 {
		t.Errorf("expected attribute %v, got %v", attr2, provider.config.attributes[1])
	}

	// Test adding more attributes
	attr3 := attribute.Bool("key3", true)
	provider = provider.WithAttributes(attr3)

	if len(provider.config.attributes) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(provider.config.attributes))
	}

	if provider.config.attributes[2] != attr3 {
		t.Errorf("expected attribute %v, got %v", attr3, provider.config.attributes[2])
	}
}

func TestMeterBuilder_WithServiceName(t *testing.T) {
	serviceName := "test-service"
	provider := NewMeterProvider().WithServiceName(serviceName)

	if provider.config.serviceName != serviceName {
		t.Errorf("expected service name %s, got %s", serviceName, provider.config.serviceName)
	}
}

func TestMeterBuilder_AddMetricOption(t *testing.T) {
	// Create a meter provider
	provider := NewMeterProvider()

	// Add some options (we don't need to test the actual options, just that they're added)
	options := []sdkmetric.Option{
		// These are just placeholders, we're not testing their functionality
		sdkmetric.WithReader(nil),
	}

	provider = provider.AddMetricOption(options...)

	if len(provider.options) != len(options) {
		t.Errorf("expected %d options, got %d", len(options), len(provider.options))
	}

	// Add more options
	moreOptions := []sdkmetric.Option{
		sdkmetric.WithReader(nil),
	}

	provider = provider.AddMetricOption(moreOptions...)

	if len(provider.options) != len(options)+len(moreOptions) {
		t.Errorf("expected %d options, got %d", len(options)+len(moreOptions), len(provider.options))
	}
}

func TestDebugMetric_apply(t *testing.T) {
	// Test with default values
	debug := DebugMetric{}
	opts := debug.apply()
	if len(opts) != 1 { // WithoutTimestamps is added by default when Timestamps is false
		t.Errorf("expected 1 option, got %d", len(opts))
	}

	// Test with all values set
	debug = DebugMetric{
		Encoder:     nil, // Can't easily create a real encoder for testing
		Writer:      io.Discard,
		PrettyPrint: true,
		Timestamps:  false,
	}

	opts = debug.apply()
	if len(opts) != 3 { // Writer, PrettyPrint, and WithoutTimestamps
		t.Errorf("expected 3 options, got %d", len(opts))
	}

	// Test with Timestamps true
	debug = DebugMetric{
		Timestamps: true,
	}

	opts = debug.apply()
	if len(opts) != 0 { // No options should be added
		t.Errorf("expected 0 options, got %d", len(opts))
	}
}
