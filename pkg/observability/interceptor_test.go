package observability

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeRequest struct{}

type fakeResponse struct{}

func TestTraceContextUnaryInterceptor(t *testing.T) {
	t.Parallel()

	t.Run("decorates logger with trace IDs", func(t *testing.T) {
		t.Parallel()

		observed, coreLogs := observer.New(zap.InfoLevel)
		baseLogger := zap.New(observed)
		baseCtx := logger.With(context.Background(), baseLogger)

		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:  [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Remote:  true,
		})
		ctx := trace.ContextWithSpanContext(baseCtx, sc)

		called := false
		next := func(c context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			called = true
			log := logger.Get(c)
			log.Info("inside handler")
			return &connect.Response[fakeResponse]{}, nil
		}

		interceptor := TraceContextUnaryInterceptor(next)
		_, err := interceptor(ctx, connect.NewRequest(&fakeRequest{}))

		require.NoError(t, err)
		require.True(t, called)
		require.Len(t, coreLogs.All(), 1)

		entry := coreLogs.All()[0]
		traceValue := entry.ContextMap()["trace_id"]
		assert.Equal(t, sc.TraceID().String(), traceValue)
	})

	t.Run("does not modify context without trace ID", func(t *testing.T) {
		t.Parallel()

		baseLogger := zap.NewNop()
		baseCtx := logger.With(context.Background(), baseLogger)

		called := false
		next := func(c context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			called = true
			assert.Equal(t, baseLogger, logger.Get(c))
			return &connect.Response[fakeResponse]{}, nil
		}

		interceptor := TraceContextUnaryInterceptor(next)
		_, err := interceptor(baseCtx, connect.NewRequest(&fakeResponse{}))

		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("propagates errors", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("handler error")
		next := func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, wantErr
		}

		interceptor := TraceContextUnaryInterceptor(next)
		_, err := interceptor(context.Background(), connect.NewRequest(&fakeResponse{}))

		assert.ErrorIs(t, err, wantErr)
	})
}
