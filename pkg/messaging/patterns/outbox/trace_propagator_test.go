package outbox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOtelTracePropagator_SaveTraceContext(t *testing.T) {
	t.Run("creates headers map when nil", func(t *testing.T) {
		propagator := newTracePropagator()

		result := propagator.SaveTraceContext(context.Background(), nil)

		assert.NotNil(t, result)
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		propagator := newTracePropagator()
		existingHeaders := map[string]string{
			"custom-header": "custom-value",
		}

		result := propagator.SaveTraceContext(context.Background(), existingHeaders)

		assert.Equal(t, "custom-value", result["custom-header"])
	})

	t.Run("modifies original headers map", func(t *testing.T) {
		propagator := newTracePropagator()
		headers := map[string]string{
			"existing": "value",
		}

		result := propagator.SaveTraceContext(context.Background(), headers)

		// Result should be the same map (modified in place)
		assert.Equal(t, headers, result)
	})
}

func TestOtelTracePropagator_StartKafkaProducerSpan(t *testing.T) {
	t.Run("returns context and span", func(t *testing.T) {
		propagator := newTracePropagator()
		headers := map[string]string{}

		ctx, span, kafkaHeaders := propagator.StartKafkaProducerSpan(headers, "test-topic", "message-123")

		assert.NotNil(t, ctx)
		assert.NotNil(t, span)
		assert.NotNil(t, kafkaHeaders)

		// Clean up span
		span.End()
	})

	t.Run("handles nil headers", func(t *testing.T) {
		propagator := newTracePropagator()

		ctx, span, kafkaHeaders := propagator.StartKafkaProducerSpan(nil, "test-topic", "message-123")

		assert.NotNil(t, ctx)
		assert.NotNil(t, span)
		assert.NotNil(t, kafkaHeaders)

		span.End()
	})

	t.Run("converts headers to kafka headers", func(t *testing.T) {
		propagator := newTracePropagator()
		headers := map[string]string{
			"header1": "value1",
			"header2": "value2",
		}

		_, span, kafkaHeaders := propagator.StartKafkaProducerSpan(headers, "test-topic", "message-123")
		defer span.End()

		// Kafka headers should contain at least the original headers
		headerMap := make(map[string]string)
		for _, h := range kafkaHeaders {
			headerMap[h.Key] = string(h.Value)
		}

		assert.Contains(t, headerMap, "header1")
		assert.Contains(t, headerMap, "header2")
	})

	t.Run("does not modify original headers", func(t *testing.T) {
		propagator := newTracePropagator()
		originalHeaders := map[string]string{
			"original": "value",
		}
		originalLen := len(originalHeaders)

		_, span, _ := propagator.StartKafkaProducerSpan(originalHeaders, "test-topic", "message-123")
		defer span.End()

		// Original headers should not be modified
		assert.Equal(t, originalLen, len(originalHeaders))
	})
}

func TestNewTracePropagator(t *testing.T) {
	t.Run("creates propagator", func(t *testing.T) {
		propagator := newTracePropagator()

		require.NotNil(t, propagator)

		// Verify it implements the interface
		var _ tracePropagator = propagator
	})
}
