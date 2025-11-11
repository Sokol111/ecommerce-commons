package middleware

import (
	"context"
	"errors"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// NewHTTPBulkheadMiddleware creates an HTTP bulkhead middleware that limits concurrent requests
func NewHTTPBulkheadMiddleware(serverConfig server.Config, log *zap.Logger, priority int) Middleware {
	config := serverConfig.Bulkhead

	// Skip if disabled
	if !config.Enabled {
		return Middleware{
			Priority: priority,
			Handler:  nil, // Will be skipped in newEngine
		}
	}

	// Create semaphore for bulkhead
	sem := semaphore.NewWeighted(int64(config.MaxConcurrent))

	log.Info("HTTP bulkhead initialized",
		zap.Int("max-concurrent", config.MaxConcurrent),
		zap.Duration("timeout", config.Timeout),
	)

	return Middleware{
		Priority: priority,
		Handler: func(c *gin.Context) {
			// Allow health checks without bulkhead
			if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
				c.Next()
				return
			}

			// Create timeout context for acquiring bulkhead slot
			ctx, cancel := context.WithTimeout(c.Request.Context(), config.Timeout)
			defer cancel()

			// Try to acquire semaphore slot
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Warn("HTTP bulkhead acquisition failed - rejecting request",
					zap.Duration("timeout", config.Timeout),
					zap.Int("max-concurrent", config.MaxConcurrent),
					zap.Error(err),
				)

				problem := problems.ServiceUnavailable("too many concurrent requests, please try again later")
				problem.Instance = c.Request.URL.Path
				c.Error(errors.New(problem.Detail)).SetMeta(problem)
				c.Abort()
				return
			}
			defer sem.Release(1)

			// Process request
			c.Next()
		},
	}
}

// HTTPBulkheadModule adds HTTP bulkhead middleware to the application
func HTTPBulkheadModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				return NewHTTPBulkheadMiddleware(serverConfig, log, priority)
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
