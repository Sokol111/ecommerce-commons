package interceptor

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

func NewRateLimitInterceptor(requestsPerSecond int, burst int) connect.Interceptor {
	limiter := rate.NewLimiter(
		rate.Limit(requestsPerSecond),
		burst,
	)
	return connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				if !limiter.Allow() {
					return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded"))
				}
				return next(ctx, req)
			}
		},
	)
}
