package outbox

import (
	"context"
	"maps"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type tracePropagator interface {
	// SaveTraceContext saves the current trace context into headers for storage in outbox.
	SaveTraceContext(ctx context.Context, headers map[string]string) map[string]string

	// StartKafkaProducerSpan restores trace context from stored headers, creates a producer span,
	// and returns Kafka-ready headers with the new span context.
	StartKafkaProducerSpan(headers map[string]string, topic, messageID string) (context.Context, trace.Span, []kafka.Header)
}

type otelTracePropagator struct {
	tracer trace.Tracer
}

func newTracePropagator(tp trace.TracerProvider) tracePropagator {
	return &otelTracePropagator{
		tracer: tp.Tracer("outbox"),
	}
}

func (t *otelTracePropagator) SaveTraceContext(ctx context.Context, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	carrier := propagation.MapCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return headers
}

// StartKafkaProducerSpan restores trace context from stored headers, creates a producer span,
// and returns Kafka-ready headers with the new span context.
// Called when sending a message to Kafka.
func (t *otelTracePropagator) StartKafkaProducerSpan(headers map[string]string, topic, messageID string) (context.Context, trace.Span, []kafka.Header) {
	// Restore trace context from stored headers
	ctx := context.Background()
	if len(headers) > 0 {
		carrier := propagation.MapCarrier(headers)
		ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	}

	// Start producer span as child of restored context
	ctx, span := t.tracer.Start(ctx, "kafka.produce",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.message.id", messageID),
		),
	)

	// Inject new span context into headers for Kafka
	updatedHeaders := maps.Clone(headers)
	if updatedHeaders == nil {
		updatedHeaders = make(map[string]string)
	}
	carrier := propagation.MapCarrier(updatedHeaders)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// Convert to Kafka headers
	kafkaHeaders := make([]kafka.Header, 0, len(updatedHeaders))
	for key, value := range updatedHeaders {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}

	return ctx, span, kafkaHeaders
}
