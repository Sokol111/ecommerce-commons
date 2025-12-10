package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Middleware represents a Gin middleware with priority.
type Middleware struct {
	Priority int
	Handler  gin.HandlerFunc
}

// requestFields returns common request fields for logging.
func requestFields(c *gin.Context) []zap.Field {
	return []zap.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.String("client_ip", c.ClientIP()),
	}
}
