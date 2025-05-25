# Telemetry Example Application

This application demonstrates the use of OpenTelemetry for metrics and tracing. It can be configured using either environment variables (via a `.env` file) or command-line flags.

## Configuration

You can configure the application using either:

1. Environment variables in a `.env` file
2. Command-line flags
3. A combination of both (command-line flags will override environment variables)

### Environment Variables

Create a `.env` file in the `example` directory with the following variables:

```
# Application configuration
APP_NAME=my-example-server-test

# Meter connector configuration
METER_ENDPOINT=localhost:4318
METER_URL_PATH=/v1/metrics
METER_INSECURE=true
METER_DEBUG=false

# Trace connector configuration
TRACE_ENDPOINT=localhost:4318
TRACE_URL_PATH=/v1/traces
TRACE_INSECURE=true
TRACE_DEBUG=false
```

### Command-line Flags

The following command-line flags are available:

```
  -app-name string
        Application name (default "my-example-server-test")
  -help
        Show help message
  -meter-debug
        Enable meter debug mode
  -meter-endpoint string
        Meter endpoint (e.g., localhost:4318)
  -meter-insecure
        Meter insecure connection (default true)
  -meter-url-path string
        Meter URL path (e.g., /v1/metrics)
  -trace-debug
        Enable trace debug mode
  -trace-endpoint string
        Trace endpoint (e.g., localhost:4318)
  -trace-insecure
        Trace insecure connection (default true)
  -trace-url-path string
        Trace URL path (e.g., /v1/traces)
```

## Running the Application

### Using Environment Variables

1. Create a `.env` file in the `example` directory with your configuration
2. Run the application:

```bash
cd example/telemetry
go run main.go
```

### Using Command-line Flags

```bash
cd example/telemetry
go run main.go -trace-endpoint=localhost:4318 -trace-url-path=/v1/traces
```

### Showing Help

```bash
cd example/telemetry
go run main.go -help
```

## Example Usage

1. Connect to a local OTLP endpoint:
```bash
go run main.go -trace-endpoint=localhost:4318 -trace-url-path=/v1/traces
```

2. Enable debug mode:
```bash
go run main.go -meter-debug=true -trace-debug=true
```

3. Use environment variables from `.env` file:
```bash
# Make sure you have a .env file in the example directory
go run main.go
```