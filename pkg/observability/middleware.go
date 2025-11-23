package observability

import (
	"strings"

	"github.com/Sokol111/ecommerce-commons/pkg/core/config"
	"github.com/Sokol111/ecommerce-commons/pkg/http/middleware"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

func NewHTTPTelemetryModule() fx.Option {
	type deps struct {
		fx.In
		Conf    Config
		AppConf config.AppConfig

		TP trace.TracerProvider `optional:"true"`
		MP metric.MeterProvider `optional:"true"`
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(d deps) middleware.Middleware {
					if !d.Conf.TracingEnabled && !d.Conf.MetricsEnabled {
						return middleware.Middleware{}
					}

					opts := []otelgin.Option{
						otelgin.WithGinFilter(func(c *gin.Context) bool {
							route := c.FullPath()
							if route == "" {
								route = c.Request.URL.Path
							}
							if strings.HasPrefix(route, "/health") || strings.HasPrefix(route, "/metrics") {
								return false
							}
							return true
						}),
					}

					if d.Conf.TracingEnabled && d.TP != nil {
						opts = append(opts, otelgin.WithTracerProvider(d.TP))
					}
					if d.Conf.MetricsEnabled && d.MP != nil {
						opts = append(opts, otelgin.WithMeterProvider(d.MP))
					}

					return middleware.Middleware{
						Priority: 5, // First middleware to create tracing span for entire request
						Handler:  otelgin.Middleware(d.AppConf.ServiceName, opts...),
					}
				},
				fx.ResultTags(`group:"gin_mw"`),
			),
		),
	)
}
