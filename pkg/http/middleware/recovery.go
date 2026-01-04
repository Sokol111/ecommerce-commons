package middleware

import (
	"errors"
	"runtime/debug"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrPanic is returned when a panic is recovered.
var ErrPanic = errors.New("internal server error")

// recoveryMiddleware handles panics and converts them to errors.
func recoveryMiddleware() middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (resp middleware.Response, err error) {
		defer func() {
			if r := recover(); r != nil {
				fields := append(requestFields(req.Raw),
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
				)
				logger.Get(req.Context).Error("Panic recovered", fields...)
				err = ErrPanic
			}
		}()
		return next(req)
	}
}

// RecoveryModule provides recovery middleware.
func RecoveryModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: priority, Handler: recoveryMiddleware()}
			},
			fx.ResultTags(`group:"ogen_mw"`),
		),
	)
}
