package consumer

import (
	"context"
	"errors"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// offsetStorer is an interface for storing message offsets
type offsetStorer interface {
	StoreMessage(m *kafka.Message) (storedOffsets []kafka.TopicPartition, err error)
}

// resultHandler handles message processing results
type resultHandler struct {
	log        *zap.Logger
	dlqHandler DLQHandler
	consumer   offsetStorer
}

func newResultHandler(
	log *zap.Logger,
	dlqHandler DLQHandler,
	consumer offsetStorer,
) *resultHandler {
	return &resultHandler{
		log:        log,
		dlqHandler: dlqHandler,
		consumer:   consumer,
	}
}

// handle processes the result of message handling and takes appropriate action
func (h *resultHandler) handle(ctx context.Context, err error, message *kafka.Message, span trace.Span) {
	defer h.storeOffset(message)

	switch {
	case err == nil:
		span.SetStatus(codes.Ok, "message processed successfully")

	case errors.Is(err, ErrSkipMessage):
		span.SetStatus(codes.Ok, "message skipped")
		h.log.Info("skipping message", h.messageFields(message)...)

	case errors.Is(err, ErrPermanent):
		span.SetStatus(codes.Error, "permanent error - sending to DLQ")
		h.log.Error("permanent error - sending message to DLQ", h.messageFieldsWithError(message, err)...)
		h.dlqHandler.SendToDLQ(ctx, message, err)

	default:
		// Retry exhausted or context cancelled
		span.RecordError(err)
		span.SetStatus(codes.Error, "message processing failed - sending to DLQ")
		h.log.Error("message processing failed after retries - sending to DLQ", h.messageFieldsWithError(message, err)...)
		h.dlqHandler.SendToDLQ(ctx, message, err)
	}
}

func (h *resultHandler) storeOffset(message *kafka.Message) {
	if _, err := h.consumer.StoreMessage(message); err != nil {
		h.log.Error("failed to store offset", h.messageFieldsWithError(message, err)...)
	}
}

func (h *resultHandler) messageFields(message *kafka.Message) []zap.Field {
	return []zap.Field{
		zap.String("key", string(message.Key)),
		zap.Int32("partition", message.TopicPartition.Partition),
		zap.Int64("offset", int64(message.TopicPartition.Offset)),
	}
}

func (h *resultHandler) messageFieldsWithError(message *kafka.Message, err error) []zap.Field {
	return append(h.messageFields(message), zap.Error(err))
}
