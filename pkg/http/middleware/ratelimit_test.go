package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/ogen-go/ogen/middleware"
	"github.com/stretchr/testify/assert"
)

// mockRateLimiter is a test double for rateLimiter interface.
type mockRateLimiter struct {
	allowResult bool
}

func (m *mockRateLimiter) Allow() bool {
	return m.allowResult
}

func TestNewRateLimitMiddleware(t *testing.T) {
	t.Run("allows request when limiter allows", func(t *testing.T) {
		limiter := &mockRateLimiter{allowResult: true}
		mw := newRateLimitMiddleware(limiter)

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

	t.Run("returns ErrRateLimitExceeded when limiter denies", func(t *testing.T) {
		limiter := &mockRateLimiter{allowResult: false}
		mw := newRateLimitMiddleware(limiter)

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

		assert.ErrorIs(t, err, ErrRateLimitExceeded)
		assert.False(t, nextCalled, "next handler should not be called when rate limited")
	})

	t.Run("propagates error from next handler", func(t *testing.T) {
		limiter := &mockRateLimiter{allowResult: true}
		mw := newRateLimitMiddleware(limiter)

		expectedErr := assert.AnError
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
		limiter := &mockRateLimiter{allowResult: true}
		mw := newRateLimitMiddleware(limiter)

		expectedResp := middleware.Response{Type: "test-response"}
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
