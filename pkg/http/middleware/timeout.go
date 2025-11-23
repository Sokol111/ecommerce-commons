package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// newTimeoutMiddleware creates a middleware that adds request timeout to all requests.
// This middleware sets a timeout context on the request. Handlers should respect this context
// and check for context.Done() or context.Err() to handle timeouts properly.
//
// Note: This middleware only sets the timeout context. If a handler doesn't check the context
// and runs longer than the timeout, the middleware will send a timeout response after the
// handler completes (if no response was written yet).
func newTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
			c.Next()
			return
		}

		// Create timeout context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context with timeout context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// After handler completes, check if context timed out and no response was sent
		if errors.Is(ctx.Err(), context.DeadlineExceeded) && !c.Writer.Written() {
			problem := problems.Problem{Detail: "request took too long to process"}
			c.AbortWithError(http.StatusGatewayTimeout, errors.New("HTTP request timeout")).SetMeta(problem)
		}
	}
}

// TimeoutModule adds timeout middleware to the application
func TimeoutModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				if !serverConfig.Timeout.Enabled {
					return Middleware{
						Priority: priority,
						Handler:  nil, // Will be skipped in newEngine
					}

				}
				log.Info("HTTP timeout middleware initialized",
					zap.Duration("request-timeout", serverConfig.Timeout.RequestTimeout),
				)
				return Middleware{
					Priority: priority,
					Handler:  newTimeoutMiddleware(serverConfig.Timeout.RequestTimeout),
				}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
