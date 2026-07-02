package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRouter_Process(t *testing.T) {
	t.Run("dispatches to correct handler", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		var calledWith *emptypb.Empty

		Register(r, func(ctx context.Context, e *emptypb.Empty) error {
			calledWith = e
			return nil
		})

		event := &emptypb.Empty{}
		err := r.Process(context.Background(), event)

		require.NoError(t, err)
		assert.NotNil(t, calledWith)
	})

	t.Run("dispatches multiple event types", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		var aCalled, bCalled bool

		Register(r, func(ctx context.Context, e *emptypb.Empty) error {
			aCalled = true
			return nil
		})
		Register(r, func(ctx context.Context, e *wrapperspb.StringValue) error {
			bCalled = true
			return nil
		})

		err := r.Process(context.Background(), &emptypb.Empty{})
		require.NoError(t, err)
		assert.True(t, aCalled)
		assert.False(t, bCalled)

		err = r.Process(context.Background(), &wrapperspb.StringValue{})
		require.NoError(t, err)
		assert.True(t, bCalled)
	})

	t.Run("returns ErrSkipMessage for unregistered event type", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		Register(r, func(ctx context.Context, e *emptypb.Empty) error {
			return nil
		})

		err := r.Process(context.Background(), &wrapperspb.StringValue{})
		assert.True(t, errors.Is(err, ErrSkipMessage))
	})

	t.Run("propagates handler error", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		expectedErr := errors.New("processing failed")

		Register(r, func(ctx context.Context, e *emptypb.Empty) error {
			return expectedErr
		})

		err := r.Process(context.Background(), &emptypb.Empty{})
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("passes context to handler", func(t *testing.T) {
		r := NewRouter(zap.NewNop())
		type ctxKey struct{}
		ctx := context.WithValue(context.Background(), ctxKey{}, "value")

		Register(r, func(ctx context.Context, e *emptypb.Empty) error {
			assert.Equal(t, "value", ctx.Value(ctxKey{}))
			return nil
		})

		err := r.Process(ctx, &emptypb.Empty{})
		require.NoError(t, err)
	})
}
