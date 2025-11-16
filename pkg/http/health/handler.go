package health

import (
	"net/http"
	"time"

	coreHealth "github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/gin-gonic/gin"
)

type ComponentStatus struct {
	Name      string    `json:"name"`
	Ready     bool      `json:"ready"`
	StartedAt time.Time `json:"started_at"`
	ReadyAt   time.Time `json:"ready_at,omitempty"`
}

type ReadinessStatus struct {
	Ready                bool              `json:"ready"`
	Components           []ComponentStatus `json:"components"`
	ReadyAt              time.Time         `json:"ready_at,omitempty"`
	KubernetesNotifiedAt time.Time         `json:"kubernetes_notified_at,omitempty"` // When K8s first got 200 OK
}

type healthHandler struct {
	readiness coreHealth.Readiness
}

func newHealthHandler(r coreHealth.Readiness) *healthHandler {
	return &healthHandler{readiness: r}
}

func (h *healthHandler) IsReady(c *gin.Context) {
	// Notify readiness tracker when we return 200 OK (Kubernetes will see we're ready)
	// The method itself checks if we're ready and if it's the first notification
	h.readiness.NotifyKubernetesProbe()

	// Support both simple text and detailed JSON responses
	if c.Query("format") == "json" || c.GetHeader("Accept") == "application/json" {
		coreStatus := h.readiness.GetStatus()
		status := convertToReadinessStatus(coreStatus)
		if status.Ready {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
		return
	}

	// Default simple response for Kubernetes probes
	if h.readiness.IsReady() {
		c.String(http.StatusOK, "ready")
	} else {
		c.String(http.StatusServiceUnavailable, "not ready")
	}
}

func (h *healthHandler) IsLive(c *gin.Context) {
	c.String(http.StatusOK, "alive")
}

func convertToReadinessStatus(coreStatus coreHealth.ReadinessStatus) ReadinessStatus {
	components := make([]ComponentStatus, len(coreStatus.Components))
	for i, comp := range coreStatus.Components {
		components[i] = ComponentStatus{
			Name:      comp.Name,
			Ready:     comp.Ready,
			StartedAt: comp.StartedAt,
			ReadyAt:   comp.ReadyAt,
		}
	}

	return ReadinessStatus{
		Ready:                coreStatus.Ready,
		Components:           components,
		ReadyAt:              coreStatus.ReadyAt,
		KubernetesNotifiedAt: coreStatus.KubernetesNotifiedAt,
	}
}
