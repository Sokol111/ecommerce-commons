package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

func TestNewCircuitBreakerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests when circuit is closed", func(t *testing.T) {
		logger := zap.NewNop()
		cb := newCircuitBreaker(5, 10*time.Second, 5*time.Second, 3, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

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

	t.Run("allows health checks without circuit breaker", func(t *testing.T) {
		logger := zap.NewNop()
		// Create circuit breaker in open state
		cb := newCircuitBreaker(1, 10*time.Second, 100*time.Millisecond, 1, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		// Force circuit to open by triggering failures
		router := gin.New()
		router.Use(middleware)
		router.GET("/fail", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "fail"})
		})
		router.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "alive"})
		})
		router.GET("/health/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		})

		// Trigger failure to open circuit
		req := httptest.NewRequest(http.MethodGet, "/fail", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Circuit should be open now, but health checks should still work
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

	t.Run("opens circuit after consecutive failures", func(t *testing.T) {
		logger := zap.NewNop()
		failureThreshold := uint32(3)
		cb := newCircuitBreaker(5, 10*time.Second, 1*time.Second, failureThreshold, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		})

		// Trigger failures to open the circuit
		for i := uint32(0); i < failureThreshold; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusInternalServerError {
				t.Errorf("failure %d: expected status %d, got %d", i+1, http.StatusInternalServerError, w.Code)
			}
		}

		// Next request should be rejected due to open circuit
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d when circuit is open, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("rejects requests when circuit is open", func(t *testing.T) {
		logger := zap.NewNop()
		cb := newCircuitBreaker(5, 10*time.Second, 100*time.Millisecond, 2, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
		})

		// Trigger failures to open circuit
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}

		// Verify circuit is open
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("does not trip circuit on 4xx errors", func(t *testing.T) {
		logger := zap.NewNop()
		cb := newCircuitBreaker(5, 10*time.Second, 5*time.Second, 3, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		})

		// Make multiple requests with 4xx errors
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusBadRequest, w.Code)
			}
		}

		// Circuit should still be closed (next request should succeed)
		router.GET("/success", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/success", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d after 4xx errors, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("trips circuit only on 5xx errors", func(t *testing.T) {
		logger := zap.NewNop()
		cb := newCircuitBreaker(5, 10*time.Second, 1*time.Second, 3, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		requestCount := 0
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			requestCount++
			// Mix 4xx and 5xx errors - only consecutive 5xx should count toward threshold
			switch requestCount {
			case 1:
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			case 2:
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			case 3:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			case 4:
				c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
			case 5:
				c.JSON(http.StatusBadGateway, gin.H{"error": "bad gateway"})
			default:
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
		})

		// Make requests: 400, 404 (4xx - should not count), then 500, 502, 503 (three consecutive 5xx)
		expectedStatuses := []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusInternalServerError,
			http.StatusNotImplemented,
			http.StatusBadGateway,
		}
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != expectedStatuses[i] {
				t.Errorf("request %d: expected status %d, got %d", i+1, expectedStatuses[i], w.Code)
			}
		}

		// Circuit should now be open after 3 consecutive 5xx errors
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d when circuit is open, got %d", http.StatusServiceUnavailable, w.Code)
		}

		// Verify circuit state
		if cb.State() != gobreaker.StateOpen {
			t.Errorf("expected circuit to be Open after 5xx errors, got %s", cb.State().String())
		}
	})

	t.Run("allows requests in half-open state after timeout", func(t *testing.T) {
		logger := zap.NewNop()
		timeout := 200 * time.Millisecond
		cb := newCircuitBreaker(1, 10*time.Second, timeout, 1, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		failureCount := 0
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			if failureCount < 1 {
				failureCount++
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
			} else {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
		})

		// Trigger failure to open circuit
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d for initial failure, got %d", http.StatusInternalServerError, w.Code)
		}

		// Verify circuit is open
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d when circuit is open, got %d", http.StatusServiceUnavailable, w.Code)
		}

		// Wait for timeout to transition to half-open
		time.Sleep(timeout + 50*time.Millisecond)

		// Request should be allowed in half-open state
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d in half-open state, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("handles aborted requests correctly", func(t *testing.T) {
		logger := zap.NewNop()
		cb := newCircuitBreaker(5, 10*time.Second, 5*time.Second, 1, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			// Abort request (e.g., by rate limiter)
			c.AbortWithStatus(http.StatusBadGateway)
		})

		// Make request that gets aborted
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadGateway {
			t.Errorf("expected status %d, got %d", http.StatusBadGateway, w.Code)
		}

		// Circuit should remain closed - next successful request should work
		router.GET("/success", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req = httptest.NewRequest(http.MethodGet, "/success", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d after aborted request, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("successful requests keep circuit closed", func(t *testing.T) {
		logger := zap.NewNop()
		cb := newCircuitBreaker(5, 10*time.Second, 5*time.Second, 3, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make multiple successful requests
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, w.Code)
			}
		}

		// Circuit should still be closed
		if cb.State() != gobreaker.StateClosed {
			t.Errorf("expected circuit state to be Closed, got %s", cb.State().String())
		}
	})

	t.Run("circuit recovers after successful request in half-open state", func(t *testing.T) {
		logger := zap.NewNop()
		timeout := 150 * time.Millisecond
		cb := newCircuitBreaker(1, 10*time.Second, timeout, 1, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		requestCount := 0
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			requestCount++
			if requestCount == 1 {
				// First request fails
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
			} else {
				// Subsequent requests succeed
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
		})

		// Trigger failure to open circuit
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if cb.State() != gobreaker.StateOpen {
			t.Errorf("expected circuit to be Open after failure, got %s", cb.State().String())
		}

		// Wait for timeout to transition to half-open
		time.Sleep(timeout + 50*time.Millisecond)

		// Successful request in half-open should close circuit
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Give circuit breaker time to update state
		time.Sleep(10 * time.Millisecond)

		if cb.State() != gobreaker.StateClosed {
			t.Errorf("expected circuit to be Closed after successful request, got %s", cb.State().String())
		}
	})

	t.Run("handles mixed success and failure scenarios", func(t *testing.T) {
		logger := zap.NewNop()
		cb := newCircuitBreaker(5, 10*time.Second, 5*time.Second, 3, logger)
		middleware := newCircuitBreakerMiddleware(cb, logger)

		requestCount := 0
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			requestCount++
			if requestCount%2 == 0 {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
			}
		})

		// Make alternating success/failure requests
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}

		// Circuit should still be closed because consecutive failures never reached threshold
		if cb.State() != gobreaker.StateClosed {
			t.Errorf("expected circuit to remain Closed with alternating results, got %s", cb.State().String())
		}
	})
}
