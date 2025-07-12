// Telemetry Example Application
//
// This application demonstrates the use of OpenTelemetry for metrics and tracing.
// It can be configured using command-line flags to connect to different backends.
//
// Example usage:
//   - Show help: go run main.go -help
//   - Connect to a local OTLP endpoint: go run main.go -trace-endpoint=localhost:4318 -trace-url-path=/v1/traces
//   - Enable debug mode: go run main.go -meter-debug=true -trace-debug=true
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/hinha/floody/telemetry"
	"github.com/hinha/floody/telemetry/builder"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Define command-line flags
var (
	// General flags
	help       = flag.Bool("help", false, "Show help message")
	appName    = flag.String("app-name", "my-example-server-test", "Application name")
	authorized = flag.String("authorized", "Bearer my-token-1234567890", "Authorization token for meter and trace connectors if needed")

	// Meter connector flags
	meterEndpoint = flag.String("meter-endpoint", "", "Meter endpoint (e.g., localhost:4318)")
	meterURLPath  = flag.String("meter-url-path", "", "Meter URL path (e.g., /v1/metrics)")
	meterInsecure = flag.Bool("meter-insecure", true, "Meter insecure connection")
	meterDebug    = flag.Bool("meter-debug", false, "Enable meter debug mode")

	// Trace connector flags
	traceEndpoint = flag.String("trace-endpoint", "", "Trace endpoint (e.g., localhost:4318)")
	traceURLPath  = flag.String("trace-url-path", "", "Trace URL path (e.g., /v1/traces)")
	traceInsecure = flag.Bool("trace-insecure", true, "Trace insecure connection")
	traceDebug    = flag.Bool("trace-debug", false, "Enable trace debug mode")
)

