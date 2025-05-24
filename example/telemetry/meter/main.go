package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/hinha/floody/telemetry/builder"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"math/rand"
	"time"
)

// This example demonstrates how to use OpenTelemetry metrics with the floody telemetry package.
// It shows how to create and use three types of metrics:
// 1. Counter - for counting events (e.g., number of requests)
// 2. Histogram - for measuring distributions (e.g., request durations)
// 3. Observable Gauge - for observing current values (e.g., memory usage)
//
// The metrics are exported using OTLP over HTTP to a collector endpoint.

func main() {
	ctx := context.Background()

	// Configure the meter connector for OTLP HTTP export
	// This defines how metrics will be exported to the collector
	config := &builder.MeterConnector{
		// The endpoint where the OTLP collector is running
		Endpoint: "localhost:4318",

		// Whether to use insecure connections (false means use TLS)
		Insecure: false,

		// Enable debug output for troubleshooting
		Debug: true,

		// Debug options when Debug is true
		DebugOption: builder.DebugMetric{
			PrettyPrint: false,
			Timestamps:  true,
		},

		// The URL path for the OTLP HTTP endpoint
		URLPath: "/v1/metrics",

		// Headers to include in the OTLP HTTP requests (e.g., for authentication)
		Headers: map[string]string{},

		// TLS configuration for the HTTP client
		HTTPTLSClientConfig: &tls.Config{InsecureSkipVerify: true},

		// Additional options (commented out for this example)
		//Timeout:             0, // Timeout for requests
		//ShutdownTimeout:     0, // Timeout for shutdown
		//ReconnectionPeriod:  0, // Period between reconnection attempts
		//HTTPCompression:     0, // Compression for HTTP requests
		//HTTPRetry:           otlpmetrichttp.RetryConfig{}, // Retry configuration
		//HTTPProxy:           nil, // Proxy configuration
	}

	// Create a meter connector with HTTP export and a 2-second collection interval
	// The interval determines how often metrics are collected and exported
	connector := builder.WithMeterHttp(config, sdkmetric.WithInterval(time.Second))

	// Create a new meter provider builder
	provider := builder.NewMeterProvider()

	// Configure the meter provider with resource attributes and build it
	// These attributes will be attached to all metrics from this provider
	meter, closer, err := provider.WithAttributes(
		// Standard OpenTelemetry semantic conventions for service information
		semconv.ServiceNameKey.String("otel-go-demo"),
		semconv.ServiceVersionKey.String("1.0.0"),
		// Custom attributes
		attribute.String("environment", "production")).Build(ctx, connector)

	if err != nil {
		panic(err)
	}
	defer closer(ctx)

	// Create a named Meter from the built MeterProvider
	// The name should identify the instrumentation scope (typically your module or package)
	m := meter.Meter("otel-go-demo")

	// 1. COUNTER METRIC
	// Counters are monotonic, they can only go up or be reset to zero
	// They're good for counting events like requests, errors, etc.
	counter, err := m.Int64Counter(
		"request.counter", // metric name
		metric.WithDescription("Counts the number of requests"), // human-readable description
		metric.WithUnit("{request}"),                            // unit of measurement
	)
	if err != nil {
		panic(err)
	}

	// 2. HISTOGRAM METRIC
	// Histograms measure distributions of values, like request durations or response sizes
	// They track statistics like count, sum, min, max, and configurable percentiles
	histogram, err := m.Float64Histogram(
		"request.duration", // metric name
		metric.WithDescription("Measures the duration of requests"), // human-readable description
		metric.WithUnit("ms"), // unit of measurement (milliseconds)
	)
	if err != nil {
		panic(err)
	}

	// 3. OBSERVABLE GAUGE METRIC
	// Gauges measure current values that can go up and down, like memory usage or queue length
	// Observable gauges use callbacks to collect values on demand during collection
	gauge, err := m.Float64ObservableGauge(
		"system.memory.usage",                          // metric name
		metric.WithDescription("Reports memory usage"), // human-readable description
		metric.WithUnit("By"),                          // unit of measurement (bytes)
	)
	if err != nil {
		panic(err)
	}

	// Register a callback function for the observable gauge
	// This function will be called whenever the metrics are collected
	_, err = m.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			// Simulate memory usage between 50-100 MB
			memoryUsage := 50.0 + rand.Float64()*50.0
			// Record the observation with attributes
			o.ObserveFloat64(gauge, memoryUsage*1024*1024, metric.WithAttributes(attribute.String("type", "heap")))
			return nil
		},
		gauge, // The gauge instrument this callback updates
	)
	if err != nil {
		panic(err)
	}

	// SIMULATION LOOP
	// In a real application, you would record metrics as part of your actual business logic
	// Here we simulate a workload to demonstrate how to record metrics
	fmt.Println("Recording metrics for 10 seconds...")
	for i := 0; i < 15; i++ {
		// 1. RECORDING COUNTER METRICS
		// Add 1 to the counter with an attribute specifying the endpoint
		// You can use attributes to add dimensions to your metrics for better analysis
		counter.Add(ctx, 1, metric.WithAttributes(attribute.String("endpoint", "/api/users")))
		counter.Add(ctx, 1, metric.WithAttributes(attribute.String("endpoint", "/api/products")))

		// 2. RECORDING HISTOGRAM METRICS
		// Generate a random duration between 10-100ms and record it
		// Histograms will automatically track the distribution statistics
		duration := 10.0 + rand.Float64()*90.0 // Random duration between 10-100ms
		histogram.Record(ctx, duration, metric.WithAttributes(attribute.String("endpoint", "/api/users")))

		// Note: We don't need to explicitly record the gauge metrics because
		// they're collected automatically via the callback we registered earlier

		// Print progress and wait before the next iteration
		fmt.Printf("Iteration %d: recorded counter and histogram metrics\n", i+1)
		time.Sleep(1 * time.Second)
	}

	// The metrics have been collected and exported to the configured endpoint
	fmt.Println("Metrics example completed. Metrics have been exported.")
	time.Sleep(2 * time.Second)

	// When the program exits, the defer closer(ctx) will be called,
	// which will shut down the meter provider gracefully
}
