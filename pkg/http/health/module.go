package health

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func NewHealthRoutesModule() fx.Option {
	return fx.Options(
		fx.Provide(newHealthHandler),
		fx.Invoke(registerHealthRoutes),
	)
}

func registerHealthRoutes(r *gin.Engine, handler *healthHandler) {
	r.GET("/health/ready", handler.IsReady)
	r.GET("/health/live", handler.IsLive)
}
