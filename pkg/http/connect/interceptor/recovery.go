package interceptor

import (
	"context"
	"errors"
	"runtime/debug"

	"connectrpc.com/connect"
	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RecoveryModule provides a panic-recovery interceptor.
// Recommended priority: 10 (first in chain).
func RecoveryModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Interceptor {
				return Interceptor{
					Priority: priority,
					Handler:  connect.UnaryInterceptorFunc(recoveryUnaryInterceptor),
				}
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

func recoveryUnaryInterceptor(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (_ connect.AnyResponse, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Get(ctx).Error("Panic recovered",
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
					zap.String("procedure", req.Spec().Procedure),
				)
				err = connect.NewError(connect.CodeInternal, errors.New("internal server error"))
			}
		}()
		return next(ctx, req)
	}
}
