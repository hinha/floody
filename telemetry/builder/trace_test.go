package builder

import (
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"testing"
)

func TestNewTracerBuilder(t *testing.T) {
	builder := NewTracerBuilder()
	if builder == nil {
		t.Error("expected non-nil builder")
	}

	// Check default logger is set
	if builder.config.log == nil {
		t.Error("expected default logger to be set")
	}
}

func TestTracerBuilder_WithLogger(t *testing.T) {
	logger := zap.NewNop()
	builder := NewTracerBuilder().WithLogger(logger)

	if builder.config.log != logger {
		t.Error("expected logger to be set")
	}
}

func TestTracerBuilder_WithAttributes(t *testing.T) {
	attr1 := attribute.String("key1", "value1")
	attr2 := attribute.Int("key2", 123)

	builder := NewTracerBuilder().WithAttributes(attr1, attr2)

	if len(builder.config.attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(builder.config.attributes))
	}

	if builder.config.attributes[0] != attr1 {
		t.Errorf("expected attribute %v, got %v", attr1, builder.config.attributes[0])
	}

	if builder.config.attributes[1] != attr2 {
		t.Errorf("expected attribute %v, got %v", attr2, builder.config.attributes[1])
	}

	// Test adding more attributes
	attr3 := attribute.Bool("key3", true)
	builder = builder.WithAttributes(attr3)

	if len(builder.config.attributes) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(builder.config.attributes))
	}

	if builder.config.attributes[2] != attr3 {
		t.Errorf("expected attribute %v, got %v", attr3, builder.config.attributes[2])
	}
}

func TestTracerBuilder_WithServiceName(t *testing.T) {
	serviceName := "test-service"
	builder := NewTracerBuilder().WithServiceName(serviceName)

	if builder.config.serviceName != serviceName {
		t.Errorf("expected service name %s, got %s", serviceName, builder.config.serviceName)
	}
}

func TestDebugTrace_apply(t *testing.T) {
	// Test with default values
	debug := DebugTrace{}
	opts := debug.apply()
	if len(opts) != 1 { // WithoutTimestamps is added by default when Timestamps is false
		t.Errorf("expected 1 option, got %d", len(opts))
	}

	// Test with all values set
	debug = DebugTrace{
		Timestamps:  true,
		PrettyPrint: true,
	}

	opts = debug.apply()
	if len(opts) != 1 { // Only PrettyPrint should be added
		t.Errorf("expected 1 option, got %d", len(opts))
	}

	// Test with only PrettyPrint
	debug = DebugTrace{
		Timestamps:  false,
		PrettyPrint: true,
	}

	opts = debug.apply()
	if len(opts) != 2 { // WithoutTimestamps and PrettyPrint
		t.Errorf("expected 2 options, got %d", len(opts))
	}
}
