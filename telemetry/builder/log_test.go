package builder

import (
	"context"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
	"testing"
)

func TestNewTraceExporter(t *testing.T) {
	// Test with nil logger
	exporter, err := NewTraceExporter(nil)
	if err == nil {
		t.Error("expected error when logger is nil, got nil")
	}
	if exporter != nil {
		t.Errorf("expected nil exporter, got %v", exporter)
	}

	// Test with valid logger
	logger := zap.NewNop()
	exporter, err = NewTraceExporter(logger)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exporter == nil {
		t.Error("expected non-nil exporter, got nil")
	}
}

func TestTraceExporter_Timestamps(t *testing.T) {
	logger := zap.NewNop()
	exporter, err := NewTraceExporter(logger)
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	tests := []struct {
		input    bool
		expected bool
	}{
		{true, true},
		{false, false},
	}

	for _, tt := range tests {
		exporter.Timestamps(tt.input)
		if exporter.timestamps != tt.expected {
			t.Errorf("expected timestamps to be %v, got %v", tt.expected, exporter.timestamps)
		}
	}
}

func TestTraceExporter_ExportSpans(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	exporter, err := NewTraceExporter(logger)
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	// Test with canceled context
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	err = exporter.ExportSpans(canceledCtx, nil)
	if err == nil {
		t.Error("expected error with canceled context, got nil")
	}

	// Test with empty spans
	err = exporter.ExportSpans(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error with empty spans: %v", err)
	}

	// Test with timestamps enabled and disabled
	exporter.Timestamps(true)
	err = exporter.ExportSpans(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error with timestamps enabled: %v", err)
	}

	exporter.Timestamps(false)
	err = exporter.ExportSpans(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error with timestamps disabled: %v", err)
	}
}

func TestTraceExporter_MarshalLog(t *testing.T) {
	logger := zap.NewNop()
	exporter, err := NewTraceExporter(logger)
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	exporter.Timestamps(true)
	result := exporter.MarshalLog()
	expected := struct {
		Type           string
		WithTimestamps bool
	}{
		Type:           "stdout",
		WithTimestamps: true,
	}

	if result != expected {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestTraceExporter_Shutdown(t *testing.T) {
	logger := zap.NewNop()
	exporter, err := NewTraceExporter(logger)
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	ctx := context.Background()
	err = exporter.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected error on shutdown: %v", err)
	}

	if !exporter.stopped {
		t.Errorf("expected exporter to be stopped")
	}

	// Test that ExportSpans returns nil when exporter is stopped
	err = exporter.ExportSpans(ctx, nil)
	if err != nil {
		t.Errorf("expected nil error when exporter is stopped, got: %v", err)
	}
}

func TestNewMetricExporter(t *testing.T) {
	// Test with nil logger
	exporter, err := NewMetricExporter(nil)
	if err == nil {
		t.Error("expected error when logger is nil, got nil")
	}
	if exporter != nil {
		t.Errorf("expected nil exporter, got %v", exporter)
	}

	// Test with valid logger
	logger := zap.NewNop()
	exporter, err = NewMetricExporter(logger)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exporter == nil {
		t.Error("expected non-nil exporter, got nil")
	}
}

func TestMetricExporter_ForceFlush(t *testing.T) {
	logger := zap.NewNop()
	mockExporter := &mockMetricExporter{}
	exporter := &MetricExporter{
		logger:   logger,
		exporter: mockExporter,
	}

	err := exporter.ForceFlush(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !mockExporter.forceFlushed {
		t.Errorf("expected ForceFlush to be called")
	}
}

func TestMetricExporter_Shutdown(t *testing.T) {
	logger := zap.NewNop()
	mockExporter := &mockMetricExporter{}
	exporter := &MetricExporter{
		logger:   logger,
		exporter: mockExporter,
	}

	err := exporter.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !mockExporter.shutdown {
		t.Errorf("expected Shutdown to be called")
	}
}

func TestMetricExporter_Export(t *testing.T) {
	logger := zap.NewNop()
	mockExporter := &mockMetricExporter{}
	exporter := &MetricExporter{
		logger:   logger,
		exporter: mockExporter,
	}

	// Create a simple ResourceMetrics
	resourceMetrics := &metricdata.ResourceMetrics{}

	// Test export
	err := exporter.Export(context.Background(), resourceMetrics)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMetricExporter_Temporality(t *testing.T) {
	logger := zap.NewNop()
	mockExporter := &mockMetricExporter{}
	exporter := &MetricExporter{
		logger:   logger,
		exporter: mockExporter,
	}

	// Test that Temporality delegates to the wrapped exporter
	temporality := exporter.Temporality(metric.InstrumentKindCounter)
	if temporality != metricdata.CumulativeTemporality {
		t.Errorf("expected CumulativeTemporality, got %v", temporality)
	}
}

func TestMetricExporter_Aggregation(t *testing.T) {
	logger := zap.NewNop()
	mockExporter := &mockMetricExporter{}
	exporter := &MetricExporter{
		logger:   logger,
		exporter: mockExporter,
	}

	// Test that Aggregation method doesn't panic
	_ = exporter.Aggregation(metric.InstrumentKindCounter)
	// We can't test the return value directly because the interface has non-exported methods
}

type mockMetricExporter struct {
	forceFlushed bool
	shutdown     bool
}

func (m *mockMetricExporter) ForceFlush(ctx context.Context) error {
	m.forceFlushed = true
	return nil
}

func (m *mockMetricExporter) Shutdown(ctx context.Context) error {
	m.shutdown = true
	return nil
}

func (m *mockMetricExporter) Temporality(k metric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (m *mockMetricExporter) Aggregation(k metric.InstrumentKind) metric.Aggregation {
	return nil
}

func (m *mockMetricExporter) Export(ctx context.Context, data *metricdata.ResourceMetrics) error {
	return nil
}
