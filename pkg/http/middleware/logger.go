package middleware

import (
	"time"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// loggerMiddleware logs incoming HTTP requests.
func loggerMiddleware() middleware.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		start := time.Now()

		resp, err := next(req)

		latency := time.Since(start)

		fields := append(requestFields(req.Raw),
			zap.String("operation", req.OperationName),
			zap.Duration("latency", latency),
			zap.String("user_agent", req.Raw.UserAgent()),
		)

		logger.Get(req.Context).Debug("Incoming request", fields...)

		return resp, err
	}
}

// LoggerModule provides logger middleware.
func LoggerModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() Middleware {
				return Middleware{Priority: priority, Handler: loggerMiddleware()}
			},
			fx.ResultTags(`group:"ogen_mw"`),
		),
	)
}
