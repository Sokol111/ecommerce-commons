package gin

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewGinModule() fx.Option {
	return fx.Provide(
		provideGinAndHandler,
	)
}

func provideGinAndHandler(log *zap.Logger) (*gin.Engine, http.Handler) {
	e := newEngine(log)
	return e, e
}

func newEngine(log *zap.Logger) *gin.Engine {
	engine := gin.New()
	engine.Use(loggerMiddleware(log))
	engine.Use(recoveryMiddleware(log))
	engine.Use(errorLoggerMiddleware(log))
	return engine
}

func loggerMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health/live" || path == "/health/ready" {
			c.Next()
			return
		}

		start := time.Now()
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Debug("Incoming request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", raw),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

func recoveryMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Panic recovered",
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.String("client_ip", c.ClientIP()),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

func errorLoggerMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Error("Request error",
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.Int("status", c.Writer.Status()),
					zap.String("error", e.Error()),
					zap.Any("meta", e.Meta),
				)
			}
		}
	}
}
