package consumer

import (
	"context"
	"errors"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// processingStrategy represents the outcome of message processing
type processingStrategy interface {
	handle(ctx context.Context, message *kafka.Message, span trace.Span)
}

// resultHandler manages processing strategies and their dependencies
type resultHandler struct {
	log        *zap.Logger
	dlqHandler DLQHandler
	consumer   *kafka.Consumer
}

func newResultHandler(
	log *zap.Logger,
	dlqHandler DLQHandler,
	consumer *kafka.Consumer,
) *resultHandler {
	return &resultHandler{
		log:        log,
		dlqHandler: dlqHandler,
		consumer:   consumer,
	}
}

// classifyAndHandle analyzes the error and executes appropriate strategy
func (h *resultHandler) classifyAndHandle(ctx context.Context, err error, message *kafka.Message, span trace.Span) {
	var strategy processingStrategy

	switch {
	case err == nil:
		strategy = &successStrategy{handler: h}
	case errors.Is(err, ErrSkipMessage):
		strategy = &skipStrategy{handler: h}
	case errors.Is(err, ErrPermanent):
		strategy = &permanentErrorStrategy{handler: h, err: err}
	default:
		// Retry exhausted or context cancelled
		strategy = &retryExhaustedStrategy{handler: h, err: err}
	}

	strategy.handle(ctx, message, span)
}

func (h *resultHandler) storeOffset(message *kafka.Message) {
	_, err := h.consumer.StoreMessage(message)
	if err != nil {
		h.log.Error("failed to store offset",
			zap.String("key", string(message.Key)),
			zap.Int32("partition", message.TopicPartition.Partition),
			zap.Int32("offset", int32(message.TopicPartition.Offset)),
			zap.Error(err))
	}
}

// successStrategy - message processed successfully
type successStrategy struct {
	handler *resultHandler
}

func (s *successStrategy) handle(ctx context.Context, message *kafka.Message, span trace.Span) {
	span.SetStatus(codes.Ok, "message processed successfully")
	s.handler.storeOffset(message)
}

// skipStrategy - message should be skipped
type skipStrategy struct {
	handler *resultHandler
}

func (s *skipStrategy) handle(ctx context.Context, message *kafka.Message, span trace.Span) {
	span.SetStatus(codes.Ok, "message skipped")
	s.handler.log.Info("skipping message",
		zap.String("key", string(message.Key)),
		zap.Int32("partition", message.TopicPartition.Partition),
		zap.Int32("offset", int32(message.TopicPartition.Offset)))
	s.handler.storeOffset(message)
}

// permanentErrorStrategy - permanent error, send to DLQ
type permanentErrorStrategy struct {
	handler *resultHandler
	err     error
}

func (s *permanentErrorStrategy) handle(ctx context.Context, message *kafka.Message, span trace.Span) {
	span.SetStatus(codes.Error, "permanent error - sending to DLQ")
	s.handler.log.Error("permanent error - sending message to DLQ",
		zap.String("key", string(message.Key)),
		zap.Int32("partition", message.TopicPartition.Partition),
		zap.Int32("offset", int32(message.TopicPartition.Offset)),
		zap.Error(s.err))
	s.handler.dlqHandler.SendToDLQ(ctx, message, s.err)
	s.handler.storeOffset(message)
}

// retryExhaustedStrategy - retries exhausted, send to DLQ
type retryExhaustedStrategy struct {
	handler *resultHandler
	err     error
}

func (s *retryExhaustedStrategy) handle(ctx context.Context, message *kafka.Message, span trace.Span) {
	span.RecordError(s.err)
	span.SetStatus(codes.Error, "message processing failed - sending to DLQ")
	s.handler.log.Error("message processing failed after retries - sending to DLQ",
		zap.String("key", string(message.Key)),
		zap.Int32("partition", message.TopicPartition.Partition),
		zap.Int32("offset", int32(message.TopicPartition.Offset)),
		zap.Error(s.err))
	s.handler.dlqHandler.SendToDLQ(ctx, message, s.err)
	s.handler.storeOffset(message)
}
