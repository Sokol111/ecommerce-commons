package middleware

import (
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// loggerMiddleware logs incoming HTTP requests.
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health/live" || path == "/health/ready" {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := append(requestFields(c),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		logger.Get(c).Debug("Incoming request", fields...)
	}
}

// LoggerModule provides logger middleware.
func LoggerModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: priority, Handler: loggerMiddleware()}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
