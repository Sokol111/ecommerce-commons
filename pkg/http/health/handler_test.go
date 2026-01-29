package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreHealth "github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReadinessChecker is a test double for ReadinessChecker.
type mockReadinessChecker struct {
	isReady bool
	status  coreHealth.ReadinessStatus
}

func (m *mockReadinessChecker) IsReady() bool {
	return m.isReady
}

func (m *mockReadinessChecker) GetStatus() coreHealth.ReadinessStatus {
	return m.status
}

// mockTrafficController is a test double for TrafficController.
type mockTrafficController struct {
	trafficReadyCalled bool
}

func (m *mockTrafficController) MarkTrafficReady() {
	m.trafficReadyCalled = true
}

func TestHealthHandler_IsReady(t *testing.T) {
	t.Run("returns 200 OK when ready (simple format)", func(t *testing.T) {
		readiness := &mockReadinessChecker{isReady: true}
		traffic := &mockTrafficController{}
		handler := newHealthHandler(readiness, traffic)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/ready", nil)

		handler.IsReady(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ready", w.Body.String())
		assert.True(t, traffic.trafficReadyCalled, "MarkTrafficReady should be called when ready")
	})

	t.Run("returns 503 Service Unavailable when not ready (simple format)", func(t *testing.T) {
		readiness := &mockReadinessChecker{isReady: false}
		traffic := &mockTrafficController{}
		handler := newHealthHandler(readiness, traffic)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/ready", nil)

		handler.IsReady(w, r)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Equal(t, "not ready", w.Body.String())
		assert.False(t, traffic.trafficReadyCalled, "MarkTrafficReady should not be called when not ready")
	})

	t.Run("returns JSON when format=json query param", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			isReady: true,
			status: coreHealth.ReadinessStatus{
				Ready:   true,
				ReadyAt: now,
				Components: []coreHealth.ComponentStatus{
					{Name: "database", Ready: true, StartedAt: now, ReadyAt: now},
					{Name: "kafka", Ready: true, StartedAt: now, ReadyAt: now},
				},
			},
		}
		traffic := &mockTrafficController{}
		handler := newHealthHandler(readiness, traffic)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/ready?format=json", nil)

		handler.IsReady(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var status coreHealth.ReadinessStatus
		err := json.Unmarshal(w.Body.Bytes(), &status)
		require.NoError(t, err)
		assert.True(t, status.Ready)
		assert.Len(t, status.Components, 2)
	})

	t.Run("returns JSON when Accept header is application/json", func(t *testing.T) {
		readiness := &mockReadinessChecker{
			isReady: true,
			status: coreHealth.ReadinessStatus{
				Ready: true,
			},
		}
		traffic := &mockTrafficController{}
		handler := newHealthHandler(readiness, traffic)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/ready", nil)
		r.Header.Set("Accept", "application/json")

		handler.IsReady(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("returns 503 with JSON when not ready and format=json", func(t *testing.T) {
		now := time.Now()
		readiness := &mockReadinessChecker{
			isReady: false,
			status: coreHealth.ReadinessStatus{
				Ready: false,
				Components: []coreHealth.ComponentStatus{
					{Name: "database", Ready: true, StartedAt: now, ReadyAt: now},
					{Name: "kafka", Ready: false, StartedAt: now},
				},
			},
		}
		traffic := &mockTrafficController{}
		handler := newHealthHandler(readiness, traffic)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/ready?format=json", nil)

		handler.IsReady(w, r)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var status coreHealth.ReadinessStatus
		err := json.Unmarshal(w.Body.Bytes(), &status)
		require.NoError(t, err)
		assert.False(t, status.Ready)
	})
}

func TestHealthHandler_IsLive(t *testing.T) {
	t.Run("always returns 200 OK", func(t *testing.T) {
		readiness := &mockReadinessChecker{isReady: false}
		traffic := &mockTrafficController{}
		handler := newHealthHandler(readiness, traffic)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/live", nil)

		handler.IsLive(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "alive", w.Body.String())
	})

	t.Run("returns alive regardless of readiness state", func(t *testing.T) {
		// Even when service is not ready, liveness should pass
		readiness := &mockReadinessChecker{isReady: false}
		traffic := &mockTrafficController{}
		handler := newHealthHandler(readiness, traffic)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/live", nil)

		handler.IsLive(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestNewHealthHandler(t *testing.T) {
	readiness := &mockReadinessChecker{}
	traffic := &mockTrafficController{}

	handler := newHealthHandler(readiness, traffic)

	assert.NotNil(t, handler)
	assert.Equal(t, readiness, handler.readiness)
	assert.Equal(t, traffic, handler.trafficControl)
}

func TestRegisterHealthRoutes(t *testing.T) {
	readiness := &mockReadinessChecker{isReady: true}
	traffic := &mockTrafficController{}
	handler := newHealthHandler(readiness, traffic)
	mux := http.NewServeMux()

	registerHealthRoutes(mux, handler)

	// Test /health/ready endpoint is registered
	t.Run("registers /health/ready endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/ready", nil)
		mux.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ready", w.Body.String())
	})

	// Test /health/live endpoint is registered
	t.Run("registers /health/live endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/health/live", nil)
		mux.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "alive", w.Body.String())
	})
}
