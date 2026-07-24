package interceptor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/zap"
)

func LoggerUnaryInterceptor(next connect.UnaryFunc) connect.UnaryFunc {
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
