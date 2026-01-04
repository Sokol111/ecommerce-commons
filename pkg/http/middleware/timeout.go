package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrRequestTimeout is returned when request times out.
var ErrRequestTimeout = errors.New("request timeout")

// newTimeoutMiddleware creates a middleware that adds request timeout to all requests.
func newTimeoutMiddleware(timeout time.Duration) middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		ctx, cancel := context.WithTimeout(req.Context, timeout)
		defer cancel()

		req.SetContext(ctx)

		resp, err := next(req)

		// Check if context timed out
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return resp, ErrRequestTimeout
		}

		return resp, err
	}
}

// TimeoutModule adds timeout middleware to the application.
func TimeoutModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Middleware {
				if serverConfig.Timeout.RequestTimeout <= 0 {
					return Middleware{
						Priority: priority,
						Handler:  nil, // Will be skipped
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
			fx.ResultTags(`group:"ogen_mw"`),
		),
	)
}
