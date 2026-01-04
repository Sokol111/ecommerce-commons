package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// ErrBulkheadFull is returned when bulkhead is full.
var ErrBulkheadFull = errors.New("too many concurrent requests")

// newHTTPBulkheadMiddleware creates an HTTP bulkhead middleware that limits concurrent requests.
func newHTTPBulkheadMiddleware(maxConcurrent int, timeout time.Duration) middleware.Middleware {
	sem := semaphore.NewWeighted(int64(maxConcurrent))

	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		ctx, cancel := context.WithTimeout(req.Context, timeout)
		defer cancel()

		if err := sem.Acquire(ctx, 1); err != nil {
			return middleware.Response{}, ErrBulkheadFull
		}
		defer sem.Release(1)

		return next(req)
	}
}

// HTTPBulkheadModule adds HTTP bulkhead middleware to the application.
func HTTPBulkheadModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				if !serverConfig.Bulkhead.Enabled {
					return Middleware{
						Priority: priority,
						Handler:  nil, // Will be skipped
					}
				}

				log.Info("HTTP bulkhead initialized",
					zap.Int("max-concurrent", serverConfig.Bulkhead.MaxConcurrent),
					zap.Duration("timeout", serverConfig.Bulkhead.Timeout),
				)
				return Middleware{
					Priority: priority,
					Handler:  newHTTPBulkheadMiddleware(serverConfig.Bulkhead.MaxConcurrent, serverConfig.Bulkhead.Timeout),
				}
			},
			fx.ResultTags(`group:"ogen_mw"`),
		),
	)
}
