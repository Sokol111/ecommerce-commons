package middleware

import (
	"context"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ogen-go/ogen/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPBulkheadMiddleware(t *testing.T) {
	t.Run("allows requests when under limit", func(t *testing.T) {
		mw := newHTTPBulkheadMiddleware(2, 100*time.Millisecond)

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

	t.Run("returns ErrBulkheadFull when limit exceeded", func(t *testing.T) {
		maxConcurrent := 2
		mw := newHTTPBulkheadMiddleware(maxConcurrent, 50*time.Millisecond)

		// Channel to control when handlers complete
		blockChan := make(chan struct{})
		var activeCount atomic.Int32

		// Slow handler that blocks until signaled
		slowNext := func(_ middleware.Request) (middleware.Response, error) {
			activeCount.Add(1)
			<-blockChan
			activeCount.Add(-1)
			return middleware.Response{}, nil
		}

		var wg sync.WaitGroup

		// Start maxConcurrent requests that will block
		for i := 0; i < maxConcurrent; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := middleware.Request{
					Context: context.Background(),
					Raw:     httptest.NewRequest("GET", "/test", nil),
				}
				_, _ = mw(req, slowNext)
			}()
		}

		// Wait for all slots to be occupied
		time.Sleep(20 * time.Millisecond)
		require.Equal(t, int32(maxConcurrent), activeCount.Load())

		// Try one more request - should fail with ErrBulkheadFull
		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}
		_, err := mw(req, slowNext)

		assert.ErrorIs(t, err, ErrBulkheadFull)

		// Cleanup: unblock all waiting goroutines
		close(blockChan)
		wg.Wait()
	})

	t.Run("returns ErrBulkheadFull on acquire timeout", func(t *testing.T) {
		mw := newHTTPBulkheadMiddleware(1, 10*time.Millisecond)

		blockChan := make(chan struct{})

		slowNext := func(_ middleware.Request) (middleware.Response, error) {
			<-blockChan
			return middleware.Response{}, nil
		}

		// Start first request that blocks
		go func() {
			req := middleware.Request{
				Context: context.Background(),
				Raw:     httptest.NewRequest("GET", "/test", nil),
			}
			_, _ = mw(req, slowNext)
		}()

		// Wait for first request to acquire semaphore
		time.Sleep(5 * time.Millisecond)

		// Second request should timeout waiting for semaphore
		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}
		_, err := mw(req, slowNext)

		assert.ErrorIs(t, err, ErrBulkheadFull)

		// Cleanup
		close(blockChan)
	})

	t.Run("releases semaphore after request completes", func(t *testing.T) {
		mw := newHTTPBulkheadMiddleware(1, 100*time.Millisecond)

		next := func(_ middleware.Request) (middleware.Response, error) {
			return middleware.Response{}, nil
		}

		req := middleware.Request{
			Context: context.Background(),
			Raw:     httptest.NewRequest("GET", "/test", nil),
		}

		// First request
		_, err := mw(req, next)
		assert.NoError(t, err)

		// Second request should also succeed (semaphore released)
		_, err = mw(req, next)
		assert.NoError(t, err)
	})

	t.Run("propagates error from next handler", func(t *testing.T) {
		mw := newHTTPBulkheadMiddleware(2, 100*time.Millisecond)

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
}
