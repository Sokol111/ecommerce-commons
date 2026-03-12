package consumer

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// MessageTracer відповідає за OpenTelemetry tracing для Kafka повідомлень.
type MessageTracer interface {
	// ExtractContext витягує trace context з Kafka headers
	ExtractContext(ctx context.Context, record *kgo.Record) context.Context

	// StartConsumerSpan створює span для обробки повідомлення
	StartConsumerSpan(ctx context.Context, record *kgo.Record) (context.Context, trace.Span)

	// StartDLQSpan створює span для відправки в DLQ
	StartDLQSpan(ctx context.Context, record *kgo.Record, dlqTopic string) (context.Context, trace.Span)

	// InjectContext додає trace context в Kafka headers
	InjectContext(ctx context.Context, record *kgo.Record)
}

type messageTracer struct {
	tracer trace.Tracer
}

func newMessageTracer(tp trace.TracerProvider) MessageTracer {
	return &messageTracer{
		tracer: tp.Tracer("kafka-consumer"),
	}
}

func (t *messageTracer) ExtractContext(ctx context.Context, record *kgo.Record) context.Context {
	if len(record.Headers) == 0 {
		return ctx
	}

	// Конвертуємо Kafka headers в map для propagator
	headersMap := make(map[string]string)
	for _, header := range record.Headers {
		headersMap[header.Key] = string(header.Value)
	}

	// Витягуємо trace context
	carrier := propagation.MapCarrier(headersMap)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

func (t *messageTracer) StartConsumerSpan(ctx context.Context, record *kgo.Record) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "kafka.consume",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", record.Topic),
			attribute.Int("messaging.partition", int(record.Partition)),
			attribute.Int64("messaging.offset", record.Offset),
			attribute.String("messaging.message.key", string(record.Key)),
		),
	)
}

func (t *messageTracer) StartDLQSpan(ctx context.Context, record *kgo.Record, dlqTopic string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "kafka.send_to_dlq",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", dlqTopic),
			attribute.String("messaging.source.topic", record.Topic),
			attribute.Int("messaging.source.partition", int(record.Partition)),
			attribute.Int64("messaging.source.offset", record.Offset),
			attribute.String("messaging.message.key", string(record.Key)),
		),
	)
}

func (t *messageTracer) InjectContext(ctx context.Context, record *kgo.Record) {
	// Конвертуємо існуючі headers в map
	headersMap := make(map[string]string)
	for _, header := range record.Headers {
		headersMap[header.Key] = string(header.Value)
	}

	// Додаємо trace context
	carrier := propagation.MapCarrier(headersMap)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// Оновлюємо headers повідомлення
	record.Headers = nil
	for key, value := range headersMap {
		record.Headers = append(record.Headers, kgo.RecordHeader{
			Key:   key,
			Value: []byte(value),
		})
	}
}
