package main

import (
	"context"
	"errors"
	"github.com/hinha/floody/telemetry/builder"
	"github.com/shirou/gopsutil/v3/cpu"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv/v1.20.0/httpconv"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	ctx := context.Background()
	attr := resource.WithAttributes(
		semconv.ServiceNameKey.String("example-http-server"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.ServiceInstanceIDKey.String("local-ID"),
		semconv.ServiceNamespaceKey.String("my-namespace"))

	res, err := resource.New(ctx, attr, resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS())

	resMerge, err := resource.Merge(resource.Default(), res)
	if err != nil {
		panic(err)
	}

	newProvider := builder.NewMeterProvider(builder.Meter{})
	meterProvider, closer, err := newProvider.AddMetricOption(sdkmetric.WithResource(resMerge)).
		Exporter(ctx,
			otlpmetrichttp.WithEndpoint("localhost:4318"),
			otlpmetrichttp.WithInsecure(),
		).Build()
	if err != nil {
		panic(err)
	}
	defer closer(ctx)

	// Create Meter from the built MeterProvider
	meter := meterProvider.Meter("example-http-server")

	// Create custom Histogram instrument
	requestDuration, err := meter.Float64Histogram(
		semconv.HTTPServerRequestBodySizeName,
		//"http_server_duration",
		// http.server.request.duration
		metric.WithDescription("HTTP Server request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}
	apiCounter, err := meter.Int64Counter( // TODO [] done
		"api.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		panic(err)
	}

	// Instrumentasi resource (CPU, RAM, GC, goroutine)
	err = otelruntime.Start( // TODO mark library
		otelruntime.WithMeterProvider(meterProvider),
		otelruntime.WithMinimumReadMemStatsInterval(time.Second*5),
	)
	if err != nil {
		log.Fatalf("Failed starting runtime instrumentation: %v", err)
	}

	cpuGauge, err := meter.Float64ObservableGauge( // TODO mark library
		"go.cpu.usage.percent",
		metric.WithDescription("Current system CPU usage percentage"),
	)
	if err != nil {
		log.Fatalf("failed to create gauge: %v", err)
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		percentages, err := cpu.Percent(0, false) // TODO mark library
		if err != nil {
			return err
		}
		if len(percentages) > 0 {
			observer.ObserveFloat64(cpuGauge, percentages[0])
		}
		return nil
	}, cpuGauge)
	if err != nil {
		log.Fatalf("failed to register callback: %v", err)
	}

	// HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCounter.Add(ctx, 1, metric.WithAttributes(httpconv.ResponseHeader(r.Header)...))
		requestDuration.Record(r.Context(), 0.05)
		w.Write([]byte("Hello, OpenTelemetry!"))

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// 🔥 Force Flush Metric langsung!
			_ = newProvider.Reader().ForceFlush(ctx)
		}()

	})

	// Bungkus pakai otelhttp.NewHandler
	wrappedHandler := otelhttp.NewHandler(handler, "root-handler")
	http.Handle("/", wrappedHandler)
	if err := http.ListenAndServe(":8081", wrappedHandler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
	log.Println("Starting HTTP server on :8081")
	os.Exit(0)
}
