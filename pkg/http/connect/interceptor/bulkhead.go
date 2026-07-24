package interceptor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/semaphore"
)

func NewBulkheadInterceptor(maxConcurrent int, timeout time.Duration) connect.Interceptor {
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
