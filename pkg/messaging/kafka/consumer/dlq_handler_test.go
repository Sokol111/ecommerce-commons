package consumer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewNoopDLQHandler(t *testing.T) {
	t.Run("creates noop handler", func(t *testing.T) {
		log := zap.NewNop()
		handler := newNoopDLQHandler(log)

		assert.NotNil(t, handler)
	})
}

func TestNoopDLQHandler_SendToDLQ(t *testing.T) {
	t.Run("logs warning but does not panic", func(t *testing.T) {
		log := zap.NewNop()
		handler := newNoopDLQHandler(log)

		msg := createTestMessage()

		// Should not panic
		assert.NotPanics(t, func() {
			handler.SendToDLQ(context.Background(), msg, assert.AnError)
		})
	})
}
