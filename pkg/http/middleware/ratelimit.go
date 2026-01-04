package middleware

import (
	"errors"

	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

// ErrRateLimitExceeded is returned when rate limit is exceeded.
var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// rateLimiter defines the interface for rate limiting.
type rateLimiter interface {
	Allow() bool
}

// newRateLimitMiddleware creates a rate limiting middleware.
func newRateLimitMiddleware(limiter rateLimiter) middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		if !limiter.Allow() {
			return middleware.Response{}, ErrRateLimitExceeded
		}
		return next(req)
	}
}

// RateLimitModule adds rate limiting middleware to the application.
func RateLimitModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(config server.Config) Middleware {
				if !config.RateLimit.Enabled {
					return Middleware{
						Priority: priority,
						Handler:  nil, // Will be skipped
					}
				}
				limiter := rate.NewLimiter(rate.Limit(config.RateLimit.RequestsPerSecond), config.RateLimit.Burst)
				return Middleware{
					Priority: priority,
					Handler:  newRateLimitMiddleware(limiter),
				}
			},
			fx.ResultTags(`group:"ogen_mw"`),
		),
	)
}
