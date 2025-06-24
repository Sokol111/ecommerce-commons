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
	if h.readiness.IsReady() {
		c.String(http.StatusOK, "ready")
	} else {
		c.String(http.StatusServiceUnavailable, "not ready")
	}
}

func (h *healthHandler) IsLive(c *gin.Context) {
	c.String(http.StatusOK, "alive")
}
