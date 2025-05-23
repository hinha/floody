package builder

import (
	"context"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSafeMeterProvider_Success(t *testing.T) {
	meter := Meter{
		ReaderInterval:  2 * time.Second,
		ShutdownTimeout: 2 * time.Second,
	}

	builder := NewMeterProvider(meter)
	builder.exporter = &fakeExporter{}

	provider, closer, err := builder.Build()

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, closer)
}
func TestMeterProvider_NoExporter(t *testing.T) {
	meter := Meter{
		ReaderInterval:  2 * time.Second,
		ShutdownTimeout: 2 * time.Second,
	}

	builder := NewMeterProvider(meter)

	provider, closer, err := builder.Build()

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Nil(t, closer)
	assert.Equal(t, "metric exporter is nil, cannot build meter provider", err.Error())
}

// fakeExporter is a mock minimal exporter for testing purpose
type fakeExporter struct{}

func (f *fakeExporter) Temporality(k sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (f *fakeExporter) Aggregation(k sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(k)
}

func (f *fakeExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	return nil
}

func (f *fakeExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func (f *fakeExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (f *fakeExporter) MarshalLog() interface{} {
	return map[string]interface{}{"exporter": "fake"}
}
