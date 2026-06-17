package interceptor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// BulkheadModule provides a bulkhead (concurrency limiter) interceptor.
// Recommended priority: 50 (last in chain, closest to handler).
func BulkheadModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Interceptor {
				if !serverConfig.Bulkhead.Enabled {
					return Interceptor{Priority: priority} // nil Handler, will be skipped
				}
				log.Info("Connect bulkhead interceptor initialized",
					zap.Int("max-concurrent", serverConfig.Bulkhead.MaxConcurrent),
					zap.Duration("timeout", serverConfig.Bulkhead.Timeout),
				)
				return Interceptor{
					Priority: priority,
					Handler:  newBulkheadInterceptor(serverConfig.Bulkhead.MaxConcurrent, serverConfig.Bulkhead.Timeout),
				}
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

func newBulkheadInterceptor(maxConcurrent int, timeout time.Duration) connect.Interceptor {
	sem := semaphore.NewWeighted(int64(maxConcurrent))
	return connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				acquireCtx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				if err := sem.Acquire(acquireCtx, 1); err != nil {
					return nil, connect.NewError(connect.CodeUnavailable, errors.New("too many concurrent requests"))
				}
				defer sem.Release(1)

				return next(ctx, req)
			}
		},
	)
}
