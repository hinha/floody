package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/hinha/floody/telemetry/builder"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"log"
	"net/http"
)

func main() {
	ctx := context.Background()
	config := &builder.TraceConnector{
		Endpoint:            "otlp.hinha.web.id",
		Insecure:            false,
		Debug:               false,
		Headers:             map[string]string{"Authorization": "$2a$04$MslvP7qnyS8DThjgAYoexOs8.SP5/9TJ19ywVCk.1sXHjJUajEmG."},
		URLPath:             "/v1/traces",
		HTTPTLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//EndpointURL: "",
		//URLPath:     "",
		//Headers:             nil,
		//Timeout:             0,
		//Debug:               false,
		//DebugTimestamp:      false,
		//ReconnectionPeriod:  0,
		//Compressor:          "",
		//TLSCredentials:      nil,
		//ServiceConfig:       "",
		//DialOption:          nil,
		//GRPCConn:            nil,
		//GRPCRetry:           otlptracegrpc.RetryConfig{},
		//HTTPCompression:     0,
		//HTTPTLSClientConfig: nil,
		//HTTPRetry:           otlptracehttp.RetryConfig{},
		//HTTPProxy:           nil,
	}
	provider := builder.NewTracerBuilder()
	trace, closer, err := provider.WithAttributes(
		semconv.ServiceNameKey.String("otel-go-demo"),
		semconv.ServiceVersionKey.String("1.0.0"),
		attribute.String("environment", "production")).Build(ctx, builder.WithTraceHttp(config))
	if err != nil {
		panic(err)
	}
	defer closer(ctx)

	// ✅ HTTP handler dengan otelhttp
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, span := trace.Tracer("otel-go-demo").Start(r.Context(), "handleRequest")
		defer span.End()

		fmt.Fprintln(w, "Hello, traced world!")
	}

	// Middleware tracing
	wrappedHandler := otelhttp.NewHandler(http.HandlerFunc(handler), "rootHandler")

	log.Println("Listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", wrappedHandler))

	//middleware := builder.NewMiddlewareWithConfig(trace.Tracer("otel-go-demo"))
	//
	//mux := http.NewServeMux()
	//mux.Handle("/", middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	time.Sleep(150 * time.Millisecond) // simulate work
	//	w.Write([]byte("Hello from OpenTelemetry!"))
	//})))
	//
	//log.Print("Listening on :8081...")
	//err = http.ListenAndServe(":8081", mux)
	//if err != nil {
	//	log.Fatal(err)
	//}

}
