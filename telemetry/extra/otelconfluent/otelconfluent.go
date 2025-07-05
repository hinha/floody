package otelconfluent

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTracerName = "confluent/v2"
)

var tracer = otel.GetTracerProvider().Tracer(defaultTracerName, trace.WithInstrumentationVersion("semver:v2"))

type ProducerFn func() error

func ProduceWithTrace(ctx context.Context, topic string, key []byte, fn ProducerFn) error {
	if !trace.SpanFromContext(ctx).IsRecording() {
		if err := fn(); err != nil {
			return nil
		}
	}

	ctx, span := tracer.Start(ctx, "KAFKA Produce"+" "+topic,
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.kafka.key", string(key)),
		),
	)
	defer span.End()

	err := fn()
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}
