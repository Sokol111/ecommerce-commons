package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestGetTraceID(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string for invalid span context", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", GetTraceID(context.Background()))
	})

	t.Run("returns trace ID from valid span context", func(t *testing.T) {
		t.Parallel()
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:  [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Remote:  true,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sc)

		assert.Equal(t, sc.TraceID().String(), GetTraceID(ctx))
	})
}

func TestGetSpanID(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string for invalid span context", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", GetSpanID(context.Background()))
	})

	t.Run("returns span ID from valid span context", func(t *testing.T) {
		t.Parallel()
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:  [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Remote:  true,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sc)

		assert.Equal(t, sc.SpanID().String(), GetSpanID(ctx))
	})
}

func TestGetTraceIDAndSpanID(t *testing.T) {
	t.Parallel()

	t.Run("returns empty values for invalid span context", func(t *testing.T) {
		t.Parallel()
		traceID, spanID := GetTraceIDAndSpanID(context.Background())
		assert.Equal(t, "", traceID)
		assert.Equal(t, "", spanID)
	})

	t.Run("returns both IDs from valid span context", func(t *testing.T) {
		t.Parallel()
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:  [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Remote:  true,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sc)

		traceID, spanID := GetTraceIDAndSpanID(ctx)
		assert.Equal(t, sc.TraceID().String(), traceID)
		assert.Equal(t, sc.SpanID().String(), spanID)
	})
}

func TestAddAttribute(t *testing.T) {
	t.Parallel()

	t.Run("does not panic without span", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		assert.NotPanics(t, func() {
			AddAttribute(ctx, "key", "value")
		})
	})
}
