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

// newTimeoutMiddleware creates a middleware that adds request timeout to all requests
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

		// Channel to track if request completed
		finished := make(chan struct{})

		// Process request in goroutine
		go func() {
			c.Next()
			close(finished)
		}()

		// Wait for completion or timeout
		select {
		case <-finished:
			// Request completed successfully
			return
		case <-ctx.Done():
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
