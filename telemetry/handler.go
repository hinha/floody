package telemetry

import (
	"fmt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semcov17 "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

type NewMiddleware struct {
	app         IFactory
	startOption []trace.SpanStartOption
}

func NewMiddlewareWithConfig(app IFactory, startOption ...trace.SpanStartOption) *NewMiddleware {
	if app == nil {
		return &NewMiddleware{}
	}

	return &NewMiddleware{app: app, startOption: startOption}
}

func (n *NewMiddleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		operation := makeTransactionName(r)

		otelHandler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}), operation,
			otelhttp.WithServerName(n.app.GetConfigs().AppName),
			otelhttp.WithMeterProvider(otel.GetMeterProvider()),
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
			otelhttp.WithSpanOptions(n.startOption...),
		)

		otelHandler.ServeHTTP(w, r)
	})
}

// SetWebRequestHTTP enriches the span with HTTP request attributes.
func (n *NewMiddleware) SetWebRequestHTTP(span trace.Span, r *http.Request) {
	if span == nil || r == nil {
		return
	}

	// Safely handle optional host
	host := r.Host
	if host == "" {
		host = r.URL.Host // Use URL host if `Host` is empty
	}

	span.SetAttributes(
		semcov17.HTTPStatusCodeKey.Int(http.StatusOK),
		semcov17.HTTPMethodKey.String(r.Method),
		semcov17.HTTPTargetKey.String(r.URL.Path),
		semcov17.HTTPSchemeKey.String(r.URL.Scheme),
		// semcov17.HTTPHostKey.String(host), // this automatically sets the net.peer.name attribute ?
		semcov17.HTTPSchemeKey.String(r.URL.Scheme),
		semcov17.HTTPTargetKey.String(r.URL.RequestURI()),
		// semcov17.NetPeerIPKey.String(r.RemoteAddr), // Peer IP this automatically sets the net.peer.ip attribute ?
	)

	// Capture headers like User-Agent, if relevant
	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		span.SetAttributes(attribute.String("http.user_agent", userAgent))
	}

}

// SetWebResponse enriches the span with HTTP response attributes and wraps the `http.ResponseWriter`.
func (n *NewMiddleware) SetWebResponse(span trace.Span, w http.ResponseWriter) http.ResponseWriter {
	if span == nil || w == nil {
		return w
	}

	return &responseWriterTracer{
		ResponseWriter: w,
		span:           span,
	}
}

// responseWriterTracer wraps http.ResponseWriter to capture attributes like HTTP status code.
type responseWriterTracer struct {
	http.ResponseWriter
	statusCode int
	span       trace.Span
}

// WriteHeader wraps the `WriteHeader` to capture HTTP status code as an attribute in the span.
func (rw *responseWriterTracer) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.span.SetAttributes(semcov17.HTTPStatusCodeKey.Int(statusCode))
	rw.ResponseWriter.WriteHeader(statusCode)
}

func makeTransactionName(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}
