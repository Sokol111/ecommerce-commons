package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestNewHTTPBulkheadMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within concurrency limit", func(t *testing.T) {
		maxConcurrent := 5
		timeout := 100 * time.Millisecond
		logger := zap.NewNop()

		middleware := newHTTPBulkheadMiddleware(maxConcurrent, timeout, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("allows health checks without bulkhead restriction", func(t *testing.T) {
		maxConcurrent := 1
		timeout := 100 * time.Millisecond
		logger := zap.NewNop()

		middleware := newHTTPBulkheadMiddleware(maxConcurrent, timeout, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})
		router.GET("/health/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		// Test /health/live
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/live, got %d", http.StatusOK, w.Code)
		}

		// Test /health/ready
		req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/ready, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("rejects requests when concurrency limit exceeded", func(t *testing.T) {
		maxConcurrent := 2
		timeout := 50 * time.Millisecond
		logger := zap.NewNop()

		middleware := newHTTPBulkheadMiddleware(maxConcurrent, timeout, logger)

		// Channel to block handlers
		blockChan := make(chan struct{})

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			<-blockChan // Block until we release it
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		var wg sync.WaitGroup
		results := make([]int, 5)

		// Start 5 concurrent requests (limit is 2)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				results[idx] = w.Code
			}(i)
		}

		// Wait a bit for requests to attempt acquisition
		time.Sleep(100 * time.Millisecond)

		// Release the blocked handlers
		close(blockChan)
		wg.Wait()

		// Count how many requests were rejected (503)
		rejectedCount := 0
		successCount := 0
		for _, code := range results {
			if code == http.StatusServiceUnavailable {
				rejectedCount++
			} else if code == http.StatusOK {
				successCount++
			}
		}

		// We expect exactly maxConcurrent successful requests
		if successCount != maxConcurrent {
			t.Errorf("expected exactly %d successful requests, got %d", maxConcurrent, successCount)
		}

		// We expect the rest to be rejected
		if rejectedCount != 5-maxConcurrent {
			t.Errorf("expected %d rejected requests, got %d", 5-maxConcurrent, rejectedCount)
		}
	})

	t.Run("rejects request when timeout exceeded waiting for slot", func(t *testing.T) {
		maxConcurrent := 1
		timeout := 50 * time.Millisecond
		logger := zap.NewNop()

		middleware := newHTTPBulkheadMiddleware(maxConcurrent, timeout, logger)

		blockChan := make(chan struct{})

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			<-blockChan
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		var wg sync.WaitGroup
		var rejectedCode atomic.Int32

		// Start first request that will hold the slot
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}()

		// Wait a bit to ensure first request has acquired the slot
		time.Sleep(10 * time.Millisecond)

		// Start second request that should timeout
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			rejectedCode.Store(int32(w.Code))
		}()

		// Wait for timeout to occur
		time.Sleep(100 * time.Millisecond)

		// Release blocked requests
		close(blockChan)
		wg.Wait()

		if rejectedCode.Load() != http.StatusServiceUnavailable {
			t.Errorf("expected rejected request to have status %d, got %d",
				http.StatusServiceUnavailable, rejectedCode.Load())
		}
	})

	t.Run("releases semaphore after request completion", func(t *testing.T) {
		maxConcurrent := 1
		timeout := 100 * time.Millisecond
		logger := zap.NewNop()

		middleware := newHTTPBulkheadMiddleware(maxConcurrent, timeout, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// First request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("first request: expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Second request (should succeed if semaphore was released)
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("second request: expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("handles cancelled context", func(t *testing.T) {
		maxConcurrent := 1
		timeout := 100 * time.Millisecond
		logger := zap.NewNop()

		middleware := newHTTPBulkheadMiddleware(maxConcurrent, timeout, logger)

		blockChan := make(chan struct{})

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			<-blockChan
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Start first request to occupy the slot
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}()

		// Wait for first request to acquire slot
		time.Sleep(10 * time.Millisecond)

		// Create request with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Release first request
		close(blockChan)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d for cancelled request, got %d",
				http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("multiple concurrent requests up to limit succeed", func(t *testing.T) {
		maxConcurrent := 10
		timeout := 200 * time.Millisecond
		logger := zap.NewNop()

		middleware := newHTTPBulkheadMiddleware(maxConcurrent, timeout, logger)

		var maxActive int32
		var currentActive int32
		var mu sync.Mutex

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			active := atomic.AddInt32(&currentActive, 1)

			mu.Lock()
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()

			time.Sleep(10 * time.Millisecond) // Simulate work
			atomic.AddInt32(&currentActive, -1)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		var wg sync.WaitGroup
		successCount := atomic.Int32{}

		// Start exactly maxConcurrent requests
		for i := 0; i < maxConcurrent; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				if w.Code == http.StatusOK {
					successCount.Add(1)
				}
			}()
		}

		wg.Wait()

		if successCount.Load() != int32(maxConcurrent) {
			t.Errorf("expected %d successful requests, got %d", maxConcurrent, successCount.Load())
		}

		if maxActive > int32(maxConcurrent) {
			t.Errorf("max concurrent execution (%d) exceeded limit (%d)", maxActive, maxConcurrent)
		}
	})
}
