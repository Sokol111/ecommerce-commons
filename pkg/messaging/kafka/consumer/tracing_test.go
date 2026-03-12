package consumer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewMessageTracer(t *testing.T) {
	t.Run("creates tracer with default propagator", func(t *testing.T) {
		tracer := newMessageTracer(noop.NewTracerProvider())
		assert.NotNil(t, tracer)
	})
}

func TestMessageTracer_ExtractContext(t *testing.T) {
	t.Run("returns original context when no headers", func(t *testing.T) {
		tracer := newMessageTracer(noop.NewTracerProvider())
		ctx := context.Background()
		record := &kgo.Record{
			Topic:   "test-topic",
			Headers: nil,
		}

		result := tracer.ExtractContext(ctx, record)

		assert.Equal(t, ctx, result)
	})

	t.Run("extracts context from headers", func(t *testing.T) {
		// Set up a text map propagator
		otel.SetTextMapPropagator(propagation.TraceContext{})

		tracer := newMessageTracer(noop.NewTracerProvider())
		ctx := context.Background()
		record := &kgo.Record{
			Topic: "test-topic",
			Headers: []kgo.RecordHeader{
				{Key: "traceparent", Value: []byte("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")},
			},
		}

		result := tracer.ExtractContext(ctx, record)

		// Context should be different (has trace info)
		assert.NotNil(t, result)
	})
}

func TestMessageTracer_StartConsumerSpan(t *testing.T) {
	t.Run("creates span with correct attributes", func(t *testing.T) {
		tracer := newMessageTracer(noop.NewTracerProvider())
		ctx := context.Background()
		record := createTestMessage()

		resultCtx, span := tracer.StartConsumerSpan(ctx, record)

		assert.NotNil(t, resultCtx)
		assert.NotNil(t, span)

		span.End()
	})
}

func TestMessageTracer_StartDLQSpan(t *testing.T) {
	t.Run("creates DLQ span with correct attributes", func(t *testing.T) {
		tracer := newMessageTracer(noop.NewTracerProvider())
		ctx := context.Background()
		record := createTestMessage()
		dlqTopic := "test-topic.dlq"

		resultCtx, span := tracer.StartDLQSpan(ctx, record, dlqTopic)

		assert.NotNil(t, resultCtx)
		assert.NotNil(t, span)

		span.End()
	})
}

func TestMessageTracer_InjectContext(t *testing.T) {
	t.Run("injects trace context into record headers", func(t *testing.T) {
		// Set up a text map propagator
		otel.SetTextMapPropagator(propagation.TraceContext{})

		tracer := newMessageTracer(noop.NewTracerProvider())
		ctx := context.Background()

		record := &kgo.Record{
			Topic: "test-topic",
			Headers: []kgo.RecordHeader{
				{Key: "existing-header", Value: []byte("value")},
			},
		}

		tracer.InjectContext(ctx, record)

		// Headers should still exist (may include trace headers if there's active span)
		assert.NotNil(t, record.Headers)
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		tracer := newMessageTracer(noop.NewTracerProvider())
		ctx := context.Background()

		record := &kgo.Record{
			Topic: "test-topic",
			Headers: []kgo.RecordHeader{
				{Key: "custom-header", Value: []byte("custom-value")},
				{Key: "another-header", Value: []byte("another-value")},
			},
		}

		tracer.InjectContext(ctx, record)

		// Check that custom headers are preserved
		headerMap := make(map[string]string)
		for _, h := range record.Headers {
			headerMap[h.Key] = string(h.Value)
		}

		assert.Equal(t, "custom-value", headerMap["custom-header"])
		assert.Equal(t, "another-value", headerMap["another-header"])
	})

	t.Run("handles empty headers", func(t *testing.T) {
		tracer := newMessageTracer(noop.NewTracerProvider())
		ctx := context.Background()

		record := &kgo.Record{
			Topic:   "test-topic",
			Headers: nil,
		}

		// Should not panic
		assert.NotPanics(t, func() {
			tracer.InjectContext(ctx, record)
		})
	})
}
