package producer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPollBrokers(t *testing.T) {
	t.Run("returns error on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := pollBrokers(ctx, nil)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestWaitForBrokers(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := waitForBrokers(ctx, nil, zap.NewNop(), 0, true)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("returns nil when timeout and failOnError is false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := waitForBrokers(ctx, nil, zap.NewNop(), 0, false)

		assert.NoError(t, err)
	})
}
