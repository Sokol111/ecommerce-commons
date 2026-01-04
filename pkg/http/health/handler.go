package health

import (
	"encoding/json"
	"net/http"

	coreHealth "github.com/Sokol111/ecommerce-commons/pkg/core/health"
)

type healthHandler struct {
	readiness      coreHealth.ReadinessChecker
	trafficControl coreHealth.TrafficController
}

func newHealthHandler(r coreHealth.ReadinessChecker, tc coreHealth.TrafficController) *healthHandler {
	return &healthHandler{
		readiness:      r,
		trafficControl: tc,
	}
}

func (h *healthHandler) IsReady(w http.ResponseWriter, r *http.Request) {
	// Support both simple text and detailed JSON responses
	if r.URL.Query().Get("format") == "json" || r.Header.Get("Accept") == "application/json" {
		status := h.readiness.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		if status.Ready {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(status)
		return
	}

	// Default simple response for Kubernetes probes
	if h.readiness.IsReady() {
		// Notify that we're ready for traffic (idempotent - safe to call multiple times)
		h.trafficControl.MarkTrafficReady()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
	}
}

func (h *healthHandler) IsLive(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("alive"))
}
