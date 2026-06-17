package interceptor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/http/server"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// TimeoutModule provides a request-timeout interceptor.
// Recommended priority: 30 (after logger).
func TimeoutModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(serverConfig server.Config, log *zap.Logger) Interceptor {
				if serverConfig.Timeout.RequestTimeout <= 0 {
					return Interceptor{Priority: priority} // nil Handler, will be skipped
				}
				log.Info("Connect timeout interceptor initialized",
					zap.Duration("request-timeout", serverConfig.Timeout.RequestTimeout),
				)
				return Interceptor{
					Priority: priority,
					Handler:  newTimeoutInterceptor(serverConfig.Timeout.RequestTimeout),
				}
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

func newTimeoutInterceptor(timeout time.Duration) connect.Interceptor {
	return connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				ctx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				resp, err := next(ctx, req)

				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return resp, connect.NewError(connect.CodeDeadlineExceeded, errors.New("request timeout"))
				}

				return resp, err
			}
		},
	)
}
