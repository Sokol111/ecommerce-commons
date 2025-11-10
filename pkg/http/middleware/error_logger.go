package middleware

import (
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// errorLoggerMiddleware logs errors from Gin context
func errorLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			log := logger.FromContext(c)
			for _, e := range c.Errors {
				fields := append(requestFields(c),
					zap.Int("status", c.Writer.Status()),
					zap.String("error", e.Error()),
					zap.Any("meta", e.Meta),
				)
				log.Error("Request error", fields...)
			}
		}
	}
}

// ErrorLoggerModule provides error logger middleware
func ErrorLoggerModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: priority, Handler: errorLoggerMiddleware()}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
