package interceptor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LoggerModule provides a request-logging interceptor.
// Recommended priority: 20 (after recovery).
func LoggerModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Interceptor {
				return Interceptor{
					Priority: priority,
					Handler:  connect.UnaryInterceptorFunc(loggerUnaryInterceptor),
				}
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

func loggerUnaryInterceptor(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()

		resp, err := next(ctx, req)

		latency := time.Since(start)

		fields := []zap.Field{
			zap.String("procedure", req.Spec().Procedure),
			zap.Duration("latency", latency),
			zap.String("peer", req.Peer().Addr),
		}

		if err != nil {
			var connectErr *connect.Error
			if errors.As(err, &connectErr) {
				fields = append(fields, zap.String("connect_code", connectErr.Code().String()))
			}
			fields = append(fields, zap.Error(err))
			logger.Get(ctx).Error("RPC error", fields...)
		} else {
			logger.Get(ctx).Debug("Incoming RPC", fields...)
		}

		return resp, err
	}
}
