package middleware

import (
	"context"
	"errors"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewTimeoutMiddleware creates a middleware that adds request timeout to all requests
func NewTimeoutMiddleware(serverConfig server.Config, log *zap.Logger, priority int) Middleware {
	config := serverConfig.Timeout

	// Skip if disabled
	if !config.Enabled {
		return Middleware{
			Priority: priority,
			Handler:  nil,
		}
	}

	log.Info("HTTP timeout middleware initialized",
		zap.Duration("request-timeout", config.RequestTimeout),
	)

	return Middleware{
		Priority: priority,
		Handler: func(c *gin.Context) {
			// Allow health checks without timeout
			if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
				c.Next()
				return
			}

			// Create timeout context
			ctx, cancel := context.WithTimeout(c.Request.Context(), config.RequestTimeout)
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
				// Timeout occurred
				log.Warn("HTTP request timeout",
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Duration("timeout", config.RequestTimeout),
				)

				problem := problems.GatewayTimeout("request took too long to process")
				problem.Instance = c.Request.URL.Path
				c.Error(errors.New(problem.Detail)).SetMeta(problem)
				c.Abort()
			}
		},
	}
}

// TimeoutModule adds timeout middleware to the application
func TimeoutModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				return NewTimeoutMiddleware(serverConfig, log, priority)
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
