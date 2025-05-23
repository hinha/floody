package telemetry

import (
	"context"
	"github.com/hinha/floody/telemetry/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"testing"
	"time"
)

// Membuat mock tracer provider untuk testing
//type mockTracerProvider struct {
//	sdktrace.TracerProvider
//}

//func newMockTracerProvider() *mockTracerProvider {
//	return &mockTracerProvider{}
//}

func (p *mockTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return p.tracer
}

func TestNewFactory(t *testing.T) {
	// Test cases tidak berubah
	tests := []struct {
		name    string
		options *Options
		want    []attribute.KeyValue
	}{
		{
			name: "creates factory with basic options",
			options: &Options{
				AppName:     "test-app",
				Version:     "1.0.0",
				Environment: "test",
			},
			want: []attribute.KeyValue{
				semconv.ServiceNameKey.String("test-app"),
				semconv.ServiceVersionKey.String("1.0.0"),
				attribute.String("service.environment", "test"),
			},
		},
		{
			name:    "creates factory with empty options",
			options: &Options{},
			want: []attribute.KeyValue{
				semconv.ServiceNameKey.String(""),
				semconv.ServiceVersionKey.String(""),
				attribute.String("service.environment", ""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory(tt.options)
			require.NotNil(t, factory)

			f, ok := factory.(*Factory)
			require.True(t, ok)
			assert.Equal(t, tt.want, f.attributes)
			assert.NotNil(t, f.app)
		})
	}
}

// Test lainnya tidak berubah
// ...

func TestFactory_StartTransaction(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*Factory)
		verifyMock  func(*testing.T, context.Context, trace.Span)
		expectError bool
	}{
		{
			name: "successful transaction start",
			setupMock: func(f *Factory) {
				// Menggunakan NoopTracerProvider untuk testing
				tp := newMockTracerProvider()
				f.app.tracer = tp.Tracer("test-tracer")
			},
			verifyMock: func(t *testing.T, ctx context.Context, span trace.Span) {
				assert.NotNil(t, ctx)
				assert.NotNil(t, span)
				// Verifikasi tambahan dapat ditambahkan di sini
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{
				Options: &Options{},
				app:     &Application[metric.Meter, trace.Tracer]{},
			}

			if tt.setupMock != nil {
				tt.setupMock(f)
			}

			ctx, span := f.StartTransaction(context.Background(), "test-transaction")

			if tt.verifyMock != nil {
				tt.verifyMock(t, ctx, span)
			}

			// Pastikan untuk end span setelah selesai
			span.End()
		})
	}
}

func TestFactory_Build(t *testing.T) {
	tests := []struct {
		name         string
		options      *Options
		expectedErr  bool
		verifyResult func(*testing.T, []builder.CloseFunc, error)
	}{
		{
			name: "successful build with basic configuration",
			options: &Options{
				AppName:       "test-app",
				Version:       "1.0.0",
				Environment:   "test",
				EndpointMeter: "localhost:4318",
				EndpointTrace: "localhost:4317",
				Insecure:      true,
			},
			verifyResult: func(t *testing.T, closers []builder.CloseFunc, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, closers)
			},
		},
		{
			name: "build with invalid meter endpoint",
			options: &Options{
				AppName:       "test-app",
				EndpointMeter: "invalid-endpoint:::4318",
				EndpointTrace: "localhost:4317",
				Insecure:      true,
			},
			expectedErr: true,
			verifyResult: func(t *testing.T, closers []builder.CloseFunc, err error) {
				// We should either get an error during Build or during closer execution
				if err == nil {
					// If Build succeeds, we expect the closer to fail
					assert.NotNil(t, closers)
					ctx := context.Background()
					closeErr := closers[0](ctx)
					assert.Error(t, closeErr)
				} else {
					// If Build fails, closers should be nil
					assert.Nil(t, closers)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFactory(tt.options)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			closers, err := f.Build(ctx)

			if tt.verifyResult != nil {
				tt.verifyResult(t, closers, err)
			}

			// Only clean up closers if we didn't expect an error
			if !tt.expectedErr && closers != nil {
				for _, closer := range closers {
					err := closer(ctx)
					assert.NoError(t, err)
				}
			}

			factory, ok := f.(*Factory)
			require.True(t, ok)

			if !tt.expectedErr {
				assert.NotNil(t, factory.logger)
				assert.NotNil(t, factory.app)
				assert.NotNil(t, factory.app.meter)
				assert.NotNil(t, factory.app.tracer)
			}
		})
	}
}

// Helper function to create mock meter provider
type mockMeterProvider struct {
	metric.MeterProvider
	meter metric.Meter
}

func newMockMeterProvider() *mockMeterProvider {
	return &mockMeterProvider{
		meter: &mockMeter{},
	}
}

type mockMeter struct {
	metric.Meter
}

// Helper function to create mock tracer provider
type mockTracerProvider struct {
	sdktrace.TracerProvider
	tracer trace.Tracer
}

func newMockTracerProvider() *mockTracerProvider {
	return &mockTracerProvider{
		tracer: &mockTracer{},
	}
}

type mockTracer struct {
	trace.Tracer
}

func (m *mockTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, &mockSpan{}
}

type mockSpan struct {
	trace.Span
}

func (s *mockSpan) End(options ...trace.SpanEndOption) {}
