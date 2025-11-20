package health

import (
	"net/http"

	coreHealth "github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"github.com/gin-gonic/gin"
)

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
		status := h.readiness.GetStatus()
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
