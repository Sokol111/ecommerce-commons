package observability

import (
	"strings"

	"github.com/Sokol111/ecommerce-commons/pkg/config"
	commongin "github.com/Sokol111/ecommerce-commons/pkg/gin"
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
		AppConf config.Config

		TP trace.TracerProvider `optional:"true"`
		MP metric.MeterProvider `optional:"true"`
	}

	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(d deps) commongin.Middleware {
					if !d.Conf.TracingEnabled && !d.Conf.MetricsEnabled {
						return commongin.Middleware{}
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

					return commongin.Middleware{
						Priority: 0,
						Handler:  otelgin.Middleware(d.AppConf.ServiceName, opts...),
					}
				},
				fx.ResultTags(`group:"gin_mw"`),
			),
		),
	)
}
