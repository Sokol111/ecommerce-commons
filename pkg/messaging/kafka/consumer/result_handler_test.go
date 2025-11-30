package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// mockSpan is a test implementation of trace.Span that records operations
type mockSpan struct {
	trace.Span
	statusCode    codes.Code
	statusMessage string
	recordedError error
}

func newMockSpan() *mockSpan {
	_, span := noop.NewTracerProvider().Tracer("test").Start(context.Background(), "test")
	return &mockSpan{
		Span: span,
	}
}

func (m *mockSpan) SetStatus(code codes.Code, description string) {
	m.statusCode = code
	m.statusMessage = description
}

func (m *mockSpan) RecordError(err error, options ...trace.EventOption) {
	m.recordedError = err
}

func (m *mockSpan) End(options ...trace.SpanEndOption) {}

// mockOffsetStorer is a test implementation of OffsetStorer
type mockOffsetStorer struct {
	storeMessageFunc func(m *kafka.Message) ([]kafka.TopicPartition, error)
	storedMessages   []*kafka.Message
}

func (m *mockOffsetStorer) StoreMessage(msg *kafka.Message) ([]kafka.TopicPartition, error) {
	m.storedMessages = append(m.storedMessages, msg)
	if m.storeMessageFunc != nil {
		return m.storeMessageFunc(msg)
	}
	return []kafka.TopicPartition{msg.TopicPartition}, nil
}

func TestNewResultHandler(t *testing.T) {
	t.Run("creates result handler with correct fields", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		consumer := &mockOffsetStorer{}

		rh := newResultHandler(log, dlqHandler, consumer)

		assert.NotNil(t, rh)
		assert.Equal(t, log, rh.log)
		assert.Equal(t, dlqHandler, rh.dlqHandler)
		assert.Equal(t, consumer, rh.consumer)
	})
}

func TestResultHandler_Handle(t *testing.T) {
	t.Run("handles successful processing", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		consumer := &mockOffsetStorer{}
		rh := newResultHandler(log, dlqHandler, consumer)

		span := newMockSpan()
		msg := createTestMessage()

		rh.handle(context.Background(), nil, msg, span)

		assert.Equal(t, codes.Ok, span.statusCode)
		assert.Equal(t, "message processed successfully", span.statusMessage)
		assert.Len(t, consumer.storedMessages, 1)
		assert.Equal(t, msg, consumer.storedMessages[0])
		assert.Equal(t, int32(0), dlqHandler.callCount.Load())
	})

	t.Run("handles ErrSkipMessage", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		consumer := &mockOffsetStorer{}
		rh := newResultHandler(log, dlqHandler, consumer)

		span := newMockSpan()
		msg := createTestMessage()

		rh.handle(context.Background(), ErrSkipMessage, msg, span)

		assert.Equal(t, codes.Ok, span.statusCode)
		assert.Equal(t, "message skipped", span.statusMessage)
		assert.Len(t, consumer.storedMessages, 1)
		assert.Equal(t, int32(0), dlqHandler.callCount.Load())
	})

	t.Run("handles ErrPermanent - sends to DLQ", func(t *testing.T) {
		log := zap.NewNop()
		dlqMessages := make(chan *kafka.Message, 1)
		dlqHandler := &mockDLQHandler{
			sendToDLQFunc: func(ctx context.Context, message *kafka.Message, processingErr error) {
				dlqMessages <- message
			},
		}
		consumer := &mockOffsetStorer{}
		rh := newResultHandler(log, dlqHandler, consumer)

		span := newMockSpan()
		msg := createTestMessage()

		rh.handle(context.Background(), ErrPermanent, msg, span)

		assert.Equal(t, codes.Error, span.statusCode)
		assert.Equal(t, "permanent error - sending to DLQ", span.statusMessage)
		assert.Len(t, consumer.storedMessages, 1)
		assert.Equal(t, int32(1), dlqHandler.callCount.Load())

		select {
		case sentMsg := <-dlqMessages:
			assert.Equal(t, msg, sentMsg)
		default:
			t.Fatal("message was not sent to DLQ")
		}
	})

	t.Run("handles retry exhausted error - sends to DLQ", func(t *testing.T) {
		log := zap.NewNop()
		dlqMessages := make(chan *kafka.Message, 1)
		dlqHandler := &mockDLQHandler{
			sendToDLQFunc: func(ctx context.Context, message *kafka.Message, processingErr error) {
				dlqMessages <- message
			},
		}
		consumer := &mockOffsetStorer{}
		rh := newResultHandler(log, dlqHandler, consumer)

		span := newMockSpan()
		msg := createTestMessage()
		someError := assert.AnError

		rh.handle(context.Background(), someError, msg, span)

		assert.Equal(t, codes.Error, span.statusCode)
		assert.Equal(t, "message processing failed - sending to DLQ", span.statusMessage)
		assert.Equal(t, someError, span.recordedError)
		assert.Len(t, consumer.storedMessages, 1)
		assert.Equal(t, int32(1), dlqHandler.callCount.Load())

		select {
		case sentMsg := <-dlqMessages:
			assert.Equal(t, msg, sentMsg)
		default:
			t.Fatal("message was not sent to DLQ")
		}
	})

	t.Run("logs error when store offset fails", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		consumer := &mockOffsetStorer{
			storeMessageFunc: func(m *kafka.Message) ([]kafka.TopicPartition, error) {
				return nil, errors.New("store failed")
			},
		}
		rh := newResultHandler(log, dlqHandler, consumer)

		span := newMockSpan()
		msg := createTestMessage()

		// Should not panic even when store fails
		rh.handle(context.Background(), nil, msg, span)

		assert.Len(t, consumer.storedMessages, 1)
	})
}

func TestResultHandler_MessageFields(t *testing.T) {
	t.Run("returns correct message fields", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		consumer := &mockOffsetStorer{}
		rh := newResultHandler(log, dlqHandler, consumer)
		msg := createTestMessage()

		fields := rh.messageFields(msg)

		assert.Len(t, fields, 3)
	})

	t.Run("returns message fields with error", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		consumer := &mockOffsetStorer{}
		rh := newResultHandler(log, dlqHandler, consumer)
		msg := createTestMessage()

		fields := rh.messageFieldsWithError(msg, assert.AnError)

		assert.Len(t, fields, 4) // 3 message fields + 1 error field
	})
}
