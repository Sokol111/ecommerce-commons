package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/kafka/producer"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// DLQHandler відповідає за відправку повідомлень, які не вдалось обробити, в Dead Letter Queue.
type DLQHandler interface {
	// SendToDLQ відправляє повідомлення в DLQ з інформацією про помилку
	SendToDLQ(ctx context.Context, record *kgo.Record, processingErr error)
}

type DlqHandler struct {
	producer producer.Producer
	dlqTopic string
	tracer   MessageTracer
	log      *zap.Logger
}

func NewDLQHandler(
	producer producer.Producer,
	dlqTopic string,
	tracer MessageTracer,
	log *zap.Logger,
) DLQHandler {
	return &DlqHandler{
		producer: producer,
		dlqTopic: dlqTopic,
		tracer:   tracer,
		log:      log,
	}
}

func (h *DlqHandler) SendToDLQ(ctx context.Context, record *kgo.Record, processingErr error) {
	// Створюємо span для операції відправки в DLQ
	ctx, span := h.tracer.StartDLQSpan(ctx, record, h.dlqTopic)
	defer span.End()

	// Створюємо DLQ повідомлення з оригінальними даними
	dlqHeaders := make([]kgo.RecordHeader, len(record.Headers), len(record.Headers)+5)
	copy(dlqHeaders, record.Headers)
	dlqHeaders = append(dlqHeaders,
		kgo.RecordHeader{Key: "dlq.original.topic", Value: []byte(record.Topic)},
		kgo.RecordHeader{Key: "dlq.original.partition", Value: []byte(fmt.Sprintf("%d", record.Partition))},
		kgo.RecordHeader{Key: "dlq.original.offset", Value: []byte(fmt.Sprintf("%d", record.Offset))},
		kgo.RecordHeader{Key: "dlq.error", Value: []byte(processingErr.Error())},
		kgo.RecordHeader{Key: "dlq.timestamp", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
	)

	dlqRecord := &kgo.Record{
		Topic:   h.dlqTopic,
		Key:     record.Key,
		Value:   record.Value,
		Headers: dlqHeaders,
	}

	// Додаємо оновлений trace context в headers DLQ повідомлення
	h.tracer.InjectContext(ctx, dlqRecord)

	// Відправляємо в DLQ синхронно
	errCh := make(chan error, 1)
	h.producer.Produce(ctx, dlqRecord, func(r *kgo.Record, err error) {
		errCh <- err
	})

	if err := <-errCh; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deliver message to DLQ")
		h.log.Error("failed to deliver message to DLQ",
			zap.String("dlq_topic", h.dlqTopic),
			zap.String("key", string(record.Key)),
			zap.Error(err))
	} else {
		span.SetStatus(codes.Ok, "message sent to DLQ")
		h.log.Info("message sent to DLQ",
			zap.String("dlq_topic", h.dlqTopic),
			zap.String("key", string(record.Key)),
			zap.Int32("original_partition", record.Partition),
			zap.Int64("original_offset", record.Offset))
	}
}

func NewNoopDLQHandler(log *zap.Logger) DLQHandler {
	return &noopDLQHandler{
		log: log,
	}
}

// noopDLQHandler - реалізація DLQHandler для випадків коли DLQ не налаштований.
type noopDLQHandler struct {
	log *zap.Logger
}

func (h *noopDLQHandler) SendToDLQ(ctx context.Context, record *kgo.Record, processingErr error) {
	h.log.Warn("DLQ producer not configured, cannot send message to DLQ",
		zap.String("key", string(record.Key)),
		zap.Int32("partition", record.Partition),
		zap.Int64("offset", record.Offset))
}
