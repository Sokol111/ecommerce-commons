package health

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	coreHealth "github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockReadinessChecker struct {
	ready  bool
	status coreHealth.ReadinessStatus
}

func (m *mockReadinessChecker) IsReady() bool {
	return m.ready
}

func (m *mockReadinessChecker) GetStatus() coreHealth.ReadinessStatus {
	return m.status
}

type mockTrafficController struct {
	trafficReadyCalled atomic.Bool
}

func (m *mockTrafficController) MarkTrafficReady() {
	m.trafficReadyCalled.Store(true)
}

func setupTestRouter(handler *healthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health/ready", handler.IsReady)
	router.GET("/health/live", handler.IsLive)
	return router
}

func TestNewHealthHandler(t *testing.T) {
	readiness := &mockReadinessChecker{}
	trafficControl := &mockTrafficController{}

	handler := newHealthHandler(readiness, trafficControl)

	assert.NotNil(t, handler)
	assert.Equal(t, readiness, handler.readiness)
	assert.Equal(t, trafficControl, handler.trafficControl)
}

func TestHealthHandler_IsReady(t *testing.T) {
	t.Run("returns 200 OK when ready (simple text response)", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ready", w.Body.String())
		assert.True(t, trafficControl.trafficReadyCalled.Load(), "Should mark traffic as ready")
	})

	t.Run("returns 503 when not ready (simple text response)", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: false}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Equal(t, "not ready", w.Body.String())
		assert.False(t, trafficControl.trafficReadyCalled.Load(), "Should not mark traffic as ready")
	})

	t.Run("returns 200 OK with JSON when format=json query parameter", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready: true,
				Components: []coreHealth.ComponentStatus{
					{
						Name:      "database",
						Ready:     true,
						StartedAt: now.Add(-5 * time.Second),
						ReadyAt:   now.Add(-2 * time.Second),
					},
					{
						Name:      "cache",
						Ready:     true,
						StartedAt: now.Add(-4 * time.Second),
						ReadyAt:   now.Add(-1 * time.Second),
					},
				},
				ReadyAt:              now.Add(-1 * time.Second),
				KubernetesNotifiedAt: now,
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), `"ready":true`)
		assert.Contains(t, w.Body.String(), `"database"`)
		assert.Contains(t, w.Body.String(), `"cache"`)
	})

	t.Run("returns 503 with JSON when format=json and not ready", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			ready: false,
			status: coreHealth.ReadinessStatus{
				Ready: false,
				Components: []coreHealth.ComponentStatus{
					{
						Name:      "database",
						Ready:     true,
						StartedAt: now.Add(-5 * time.Second),
						ReadyAt:   now.Add(-2 * time.Second),
					},
					{
						Name:      "cache",
						Ready:     false,
						StartedAt: now.Add(-4 * time.Second),
					},
				},
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), `"ready":false`)
		assert.Contains(t, w.Body.String(), `"database"`)
		assert.Contains(t, w.Body.String(), `"cache"`)
	})

	t.Run("returns 200 OK with JSON when Accept header is application/json", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready: true,
				Components: []coreHealth.ComponentStatus{
					{
						Name:      "database",
						Ready:     true,
						StartedAt: now.Add(-3 * time.Second),
						ReadyAt:   now.Add(-1 * time.Second),
					},
				},
				ReadyAt: now.Add(-1 * time.Second),
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), `"ready":true`)
		assert.Contains(t, w.Body.String(), `"database"`)
	})

	t.Run("returns 503 with JSON when Accept header is application/json and not ready", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			ready: false,
			status: coreHealth.ReadinessStatus{
				Ready: false,
				Components: []coreHealth.ComponentStatus{
					{
						Name:      "database",
						Ready:     false,
						StartedAt: now,
					},
				},
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), `"ready":false`)
	})

	t.Run("prefers format=json query parameter over Accept header", func(t *testing.T) {
		readiness := &mockReadinessChecker{
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready:      true,
				Components: []coreHealth.ComponentStatus{},
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		req.Header.Set("Accept", "text/plain")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	})

	t.Run("calls MarkTrafficReady only when ready (simple response)", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.True(t, trafficControl.trafficReadyCalled.Load())
	})

	t.Run("does not call MarkTrafficReady when not ready (simple response)", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: false}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.False(t, trafficControl.trafficReadyCalled.Load())
	})

	t.Run("does not call MarkTrafficReady when using JSON response format", func(t *testing.T) {
		readiness := &mockReadinessChecker{
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready:      true,
				Components: []coreHealth.ComponentStatus{},
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.False(t, trafficControl.trafficReadyCalled.Load(), "Should not call MarkTrafficReady for JSON responses")
	})

	t.Run("MarkTrafficReady is idempotent (can be called multiple times)", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Second request
		req2 := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		// Both should succeed
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.True(t, trafficControl.trafficReadyCalled.Load())
	})
}

