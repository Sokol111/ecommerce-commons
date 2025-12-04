package consumer

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// MessageTracer відповідає за OpenTelemetry tracing для Kafka повідомлень.
type MessageTracer interface {
	// ExtractContext витягує trace context з Kafka headers
	ExtractContext(ctx context.Context, message *kafka.Message) context.Context

	// StartConsumerSpan створює span для обробки повідомлення
	StartConsumerSpan(ctx context.Context, message *kafka.Message) (context.Context, trace.Span)

	// StartDLQSpan створює span для відправки в DLQ
	StartDLQSpan(ctx context.Context, message *kafka.Message, dlqTopic string) (context.Context, trace.Span)

	// InjectContext додає trace context в Kafka headers
	InjectContext(ctx context.Context, message *kafka.Message)
}

type messageTracer struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func newMessageTracer() MessageTracer {
	return &messageTracer{
		tracer:     otel.Tracer("kafka-consumer"),
		propagator: otel.GetTextMapPropagator(),
	}
}

func (t *messageTracer) ExtractContext(ctx context.Context, message *kafka.Message) context.Context {
	if len(message.Headers) == 0 {
		return ctx
	}

	// Конвертуємо Kafka headers в map для propagator
	headersMap := make(map[string]string)
	for _, header := range message.Headers {
		headersMap[header.Key] = string(header.Value)
	}

	// Витягуємо trace context
	carrier := propagation.MapCarrier(headersMap)
	return t.propagator.Extract(ctx, carrier)
}

func (t *messageTracer) StartConsumerSpan(ctx context.Context, message *kafka.Message) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "kafka.consume",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", *message.TopicPartition.Topic),
			attribute.Int("messaging.partition", int(message.TopicPartition.Partition)),
			attribute.Int64("messaging.offset", int64(message.TopicPartition.Offset)),
			attribute.String("messaging.message.key", string(message.Key)),
		),
	)
}

func (t *messageTracer) StartDLQSpan(ctx context.Context, message *kafka.Message, dlqTopic string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "kafka.send_to_dlq",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", dlqTopic),
			attribute.String("messaging.source.topic", *message.TopicPartition.Topic),
			attribute.Int("messaging.source.partition", int(message.TopicPartition.Partition)),
			attribute.Int64("messaging.source.offset", int64(message.TopicPartition.Offset)),
			attribute.String("messaging.message.key", string(message.Key)),
		),
	)
}

func (t *messageTracer) InjectContext(ctx context.Context, message *kafka.Message) {
	// Конвертуємо існуючі headers в map
	headersMap := make(map[string]string)
	for _, header := range message.Headers {
		headersMap[header.Key] = string(header.Value)
	}

	// Додаємо trace context
	carrier := propagation.MapCarrier(headersMap)
	t.propagator.Inject(ctx, carrier)

	// Оновлюємо headers повідомлення
	message.Headers = nil
	for key, value := range headersMap {
		message.Headers = append(message.Headers, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}
}
