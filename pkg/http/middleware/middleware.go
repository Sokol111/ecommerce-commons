package middleware

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Middleware represents a Gin middleware with priority
type Middleware struct {
	Priority int
	Handler  gin.HandlerFunc
}

// mwIn is used for dependency injection of all middlewares
type mwIn struct {
	fx.In
	Middlewares []Middleware `group:"gin_mw"`
}

// provideGinAndHandler creates Gin engine with all registered middlewares
func provideGinAndHandler(in mwIn) (*gin.Engine, http.Handler) {
	e := newEngine(in.Middlewares)
	return e, e
}

// newEngine creates a new Gin engine and registers middlewares sorted by priority
func newEngine(mws []Middleware) *gin.Engine {
	engine := gin.New(func(e *gin.Engine) {
		e.ContextWithFallback = true
	})

	// Sort middlewares by priority (lower priority = executes first)
	sort.Slice(mws, func(i, j int) bool { return mws[i].Priority < mws[j].Priority })

	// Register middlewares
	for _, m := range mws {
		if m.Handler == nil {
			continue // Skip disabled middlewares
		}
		engine.Use(m.Handler)
	}

	return engine
}

// requestFields returns common request fields for logging
func requestFields(c *gin.Context) []zap.Field {
	return []zap.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.String("client_ip", c.ClientIP()),
	}
}