func TestHealthHandler_IsLive(t *testing.T) {
	t.Run("always returns 200 OK", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: false}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "alive", w.Body.String())
		assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	})

	t.Run("returns alive regardless of readiness state", func(t *testing.T) {
		testCases := []struct {
			name  string
			ready bool
		}{
			{"when ready", true},
			{"when not ready", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				readiness := &mockReadinessChecker{ready: tc.ready}
				trafficControl := &mockTrafficController{}
				handler := newHealthHandler(readiness, trafficControl)
				router := setupTestRouter(handler)

				req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "alive", w.Body.String())
			})
		}
	})

	t.Run("does not call MarkTrafficReady", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.False(t, trafficControl.trafficReadyCalled.Load())
	})
}

func TestHealthHandler_ComplexStatus(t *testing.T) {
	t.Run("returns complete status with multiple components", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready: true,
				Components: []coreHealth.ComponentStatus{
					{
						Name:      "database",
						Ready:     true,
						StartedAt: now.Add(-10 * time.Second),
						ReadyAt:   now.Add(-5 * time.Second),
					},
					{
						Name:      "cache",
						Ready:     true,
						StartedAt: now.Add(-8 * time.Second),
						ReadyAt:   now.Add(-3 * time.Second),
					},
					{
						Name:      "message-queue",
						Ready:     true,
						StartedAt: now.Add(-6 * time.Second),
						ReadyAt:   now.Add(-1 * time.Second),
					},
				},
				ReadyAt:              now.Add(-1 * time.Second),
				KubernetesNotifiedAt: now,
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, `"ready":true`)
		assert.Contains(t, body, `"database"`)
		assert.Contains(t, body, `"cache"`)
		assert.Contains(t, body, `"message-queue"`)
	})

	t.Run("returns partial ready status when some components not ready", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			ready: false,
			status: coreHealth.ReadinessStatus{
				Ready: false,
				Components: []coreHealth.ComponentStatus{
					{
						Name:      "database",
						Ready:     true,
						StartedAt: now.Add(-5 * time.Second),
						ReadyAt:   now.Add(-2 * time.Second),
					},
					{
						Name:      "cache",
						Ready:     false,
						StartedAt: now.Add(-3 * time.Second),
					},
				},
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, `"ready":false`)
		assert.Contains(t, body, `"database"`)
		assert.Contains(t, body, `"cache"`)
	})

	t.Run("returns empty components list when no components registered", func(t *testing.T) {
		readiness := &mockReadinessChecker{
			ready: false,
			status: coreHealth.ReadinessStatus{
				Ready:      false,
				Components: []coreHealth.ComponentStatus{},
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, `"ready":false`)
		assert.Contains(t, body, `"components":[]`)
	})
}

func TestHealthHandler_EdgeCases(t *testing.T) {
	t.Run("handles empty format parameter", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ready", w.Body.String())
	})

	t.Run("handles invalid format parameter", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=xml", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ready", w.Body.String())
	})

	t.Run("handles Accept header with multiple values", func(t *testing.T) {
		readiness := &mockReadinessChecker{
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready:      true,
				Components: []coreHealth.ComponentStatus{},
			},
		}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		req.Header.Set("Accept", "text/html, application/json, */*")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should return simple text response as Accept doesn't exactly match "application/json"
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ready", w.Body.String())
	})

	t.Run("concurrent requests", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		const numRequests = 50
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		for i := 0; i < numRequests; i++ {
			code := <-results
			assert.Equal(t, http.StatusOK, code)
		}
	})
}

