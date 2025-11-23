package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/http/problems"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// newHTTPBulkheadMiddleware creates an HTTP bulkhead middleware that limits concurrent requests
func newHTTPBulkheadMiddleware(maxConcurrent int, timeout time.Duration, log *zap.Logger) gin.HandlerFunc {
	// Create semaphore for bulkhead
	sem := semaphore.NewWeighted(int64(maxConcurrent))

	log.Info("HTTP bulkhead initialized",
		zap.Int("max-concurrent", maxConcurrent),
		zap.Duration("timeout", timeout),
	)

	return func(c *gin.Context) {
		// Allow health checks without bulkhead
		if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
			c.Next()
			return
		}

		// Create timeout context for acquiring bulkhead slot
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Try to acquire semaphore slot
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Warn("HTTP bulkhead acquisition failed - rejecting request",
				zap.Duration("timeout", timeout),
				zap.Int("max-concurrent", maxConcurrent),
				zap.Error(err),
			)

			problem := problems.New(http.StatusServiceUnavailable, "too many concurrent requests, please try again later")
			problem.Instance = c.Request.URL.Path
			c.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("bulkhead acquisition failed: %w", err)).SetMeta(problem)
			return
		}
		defer sem.Release(1)

		// Process request
		c.Next()
	}
}

// HTTPBulkheadModule adds HTTP bulkhead middleware to the application
func HTTPBulkheadModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				// Skip if disabled
				if !serverConfig.Bulkhead.Enabled {
					return Middleware{
						Priority: priority,
						Handler:  nil, // Will be skipped in newEngine
					}
				}
				return Middleware{
					Priority: priority,
					Handler:  newHTTPBulkheadMiddleware(serverConfig.Bulkhead.MaxConcurrent, serverConfig.Bulkhead.Timeout, log),
				}
			},
			fx.ResultTags(`group:"gin_mw"`),
		),
	)
}
