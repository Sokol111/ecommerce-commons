package middleware

import (
	"net/http"

	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/zap"
)

// Middleware represents an ogen middleware with priority.
type Middleware struct {
	Priority int
	Handler  middleware.Middleware
}

// requestFields returns common request fields for logging.
func requestFields(r *http.Request) []zap.Field {
	return []zap.Field{
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("query", r.URL.RawQuery),
	}
}
