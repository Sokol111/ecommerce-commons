package interceptor

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

// RateLimitModule provides a rate-limiting interceptor.
// Recommended priority: 40 (after timeout).
func RateLimitModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(config server.Config) Interceptor {
				if !config.RateLimit.Enabled {
					return Interceptor{Priority: priority} // nil Handler, will be skipped
				}
				limiter := rate.NewLimiter(
					rate.Limit(config.RateLimit.RequestsPerSecond),
					config.RateLimit.Burst,
				)
				return Interceptor{
					Priority: priority,
					Handler:  newRateLimitInterceptor(limiter),
				}
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

func newRateLimitInterceptor(limiter *rate.Limiter) connect.Interceptor {
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
