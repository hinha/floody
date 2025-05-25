# Floody

Floody is a Go library that provides robust logging and telemetry capabilities for Go applications. It offers a unified interface for both logging and telemetry, making it easy to instrument your applications with observability features.

## Features

### Logging

Floody provides a high-performance logging system built on top of [Uber's zap](https://github.com/uber-go/zap) with additional features:

- Multiple output formats (console and JSON)
- Log rotation using [lumberjack](https://github.com/natefinch/lumberjack)
- Configurable log levels
- Named loggers for better organization
- Alerting for missed logs
- High-performance asynchronous logging with diode writer

### Telemetry with OpenTelemetry

Floody integrates with [OpenTelemetry](https://opentelemetry.io/) to provide comprehensive telemetry capabilities:

- Metrics collection and export
- Distributed tracing
- HTTP middleware for automatic request tracing
- Support for both HTTP and gRPC exporters
- Runtime metrics collection (CPU, memory, etc.)
- Configurable sampling and exemplar filtering
- Resource attributes for service identification

## Understanding OpenTelemetry

OpenTelemetry is an observability framework and toolkit designed to create and manage telemetry data such as traces, metrics, and logs. Floody makes it easy to integrate OpenTelemetry into your Go applications.

### Key Concepts

#### Traces and Spans

A **trace** represents the entire journey of a request as it moves through your distributed system. Each trace consists of one or more **spans**, which represent individual operations within that journey.

- **Span**: A named, timed operation representing a piece of work in a trace
- **Trace Context**: Information that identifies which trace a span belongs to, allowing for distributed tracing across service boundaries
- **Attributes**: Key-value pairs that provide additional context about a span

#### Metrics

Metrics are measurements of system behavior collected at runtime:

- **Counter**: A cumulative metric that represents a single monotonically increasing value (e.g., number of requests)
- **Histogram**: A metric that tracks the distribution of values (e.g., request durations)
- **Gauge**: A metric that represents a single numerical value that can go up and down (e.g., memory usage)

#### Resource and Instrumentation

- **Resource**: Represents the entity producing the telemetry (e.g., your service)
- **Instrumentation**: The code that creates telemetry data

## Installation

```bash
go get github.com/hinha/floody
```

## Usage

### Logging

```go
package main

import (
    "github.com/hinha/floody/log"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func main() {
    // Create a development configuration
    cfg := log.NewDevelopmentConfig()

    // For production, you might want to use JSON encoding and log rotation
    // cfg.Base.Encoding = "all"

    // Create a new logger
    logger := log.NewLogger(cfg)
    defer logger.Sync() // Ensure logs are flushed

    // Add caller information and stack traces for errors
    zapLogger := logger.WithOptions(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

    // Use named loggers for better organization
    appLogger := zapLogger.Named("app")

    // Log at different levels
    appLogger.Info("Application started", zap.String("version", "1.0.0"))
    appLogger.Debug("Debug information")
    appLogger.Error("An error occurred", zap.Error(err))
}
```

### Telemetry

#### Step-by-Step Guide to Using OpenTelemetry with Floody

1. **Configure Telemetry Options**

```go
package main

import (
    "context"
    "github.com/hinha/floody/telemetry"
    "github.com/hinha/floody/telemetry/builder"
    "go.uber.org/zap"
    "net/http"
    "time"
)

func main() {
    ctx := context.Background()

    // Configure telemetry options
    options := &telemetry.Options{
        // Service identification
        AppName: "my-service",

        // Meter endpoint configuration (HTTP in this example)
        EndpointMeter: telemetry.Endpoint{
            Endpoint: "otel-collector:4318", // Address of your collector
            WithHTTP: true,                  // Use HTTP protocol (false for gRPC)
        },

        // Trace endpoint configuration (gRPC in this example)
        EndpointTrace: telemetry.Endpoint{
            Endpoint: "otel-collector:4317", // Address of your collector
            WithHTTP: false,                 // Use gRPC protocol
        },

        // Additional meter configuration
        Meter: builder.Meter{},

        // Security settings
        Insecure: true, // Use insecure connection (no TLS)
    }
```

2. **Initialize Telemetry and Build the Provider**

```go
    // Initialize telemetry factory
    otl := telemetry.NewFactory(options)

    // Configure and build the telemetry provider
    // This sets up both metrics and tracing
    closers, err := otl.MeterOptions().  // Configure metrics
        TracerOptions().                 // Configure tracing
        Build(ctx)                       // Build the provider

    if err != nil {
        otl.GetLogger().Fatal("Telemetry error", zap.Error(err))
    }

    // Ensure proper cleanup when your application exits
    defer func() {
        for _, closer := range closers {
            closer(ctx)
        }
    }()
```

3. **Create HTTP Middleware for Request Tracing**

```go
    // Create HTTP middleware for automatic request tracing
    middleware := telemetry.NewMiddlewareWithConfig(otl)

    // Use middleware with HTTP handlers
    mux := http.NewServeMux()
    mux.Handle("/", middleware.Handler("index", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })))

    http.ListenAndServe(":8080", mux)
}
```

4. **Creating and Using Metrics**

```go
// Get the metric provider
m := telemetry.GetMetricTelemetry()

// Create a counter metric
counter, err := m.Int64Counter(
    "request.counter",                           // metric name
    metric.WithDescription("Request counter"),   // description
    metric.WithUnit("{request}"),                // unit
)
if err != nil {
    panic(err)
}

// Increment the counter with attributes
counter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("endpoint", "/api/users"),
    attribute.String("method", "GET"),
))

// Create a histogram metric for measuring durations
histogram, err := m.Float64Histogram(
    "request.duration",
    metric.WithDescription("Request duration"),
    metric.WithUnit("ms"),
)
if err != nil {
    panic(err)
}

// Record a duration
histogram.Record(ctx, 42.5, metric.WithAttributes(
    attribute.String("endpoint", "/api/users"),
))
```

5. **Manual Tracing**

```go
// Get a tracer
tracer := telemetry.GetTracerTelemetry()

// Create a span
ctx, span := tracer.Start(ctx, "operation_name")
defer span.End()

// Add attributes to the span
span.SetAttributes(
    attribute.String("key", "value"),
    attribute.Int("count", 42),
)

// Create a child span
ctx, childSpan := tracer.Start(ctx, "child_operation")
defer childSpan.End()

// Record an error
childSpan.RecordError(err)
childSpan.SetStatus(codes.Error, "Operation failed")
```

## Advanced Configuration

### Logging Configuration

```go
// Create a production configuration
cfg := log.NewProductionConfig()

// Configure log rotation
cfg.RotateConfig = log.RotateConfig{
    Filename:   "/var/log/myapp.log",
    MaxSize:    100, // megabytes
    MaxBackups: 3,
    MaxAge:     28, // days
    Compress:   true,
}

// Use both console and JSON output
cfg.Base.Encoding = "all"

// Configure log level
cfg.Base.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

// Create logger with alerter for missed logs
logger := log.NewLogger(cfg, log.WithAlerter(func(missed int) {
    fmt.Printf("Missed %d log entries due to buffer overflow\n", missed)
}))
```

### Telemetry Configuration

#### Resource Attributes

Resource attributes identify your service and provide context for your telemetry data:

```go
// Configure resource attributes
options := &telemetry.Options{
    // Basic service identification
    AppName:     "my-service",       // Name of your service
    Version:     "1.0.0",            // Version of your service
    Environment: "production",       // Environment (production, staging, development)

    // Schema URL for OpenTelemetry semantic conventions
    SchemaURL:   "https://opentelemetry.io/schemas/1.4.0",

    // Additional instrumentation attributes
    Instrumentation: []attribute.KeyValue{
        attribute.String("library.name", "floody"),
        attribute.String("library.version", "0.1.0"),
    },
    // ... other options
}
```

#### Export Configuration

Configure how telemetry data is exported to your backend:

```go
// HTTP export for metrics
options.EndpointMeter = telemetry.Endpoint{
    Endpoint: "otel-collector:4318",  // Collector address
    WithHTTP: true,                   // Use HTTP protocol
    URLPath:  "/v1/metrics",          // URL path (optional)
}

// gRPC export for traces
options.EndpointTrace = telemetry.Endpoint{
    Endpoint: "otel-collector:4317",  // Collector address
    WithHTTP: false,                  // Use gRPC protocol
}

// Security settings
options.Insecure = false              // Use TLS (secure connection)
options.Headers = map[string]string{  // Custom headers (e.g., for authentication)
    "Authorization": "Bearer token",
}
```

#### Advanced Options

Fine-tune the behavior of your telemetry:

```go
// Configure retry behavior for failed exports
options.Retry = telemetry.RetryConfig{
    Enabled:         true,               // Enable retry
    InitialInterval: time.Second,        // Start with 1s between retries
    MaxInterval:     time.Second * 10,   // Maximum 10s between retries
    MaxElapsedTime:  time.Minute,        // Give up after 1 minute
}

// Debug options
options.MeterConnector = &builder.MeterConnector{
    Debug: true,                         // Enable debug output
    DebugOption: builder.DebugMetric{    // Debug configuration
        PrettyPrint: true,               // Format JSON output
        Timestamps:  true,               // Include timestamps
    },
}

// Build with additional options
otl := telemetry.NewFactory(options)
closers, err := otl.Build(ctx, 
    // Configure exemplar filtering for metrics
    telemetry.WithExemplarFilter(exemplar.TraceBasedFilter),

    // Add global attributes to all telemetry
    telemetry.WithAttribute(attribute.String("deployment.region", "us-west-1")),

    // Configure metric reader options
    telemetry.WithMeterReaderOption(
        sdkmetric.WithInterval(time.Second*30),  // Collection interval
        sdkmetric.WithTimeout(time.Minute),      // Export timeout
    ),
)
```

## Examples

See the `example` directory for complete working examples:

- `example/log/main.go` - Demonstrates logging features
- `example/telemetry/main.go` - Demonstrates telemetry features
- `example/telemetry/meter/main.go` - Demonstrates metrics collection
- `example/telemetry/trace/main.go` - Demonstrates distributed tracing

## Support & Contribute

We welcome contributions to Floody! Here's how you can help:

### Reporting Issues

If you encounter a bug or have a feature request, please open an issue on GitHub with:

- A clear description of the problem or feature
- Steps to reproduce (for bugs)
- Expected vs. actual behavior
- Version information (Go version, Floody version, etc.)

### Contributing Code

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests to ensure they pass
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Guidelines

- Follow Go best practices and coding conventions
- Write tests for new features and bug fixes
- Update documentation for any changes
- Add examples for new functionality

### Getting Help

If you have questions about using Floody, you can:

- Check the examples in the `example` directory
- Open a discussion on GitHub
- Reach out to the maintainers

## License

This project is licensed under the MIT License - see the LICENSE file for details.
