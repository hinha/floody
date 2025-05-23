package record

import "net/http"

// secureAgent is a global interface point for the nrsecureagent's hooks into the go agent.
// The default value for this is a noOpSecurityAgent value, which has null definitions for
// the methods. The Go compiler is expected to optimize away all the securityAgent method
// calls in this case, effectively removing the hooks from the running agent.
//
// If the nrsecureagent integration is initialized, it will register a real securityAgent
// value in the secureAgent variable, thus "activating" the hooks.
var secureAgent securityAgent = noOpSecurityAgent{}

type securityAgent interface {
	RefreshState(map[string]string) bool
	DeactivateSecurity()
	SendEvent(string, ...any) any
	IsSecurityActive() bool
	DistributedTraceHeaders(hdrs *http.Request, secureAgentevent any)
	SendExitEvent(any, error)
	RequestBodyReadLimit() int
}

// IsSecurityAgentPresent returns true if there's an actual security agent hooked in to the
// Go APM agent, whether or not it's enabled or operating in any particular mode. It returns
// false only if the hook-in interface for those functions is a No-Op will null functionality.
func IsSecurityAgentPresent() bool {
	_, isNoOp := secureAgent.(noOpSecurityAgent)
	return !isNoOp
}

// noOpSecurityAgent satisfies the secureAgent interface but is a null implementation
// that will largely be optimized away at compile time.
type noOpSecurityAgent struct {
}

func (t noOpSecurityAgent) RefreshState(connectionData map[string]string) bool {
	return false
}

func (t noOpSecurityAgent) DeactivateSecurity() {
}

func (t noOpSecurityAgent) SendEvent(caseType string, data ...any) any {
	return nil
}

func (t noOpSecurityAgent) IsSecurityActive() bool {
	return false
}

func (t noOpSecurityAgent) DistributedTraceHeaders(hdrs *http.Request, secureAgentevent any) {
}

func (t noOpSecurityAgent) SendExitEvent(secureAgentevent any, err error) {
}
func (t noOpSecurityAgent) RequestBodyReadLimit() int {
	return 300 * 1000
}

const (
	// DistributedTraceTelemetryHeader is the header used by New Relic agents
	// for automatic trace payload instrumentation.
	DistributedTraceTelemetryHeader = "Telemetry"
	// DistributedTraceW3CTraceStateHeader is one of two headers used by W3C
	// trace context
	DistributedTraceW3CTraceStateHeader = "Tracestate"
	// DistributedTraceW3CTraceParentHeader is one of two headers used by W3C
	// trace context
	DistributedTraceW3CTraceParentHeader = "Traceparent"
)

// TransportType is used in Transaction.AcceptDistributedTraceHeaders to
// represent the type of connection that the trace payload was transported
// over.
type TransportType string

// TransportType names used across New Relic agents:
const (
	TransportUnknown TransportType = "Unknown"
	TransportHTTP    TransportType = "HTTP"
	TransportHTTPS   TransportType = "HTTPS"
	TransportKafka   TransportType = "Kafka"
	TransportJMS     TransportType = "JMS"
	TransportIronMQ  TransportType = "IronMQ"
	TransportAMQP    TransportType = "AMQP"
	TransportQueue   TransportType = "Queue"
	TransportOther   TransportType = "Other"
)

func (tt TransportType) toString() string {
	switch tt {
	case TransportHTTP, TransportHTTPS, TransportKafka, TransportJMS, TransportIronMQ, TransportAMQP,
		TransportQueue, TransportOther:
		return string(tt)
	default:
		return string(TransportUnknown)
	}
}

type BodyBuffer struct {
	buf             []byte
	isDataTruncated bool
}

func (b *BodyBuffer) Write(p []byte) (int, error) {
	if l := len(b.buf); len(p) <= secureAgent.RequestBodyReadLimit()-l {
		b.buf = append(b.buf, p...)
		return len(p), nil
	} else if l := len(b.buf); secureAgent.RequestBodyReadLimit()-l > 1 {
		end := secureAgent.RequestBodyReadLimit() - l
		b.buf = append(b.buf, p[:end-1]...)
		return end, nil
	} else {
		b.isDataTruncated = true
		return 0, nil
	}
}

func (b *BodyBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.buf)

}

func (b *BodyBuffer) read() []byte {
	if b == nil {
		return make([]byte, 0)
	}
	return b.buf
}

func (b *BodyBuffer) isBodyTruncated() bool {
	if b == nil {
		return false
	}
	return b.isDataTruncated
}
func (b *BodyBuffer) String() (string, bool) {
	if b == nil {
		return "", false
	}
	return string(b.buf), b.isDataTruncated

}
