package consumer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
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

// mockOffsetMarker is a test implementation of offsetMarker
type mockOffsetMarker struct {
	markedRecords []*kgo.Record
}

func (m *mockOffsetMarker) MarkCommitRecords(records ...*kgo.Record) {
	m.markedRecords = append(m.markedRecords, records...)
}

func TestNewResultHandler(t *testing.T) {
	t.Run("creates result handler with correct fields", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		marker := &mockOffsetMarker{}

		rh := &resultHandler{log: log, dlqHandler: dlqHandler, offsetMarker: marker}

		assert.NotNil(t, rh)
		assert.Equal(t, log, rh.log)
		assert.Equal(t, dlqHandler, rh.dlqHandler)
		assert.Equal(t, marker, rh.offsetMarker)
	})
}

func TestResultHandler_Handle(t *testing.T) {
	t.Run("handles successful processing", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		marker := &mockOffsetMarker{}
		rh := &resultHandler{log: log, dlqHandler: dlqHandler, offsetMarker: marker}

		span := newMockSpan()
		record := createTestMessage()

		rh.handle(context.Background(), nil, record, span)

		assert.Equal(t, codes.Ok, span.statusCode)
		assert.Equal(t, "message processed successfully", span.statusMessage)
		assert.Len(t, marker.markedRecords, 1)
		assert.Equal(t, record, marker.markedRecords[0])
		assert.Equal(t, int32(0), dlqHandler.callCount.Load())
	})

	t.Run("handles ErrSkipMessage", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		marker := &mockOffsetMarker{}
		rh := &resultHandler{log: log, dlqHandler: dlqHandler, offsetMarker: marker}

		span := newMockSpan()
		record := createTestMessage()

		rh.handle(context.Background(), ErrSkipMessage, record, span)

		assert.Equal(t, codes.Ok, span.statusCode)
		assert.Equal(t, "message skipped", span.statusMessage)
		assert.Len(t, marker.markedRecords, 1)
		assert.Equal(t, int32(0), dlqHandler.callCount.Load())
	})

	t.Run("handles ErrPermanent - sends to DLQ", func(t *testing.T) {
		log := zap.NewNop()
		dlqRecords := make(chan *kgo.Record, 1)
		dlqHandler := &mockDLQHandler{
			sendToDLQFunc: func(ctx context.Context, record *kgo.Record, processingErr error) {
				dlqRecords <- record
			},
		}
		marker := &mockOffsetMarker{}
		rh := &resultHandler{log: log, dlqHandler: dlqHandler, offsetMarker: marker}

		span := newMockSpan()
		record := createTestMessage()

		rh.handle(context.Background(), ErrPermanent, record, span)

		assert.Equal(t, codes.Error, span.statusCode)
		assert.Equal(t, "permanent error - sending to DLQ", span.statusMessage)
		assert.Len(t, marker.markedRecords, 1)
		assert.Equal(t, int32(1), dlqHandler.callCount.Load())

		select {
		case sentRecord := <-dlqRecords:
			assert.Equal(t, record, sentRecord)
		default:
			t.Fatal("record was not sent to DLQ")
		}
	})

	t.Run("handles retry exhausted error - sends to DLQ", func(t *testing.T) {
		log := zap.NewNop()
		dlqRecords := make(chan *kgo.Record, 1)
		dlqHandler := &mockDLQHandler{
			sendToDLQFunc: func(ctx context.Context, record *kgo.Record, processingErr error) {
				dlqRecords <- record
			},
		}
		marker := &mockOffsetMarker{}
		rh := &resultHandler{log: log, dlqHandler: dlqHandler, offsetMarker: marker}

		span := newMockSpan()
		record := createTestMessage()
		someError := assert.AnError

		rh.handle(context.Background(), someError, record, span)

		assert.Equal(t, codes.Error, span.statusCode)
		assert.Equal(t, "message processing failed - sending to DLQ", span.statusMessage)
		assert.Equal(t, someError, span.recordedError)
		assert.Len(t, marker.markedRecords, 1)
		assert.Equal(t, int32(1), dlqHandler.callCount.Load())

		select {
		case sentRecord := <-dlqRecords:
			assert.Equal(t, record, sentRecord)
		default:
			t.Fatal("record was not sent to DLQ")
		}
	})
}

func TestResultHandler_RecordFields(t *testing.T) {
	t.Run("returns correct record fields", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		marker := &mockOffsetMarker{}
		rh := &resultHandler{log: log, dlqHandler: dlqHandler, offsetMarker: marker}
		record := createTestMessage()

		fields := rh.recordFields(record)

		assert.Len(t, fields, 3)
	})

	t.Run("returns record fields with error", func(t *testing.T) {
		log := zap.NewNop()
		dlqHandler := &mockDLQHandler{}
		marker := &mockOffsetMarker{}
		rh := &resultHandler{log: log, dlqHandler: dlqHandler, offsetMarker: marker}
		record := createTestMessage()

		fields := rh.recordFieldsWithError(record, assert.AnError)

		assert.Len(t, fields, 4) // 3 record fields + 1 error field
	})
}
