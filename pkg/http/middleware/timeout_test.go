package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewTimeoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests that complete within timeout", func(t *testing.T) {
		middleware := newTimeoutMiddleware(100 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Fast handler that completes immediately
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("returns timeout error when request exceeds timeout", func(t *testing.T) {
		middleware := newTimeoutMiddleware(50 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Slow handler that exceeds timeout and doesn't write response
			time.Sleep(100 * time.Millisecond)
			// Don't write response to simulate a handler that respects context
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("expected status %d, got %d", http.StatusGatewayTimeout, w.Code)
		}
	})

	t.Run("bypasses timeout for /health/live endpoint", func(t *testing.T) {
		middleware := newTimeoutMiddleware(50 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/health/live", func(c *gin.Context) {
			// Slow handler that would exceed timeout
			time.Sleep(100 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/live, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("bypasses timeout for /health/ready endpoint", func(t *testing.T) {
		middleware := newTimeoutMiddleware(50 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/health/ready", func(c *gin.Context) {
			// Slow handler that would exceed timeout
			time.Sleep(100 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for /health/ready, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("sets timeout context on request", func(t *testing.T) {
		middleware := newTimeoutMiddleware(100 * time.Millisecond)

		var requestContext context.Context
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			requestContext = c.Request.Context()
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if requestContext == nil {
			t.Fatal("request context should not be nil")
		}

		deadline, ok := requestContext.Deadline()
		if !ok {
			t.Error("expected context to have a deadline")
		}

		if time.Until(deadline) > 100*time.Millisecond {
			t.Error("deadline should be set to the timeout duration")
		}
	})

	t.Run("does not set timeout context for health endpoints", func(t *testing.T) {
		middleware := newTimeoutMiddleware(100 * time.Millisecond)

		var requestContext context.Context
		router := gin.New()
		router.Use(middleware)
		router.GET("/health/live", func(c *gin.Context) {
			requestContext = c.Request.Context()
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if requestContext == nil {
			t.Fatal("request context should not be nil")
		}

		_, ok := requestContext.Deadline()
		if ok {
			t.Error("health endpoint should not have a timeout deadline")
		}
	})

	t.Run("handler can check context for timeout", func(t *testing.T) {
		middleware := newTimeoutMiddleware(50 * time.Millisecond)

		contextTimedOut := false
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Simulate work and check for timeout
			for i := 0; i < 10; i++ {
				select {
				case <-c.Request.Context().Done():
					contextTimedOut = true
					return
				default:
					time.Sleep(20 * time.Millisecond)
				}
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if !contextTimedOut {
			t.Error("handler should detect context timeout")
		}
	})

	t.Run("does not send timeout response if handler already wrote response", func(t *testing.T) {
		middleware := newTimeoutMiddleware(50 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Write response before timeout
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			// Then sleep to exceed timeout
			time.Sleep(100 * time.Millisecond)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should keep the original status, not timeout
		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("works with different HTTP methods", func(t *testing.T) {
		middleware := newTimeoutMiddleware(100 * time.Millisecond)

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
		router.DELETE("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"method": "DELETE"})
		})

		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s: expected status %d, got %d", method, http.StatusOK, w.Code)
			}
		}
	})

	t.Run("applies timeout to regular endpoints but not health checks", func(t *testing.T) {
		middleware := newTimeoutMiddleware(50 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/users", func(c *gin.Context) {
			time.Sleep(100 * time.Millisecond)
			// Don't write response to simulate timeout
		})
		router.GET("/health/live", func(c *gin.Context) {
			time.Sleep(100 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})
		router.GET("/health/ready", func(c *gin.Context) {
			time.Sleep(100 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		// Regular endpoint should timeout
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("expected status %d for /api/users, got %d", http.StatusGatewayTimeout, w.Code)
		}

		// Health endpoints should not timeout
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

	t.Run("multiple requests with varying completion times", func(t *testing.T) {
		middleware := newTimeoutMiddleware(100 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Check if delay parameter is provided
			delay := c.Query("delay")
			if delay != "" {
				d, _ := time.ParseDuration(delay)
				time.Sleep(d)
			}
			// Only write response if context is not done
			if c.Request.Context().Err() == nil {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
		})

		// Fast request - should succeed
		req := httptest.NewRequest(http.MethodGet, "/test?delay=20ms", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("fast request: expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Slow request - should timeout
		req = httptest.NewRequest(http.MethodGet, "/test?delay=150ms", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("slow request: expected status %d, got %d", http.StatusGatewayTimeout, w.Code)
		}

		// Another fast request - should succeed
		req = httptest.NewRequest(http.MethodGet, "/test?delay=30ms", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("second fast request: expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("timeout context is properly cancelled", func(t *testing.T) {
		middleware := newTimeoutMiddleware(100 * time.Millisecond)

		var capturedContext context.Context
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			capturedContext = c.Request.Context()
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Context should be cancelled after request completes
		time.Sleep(10 * time.Millisecond)
		if capturedContext.Err() == nil {
			t.Error("context should be cancelled after request completes")
		}
	})

	t.Run("very short timeout triggers immediately", func(t *testing.T) {
		middleware := newTimeoutMiddleware(1 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			time.Sleep(50 * time.Millisecond)
			// Don't write response to simulate timeout
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusGatewayTimeout {
			t.Errorf("expected status %d, got %d", http.StatusGatewayTimeout, w.Code)
		}
	})

	t.Run("very long timeout allows slow requests", func(t *testing.T) {
		middleware := newTimeoutMiddleware(500 * time.Millisecond)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			time.Sleep(100 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("context error is deadline exceeded", func(t *testing.T) {
		middleware := newTimeoutMiddleware(50 * time.Millisecond)

		var contextError error
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			time.Sleep(100 * time.Millisecond)
			contextError = c.Request.Context().Err()
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if contextError != context.DeadlineExceeded {
			t.Errorf("expected context error to be DeadlineExceeded, got %v", contextError)
		}
	})
}
