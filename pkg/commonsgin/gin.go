package commonsgin

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewEngine(log *zap.Logger) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(errorLoggerMiddleware(log))
	return engine
}

func errorLoggerMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Error("Request error",
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.Int("status", c.Writer.Status()),
					zap.Duration("latency", time.Since(start)),
					zap.String("error", e.Error()),
					zap.Any("meta", e.Meta),
				)
			}
		}
	}
}
