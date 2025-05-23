package record

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type WebRequest struct {
	Header        http.Header
	URL           *url.URL
	Method        string
	Transport     TransportType
	Host          string
	Body          *BodyBuffer
	ServerName    string
	Type          string
	RemoteAddress string
}

func SetWebRequestHTTP(name string, r *http.Request) {
	if r == nil {
		SetWebRequest(WebRequest{})
		return
	}
	wr := WebRequest{
		Header:        r.Header,
		URL:           r.URL,
		Method:        r.Method,
		Transport:     transport(r),
		Host:          r.Host,
		Body:          reqBody(r),
		ServerName:    serverName(name, r),
		Type:          "HTTP",
		RemoteAddress: r.RemoteAddr,
	}
	SetWebRequest(wr)

	RecordRequest(r.Context(), otel.GetMeterProvider().Meter(fmt.Sprintf("%s %s", r.Method, r.URL.Path)), wr)
}

// SetWebRequest marks the transaction as a web transaction.  SetWebRequest
// additionally collects details on request attributes, url, and method if
// these fields are set.  If headers are present, the agent will look for
// distributed tracing headers using Transaction.AcceptDistributedTraceHeaders.
// Use Transaction.SetWebRequestHTTP if you have a *http.Request.
func SetWebRequest(r WebRequest) {
	if IsSecurityAgentPresent() {
		secureAgent.SendEvent("INBOUND", r)
	}

}

func WebRequestAttributes(wr WebRequest) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.method", wr.Method),
		attribute.String("http.url", wr.URL.String()),
		attribute.String("http.host", wr.Host),
		attribute.String("net.transport", string(wr.Transport)),
		attribute.String("net.peer.ip", wr.RemoteAddress),
		attribute.String("server.name", wr.ServerName),
	}
}

func RecordRequest(ctx context.Context, meter metric.Meter, wr WebRequest) {
	counter, _ := meter.Int64Counter(
		"http.request.count",
		metric.WithDescription("Count of incoming web requests"),
	)

	counter.Add(ctx, 1, metric.WithAttributes(WebRequestAttributes(wr)...))
}

func transport(r *http.Request) TransportType {
	if strings.HasPrefix(r.Proto, "HTTP") {
		if r.TLS != nil {
			return TransportHTTPS
		}
		return TransportHTTP
	}
	return TransportUnknown
}

func serverName(n string, r *http.Request) string {
	if strings.HasPrefix(r.Proto, "HTTP") {
		if r.TLS != nil {
			return r.TLS.ServerName
		}
	}
	return n
}

func reqBody(req *http.Request) *BodyBuffer {
	if req.Body != nil && req.Body != http.NoBody {
		buf := &BodyBuffer{buf: make([]byte, 0, 100)}
		tee := io.TeeReader(req.Body, buf)
		req.Body = io.NopCloser(tee)
		return buf
	}
	return nil
}
