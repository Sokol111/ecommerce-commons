package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type healthHandler struct {
	readiness Readiness
}

func newHealthHandler(r Readiness) *healthHandler {
	return &healthHandler{readiness: r}
}

func (h *healthHandler) IsReady(c *gin.Context) {
	status := h.readiness.getStatus()

	// Notify readiness tracker when probe returns 200 OK (first time Kubernetes knows we're ready)
	if status.Ready {
		h.readiness.notifyKubernetesProbe()
	}

	// Support both simple text and detailed JSON responses
	if c.Query("format") == "json" || c.GetHeader("Accept") == "application/json" {
		if status.Ready {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
		return
	}

	// Default simple response for Kubernetes probes
	if status.Ready {
		c.String(http.StatusOK, "ready")
	} else {
		c.String(http.StatusServiceUnavailable, "not ready")
	}
}

func (h *healthHandler) IsLive(c *gin.Context) {
	c.String(http.StatusOK, "alive")
}