// run -meter-endpoint=otel.hinha.web.id -meter-url-path=/v1/metrics -meter-insecure=false -meter-debug=true -trace-endpoint=otel.hinha.web.id -trace-url-path=/v1/traces -trace-insecure=false -trace-debug=true
// run -meter-endpoint=otel.hinha.web.id -meter-url-path=/v1/metrics -meter-insecure=false -meter-debug=true -trace-endpoint=otel.hinha.web.id -trace-url-path=/v1/traces -trace-insecure=false -trace-debug=true -authorized=Bearer my-token-1234567890
// getEnvOrDefault returns the value of the environment variable or the default value if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvBoolOrDefault returns the boolean value of the environment variable or the default value if not set
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Default options that can be overridden by environment variables or command-line flags
var options = &telemetry.Options{
	MeterHTTP: true,
	MeterConnector: &builder.MeterConnector{
		Insecure:            true,
		HTTPTLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Headers:             map[string]string{"Authorization": "Bearer my-token-1234567890"},
		Debug:               false,
		DebugOption:         builder.DebugMetric{Timestamps: true},
	},
	TraceHTTP: true,
	TraceConnector: &builder.TraceConnector{
		Insecure:            true,
		Headers:             map[string]string{},
		HTTPTLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Debug:               false,
		DebugOption:         builder.DebugTrace{Timestamps: true},
	},
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load("../../example/.env"); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	} else {
		log.Printf("Successfully loaded .env file")
	}

	// Parse command-line flags
	flag.Parse()

	// Show help message if requested
	if *help {
		fmt.Println("Telemetry Example Application")
		fmt.Println("This application demonstrates the use of OpenTelemetry for metrics and tracing.")
		fmt.Println("\nAvailable flags:")
		flag.PrintDefaults()
		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  APP_NAME                - Application name")
		fmt.Println("  METER_ENDPOINT          - Meter endpoint (e.g., localhost:4318)")
		fmt.Println("  METER_URL_PATH          - Meter URL path (e.g., /v1/metrics)")
		fmt.Println("  METER_INSECURE          - Meter insecure connection (true/false)")
		fmt.Println("  METER_DEBUG             - Enable meter debug mode (true/false)")
		fmt.Println("  TRACE_ENDPOINT          - Trace endpoint (e.g., localhost:4318)")
		fmt.Println("  TRACE_URL_PATH          - Trace URL path (e.g., /v1/traces)")
		fmt.Println("  TRACE_INSECURE          - Trace insecure connection (true/false)")
		fmt.Println("  TRACE_DEBUG             - Enable trace debug mode (true/false)")
		return
	}

	options.MeterConnector.Headers["Authorization"] = *authorized
	options.TraceConnector.Headers["Authorization"] = *authorized
	// Set options from environment variables first, then override with command-line flags if provided
	options.AppName = *appName

	// Meter connector options
	options.MeterConnector.Endpoint = *meterEndpoint
	options.MeterConnector.URLPath = *meterURLPath
	options.MeterConnector.Insecure = *meterInsecure
	options.MeterConnector.Debug = *meterDebug

	// Trace connector options
	options.TraceConnector.Endpoint = *traceEndpoint
	options.TraceConnector.URLPath = *traceURLPath
	options.TraceConnector.Insecure = *traceInsecure
	options.TraceConnector.Debug = *traceDebug

	// Log the current configuration
	log.Printf("Starting with configuration:")
	log.Printf("  AppName: %s", options.AppName)
	log.Printf("  Meter Endpoint: %s", options.MeterConnector.Endpoint)
	log.Printf("  Meter URL Path: %s", options.MeterConnector.URLPath)
	log.Printf("  Meter Insecure: %v", options.MeterConnector.Insecure)
	log.Printf("  Meter Debug: %v", options.MeterConnector.Debug)
	log.Printf("  Trace Endpoint: %s", options.TraceConnector.Endpoint)
	log.Printf("  Trace URL Path: %s", options.TraceConnector.URLPath)
	log.Printf("  Trace Insecure: %v", options.TraceConnector.Insecure)
	log.Printf("  Trace Debug: %v", options.TraceConnector.Debug)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Init Telemetry
	otl := telemetry.NewFactory(options)
	loc, _ := time.LoadLocation("Asia/Jakarta")

	// set logger for telemetry
	otl.SetLogger(func() zerolog.LevelWriter {
		return zerolog.MultiLevelWriter(zerolog.ConsoleWriter{
			Out:          os.Stdout,
			TimeFormat:   "2006-01-02 15:04:05",
			TimeLocation: loc,
			//FormatTimestamp: func(i interface{}) string {
			//	loc, _ := time.LoadLocation("Asia/Jakarta")
			//	t := zerolog.TimeFieldFormat
			//	if str, ok := i.(string); ok {
			//		parsed, err := time.Parse(t, str)
			//		if err == nil {
			//			return parsed.In(loc).Format("2006-01-02 15:04:05")
			//		}
			//	}
			//	return ""
			//},
		})
	})

	closers, err := otl.Build(ctx,
		telemetry.WithAttribute(semconv.SessionIDKey.String("1234567890")),
		telemetry.WithMeterReaderOption(
			sdkmetric.WithInterval(time.Second*5),
			sdkmetric.WithTimeout(30*time.Second)),
	)

	log := otl.GetLogger()
	if err != nil {
		log.Fatal().Err(err).Msg("Telemetry error")
	}
	defer func() {
		for _, closer := range closers {
			closer(ctx)
		}
	}()

	m := telemetry.GetMetricTelemetry()
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

	middleware := telemetry.NewMiddlewareWithConfig(otl, trace.WithSpanKind(trace.SpanKindServer))
	mux := http.NewServeMux()

	mux.Handle("/", middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Seed dengan waktu saat ini
		rand.Seed(time.Now().UnixNano())
		randomInt := rand.Intn(200)
		counter.Add(r.Context(), 1, metric.WithAttributes(attribute.String("endpoint", "/api/example")))

		time.Sleep(time.Duration(randomInt) * time.Millisecond) // simulate work
		w.Write([]byte(fmt.Sprintf("Hello from OpenTelemetry! duration: %d", randomInt)))
	})))

	log.Print("Listening on :8081...")
	err = http.ListenAndServe(":8081", mux)
	if err != nil {
		log.Fatal().Err(err)
	}
}
