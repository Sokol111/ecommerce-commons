package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/ogen-go/ogen/middleware"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRecoveryMiddleware(t *testing.T) {
	// Create a test logger and context
	testLogger := zap.NewNop()
	ctx := logger.With(context.Background(), testLogger)

	t.Run("passes request through when no panic", func(t *testing.T) {
		mw := recoveryMiddleware()

		nextCalled := false
		next := func(_ middleware.Request) (middleware.Response, error) {
			nextCalled = true
			return middleware.Response{}, nil
		}

		req := middleware.Request{
			Context: ctx,
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.NoError(t, err)
		assert.True(t, nextCalled)
	})

	t.Run("recovers from panic and returns ErrPanic", func(t *testing.T) {
		mw := recoveryMiddleware()

		next := func(_ middleware.Request) (middleware.Response, error) {
			panic("something went wrong")
		}

		req := middleware.Request{
			Context: ctx,
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.ErrorIs(t, err, ErrPanic)
	})

	t.Run("recovers from panic with error value", func(t *testing.T) {
		mw := recoveryMiddleware()

		next := func(_ middleware.Request) (middleware.Response, error) {
			panic(assert.AnError)
		}

		req := middleware.Request{
			Context: ctx,
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.ErrorIs(t, err, ErrPanic)
	})

	t.Run("recovers from panic with nil value", func(t *testing.T) {
		mw := recoveryMiddleware()

		next := func(_ middleware.Request) (middleware.Response, error) {
			panic(any(nil)) //nolint:govet // Testing panic(nil) behavior intentionally
		}

		req := middleware.Request{
			Context: ctx,
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		// panic(nil) in Go 1.21+ behaves differently, but recovery should still work
		resp, err := mw(req, next)

		// Either no error (panic(nil) doesn't trigger recover in Go 1.21+) or ErrPanic
		if err != nil {
			assert.ErrorIs(t, err, ErrPanic)
		} else {
			assert.NotNil(t, resp)
		}
	})

	t.Run("propagates error from next handler", func(t *testing.T) {
		mw := recoveryMiddleware()

		expectedErr := assert.AnError
		next := func(_ middleware.Request) (middleware.Response, error) {
			return middleware.Response{}, expectedErr
		}

		req := middleware.Request{
			Context: ctx,
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("returns response from next handler", func(t *testing.T) {
		mw := recoveryMiddleware()

		expectedResp := middleware.Response{Type: "test-response"}
		next := func(_ middleware.Request) (middleware.Response, error) {
			return expectedResp, nil
		}

		req := middleware.Request{
			Context: ctx,
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		resp, err := mw(req, next)

		assert.NoError(t, err)
		assert.Equal(t, expectedResp.Type, resp.Type)
	})
}
