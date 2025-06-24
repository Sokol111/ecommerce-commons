package health

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func NewHealthModule() fx.Option {
	return fx.Options(
		fx.Provide(
			provideReadiness,
			newHealthHandler,
		),
		fx.Invoke(func(lc fx.Lifecycle, r *readiness) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					r.StartWatching()
					return nil
				},
			})
		}),
		fx.Invoke(registerHealthRoutes),
	)
}

func provideReadiness() (*readiness, Readiness) {
	r := newReadiness()
	return r, r
}

func registerHealthRoutes(r *gin.Engine, handler *healthHandler) {
	r.GET("/health/ready", handler.IsReady)
	r.GET("/health/live", handler.IsLive)
}
