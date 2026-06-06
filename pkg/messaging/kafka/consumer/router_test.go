package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testEventA struct {
	ID string
}

type testEventB struct {
	Name string
}

type testEventUnregistered struct{}

func TestRouter_Process(t *testing.T) {
	t.Run("dispatches to correct handler", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		var calledWith *testEventA

		Register(r, func(ctx context.Context, e *testEventA) error {
			calledWith = e
			return nil
		})

		event := &testEventA{ID: "123"}
		err := r.Process(context.Background(), event)

		require.NoError(t, err)
		assert.Equal(t, "123", calledWith.ID)
	})

	t.Run("dispatches multiple event types", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		var aCalled, bCalled bool

		Register(r, func(ctx context.Context, e *testEventA) error {
			aCalled = true
			return nil
		})
		Register(r, func(ctx context.Context, e *testEventB) error {
			bCalled = true
			return nil
		})

		err := r.Process(context.Background(), &testEventA{})
		require.NoError(t, err)
		assert.True(t, aCalled)
		assert.False(t, bCalled)

		err = r.Process(context.Background(), &testEventB{})
		require.NoError(t, err)
		assert.True(t, bCalled)
	})

	t.Run("returns ErrSkipMessage for unregistered event type", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		Register(r, func(ctx context.Context, e *testEventA) error {
			return nil
		})

		err := r.Process(context.Background(), &testEventUnregistered{})
		assert.True(t, errors.Is(err, ErrSkipMessage))
	})

	t.Run("propagates handler error", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		expectedErr := errors.New("processing failed")

		Register(r, func(ctx context.Context, e *testEventA) error {
			return expectedErr
		})

		err := r.Process(context.Background(), &testEventA{})
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("passes context to handler", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		type ctxKey struct{}
		ctx := context.WithValue(context.Background(), ctxKey{}, "value")

		Register(r, func(ctx context.Context, e *testEventA) error {
			assert.Equal(t, "value", ctx.Value(ctxKey{}))
			return nil
		})

		err := r.Process(ctx, &testEventA{})
		require.NoError(t, err)
	})
}
