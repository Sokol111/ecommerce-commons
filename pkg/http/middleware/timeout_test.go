package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ogen-go/ogen/middleware"
	"github.com/stretchr/testify/assert"
)

func TestNewTimeoutMiddleware(t *testing.T) {
	t.Run("passes request through when no timeout", func(t *testing.T) {
		mw := newTimeoutMiddleware(100 * time.Millisecond)

		nextCalled := false
		next := func(_ middleware.Request) (middleware.Response, error) {
			nextCalled = true
			return middleware.Response{}, nil
		}

		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.NoError(t, err)
		assert.True(t, nextCalled)
	})

	t.Run("returns ErrRequestTimeout when handler exceeds timeout", func(t *testing.T) {
		mw := newTimeoutMiddleware(10 * time.Millisecond)

		next := func(req middleware.Request) (middleware.Response, error) {
			// Simulate slow handler
			select {
			case <-time.After(50 * time.Millisecond):
				return middleware.Response{}, nil
			case <-req.Context.Done():
				return middleware.Response{}, req.Context.Err()
			}
		}

		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.ErrorIs(t, err, ErrRequestTimeout)
	})

	t.Run("sets context with timeout on request", func(t *testing.T) {
		timeout := 100 * time.Millisecond
		mw := newTimeoutMiddleware(timeout)

		var capturedDeadline time.Time
		var hasDeadline bool

		next := func(req middleware.Request) (middleware.Response, error) {
			capturedDeadline, hasDeadline = req.Context.Deadline()
			return middleware.Response{}, nil
		}

		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.NoError(t, err)
		assert.True(t, hasDeadline)
		// Deadline should be approximately timeout from now (with some tolerance)
		assert.WithinDuration(t, time.Now().Add(timeout), capturedDeadline, 50*time.Millisecond)
	})

	t.Run("propagates error from next handler", func(t *testing.T) {
		mw := newTimeoutMiddleware(100 * time.Millisecond)

		expectedErr := errors.New("handler error")
		next := func(_ middleware.Request) (middleware.Response, error) {
			return middleware.Response{}, expectedErr
		}

		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		_, err := mw(req, next)

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("returns response from next handler", func(t *testing.T) {
		mw := newTimeoutMiddleware(100 * time.Millisecond)

		expectedResp := middleware.Response{Type: "test-type"}
		next := func(_ middleware.Request) (middleware.Response, error) {
			return expectedResp, nil
		}

		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		resp, err := mw(req, next)

		assert.NoError(t, err)
		assert.Equal(t, expectedResp.Type, resp.Type)
	})
}
