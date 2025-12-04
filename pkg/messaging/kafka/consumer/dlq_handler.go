package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/messaging/kafka/producer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// DLQHandler відповідає за відправку повідомлень, які не вдалось обробити, в Dead Letter Queue.
type DLQHandler interface {
	// SendToDLQ відправляє повідомлення в DLQ з інформацією про помилку
	SendToDLQ(ctx context.Context, message *kafka.Message, processingErr error)
}

type dlqHandler struct {
	producer producer.Producer
	dlqTopic string
	tracer   MessageTracer
	log      *zap.Logger
}

func newDLQHandler(
	producer producer.Producer,
	dlqTopic string,
	tracer MessageTracer,
	log *zap.Logger,
) DLQHandler {
	return &dlqHandler{
		producer: producer,
		dlqTopic: dlqTopic,
		tracer:   tracer,
		log:      log,
	}
}

func (h *dlqHandler) SendToDLQ(ctx context.Context, message *kafka.Message, processingErr error) {
	// Створюємо span для операції відправки в DLQ
	ctx, span := h.tracer.StartDLQSpan(ctx, message, h.dlqTopic)
	defer span.End()

	// Створюємо DLQ повідомлення з оригінальними даними
	dlqMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &h.dlqTopic,
			Partition: kafka.PartitionAny,
		},
		Key:   message.Key,
		Value: message.Value,
		Headers: append(message.Headers,
			kafka.Header{Key: "dlq.original.topic", Value: []byte(*message.TopicPartition.Topic)},
			kafka.Header{Key: "dlq.original.partition", Value: []byte(fmt.Sprintf("%d", message.TopicPartition.Partition))},
			kafka.Header{Key: "dlq.original.offset", Value: []byte(fmt.Sprintf("%d", message.TopicPartition.Offset))},
			kafka.Header{Key: "dlq.error", Value: []byte(processingErr.Error())},
			kafka.Header{Key: "dlq.timestamp", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
		),
	}

	// Додаємо оновлений trace context в headers DLQ повідомлення
	h.tracer.InjectContext(ctx, dlqMessage)

	// Відправляємо в DLQ синхронно
	deliveryChan := make(chan kafka.Event, 1)
	err := h.producer.Produce(dlqMessage, deliveryChan)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send message to DLQ")
		h.log.Error("failed to send message to DLQ",
			zap.String("dlq_topic", h.dlqTopic),
			zap.String("key", string(message.Key)),
			zap.Error(err))
		return
	}

	// Чекаємо на delivery report
	e := <-deliveryChan
	m, ok := e.(*kafka.Message)
	if !ok {
		h.log.Error("unexpected event type from delivery channel")
		close(deliveryChan)
		return
	}
	if m.TopicPartition.Error != nil {
		span.RecordError(m.TopicPartition.Error)
		span.SetStatus(codes.Error, "failed to deliver message to DLQ")
		h.log.Error("failed to deliver message to DLQ",
			zap.String("dlq_topic", h.dlqTopic),
			zap.String("key", string(message.Key)),
			zap.Error(m.TopicPartition.Error))
	} else {
		span.SetStatus(codes.Ok, "message sent to DLQ")
		h.log.Info("message sent to DLQ",
			zap.String("dlq_topic", h.dlqTopic),
			zap.String("key", string(message.Key)),
			zap.Int32("original_partition", message.TopicPartition.Partition),
			zap.Int64("original_offset", int64(message.TopicPartition.Offset)))
	}
	close(deliveryChan)
}

func newNoopDLQHandler(log *zap.Logger) DLQHandler {
	return &noopDLQHandler{
		log: log,
	}
}

// noopDLQHandler - реалізація DLQHandler для випадків коли DLQ не налаштований.
type noopDLQHandler struct {
	log *zap.Logger
}

func (h *noopDLQHandler) SendToDLQ(ctx context.Context, message *kafka.Message, processingErr error) {
	h.log.Warn("DLQ producer not configured, cannot send message to DLQ",
		zap.String("key", string(message.Key)),
		zap.Int32("partition", message.TopicPartition.Partition),
		zap.Int64("offset", int64(message.TopicPartition.Offset)))
}