func TestHealthHandler_KubernetesIntegration(t *testing.T) {
	t.Run("simple text response suitable for Kubernetes probes", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		// Kubernetes typically doesn't send special headers or query params
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ready", w.Body.String())
		assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})

	t.Run("liveness probe always succeeds", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: false}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "alive", w.Body.String())
	})
}

func TestHealthHandler_HTTPMethods(t *testing.T) {
	t.Run("GET method works for readiness", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET method works for liveness", func(t *testing.T) {
		readiness := &mockReadinessChecker{ready: true}
		trafficControl := &mockTrafficController{}
		handler := newHealthHandler(readiness, trafficControl)
		router := setupTestRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Table-driven tests for various scenarios
func TestHealthHandler_IsReady_TableDriven(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name                string
		ready               bool
		status              coreHealth.ReadinessStatus
		queryParams         string
		acceptHeader        string
		expectedStatusCode  int
		expectedBody        string
		expectedContentType string
		shouldMarkTraffic   bool
	}{
		{
			name:                "ready - simple text",
			ready:               true,
			queryParams:         "",
			acceptHeader:        "",
			expectedStatusCode:  http.StatusOK,
			expectedBody:        "ready",
			expectedContentType: "text/plain",
			shouldMarkTraffic:   true,
		},
		{
			name:                "not ready - simple text",
			ready:               false,
			queryParams:         "",
			acceptHeader:        "",
			expectedStatusCode:  http.StatusServiceUnavailable,
			expectedBody:        "not ready",
			expectedContentType: "text/plain",
			shouldMarkTraffic:   false,
		},
		{
			name:  "ready - JSON format",
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready: true,
				Components: []coreHealth.ComponentStatus{
					{Name: "db", Ready: true, StartedAt: now, ReadyAt: now},
				},
			},
			queryParams:         "format=json",
			acceptHeader:        "",
			expectedStatusCode:  http.StatusOK,
			expectedBody:        `"ready":true`,
			expectedContentType: "application/json",
			shouldMarkTraffic:   false,
		},
		{
			name:  "not ready - JSON format",
			ready: false,
			status: coreHealth.ReadinessStatus{
				Ready: false,
				Components: []coreHealth.ComponentStatus{
					{Name: "db", Ready: false, StartedAt: now},
				},
			},
			queryParams:         "format=json",
			acceptHeader:        "",
			expectedStatusCode:  http.StatusServiceUnavailable,
			expectedBody:        `"ready":false`,
			expectedContentType: "application/json",
			shouldMarkTraffic:   false,
		},
		{
			name:  "ready - Accept JSON header",
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready:      true,
				Components: []coreHealth.ComponentStatus{},
			},
			queryParams:         "",
			acceptHeader:        "application/json",
			expectedStatusCode:  http.StatusOK,
			expectedBody:        `"ready":true`,
			expectedContentType: "application/json",
			shouldMarkTraffic:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			readiness := &mockReadinessChecker{
				ready:  tc.ready,
				status: tc.status,
			}
			trafficControl := &mockTrafficController{}
			handler := newHealthHandler(readiness, trafficControl)
			router := setupTestRouter(handler)

			url := "/health/ready"
			if tc.queryParams != "" {
				url += "?" + tc.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tc.acceptHeader != "" {
				req.Header.Set("Accept", tc.acceptHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
			assert.Contains(t, w.Header().Get("Content-Type"), tc.expectedContentType)
			assert.Equal(t, tc.shouldMarkTraffic, trafficControl.trafficReadyCalled.Load())
		})
	}
}

// Test handler behavior with nil dependencies (should panic in production, but let's document the expected behavior)
func TestHealthHandler_NilDependencies(t *testing.T) {
	t.Run("handler works with nil traffic controller in edge case", func(t *testing.T) {
		// Note: This is not recommended, just documenting behavior
		readiness := &mockReadinessChecker{
			ready: true,
			status: coreHealth.ReadinessStatus{
				Ready:      true,
				Components: []coreHealth.ComponentStatus{},
			},
		}
		handler := &healthHandler{
			readiness:      readiness,
			trafficControl: nil, // This would panic if MarkTrafficReady is called
		}
		router := setupTestRouter(handler)

		// Using JSON format to avoid calling MarkTrafficReady
		req := httptest.NewRequest(http.MethodGet, "/health/ready?format=json", nil)
		w := httptest.NewRecorder()

		// Should not panic
		require.NotPanics(t, func() {
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
