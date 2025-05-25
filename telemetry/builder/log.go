package builder

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"sync"
	"time"
)

var zeroTime time.Time

type TraceExporter struct {
	logger    *zap.Logger
	encoderMu sync.Mutex
	exporter  trace.SpanExporter

	stoppedMu  sync.RWMutex
	stopped    bool
	timestamps bool
}

func NewTraceExporter(logger *zap.Logger, options ...stdouttrace.Option) (*TraceExporter, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	exporter, err := stdouttrace.New(options...)
	if err != nil {
		return nil, fmt.Errorf("creating stdout exporter: %w", err)
	}

	return &TraceExporter{
		logger:   logger,
		exporter: exporter,
	}, nil
}
func (e *TraceExporter) Timestamps(ts bool) {
	e.timestamps = ts
}

func (e *TraceExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.stoppedMu.RLock()
	stopped := e.stopped
	e.stoppedMu.RUnlock()
	if stopped {
		return nil
	}

	if len(spans) == 0 {
		return nil
	}

	stubs := tracetest.SpanStubsFromReadOnlySpans(spans)

	e.encoderMu.Lock()
	defer e.encoderMu.Unlock()

	for i := range stubs {
		stub := &stubs[i]
		// Remove timestamps
		if !e.timestamps {
			stub.StartTime = zeroTime
			stub.EndTime = zeroTime
			for j := range stub.Events {
				ev := &stub.Events[j]
				ev.Time = zeroTime
			}
		}

		e.logger.Debug("exporting traces", zap.Any("spans", []zap.Field{
			zap.Any("name", stub.Name),
			zap.String("traceId", stub.SpanContext.TraceID().String()),
			zap.Any("spanId", stub.SpanContext.SpanID().String()),
		}))
	}

	return nil
}

// MarshalLog is the marshaling function used by the logging system to represent this Exporter.
func (e *TraceExporter) MarshalLog() interface{} {
	return struct {
		Type           string
		WithTimestamps bool
	}{
		Type:           "stdout",
		WithTimestamps: e.timestamps,
	}
}

// Shutdown is called to stop the exporter, it performs no action.
func (e *TraceExporter) Shutdown(ctx context.Context) error {
	e.stoppedMu.Lock()
	e.stopped = true
	e.stoppedMu.Unlock()

	return nil
}

// MetricExporter implements the metric.Exporter interface
type MetricExporter struct {
	logger   *zap.Logger
	exporter metric.Exporter
}

func (e *MetricExporter) ForceFlush(ctx context.Context) error {
	return e.exporter.ForceFlush(ctx)
}

func (e *MetricExporter) Shutdown(ctx context.Context) error {
	return e.exporter.Shutdown(ctx)
}

// NewMetricExporter creates a new metricExporter that wraps a stdoutmetric exporter
// It accepts a zap logger and optional stdoutmetric.Option parameters
func NewMetricExporter(logger *zap.Logger, options ...stdoutmetric.Option) (*MetricExporter, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Create the stdoutmetric exporter with the provided options
	exporter, err := stdoutmetric.New(options...)
	if err != nil {
		return nil, fmt.Errorf("creating stdout exporter: %w", err)
	}

	return &MetricExporter{
		logger:   logger,
		exporter: exporter,
	}, nil
}

// Temporality returns the Temporality for the given InstrumentKind.
// Delegates to the wrapped exporter's Temporality method
func (e *MetricExporter) Temporality(k metric.InstrumentKind) metricdata.Temporality {
	return e.exporter.Temporality(k)
}

// Aggregation returns the Aggregation for the given InstrumentKind.
// Delegates to the wrapped exporter's Aggregation method
func (e *MetricExporter) Aggregation(k metric.InstrumentKind) metric.Aggregation {
	return e.exporter.Aggregation(k)
}

// Export exports metric data through the wrapped exporter
// Also logs the metrics using the zap logger
func (e *MetricExporter) Export(ctx context.Context, data *metricdata.ResourceMetrics) error {
	// Log the entire data structure as JSON
	fields := []zap.Field{
		zap.Any("length", data.Resource.Len()),
	}
	pid, ok := data.Resource.Set().Value(semconv.ProcessPIDKey)
	if ok {
		fields = append(fields, zap.String("pid", pid.Emit()))
	}
	executable, ok := data.Resource.Set().Value(semconv.ProcessExecutablePathKey)
	if ok {
		fields = append(fields, zap.String("executablePath", executable.Emit()))
	}

	e.logger.Debug("exporting metrics", fields...)

	// Forward to the wrapped exporter
	//return e.exporter.Export(ctx, data)
	return nil
}
