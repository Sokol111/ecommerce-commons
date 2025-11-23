package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockRateLimiter is a mock implementation of the rateLimiter interface
type mockRateLimiter struct {
	allowFunc func() bool
}

func (m *mockRateLimiter) Allow() bool {
	return m.allowFunc()
}

func TestNewRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests when rate limit is not exceeded", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				return true
			},
		}
		middleware := newRateLimitMiddleware(limiter)

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

	t.Run("rejects requests when rate limit is exceeded", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				return false
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})

	t.Run("bypasses rate limiting for /health/live endpoint", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				return false // Rate limit exceeded
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		router := gin.New()
		router.Use(middleware)
		router.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/live, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("bypasses rate limiting for /health/ready endpoint", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				return false // Rate limit exceeded
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		router := gin.New()
		router.Use(middleware)
		router.GET("/health/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/ready, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("rate limits regular endpoints but not health checks", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				return false // Rate limit exceeded
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": "users"})
		})
		router.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})
		router.GET("/health/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		// Regular endpoint should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d for /api/users, got %d", http.StatusTooManyRequests, w.Code)
		}

		// Health endpoints should not be rate limited
		req = httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/live, got %d", http.StatusOK, w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/ready, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("handles multiple requests with varying rate limit states", func(t *testing.T) {
		callCount := 0
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				callCount++
				// Allow first 3 requests, then deny
				return callCount <= 3
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// First 3 requests should succeed
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, w.Code)
			}
		}

		// 4th request should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d for 4th request, got %d", http.StatusTooManyRequests, w.Code)
		}
	})

	t.Run("does not call next handler when rate limited", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				return false
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		handlerCalled := false
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("handler should not be called when rate limited")
		}

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})

	t.Run("calls next handler when rate limit allows", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				return true
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		handlerCalled := false
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("handler should be called when rate limit allows")
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("works with different HTTP methods", func(t *testing.T) {
		callCount := 0
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				callCount++
				return callCount <= 2
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"method": "GET"})
		})
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"method": "POST"})
		})
		router.PUT("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"method": "PUT"})
		})

		// GET request (allowed)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET: expected status %d, got %d", http.StatusOK, w.Code)
		}

		// POST request (allowed)
		req = httptest.NewRequest(http.MethodPost, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("POST: expected status %d, got %d", http.StatusOK, w.Code)
		}

		// PUT request (rate limited)
		req = httptest.NewRequest(http.MethodPut, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("PUT: expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})

	t.Run("rate limiter is called for each request except health checks", func(t *testing.T) {
		callCount := 0
		limiter := &mockRateLimiter{
			allowFunc: func() bool {
				callCount++
				return true
			},
		}
		middleware := newRateLimitMiddleware(limiter)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		router.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})
		router.GET("/health/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		// Regular request - limiter should be called
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if callCount != 1 {
			t.Errorf("expected limiter to be called once, but was called %d times", callCount)
		}

		// Health endpoints - limiter should not be called
		req = httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if callCount != 1 {
			t.Errorf("expected limiter not to be called for /health/live, but was called %d times total", callCount)
		}

		req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if callCount != 1 {
			t.Errorf("expected limiter not to be called for /health/ready, but was called %d times total", callCount)
		}

		// Another regular request - limiter should be called again
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if callCount != 2 {
			t.Errorf("expected limiter to be called twice for regular requests, but was called %d times", callCount)
		}
	})
}
