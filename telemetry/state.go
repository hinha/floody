package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"sync/atomic"
)

type (
	meterHolder struct {
		m metric.Meter
	}
	traceHolder struct {
		t trace.Tracer
	}
)

var (
	globalMeter = defaultMetricTelemetry()
	globalTrace = defaultTraceTelemetry()
)

// GetMetricTelemetry returns the global Meter instance.
func GetMetricTelemetry() metric.Meter {
	return globalMeter.Load().(meterHolder).m
}

// SetMetricTelemetry sets the global Meter to meter.
func SetMetricTelemetry(meter metric.Meter) {
	globalMeter.Store(meterHolder{m: meter})
}

// GetTraceTelemetry returns the global Trace instance.
func GetTraceTelemetry() trace.Tracer {
	return globalTrace.Load().(traceHolder).t
}

// SetTraceTelemetry sets the global Trace to trace.
func SetTraceTelemetry(trace trace.Tracer) {
	globalTrace.Store(traceHolder{t: trace})
}

func defaultMetricTelemetry() *atomic.Value {
	v := &atomic.Value{}
	// Use the default meter from otel package
	defaultMeter := otel.Meter("floody")
	v.Store(meterHolder{m: defaultMeter})
	return v
}

func defaultTraceTelemetry() *atomic.Value {
	v := &atomic.Value{}
	// Use the default trace from otel package
	defaultTrace := otel.Tracer("floody")
	v.Store(traceHolder{t: defaultTrace})
	return v
}
