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

### Telemetry

Floody integrates with [OpenTelemetry](https://opentelemetry.io/) to provide comprehensive telemetry capabilities:

- Metrics collection and export
- Distributed tracing
- HTTP middleware for automatic request tracing
- Support for both HTTP and gRPC exporters
- Runtime metrics collection (CPU, memory, etc.)
- Configurable sampling and exemplar filtering
- Resource attributes for service identification

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
        AppName: "my-service",
        EndpointMeter: telemetry.Endpoint{
            Endpoint: "otel-collector:4318",
            WithHTTP: true,
        },
        EndpointTrace: telemetry.Endpoint{
            Endpoint: "otel-collector:4317",
            WithHTTP: false,
        },
        Meter:    builder.Meter{},
        Insecure: true,
    }
    
    // Initialize telemetry
    otl := telemetry.NewFactory(options)
    closers, err := otl.MeterOptions().
        TracerOptions().
        Build(ctx)
    if err != nil {
        otl.GetLogger().Fatal("Telemetry error", zap.Error(err))
    }
    defer func() {
        for _, closer := range closers {
            closer(ctx)
        }
    }()
    
    // Create HTTP middleware for request tracing
    middleware := telemetry.NewMiddlewareWithConfig(otl)
    
    // Use middleware with HTTP handlers
    mux := http.NewServeMux()
    mux.Handle("/", middleware.Handler("index", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })))
    
    http.ListenAndServe(":8080", mux)
}
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

```go
// Configure resource attributes
options := &telemetry.Options{
    AppName:     "my-service",
    Version:     "1.0.0",
    Environment: "production",
    SchemaURL:   "https://opentelemetry.io/schemas/1.4.0",
    Instrumentation: []attribute.KeyValue{
        attribute.String("library.name", "floody"),
        attribute.String("library.version", "0.1.0"),
    },
    // ... other options
}

// Configure retry behavior
options.Retry = telemetry.RetryConfig{
    Enabled:         true,
    InitialInterval: time.Second,
    MaxInterval:     time.Second * 10,
    MaxElapsedTime:  time.Minute,
}

// Build with exemplar filter
otl := telemetry.NewFactory(options)
closers, err := otl.Build(ctx, telemetry.WithExemplarFilter(exemplar.TraceBasedFilter))
```

## Examples

See the `example` directory for complete working examples:

- `example/log/main.go` - Demonstrates logging features
- `example/telemetry/main.go` - Demonstrates telemetry features
- `example/telemetry/meter/main.go` - Demonstrates metrics collection
- `example/telemetry/trace/main.go` - Demonstrates distributed tracing

## License

This project is licensed under the MIT License - see the LICENSE file for details.
